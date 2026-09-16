from __future__ import annotations

import json
import uuid
from typing import Any
from uuid import UUID

from app.models.contracts import TOOL_SOURCE, utcnow


def evidence_id(investigation_id: UUID, tool: str, summary: str, source_ref: str | None) -> UUID:
    key = f"{investigation_id}:{tool}:{source_ref or ''}:{summary[:160]}"
    return uuid.uuid5(uuid.NAMESPACE_OID, key)


def normalize_tool_result(
    investigation_id: UUID,
    tool: str,
    result: dict[str, Any],
) -> list[dict[str, Any]]:
    source = TOOL_SOURCE.get(tool, "logs")
    if result.get("code") == "no_evidence" or result.get("available") is False:
        summary = result.get("message") or "no evidence available"
        eid = evidence_id(investigation_id, tool, summary, None)
        return [
            {
                "id": eid,
                "tool_name": tool,
                "source_type": source,
                "summary": summary[:500],
                "artifact_uri": None,
                "source_ref": None,
                "metadata": {"observed": False, "inference": False, "kind": "no_evidence"},
                "captured_at": utcnow(),
            }
        ]
    rows: list[dict[str, Any]] = []
    if tool == "get_recent_logs":
        entries = result.get("entries") or []
        n = len(entries)
        msg = entries[0]["message"] if entries else ""
        summary = f"{n} log entries. Example: {msg[:200]}"
        ref = entries[0].get("event_id") if entries else None
        rows.append(_row(investigation_id, tool, source, summary, ref, {"count": n, "observed": True}))
    elif tool == "get_metrics":
        points = result.get("points") or []
        last = points[-1]["v"] if points else None
        summary = f"Metric {result.get('name')} samples={len(points)} last={last}"
        rows.append(_row(investigation_id, tool, source, summary, None, {"name": result.get("name"), "observed": True}))
    elif tool == "get_service_health":
        summary = f"Catalog health_status={result.get('health_status')}"
        rows.append(
            _row(investigation_id, tool, source, summary, result.get("service_id"), {"observed": True})
        )
    elif tool == "get_recent_deployments":
        deps = result.get("deployments") or []
        versions = ", ".join(d.get("version", "") for d in deps[:3])
        summary = f"{len(deps)} recent deployments: {versions}"
        ref = deps[0]["id"] if deps else None
        rows.append(_row(investigation_id, tool, source, summary, ref, {"versions": [d.get("version") for d in deps], "observed": True}))
    elif tool == "get_trace":
        spans = result.get("spans") or []
        summary = f"Trace {result.get('trace_id')} spans={len(spans)}"
        rows.append(_row(investigation_id, tool, source, summary, result.get("trace_id"), {"observed": True}))
    elif tool == "search_runbooks":
        hits = result.get("hits") or []
        top = hits[0] if hits else {}
        summary = f"Runbook {top.get('slug')}: {top.get('title')}" if top else "runbook hits"
        rows.append(
            _row(
                investigation_id,
                tool,
                source,
                summary,
                top.get("runbook_id"),
                {"hits": hits[:5], "observed": True, "guidance_only": True},
            )
        )
    elif tool == "search_previous_incidents":
        hits = result.get("hits") or []
        top = hits[0] if hits else {}
        summary = (
            f"Historical {top.get('reference')}: {top.get('title')} (supporting evidence only)"
            if top
            else "historical hits"
        )
        rows.append(
            _row(
                investigation_id,
                tool,
                source,
                summary,
                top.get("incident_id"),
                {"hits": hits[:5], "observed": True, "does_not_prove_root_cause": True},
            )
        )
    elif tool == "get_deployment_diff":
        summary = result.get("summary") or "deployment metadata"
        rows.append(
            _row(
                investigation_id,
                tool,
                source,
                summary,
                result.get("deployment_id"),
                {
                    "from_version": result.get("from_version"),
                    "to_version": result.get("to_version"),
                    "observed": True,
                    "not_root_cause": True,
                },
            )
        )
    else:
        summary = json.dumps(result, default=str)[:400]
        rows.append(_row(investigation_id, tool, source, summary, None, {"observed": True}))
    return rows


def _row(
    investigation_id: UUID,
    tool: str,
    source: str,
    summary: str,
    source_ref: str | None,
    metadata: dict[str, Any],
) -> dict[str, Any]:
    ref_uuid = None
    if source_ref:
        try:
            ref_uuid = UUID(str(source_ref))
        except ValueError:
            metadata = {**metadata, "source_ref_text": str(source_ref)}
    return {
        "id": evidence_id(investigation_id, tool, summary, source_ref),
        "tool_name": tool,
        "source_type": source,
        "summary": summary[:500],
        "artifact_uri": None,
        "source_ref": ref_uuid,
        "metadata": metadata,
        "captured_at": utcnow(),
    }


def build_context(
    request_dump: dict[str, Any],
    evidence_rows: list[dict[str, Any]],
    timeline: list[dict[str, Any]],
    max_chars: int,
) -> str:
    parts = [
        "INVESTIGATION CONTEXT (bounded; do not assume unseen data exists).",
        json.dumps(
            {
                "incident": request_dump.get("incident"),
                "service": request_dump.get("service"),
                "alerts": request_dump.get("alerts", [])[:8],
                "recent_deployments": request_dump.get("recent_deployments", [])[:5],
                "time_window": request_dump.get("time_window"),
            },
            default=str,
        ),
        "TIMELINE:",
        json.dumps(
            [{"kind": t.get("kind"), "at": str(t.get("occurred_at")), "payload": t.get("payload")} for t in timeline[:20]],
            default=str,
        ),
        "OBSERVED EVIDENCE (tool outputs, not hypotheses):",
    ]
    for ev in evidence_rows:
        if (ev.get("metadata") or {}).get("kind") == "no_evidence":
            parts.append(f"- [{ev['tool_name']}] NO EVIDENCE: {ev['summary']}")
        else:
            parts.append(f"- [{ev['tool_name']}/{ev['source_type']}] {ev['summary']}")
    text = "\n".join(parts)
    if len(text) > max_chars:
        text = text[: max_chars - 20] + "\n[truncated]"
    return text
