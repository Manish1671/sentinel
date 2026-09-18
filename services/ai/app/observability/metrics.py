from __future__ import annotations

from prometheus_client import Counter, Gauge, Histogram

SERVICE = "sentinel-ai"

HTTP_REQUESTS = Counter(
    "sentinel_http_requests",
    "HTTP requests",
    ["service", "environment", "method", "route", "status"],
)
HTTP_DURATION = Histogram(
    "sentinel_http_request_duration_seconds",
    "HTTP request duration",
    ["service", "environment", "method", "route", "status"],
)
HTTP_ACTIVE = Gauge(
    "sentinel_http_requests_active",
    "In-flight HTTP requests",
    ["service", "environment", "method", "route"],
)
KAFKA_PUBLISHED = Counter(
    "sentinel_kafka_messages_published",
    "Kafka publishes",
    ["service", "environment", "topic", "operation"],
)
KAFKA_CONSUMED = Counter(
    "sentinel_kafka_messages_consumed",
    "Kafka consumes",
    ["service", "environment", "topic", "operation"],
)
KAFKA_FAILURES = Counter(
    "sentinel_kafka_processing_failures",
    "Kafka processing failures",
    ["service", "environment", "operation", "topic"],
)
AI_STARTED = Counter(
    "sentinel_ai_investigations_started",
    "Investigations started",
    ["service", "environment", "status"],
)
AI_COMPLETED = Counter(
    "sentinel_ai_investigations_completed",
    "Investigations completed",
    ["service", "environment", "status"],
)
AI_FAILED = Counter(
    "sentinel_ai_investigations_failed",
    "Investigations failed",
    ["service", "environment", "status"],
)
AI_DURATION = Histogram(
    "sentinel_ai_investigation_duration_seconds",
    "Investigation duration",
    ["service", "environment", "status"],
)
AI_TOOL_CALLS = Counter(
    "sentinel_ai_tool_calls",
    "Tool calls",
    ["service", "environment", "tool", "status"],
)
AI_TOOL_FAILURES = Counter(
    "sentinel_ai_tool_failures",
    "Tool failures",
    ["service", "environment", "tool", "status"],
)
AI_MODEL_REQUESTS = Counter(
    "sentinel_ai_model_requests",
    "Model requests",
    ["service", "environment", "provider"],
)
AI_MODEL_FAILURES = Counter(
    "sentinel_ai_model_failures",
    "Model failures",
    ["service", "environment", "provider"],
)
AI_TOKENS = Counter(
    "sentinel_ai_tokens",
    "Model tokens when the provider reports usage",
    ["service", "environment", "kind"],
)
DB_DURATION = Histogram(
    "sentinel_db_operation_duration_seconds",
    "Database operation duration",
    ["service", "environment", "operation"],
)
DB_ERRORS = Counter(
    "sentinel_db_errors",
    "Database errors",
    ["service", "environment", "operation"],
)
