from __future__ import annotations

from typing import Any, Callable

from app.config import ALLOWED_TOOLS

TOOL_DESCRIPTIONS: dict[str, str] = {
    "get_recent_logs": "Read-only logs for a service inside a time window. No shell.",
    "get_metrics": "Read-only metrics samples for a named series in a time window.",
    "get_service_health": "Catalog health_status for a service. Not a live probe of production.",
    "get_recent_deployments": "Recent deployment rows for a service from PostgreSQL.",
    "get_trace": "Trace/span samples retained in telemetry_events.",
    "search_runbooks": "Keyword/full-text search over published runbooks.",
    "search_previous_incidents": "Similar resolved/closed incidents. Supporting evidence only.",
    "get_deployment_diff": "Metadata/changelog comparison between deployments. Not a git checkout.",
}

REQUIRED_ARGS: dict[str, tuple[str, ...]] = {
    "get_recent_logs": ("service_id", "from", "to"),
    "get_metrics": ("service_id", "name", "from", "to"),
    "get_service_health": ("service_id",),
    "get_recent_deployments": ("service_id",),
    "get_trace": (),
    "search_runbooks": ("query",),
    "search_previous_incidents": ("query",),
    "get_deployment_diff": ("deployment_id",),
}

UUID_ARGS = {"service_id", "deployment_id", "against_deployment_id"}


class ToolDenied(Exception):
    def __init__(self, code: str, message: str):
        super().__init__(message)
        self.code = code
        self.message = message


def authorize(name: str, allowed: list[str]) -> None:
    if name not in ALLOWED_TOOLS:
        raise ToolDenied("unknown_tool", f"{name} is not an allowlisted tool")
    if name not in allowed:
        raise ToolDenied("tool_not_allowed", f"{name} is not allowed for this investigation")


def validate_args(name: str, args: dict[str, Any]) -> dict[str, Any]:
    if not isinstance(args, dict):
        raise ToolDenied("invalid_args", "tool arguments must be an object")
    for req in REQUIRED_ARGS.get(name, ()):
        if not args.get(req):
            raise ToolDenied("invalid_args", f"{req} is required")
    cleaned = dict(args)
    if "query" in cleaned and isinstance(cleaned["query"], str) and len(cleaned["query"]) > 256:
        cleaned["query"] = cleaned["query"][:256]
    if "limit" in cleaned:
        try:
            lim = int(cleaned["limit"])
        except (TypeError, ValueError) as exc:
            raise ToolDenied("invalid_args", "limit must be an integer") from exc
        cap = 200 if name == "get_recent_logs" else 20
        if name in ("search_runbooks", "search_previous_incidents"):
            cap = 10
        if name == "get_recent_deployments":
            cap = 20
        cleaned["limit"] = max(1, min(lim, cap))
    return cleaned


Handler = Callable[[Any, dict[str, Any], dict[str, Any]], dict[str, Any]]
