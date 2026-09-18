from __future__ import annotations

import os

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.propagate import set_global_textmap
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

_tracer: trace.Tracer | None = None


def setup_tracing(service_name: str = "sentinel-ai", environment: str = "development") -> None:
    global _tracer
    resource = Resource.create({"service.name": service_name, "deployment.environment": environment})
    provider = TracerProvider(resource=resource)
    endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "").strip()
    if endpoint and os.getenv("OTEL_SDK_DISABLED") != "true":
        url = endpoint if endpoint.startswith("http") else f"http://{endpoint}"
        exporter = OTLPSpanExporter(endpoint=f"{url.rstrip('/')}/v1/traces")
        provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)
    set_global_textmap(TraceContextTextMapPropagator())
    _tracer = trace.get_tracer("sentinel")


def tracer() -> trace.Tracer:
    global _tracer
    if _tracer is None:
        _tracer = trace.get_tracer("sentinel")
    return _tracer
