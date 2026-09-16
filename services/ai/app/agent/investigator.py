from __future__ import annotations

from typing import Any

from app.config import Settings
from app.evidence.normalize import build_context, normalize_tool_result
from app.models.contracts import InvestigationRequest, InvestigationResult
from app.models.llm import HeuristicProvider, LLMProvider, build_provider, validate_result
from app.tools.runtime import ToolRuntime

METRIC_NAMES = ("latency", "error_rate", "connection", "db")


def planned_calls(req: InvestigationRequest) -> list[tuple[str, dict[str, Any]]]:
    sid = str(req.service.id)
    window = req.time_window or {}
    start, end = window.get("from"), window.get("to")
    calls: list[tuple[str, dict[str, Any]]] = [
        ("get_service_health", {"service_id": sid}),
        ("get_recent_deployments", {"service_id": sid, "limit": 5}),
    ]
    for name in METRIC_NAMES:
        calls.append(
            ("get_metrics", {"service_id": sid, "name": name, "from": start, "to": end})
        )
    calls.append(
        ("get_recent_logs", {"service_id": sid, "from": start, "to": end, "limit": 50})
    )
    calls.append(("get_trace", {"service_id": sid, "error_example": True}))
    detectors = " ".join(a.detector_id for a in req.alerts) or req.incident.title
    q = f"{req.service.slug} {req.incident.title} {detectors}"
    calls.append(("search_runbooks", {"query": q[:256], "service_id": sid, "limit": 5}))
    calls.append(("search_previous_incidents", {"query": q[:256], "service_id": sid, "limit": 5}))
    if req.recent_deployments:
        calls.append(("get_deployment_diff", {"deployment_id": str(req.recent_deployments[0].id)}))
    return calls


def investigate(
    req: InvestigationRequest,
    tools: ToolRuntime,
    settings: Settings,
    timeline: list[dict[str, Any]],
    provider: LLMProvider | None = None,
) -> tuple[InvestigationResult, list[dict[str, Any]]]:
    ctx = {
        "service_id": str(req.service.id),
        "incident_id": str(req.incident_id),
        "time_window": req.time_window,
    }
    evidence_rows: list[dict[str, Any]] = []
    allowed = set(req.allowed_tools)
    for name, args in planned_calls(req):
        if name not in allowed:
            continue
        if tools.calls >= settings.max_tool_calls:
            break
        result = tools.call(name, args, ctx)
        evidence_rows.extend(normalize_tool_result(req.investigation_id, name, result))
        if tools.calls >= settings.max_investigation_steps + 4:
            break

    context = build_context(req.model_dump(mode="json"), evidence_rows, timeline, settings.max_context_chars)
    llm = provider or build_provider(settings)
    try:
        raw = llm.complete(context, req.investigation_id, req.incident_id, req.service.id)
        result = validate_result(raw, req.investigation_id, req.incident_id, evidence_rows, tools.usage_list(), settings)
    except Exception:
        raw = HeuristicProvider(settings).complete(context, req.investigation_id, req.incident_id, req.service.id)
        result = validate_result(raw, req.investigation_id, req.incident_id, evidence_rows, tools.usage_list(), settings)
    return result, evidence_rows
