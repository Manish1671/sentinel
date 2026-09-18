"""Telemetry injection through services/ingestion only."""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any
from uuid import uuid4

from evaluation.chaos.client import HTTPClient

PAYMENTS_ID = "22222222-2222-4222-8222-222222222221"
PAYMENTS_SLUG = "payments-api"


def _now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def post_ingest(
    client: HTTPClient,
    base: str,
    path: str,
    body: dict[str, Any],
    headers: dict[str, str] | None = None,
) -> tuple[int, dict[str, Any]]:
    code, data = client.request("POST", base.rstrip("/") + path, body, headers)
    if not isinstance(data, dict):
        raise RuntimeError(f"ingest {path} returned {code}: {data}")
    return code, data


def emit_payments(
    client: HTTPClient,
    ingestion_url: str,
    *,
    latency_ms: float,
    error_rate: float,
    db_utilization: float,
    version: str | None,
    log_error: bool,
    event_id: str | None = None,
    idempotency_key: str | None = None,
) -> dict[str, Any]:
    ts = _now()
    sid = PAYMENTS_ID
    slug = PAYMENTS_SLUG
    headers = {}
    if idempotency_key:
        headers["Idempotency-Key"] = idempotency_key
    events: list[dict[str, Any]] = []
    if version:
        body: dict[str, Any] = {
            "service_id": sid,
            "service_slug": slug,
            "environment": "production",
            "version": version,
            "status": "succeeded",
            "started_at": ts,
            "completed_at": ts,
        }
        if event_id:
            body["event_id"] = str(uuid4())
        code, data = post_ingest(client, ingestion_url, "/api/v1/deployments", body, headers or None)
        events.append({"path": "deployments", "code": code, "replayed": data.get("data", {}).get("replayed"), "event": data.get("data", {}).get("event")})
    metric_specs = [
        (f"{slug.replace('-', '_')}.latency_p99", latency_ms, "ms"),
        (f"{slug.replace('-', '_')}.error_rate", error_rate, None),
        ("db_connection_utilization", db_utilization, None),
    ]
    first_event_id = event_id
    for name, value, unit in metric_specs:
        body = {
            "service_id": sid,
            "service_slug": slug,
            "environment": "production",
            "name": name,
            "value": value,
            "occurred_at": ts,
            "labels": {"env": "production", "eval": "phase11"},
        }
        if unit:
            body["unit"] = unit
        if first_event_id:
            body["event_id"] = first_event_id
            first_event_id = None
        code, data = post_ingest(client, ingestion_url, "/api/v1/telemetry/metric", body, headers or None)
        events.append({"path": name, "code": code, "replayed": data.get("data", {}).get("replayed"), "event": data.get("data", {}).get("event")})
    log_body = {
        "service_id": sid,
        "service_slug": slug,
        "environment": "production",
        "severity": "error" if log_error else "info",
        "message": "inventory reservation timeout after 150ms" if log_error else "request completed",
        "occurred_at": ts,
    }
    code, data = post_ingest(client, ingestion_url, "/api/v1/telemetry/log", log_body)
    events.append({"path": "log", "code": code, "replayed": data.get("data", {}).get("replayed"), "event": data.get("data", {}).get("event")})
    return {"emitted_at": ts, "events": events}


def healthy_cleanup(client: HTTPClient, ingestion_url: str) -> None:
    emit_payments(
        client,
        ingestion_url,
        latency_ms=24.0,
        error_rate=0.002,
        db_utilization=0.35,
        version=None,
        log_error=False,
    )
