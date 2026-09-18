from __future__ import annotations

import json
import logging
import os
from datetime import datetime, timezone
from typing import Any

from opentelemetry import trace


class JSONLogFormatter(logging.Formatter):
    def __init__(self, service: str, environment: str):
        super().__init__()
        self.service = service
        self.environment = environment

    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "level": record.levelname.lower(),
            "message": record.getMessage(),
            "service": self.service,
            "environment": self.environment,
            "logger": record.name,
        }
        span = trace.get_current_span()
        ctx = span.get_span_context()
        if ctx and ctx.is_valid:
            payload["trace_id"] = format(ctx.trace_id, "032x")
            payload["span_id"] = format(ctx.span_id, "016x")
        for key in (
            "request_id",
            "event_id",
            "incident_id",
            "investigation_id",
            "remediation_id",
        ):
            val = getattr(record, key, None)
            if val:
                payload[key] = str(val)
        if record.exc_info:
            payload["error"] = self.formatException(record.exc_info)
        return json.dumps(payload, default=str)


_configured = False


def setup_logging(level: str = "info", service: str = "sentinel-ai", environment: str = "development") -> None:
    global _configured
    root = logging.getLogger()
    root.handlers.clear()
    handler = logging.StreamHandler()
    handler.setFormatter(JSONLogFormatter(service, environment or os.getenv("ENVIRONMENT", "development")))
    root.addHandler(handler)
    root.setLevel(getattr(logging, level.upper(), logging.INFO))
    _configured = True


def get_logger(name: str) -> logging.Logger:
    if not _configured:
        setup_logging(os.getenv("LOG_LEVEL", "info"))
    return logging.getLogger(name)
