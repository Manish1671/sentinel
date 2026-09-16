from __future__ import annotations

import json
import uuid
from datetime import datetime, timedelta, timezone
from typing import Any
from uuid import UUID

import psycopg

from app.config import ALLOWED_TOOLS
from app.models.contracts import InvestigationRequest, InvestigationResult, utcnow
from app.storage.db import as_json


def load_incident_bundle(conn: psycopg.Connection, incident_id: UUID) -> dict[str, Any] | None:
    inc = conn.execute(
        """
        SELECT i.id, i.reference, i.service_id, i.title, i.summary, i.severity::text, i.status::text,
               i.detected_at, s.slug, s.name, s.environment::text, s.health_status::text
        FROM incidents i
        JOIN services s ON s.id = i.service_id
        WHERE i.id = %s
        """,
        (incident_id,),
    ).fetchone()
    if not inc:
        return None
    alerts = conn.execute(
        """
        SELECT a.id, a.detector_id, a.severity::text, a.title, a.summary, a.started_at, a.labels
        FROM incident_alerts ia
        JOIN alerts a ON a.id = ia.alert_id
        WHERE ia.incident_id = %s
        ORDER BY ia.attached_at
        """,
        (incident_id,),
    ).fetchall()
    deps = conn.execute(
        """
        SELECT id, version, git_sha, status::text, started_at, changelog, metadata
        FROM deployments
        WHERE service_id = %s
        ORDER BY started_at DESC
        LIMIT 5
        """,
        (inc["service_id"],),
    ).fetchall()
    timeline = conn.execute(
        """
        SELECT kind::text, occurred_at, payload
        FROM incident_events
        WHERE incident_id = %s
        ORDER BY occurred_at ASC
        LIMIT 50
        """,
        (incident_id,),
    ).fetchall()
    return {"incident": inc, "alerts": alerts, "deployments": deps, "timeline": timeline}


def default_window(detected_at: datetime) -> dict[str, str]:
    if detected_at.tzinfo is None:
        detected_at = detected_at.replace(tzinfo=timezone.utc)
    start = detected_at - timedelta(hours=1)
    end = utcnow()
    return {"from": start.isoformat(), "to": end.isoformat()}


def request_from_bundle(investigation_id: UUID, bundle: dict[str, Any], correlation_id: UUID) -> InvestigationRequest:
    inc = bundle["incident"]
    alerts = []
    for a in bundle["alerts"]:
        alerts.append(
            {
                "id": str(a["id"]),
                "detector_id": a["detector_id"],
                "severity": a["severity"],
                "title": a["title"],
                "summary": a.get("summary") or "",
                "started_at": a["started_at"].isoformat(),
            }
        )
    deps = []
    for d in bundle["deployments"]:
        deps.append(
            {
                "id": str(d["id"]),
                "version": d["version"],
                "git_sha": d.get("git_sha"),
                "status": d["status"],
                "started_at": d["started_at"].isoformat(),
            }
        )
    payload = {
        "investigation_id": str(investigation_id),
        "incident_id": str(inc["id"]),
        "correlation_id": str(correlation_id),
        "service": {
            "id": str(inc["service_id"]),
            "slug": inc["slug"],
            "name": inc["name"],
            "environment": inc["environment"],
            "health_status": inc["health_status"],
        },
        "incident": {
            "reference": inc["reference"],
            "title": inc["title"],
            "summary": inc.get("summary") or "",
            "severity": inc["severity"],
            "status": inc["status"],
            "detected_at": inc["detected_at"].isoformat(),
        },
        "alerts": alerts,
        "recent_deployments": deps,
        "time_window": default_window(inc["detected_at"]),
        "allowed_tools": list(ALLOWED_TOOLS),
    }
    return InvestigationRequest.model_validate(payload)


def get_investigation(conn: psycopg.Connection, investigation_id: UUID) -> dict[str, Any] | None:
    return conn.execute("SELECT * FROM investigations WHERE id = %s", (investigation_id,)).fetchone()


def get_processed(conn: psycopg.Connection, event_id: UUID) -> dict[str, Any] | None:
    return conn.execute(
        "SELECT * FROM ai_processed_events WHERE event_id = %s", (event_id,)
    ).fetchone()


def upsert_investigation_requested(
    conn: psycopg.Connection,
    req: InvestigationRequest,
    idempotency_key: str,
) -> dict[str, Any]:
    existing = conn.execute(
        "SELECT * FROM investigations WHERE id = %s OR idempotency_key = %s",
        (req.investigation_id, idempotency_key),
    ).fetchone()
    if existing:
        return existing
    conn.execute(
        """
        INSERT INTO investigations (
            id, incident_id, status, requested_by_user_id, idempotency_key, requested_at
        ) VALUES (%s, %s, 'requested', %s, %s, now())
        ON CONFLICT (idempotency_key) DO NOTHING
        """,
        (req.investigation_id, req.incident_id, req.requested_by_user_id, idempotency_key),
    )
    row = conn.execute(
        "SELECT * FROM investigations WHERE id = %s OR idempotency_key = %s",
        (req.investigation_id, idempotency_key),
    ).fetchone()
    if not row:
        raise RuntimeError("failed to persist investigation request")
    return row


