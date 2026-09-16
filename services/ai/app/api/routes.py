from __future__ import annotations

from uuid import UUID

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import JSONResponse

from app.storage import investigations as store
from app.storage.db import ping

router = APIRouter()


@router.get("/health")
def health() -> dict:
    return {"status": "ok"}


@router.get("/ready")
def ready(request: Request):
    checks = {"postgres": "ok", "kafka": "ok"}
    ok = True
    try:
        ping(request.app.state.conn)
    except Exception:
        checks["postgres"] = "unavailable"
        ok = False
    try:
        if request.app.state.kafka.producer is None:
            raise RuntimeError("no producer")
    except Exception:
        checks["kafka"] = "unavailable"
        ok = False
    status = "ok" if ok else "unavailable"
    code = 200 if ok else 503
    return JSONResponse({"status": status, "checks": checks}, status_code=code)


@router.post("/api/v1/incidents/{incident_id}/investigations")
async def create_investigation(incident_id: UUID, request: Request):
    proc = request.app.state.processor
    out = await proc.request_for_incident(incident_id)
    if out.get("error") == "incident_not_found":
        raise HTTPException(status_code=404, detail="incident not found")
    return {"data": out}


@router.post("/api/v1/investigations/{investigation_id}/run")
async def run_investigation(investigation_id: UUID, request: Request):
    conn = request.app.state.conn
    row = store.get_investigation(conn, investigation_id)
    if not row:
        raise HTTPException(status_code=404, detail="investigation not found")
    bundle = store.load_incident_bundle(conn, row["incident_id"])
    if not bundle:
        raise HTTPException(status_code=404, detail="incident not found")
    from uuid import uuid4

    req = store.request_from_bundle(investigation_id, bundle, uuid4())
    # keep persisted investigation id
    req.investigation_id = investigation_id
    proc = request.app.state.processor
    out = await proc.run(req, causation_id=None, kafka_event_id=None)
    return {"data": out}


@router.get("/api/v1/investigations/{investigation_id}")
def get_investigation(investigation_id: UUID, request: Request):
    conn = request.app.state.conn
    row = store.get_investigation(conn, investigation_id)
    if not row:
        raise HTTPException(status_code=404, detail="investigation not found")
    evidence = store.list_evidence(conn, investigation_id)
    return {
        "data": {
            "id": str(row["id"]),
            "incident_id": str(row["incident_id"]),
            "status": row["status"],
            "model_name": row.get("model_name"),
            "model_version": row.get("model_version"),
            "root_cause_hypothesis": row.get("root_cause_hypothesis"),
            "reasoning_summary": row.get("reasoning_summary"),
            "confidence": float(row["confidence"]) if row.get("confidence") is not None else None,
            "risk_level": row.get("risk_level"),
            "tool_usage": row.get("tool_usage"),
            "error_message": row.get("error_message"),
            "requested_at": row["requested_at"].isoformat() if row.get("requested_at") else None,
            "started_at": row["started_at"].isoformat() if row.get("started_at") else None,
            "completed_at": row["completed_at"].isoformat() if row.get("completed_at") else None,
            "evidence": [
                {
                    "id": str(e["id"]),
                    "tool_name": e["tool_name"],
                    "source_type": e["source_type"],
                    "summary": e["summary"],
                    "metadata": e.get("metadata"),
                }
                for e in evidence
            ],
        }
    }
