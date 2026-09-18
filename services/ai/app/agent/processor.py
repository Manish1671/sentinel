from __future__ import annotations

import logging
import os
import time
from uuid import UUID, uuid4

from app.agent.investigator import investigate
from app.config import Settings
from app.kafka.envelope import (
    TYPE_COMPLETED,
    TYPE_REQUESTED,
    TOPIC_COMPLETED,
    TOPIC_REQUESTED,
    build_envelope,
    completed_event_id,
    parse_envelope,
    requested_event_id,
)
from app.models.contracts import InvestigationRequest, utcnow
from app.storage import investigations as store
from app.observability import metrics as m
from app.observability.otel import tracer
from app.tools.runtime import ToolRuntime

log = logging.getLogger("sentinel.ai")
_ENV = os.getenv("ENVIRONMENT", "development")


class Processor:
    def __init__(self, conn, kafka, settings: Settings):
        self.conn = conn
        self.kafka = kafka
        self.settings = settings

    async def handle_kafka_message(self, msg) -> None:
        try:
            env = parse_envelope(msg.value)
        except Exception as exc:  # poison
            log.error("poison_message error=%s topic=%s offset=%s", exc, msg.topic, msg.offset)
            return
        if env.get("event_type") != TYPE_REQUESTED:
            log.info("ignored_type event_type=%s event_id=%s", env.get("event_type"), env.get("event_id"))
            return
        event_id = UUID(env["event_id"])
        prev = store.get_processed(self.conn, event_id)
        if prev and prev.get("published"):
            log.info("event_duplicate event_id=%s investigation_id=%s", event_id, prev["investigation_id"])
            return
        req = InvestigationRequest.model_validate(env["payload"])
        log.info(
            "investigation_requested",
            extra={"event_id": str(event_id), "investigation_id": str(req.investigation_id), "incident_id": str(req.incident_id)},
        )
        await self.run(req, causation_id=event_id, kafka_event_id=event_id)

    async def run(
        self,
        req: InvestigationRequest,
        causation_id: UUID | None,
        kafka_event_id: UUID | None,
    ) -> dict:
        idem = f"investigation:{req.investigation_id}"
        row = store.upsert_investigation_requested(self.conn, req, idem)
        started = time.perf_counter()
        with tracer().start_as_current_span("sentinel.ai.investigation"):
            if row["status"] in ("completed", "failed"):
                await self._publish_terminal(row, req, causation_id, kafka_event_id)
                return {"id": str(row["id"]), "status": row["status"], "duplicate": True}

            if not store.mark_running(self.conn, req.investigation_id):
                row = store.get_investigation(self.conn, req.investigation_id)
                return {"id": str(req.investigation_id), "status": row["status"] if row else "unknown"}

            m.AI_STARTED.labels(service=m.SERVICE, environment=_ENV, status="running").inc()
            bundle = store.load_incident_bundle(self.conn, req.incident_id) or {"timeline": []}
            tools = ToolRuntime(
                self.conn,
                req.allowed_tools,
                self.settings.tool_timeout_seconds,
                self.settings.max_tool_calls,
            )
            try:
                result, evidence = investigate(req, tools, self.settings, bundle.get("timeline") or [])
                with tracer().start_as_current_span("sentinel.db.persist"):
                    store.persist_success(self.conn, result, evidence, kafka_event_id)
                payload = {
                    "investigation_id": str(result.investigation_id),
                    "incident_id": str(result.incident_id),
                    "status": "completed",
                    "error_message": None,
                    "result": result.model_dump(mode="json"),
                }
                await self._emit_completed(req, payload, causation_id, kafka_event_id, "completed")
                m.AI_COMPLETED.labels(service=m.SERVICE, environment=_ENV, status="completed").inc()
                m.AI_DURATION.labels(service=m.SERVICE, environment=_ENV, status="completed").observe(time.perf_counter() - started)
                log.info(
                    "investigation_complete",
                    extra={"investigation_id": str(result.investigation_id), "incident_id": str(result.incident_id)},
                )
                return {"id": str(result.investigation_id), "status": "completed", "result": result.model_dump(mode="json")}
            except Exception as exc:  # noqa: BLE001
                with tracer().start_as_current_span("sentinel.db.persist"):
                    store.persist_failure(self.conn, req.investigation_id, req.incident_id, str(exc), kafka_event_id)
                payload = {
                    "investigation_id": str(req.investigation_id),
                    "incident_id": str(req.incident_id),
                    "status": "failed",
                    "error_message": str(exc)[:2000],
                    "result": None,
                }
                await self._emit_completed(req, payload, causation_id, kafka_event_id, "failed")
                m.AI_FAILED.labels(service=m.SERVICE, environment=_ENV, status="failed").inc()
                m.AI_DURATION.labels(service=m.SERVICE, environment=_ENV, status="failed").observe(time.perf_counter() - started)
                log.error(
                    "investigation_failed",
                    extra={"investigation_id": str(req.investigation_id), "incident_id": str(req.incident_id)},
                    exc_info=True,
                )
                return {"id": str(req.investigation_id), "status": "failed", "error": str(exc)}

    async def request_for_incident(self, incident_id: UUID) -> dict:
        bundle = store.load_incident_bundle(self.conn, incident_id)
        if not bundle:
            return {"error": "incident_not_found"}
        investigation_id = uuid4()
        corr = uuid4()
        req = store.request_from_bundle(investigation_id, bundle, corr)
        idem = f"investigation:{investigation_id}"
        store.upsert_investigation_requested(self.conn, req, idem)
        event_id = requested_event_id(investigation_id)
        env = build_envelope(
            TYPE_REQUESTED,
            event_id,
            corr,
            None,
            utcnow(),
            req.model_dump(mode="json"),
        )
        await self.kafka.publish(TOPIC_REQUESTED, str(investigation_id), env)
        return {"id": str(investigation_id), "incident_id": str(incident_id), "status": "requested", "event_id": str(event_id)}

    async def _publish_terminal(self, row, req: InvestigationRequest, causation_id, kafka_event_id) -> None:
        status = row["status"]
        result_payload = None
        if status == "completed":
            result_payload = {
                "investigation_id": str(row["id"]),
                "incident_id": str(row["incident_id"]),
                "root_cause_hypothesis": row.get("root_cause_hypothesis") or "",
                "confidence": float(row["confidence"] or 0),
                "reasoning_summary": row.get("reasoning_summary") or "",
                "risk_level": row.get("risk_level") or "medium",
                "evidence_refs": [
                    {
                        "evidence_id": str(ev["id"]),
                        "tool_name": ev["tool_name"],
                        "source_type": ev["source_type"],
                        "summary": ev["summary"],
                    }
                    for ev in store.list_evidence(self.conn, row["id"])
                ],
                "recommendations": [],
                "tool_usage": row.get("tool_usage") or [],
                "model": {"name": row.get("model_name") or "unknown", "version": row.get("model_version") or "unknown"},
            }
        payload = {
            "investigation_id": str(row["id"]),
            "incident_id": str(row["incident_id"]),
            "status": "failed" if status == "failed" else "completed",
            "error_message": row.get("error_message"),
            "result": result_payload,
        }
        await self._emit_completed(req, payload, causation_id, kafka_event_id, payload["status"])

    async def _emit_completed(self, req, payload, causation_id, kafka_event_id, status: str) -> None:
        eid = completed_event_id(req.investigation_id, status)
        env = build_envelope(
            TYPE_COMPLETED,
            eid,
            req.correlation_id,
            causation_id,
            utcnow(),
            payload,
        )
        await self.kafka.publish(TOPIC_COMPLETED, str(req.investigation_id), env)
        if kafka_event_id:
            store.mark_published(self.conn, kafka_event_id)
        else:
            store.record_processed(self.conn, eid, req.investigation_id, status, True)