def mark_running(conn: psycopg.Connection, investigation_id: UUID) -> bool:
    cur = conn.execute(
        """
        UPDATE investigations
        SET status = 'running', started_at = COALESCE(started_at, now())
        WHERE id = %s AND status IN ('requested', 'running')
        RETURNING id
        """,
        (investigation_id,),
    )
    return cur.fetchone() is not None


def persist_success(
    conn: psycopg.Connection,
    result: InvestigationResult,
    evidence_rows: list[dict[str, Any]],
    event_id: UUID | None,
) -> None:
    with conn.transaction():
        conn.execute(
            """
            UPDATE investigations SET
                status = 'completed',
                completed_at = now(),
                model_name = %s,
                model_version = %s,
                root_cause_hypothesis = %s,
                reasoning_summary = %s,
                confidence = %s,
                risk_level = %s::risk_level,
                tool_usage = %s,
                error_message = NULL
            WHERE id = %s
            """,
            (
                result.model.name,
                result.model.version,
                result.root_cause_hypothesis,
                result.reasoning_summary,
                result.confidence,
                result.risk_level,
                as_json([u.model_dump() for u in result.tool_usage]),
                result.investigation_id,
            ),
        )
        for ev in evidence_rows:
            conn.execute(
                """
                INSERT INTO evidence (
                    id, investigation_id, tool_name, source_type, summary, artifact_uri, source_ref, metadata, captured_at
                ) VALUES (%s, %s, %s, %s::evidence_source_type, %s, %s, %s, %s, %s)
                ON CONFLICT (id) DO NOTHING
                """,
                (
                    ev["id"],
                    result.investigation_id,
                    ev["tool_name"],
                    ev["source_type"],
                    ev["summary"],
                    ev.get("artifact_uri"),
                    ev.get("source_ref"),
                    as_json(ev.get("metadata") or {}),
                    ev.get("captured_at") or utcnow(),
                ),
            )
        for rec in result.recommendations:
            rid = uuid.uuid5(uuid.NAMESPACE_OID, f"rec:{result.investigation_id}:{rec.action_type}:{rec.title}")
            conn.execute(
                """
                INSERT INTO recommendations (
                    id, incident_id, investigation_id, action_type, title, rationale,
                    target_service_id, parameters, confidence, risk_level, required_approval_role, status
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s::risk_level, %s::approval_role, 'proposed')
                ON CONFLICT (id) DO NOTHING
                """,
                (
                    rid,
                    result.incident_id,
                    result.investigation_id,
                    rec.action_type,
                    rec.title,
                    rec.rationale,
                    rec.target_service_id or result_service_fallback(conn, result.incident_id),
                    as_json(rec.parameters),
                    rec.confidence,
                    rec.risk_level,
                    rec.required_approval_role,
                ),
            )
        if event_id:
            conn.execute(
                """
                INSERT INTO ai_processed_events (event_id, investigation_id, outcome, published)
                VALUES (%s, %s, 'completed', false)
                ON CONFLICT (event_id) DO UPDATE SET outcome = EXCLUDED.outcome
                """,
                (event_id, result.investigation_id),
            )


def persist_failure(
    conn: psycopg.Connection,
    investigation_id: UUID,
    incident_id: UUID,
    message: str,
    event_id: UUID | None,
) -> None:
    with conn.transaction():
        conn.execute(
            """
            UPDATE investigations
            SET status = 'failed', completed_at = now(), error_message = %s
            WHERE id = %s AND status <> 'completed'
            """,
            (message[:2000], investigation_id),
        )
        if event_id:
            conn.execute(
                """
                INSERT INTO ai_processed_events (event_id, investigation_id, outcome, published)
                VALUES (%s, %s, 'failed', false)
                ON CONFLICT (event_id) DO UPDATE SET outcome = EXCLUDED.outcome
                """,
                (event_id, investigation_id),
            )


def mark_published(conn: psycopg.Connection, event_id: UUID) -> None:
    conn.execute("UPDATE ai_processed_events SET published = true WHERE event_id = %s", (event_id,))


def record_processed(
    conn: psycopg.Connection, event_id: UUID, investigation_id: UUID, outcome: str, published: bool
) -> None:
    conn.execute(
        """
        INSERT INTO ai_processed_events (event_id, investigation_id, outcome, published)
        VALUES (%s, %s, %s, %s)
        ON CONFLICT (event_id) DO UPDATE SET published = EXCLUDED.published, outcome = EXCLUDED.outcome
        """,
        (event_id, investigation_id, outcome, published),
    )


def list_evidence(conn: psycopg.Connection, investigation_id: UUID) -> list[dict[str, Any]]:
    return conn.execute(
        "SELECT * FROM evidence WHERE investigation_id = %s ORDER BY captured_at",
        (investigation_id,),
    ).fetchall()


def result_service_fallback(conn: psycopg.Connection, incident_id: UUID) -> UUID:
    row = conn.execute("SELECT service_id FROM incidents WHERE id = %s", (incident_id,)).fetchone()
    return row["service_id"]


def dump_json(value: Any) -> str:
    return json.dumps(value, default=str)
