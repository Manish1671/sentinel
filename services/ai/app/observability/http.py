from __future__ import annotations

import os
import time

from opentelemetry.propagate import extract
from opentelemetry.context import attach, detach
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response

from app.observability import metrics as m
from app.observability.otel import tracer


def _route(path: str) -> str:
    parts = path.strip("/").split("/")
    out = []
    for p in parts:
        if len(p) == 36 and p.count("-") == 4:
            out.append("{id}")
        else:
            out.append(p)
    return "/" + "/".join(out) if out and out != [""] else "/"


class ObservabilityMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        if request.url.path == "/metrics":
            return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)
        carrier = {k: v for k, v in request.headers.items()}
        token = attach(extract(carrier))
        route = _route(request.url.path)
        env = os.getenv("ENVIRONMENT", "development")
        start = time.perf_counter()
        status = "500"
        try:
            with tracer().start_as_current_span("sentinel.http.request") as span:
                span.set_attribute("http.method", request.method)
                span.set_attribute("http.route", route)
                m.HTTP_ACTIVE.labels(service=m.SERVICE, environment=env, method=request.method, route=route).inc()
                try:
                    response = await call_next(request)
                    status = str(response.status_code)
                    return response
                finally:
                    m.HTTP_ACTIVE.labels(service=m.SERVICE, environment=env, method=request.method, route=route).dec()
        finally:
            elapsed = time.perf_counter() - start
            m.HTTP_REQUESTS.labels(service=m.SERVICE, environment=env, method=request.method, route=route, status=status).inc()
            m.HTTP_DURATION.labels(service=m.SERVICE, environment=env, method=request.method, route=route, status=status).observe(elapsed)
            detach(token)
