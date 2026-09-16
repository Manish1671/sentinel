from __future__ import annotations

import time
from typing import Any

import psycopg

from app.tools import impl
from app.tools.common import no_evidence
from app.tools.schemas import ToolDenied, authorize, validate_args


class ToolRuntime:
    def __init__(self, conn: psycopg.Connection, allowed: list[str], timeout_s: float, max_calls: int):
        self.conn = conn
        self.allowed = allowed
        self.timeout_s = timeout_s
        self.max_calls = max_calls
        self.calls = 0
        self.usage: dict[str, dict[str, Any]] = {}

    def call(self, name: str, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
        if self.calls >= self.max_calls:
            raise ToolDenied("budget_exceeded", "maximum tool calls reached")
        authorize(name, self.allowed)
        cleaned = validate_args(name, args)
        handler = impl.HANDLERS[name]
        start = time.perf_counter()
        status = "ok"
        try:
            result = handler(self.conn, cleaned, ctx)
            if result.get("code") == "no_evidence":
                status = "ok"
            return result
        except ToolDenied:
            raise
        except Exception as exc:  # noqa: BLE001 — tools must not abort the investigation
            status = "error"
            return no_evidence(f"tool error: {exc}")
        finally:
            self.calls += 1
            ms = int((time.perf_counter() - start) * 1000)
            rec = self.usage.setdefault(name, {"tool": name, "call_count": 0, "duration_ms": 0, "status": "ok"})
            rec["call_count"] += 1
            rec["duration_ms"] += ms
            if status != "ok":
                rec["status"] = status

    def usage_list(self) -> list[dict[str, Any]]:
        return list(self.usage.values())
