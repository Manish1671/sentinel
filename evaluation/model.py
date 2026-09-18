"""Structured evaluation records. Missing timestamps stay null; durations are omitted."""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any
from uuid import uuid4


CATEGORIES = (
    "detection_correctness",
    "correlation_correctness",
    "investigation_completion",
    "evidence_grounding",
    "recommendation_validity",
    "approval_enforcement",
    "remediation_correctness",
    "recovery_verification_correctness",
    "idempotency_correctness",
    "failure_handling_correctness",
)

REMEDIATION_ACTIONS = frozenset(
    {"rollback_deployment", "restart_service", "scale_replicas", "scale_service"}
)

REASONING_MARKERS = ("OBSERVED EVIDENCE", "INFERENCE", "RECOMMENDATION")


def utcnow() -> datetime:
    return datetime.now(timezone.utc)


def iso(ts: datetime | None) -> str | None:
    if ts is None:
        return None
    return ts.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")


def parse_iso(raw: str | None) -> datetime | None:
    if not raw:
        return None
    text = raw.replace("Z", "+00:00")
    try:
        return datetime.fromisoformat(text)
    except ValueError:
        return None


def ms_between(start: datetime | None, end: datetime | None) -> int | None:
    if start is None or end is None:
        return None
    delta = end - start
    if delta.total_seconds() < 0:
        return None
    return int(delta.total_seconds() * 1000)


def new_run_id() -> str:
    return datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8]


def empty_stage() -> dict[str, Any]:
    return {"status": "not_run", "detail": None, "passed": None}


def new_record(scenario_id: str, service: str = "payments-api", environment: str = "production") -> dict[str, Any]:
    return {
        "run_id": new_run_id(),
        "scenario_id": scenario_id,
        "started_at": None,
        "completed_at": None,
        "service": service,
        "environment": environment,
        "execution": "not_started",
        "passed": None,
        "failure_reason": None,
        "detection_result": empty_stage(),
        "incident_result": empty_stage(),
        "investigation_result": empty_stage(),
        "recommendation_result": empty_stage(),
        "approval_result": empty_stage(),
        "remediation_result": empty_stage(),
        "verification_result": empty_stage(),
        "timestamps": {
            "scenario_started_at": None,
            "telemetry_emitted_at": None,
            "alert_created_at": None,
            "incident_created_at": None,
            "investigation_started_at": None,
            "investigation_completed_at": None,
            "recommendation_created_at": None,
            "approval_at": None,
            "remediation_started_at": None,
            "remediation_completed_at": None,
            "verification_at": None,
            "incident_resolved_at": None,
        },
        "durations_ms": {},
        "categories": {name: {"passed": None, "detail": None} for name in CATEGORIES},
        "expected": {},
        "actual": {},
        "assertions": [],
    }


def compute_durations(record: dict[str, Any]) -> dict[str, int]:
    ts = {k: parse_iso(v) if isinstance(v, str) else v for k, v in (record.get("timestamps") or {}).items()}
    mapping = {
        "detection_latency_ms": ("telemetry_emitted_at", "alert_created_at"),
        "correlation_latency_ms": ("alert_created_at", "incident_created_at"),
        "investigation_latency_ms": ("investigation_started_at", "investigation_completed_at"),
        "remediation_latency_ms": ("remediation_started_at", "remediation_completed_at"),
        "end_to_end_recovery_ms": ("scenario_started_at", "incident_resolved_at"),
    }
    out: dict[str, int] = {}
    for name, (a, b) in mapping.items():
        value = ms_between(ts.get(a), ts.get(b))
        if value is not None:
            out[name] = value
    record["durations_ms"] = out
    return out


def stage(status: str, passed: bool | None, detail: str | None = None) -> dict[str, Any]:
    return {"status": status, "passed": passed, "detail": detail}
