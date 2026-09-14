#!/usr/bin/env python3
"""Send sample telemetry through the Sentinel ingestion API."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from uuid import uuid4

SERVICES = {
    "payments-api": "22222222-2222-4222-8222-222222222221",
    "orders-api": "22222222-2222-4222-8222-222222222226",
    "auth-service": "22222222-2222-4222-8222-222222222224",
}


def now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def post(base: str, path: str, body: dict) -> dict:
    req = urllib.request.Request(
        base.rstrip("/") + path,
        data=json.dumps(body).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        print(exc.read().decode("utf-8"), file=sys.stderr)
        raise


def samples(abnormal: bool) -> list[tuple[str, dict]]:
    ts = now()
    out: list[tuple[str, dict]] = []
    for slug, sid in SERVICES.items():
        latency = 920.0 if abnormal and slug == "payments-api" else 24.0
        errors = 0.11 if abnormal and slug == "payments-api" else 0.002
        out.append(
            (
                "/api/v1/telemetry/metric",
                {
                    "service_id": sid,
                    "service_slug": slug,
                    "environment": "production",
                    "name": f"{slug.replace('-', '_')}.latency_p99",
                    "value": latency,
                    "unit": "ms",
                    "occurred_at": ts,
                    "labels": {"env": "production"},
                },
            )
        )
        out.append(
            (
                "/api/v1/telemetry/log",
                {
                    "service_id": sid,
                    "service_slug": slug,
                    "environment": "production",
                    "severity": "error" if abnormal and slug == "payments-api" else "info",
                    "message": "inventory reservation timeout after 150ms"
                    if abnormal and slug == "payments-api"
                    else "request completed",
                    "occurred_at": ts,
                },
            )
        )
        out.append(
            (
                "/api/v1/telemetry/trace",
                {
                    "service_id": sid,
                    "service_slug": slug,
                    "environment": "production",
                    "trace_id": uuid4().hex[:16],
                    "span_id": uuid4().hex[:8],
                    "name": "http.server",
                    "duration_ms": latency,
                    "status": "error" if abnormal and slug == "payments-api" else "ok",
                    "occurred_at": ts,
                },
            )
        )
        out.append(
            (
                "/api/v1/deployments",
                {
                    "service_id": sid,
                    "service_slug": slug,
                    "environment": "production",
                    "version": "1.18.0" if slug == "payments-api" else "1.0.0",
                    "status": "succeeded",
                    "started_at": ts,
                    "completed_at": ts,
                },
            )
        )
    return out


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://localhost:8090")
    parser.add_argument("--abnormal", action="store_true", help="include a payments-api failure sample")
    args = parser.parse_args()
    for path, body in samples(args.abnormal):
        result = post(args.base_url, path, body)
        event = result["data"]["event"]
        print(f"{event['event_type']} -> {result['data']['topic']} {event['event_id']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
