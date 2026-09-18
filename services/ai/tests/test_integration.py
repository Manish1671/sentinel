from __future__ import annotations

import os
from uuid import UUID, uuid4

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.agent.investigator import investigate
from app.agent.processor import Processor
from app.api.routes import router
from app.config import Settings
from app.storage.db import connect, migrate, ping
from app.storage.investigations import (
    load_incident_bundle,
    persist_success,
    request_from_bundle,
    upsert_investigation_requested,
)
from app.tools.runtime import ToolRuntime


def _db_url() -> str | None:
    return os.getenv("DATABASE_URL") or None


@pytest.fixture
def conn():
    url = _db_url()
    if not url:
        pytest.skip("DATABASE_URL not set")
    try:
        c = connect(url)
        ping(c)
    except Exception as exc:
        pytest.skip(f"postgres: {exc}")
    mig = os.getenv("MIGRATIONS_PATH")
    if not mig:
        here = __import__("pathlib").Path(__file__).resolve()
        mig = ""
        for p in here.parents:
            candidate = p / "database" / "migrations"
            if candidate.is_dir():
                mig = str(candidate)
                break
        if not mig:
            pytest.skip("migrations path not found")
    try:
        migrate(url, mig)
    except Exception as exc:
        pytest.skip(f"migrate: {exc}")
    yield c
    c.close()


def test_health_endpoint():
    app = FastAPI()
    app.include_router(router)
    client = TestClient(app)
    res = client.get("/health")
    assert res.status_code == 200
    assert res.json()["status"] == "ok"


def test_tool_runtime_no_sql_or_shell():
    from app.tools.schemas import authorize, ToolDenied

    for banned in ("psql", "kubectl", "shell", "http_request"):
        with pytest.raises(ToolDenied):
            authorize(banned, ["get_recent_logs"])


def test_persist_and_investigate_seed_incident(conn):
    settings = Settings(
        database_url=os.environ.get("DATABASE_URL", "postgres://x"),
        kafka_brokers="localhost:9092",
        ai_api_key="",
        ai_model="sentinel-investigator",
    )
    incident_id = __import__("uuid").UUID("33333333-3333-4333-8333-333333333334")
    bundle = load_incident_bundle(conn, incident_id)
    if not bundle:
        pytest.skip("seed incident missing")
    inv_id = uuid4()
    req = request_from_bundle(inv_id, bundle, uuid4())
    row = upsert_investigation_requested(conn, req, f"investigation:{inv_id}")
    assert row["status"] == "requested"
    tools = ToolRuntime(conn, req.allowed_tools, 10, 16)
    result, evidence = investigate(req, tools, settings, bundle.get("timeline") or [])
    persist_success(conn, result, evidence, None)
    again, evidence2 = investigate(req, tools, settings, bundle.get("timeline") or [])
    persist_success(conn, again, evidence2, None)
    count = conn.execute(
        "SELECT count(*) AS n FROM evidence WHERE investigation_id = %s", (inv_id,)
    ).fetchone()["n"]
    first_n = len({e["id"] for e in evidence})
    assert count == first_n
    recs = conn.execute(
        "SELECT count(*) AS n FROM recommendations WHERE investigation_id = %s", (inv_id,)
    ).fetchone()["n"]
    assert recs >= 1
    assert result.confidence <= 1
    assert "OBSERVED" in result.reasoning_summary


class FakeKafka:
    def __init__(self):
        self.messages = []

    async def publish(self, topic, key, value):
        self.messages.append((topic, key, value))


@pytest.mark.asyncio
async def test_processor_idempotent_run(conn):
    settings = Settings(
        database_url=os.environ.get("DATABASE_URL", "postgres://x"),
        kafka_brokers="localhost:9092",
        ai_api_key="",
    )
    incident_id = UUID("33333333-3333-4333-8333-333333333334")
    bundle = load_incident_bundle(conn, incident_id)
    if not bundle:
        pytest.skip("seed incident missing")
    inv_id = uuid4()
    req = request_from_bundle(inv_id, bundle, uuid4())
    kafka = FakeKafka()
    proc = Processor(conn, kafka, settings)
    first = await proc.run(req, causation_id=None, kafka_event_id=None)
    second = await proc.run(req, causation_id=None, kafka_event_id=None)
    assert first["status"] == "completed"
    assert second.get("duplicate") is True
    completed = [m for m in kafka.messages if m[0] == "investigations.completed"]
    assert completed
    ev = conn.execute(
        "SELECT count(*) AS n FROM evidence WHERE investigation_id = %s", (inv_id,)
    ).fetchone()["n"]
    after = conn.execute(
        "SELECT count(*) AS n FROM evidence WHERE investigation_id = %s", (inv_id,)
    ).fetchone()["n"]
    assert ev == after
    recs = conn.execute(
        "SELECT count(*) AS n FROM recommendations WHERE investigation_id = %s", (inv_id,)
    ).fetchone()["n"]
    assert recs >= 1


@pytest.mark.asyncio
async def test_fault_injection_fails_investigation(monkeypatch, conn):
    monkeypatch.setenv("SENTINEL_FAULT_INJECTION", "true")
    monkeypatch.setenv("AI_FAULT_FAIL_INVESTIGATION", "true")
    from app.agent.faults import fail_investigation

    assert fail_investigation() is True
    settings = Settings(
        database_url=os.environ.get("DATABASE_URL", "postgres://x"),
        kafka_brokers="localhost:9092",
        ai_api_key="",
    )
    incident_id = UUID("33333333-3333-4333-8333-333333333334")
    bundle = load_incident_bundle(conn, incident_id)
    if not bundle:
        pytest.skip("seed incident missing")
    inv_id = uuid4()
    req = request_from_bundle(inv_id, bundle, uuid4())
    upsert_investigation_requested(conn, req, f"investigation:{inv_id}")
    proc = Processor(conn, FakeKafka(), settings)
    out = await proc.run(req, causation_id=None, kafka_event_id=None)
    assert out["status"] == "failed"
    assert "fault injection" in (out.get("error") or "")
    row = conn.execute("SELECT status FROM investigations WHERE id = %s", (inv_id,)).fetchone()
    assert row["status"] == "failed"
    recs = conn.execute(
        "SELECT count(*) AS n FROM recommendations WHERE investigation_id = %s", (inv_id,)
    ).fetchone()["n"]
    assert recs == 0
    monkeypatch.delenv("SENTINEL_FAULT_INJECTION", raising=False)
    monkeypatch.delenv("AI_FAULT_FAIL_INVESTIGATION", raising=False)
