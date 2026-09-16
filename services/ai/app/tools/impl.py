from __future__ import annotations

from typing import Any

import psycopg

from app.retrieval.incidents import search_previous_incidents
from app.retrieval.runbooks import search_runbooks
from app.tools.common import no_evidence, ok, parse_window


def get_recent_logs(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    start, end = parse_window(args, ctx.get("time_window"))
    limit = int(args.get("limit") or 50)
    severity = args.get("severity")
    query = args.get("query")
    sql = """
        SELECT id, occurred_at, payload
        FROM telemetry_events
        WHERE service_id = %s AND kind = 'log' AND occurred_at >= %s AND occurred_at <= %s
    """
    params: list[Any] = [args["service_id"], start, end]
    if severity:
        sql += " AND COALESCE(payload->>'severity','') ILIKE %s"
        params.append(severity)
    if query:
        sql += " AND COALESCE(payload->>'message','') ILIKE %s"
        params.append(f"%{query}%")
    sql += " ORDER BY occurred_at DESC LIMIT %s"
    params.append(limit)
    rows = conn.execute(sql, params).fetchall()
    entries = []
    for row in rows:
        p = row["payload"] or {}
        entries.append(
            {
                "event_id": str(row["id"]),
                "occurred_at": row["occurred_at"].isoformat(),
                "severity": p.get("severity") or "info",
                "message": p.get("message") or "",
                "trace_id": p.get("trace_id"),
            }
        )
    if not entries:
        return no_evidence("no log samples in telemetry_events for this window")
    return ok({"entries": entries})


def get_metrics(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    start, end = parse_window(args, ctx.get("time_window"))
    name = args["name"]
    rows = conn.execute(
        """
        SELECT occurred_at, payload
        FROM telemetry_events
        WHERE service_id = %s AND kind = 'metric'
          AND occurred_at >= %s AND occurred_at <= %s
          AND (
            payload->>'name' ILIKE %s
            OR COALESCE(payload->>'name','') ILIKE %s
          )
        ORDER BY occurred_at ASC
        LIMIT 200
        """,
        (args["service_id"], start, end, name, f"%{name}%"),
    ).fetchall()
    points = []
    unit = None
    used = name
    for row in rows:
        p = row["payload"] or {}
        used = p.get("name") or used
        unit = p.get("unit") or unit
        try:
            v = float(p.get("value"))
        except (TypeError, ValueError):
            continue
        points.append({"t": row["occurred_at"].isoformat(), "v": v})
    if not points:
        return no_evidence(f"no metric samples matching {name} in this window")
    return ok({"name": used, "unit": unit or "", "points": points})


def get_service_health(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    row = conn.execute(
        """
        SELECT id, slug, health_status::text, updated_at
        FROM services WHERE id = %s
        """,
        (args["service_id"],),
    ).fetchone()
    if not row:
        return no_evidence("service not found")
    return ok(
        {
            "service_id": str(row["id"]),
            "health_status": row["health_status"],
            "checked_at": row["updated_at"].isoformat(),
            "checks": {"source": "catalog", "slug": row["slug"]},
        }
    )


def get_recent_deployments(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    limit = int(args.get("limit") or 5)
    rows = conn.execute(
        """
        SELECT id, version, git_sha, status::text, changelog, started_at, completed_at, metadata
        FROM deployments
        WHERE service_id = %s
        ORDER BY started_at DESC
        LIMIT %s
        """,
        (args["service_id"], limit),
    ).fetchall()
    if not rows:
        return no_evidence("no deployments recorded for this service")
    deployments = []
    for row in rows:
        deployments.append(
            {
                "id": str(row["id"]),
                "version": row["version"],
                "git_sha": row["git_sha"],
                "status": row["status"],
                "changelog": row["changelog"],
                "started_at": row["started_at"].isoformat(),
                "completed_at": row["completed_at"].isoformat() if row["completed_at"] else None,
            }
        )
    return ok({"deployments": deployments})


def get_trace(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    start, end = parse_window(ctx.get("time_window") or {}, None)
    trace_id = args.get("trace_id")
    service_id = args.get("service_id") or ctx.get("service_id")
    if trace_id:
        rows = conn.execute(
            """
            SELECT id, occurred_at, payload
            FROM telemetry_events
            WHERE kind = 'trace' AND payload->>'trace_id' = %s
            ORDER BY occurred_at
            LIMIT 50
            """,
            (trace_id,),
        ).fetchall()
    elif service_id:
        sql = """
            SELECT id, occurred_at, payload
            FROM telemetry_events
            WHERE kind = 'trace' AND service_id = %s
        """
        params: list[Any] = [service_id]
        if start and end:
            sql += " AND occurred_at >= %s AND occurred_at <= %s"
            params.extend([start, end])
        if args.get("error_example"):
            sql += " AND COALESCE(payload->>'status','') ILIKE 'error'"
        sql += " ORDER BY occurred_at DESC LIMIT 20"
        rows = conn.execute(sql, params).fetchall()
    else:
        return no_evidence("trace_id or service_id is required")
    if not rows:
        return no_evidence("no trace samples available")
    spans = []
    found = trace_id or ""
    for row in rows:
        p = row["payload"] or {}
        found = p.get("trace_id") or found
        spans.append(
            {
                "event_id": str(row["id"]),
                "occurred_at": row["occurred_at"].isoformat(),
                "name": p.get("name"),
                "status": p.get("status"),
                "duration_ms": p.get("duration_ms"),
                "trace_id": p.get("trace_id"),
                "span_id": p.get("span_id"),
            }
        )
    return ok({"trace_id": found or "unknown", "spans": spans})


def get_deployment_diff(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    current = conn.execute(
        """
        SELECT id, service_id, version, git_sha, changelog, started_at, metadata
        FROM deployments WHERE id = %s
        """,
        (args["deployment_id"],),
    ).fetchone()
    if not current:
        return no_evidence("deployment not found")
    against_id = args.get("against_deployment_id")
    if against_id:
        previous = conn.execute("SELECT * FROM deployments WHERE id = %s", (against_id,)).fetchone()
    else:
        previous = conn.execute(
            """
            SELECT * FROM deployments
            WHERE service_id = %s AND started_at < %s
            ORDER BY started_at DESC
            LIMIT 1
            """,
            (current["service_id"], current["started_at"]),
        ).fetchone()
    from_version = previous["version"] if previous else "unknown"
    summary = (
        f"Deployment {current['version']} started {current['started_at'].isoformat()}."
        f" Previous recorded version: {from_version}."
        " This is catalog metadata, not proof of root cause."
    )
    return ok(
        {
            "deployment_id": str(current["id"]),
            "from_version": from_version,
            "to_version": current["version"],
            "summary": summary,
            "changelog": current.get("changelog") or "",
            "git_sha": current.get("git_sha"),
        }
    )


HANDLERS = {
    "get_recent_logs": get_recent_logs,
    "get_metrics": get_metrics,
    "get_service_health": get_service_health,
    "get_recent_deployments": get_recent_deployments,
    "get_trace": get_trace,
    "search_runbooks": search_runbooks,
    "search_previous_incidents": search_previous_incidents,
    "get_deployment_diff": get_deployment_diff,
}
