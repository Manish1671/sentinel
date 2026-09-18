from __future__ import annotations

import json
import logging
from io import StringIO
from uuid import uuid4

from prometheus_client import REGISTRY, generate_latest

from app.observability.log import JSONLogFormatter, setup_logging
from app.observability import metrics as m
from app.observability.otel import setup_tracing, tracer


def test_json_log_fields():
    setup_logging("info", service="sentinel-ai", environment="test")
    stream = StringIO()
    handler = logging.StreamHandler(stream)
    handler.setFormatter(JSONLogFormatter("sentinel-ai", "test"))
    logger = logging.getLogger("test.json.fields")
    logger.handlers = [handler]
    logger.setLevel(logging.INFO)
    logger.propagate = False
    logger.info("hello", extra={"request_id": "req-1", "incident_id": "inc-1", "investigation_id": "inv-1"})
    row = json.loads(stream.getvalue())
    assert row["service"] == "sentinel-ai"
    assert row["environment"] == "test"
    assert row["message"] == "hello"
    assert row["request_id"] == "req-1"
    assert row["incident_id"] == "inc-1"
    assert row["investigation_id"] == "inv-1"
    assert "level" in row and "timestamp" in row


def test_ai_and_error_metrics():
    env = "test"
    m.AI_STARTED.labels(service=m.SERVICE, environment=env, status="running").inc()
    m.AI_COMPLETED.labels(service=m.SERVICE, environment=env, status="completed").inc()
    m.AI_FAILED.labels(service=m.SERVICE, environment=env, status="failed").inc()
    m.AI_DURATION.labels(service=m.SERVICE, environment=env, status="failed").observe(0.2)
    m.AI_TOOL_CALLS.labels(service=m.SERVICE, environment=env, tool="get_metrics", status="ok").inc()
    m.AI_TOOL_FAILURES.labels(service=m.SERVICE, environment=env, tool="get_metrics", status="error").inc()
    m.AI_MODEL_REQUESTS.labels(service=m.SERVICE, environment=env, provider="heuristic").inc()
    m.KAFKA_FAILURES.labels(service=m.SERVICE, environment=env, operation="consume", topic="investigations.requested").inc()
    body = generate_latest(REGISTRY).decode("utf-8")
    for name in (
        "sentinel_ai_investigations_started",
        "sentinel_ai_investigations_completed",
        "sentinel_ai_investigations_failed",
        "sentinel_ai_tool_calls",
        "sentinel_ai_tool_failures",
        "sentinel_ai_model_requests",
        "sentinel_kafka_processing_failures",
    ):
        assert name in body, body


def test_investigation_span_created():
    setup_tracing("sentinel-ai", "test")
    with tracer().start_as_current_span("sentinel.ai.investigation") as span:
        span.set_attribute("investigation_id", str(uuid4()))
        assert span.is_recording()
        assert span.name == "sentinel.ai.investigation"
