from __future__ import annotations

import sys
from pathlib import Path
from uuid import uuid4

from app.config import ALLOWED_TOOLS, Settings
from app.evidence.normalize import build_context, normalize_tool_result
from app.models.contracts import confidence_label
from app.models.llm import HeuristicProvider, extract_json, validate_result
from app.tools.schemas import ToolDenied, authorize, validate_args


def _repo_root() -> Path:
    here = Path(__file__).resolve()
    for p in [here.parent, *here.parents]:
        if (p / "evaluation" / "cases" / "payments_api_1180.json").is_file():
            return p
    raise RuntimeError("cannot locate evaluation/cases")


ROOT = _repo_root()
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))
from evaluation.runner import evaluate_result, load_case


def _settings() -> Settings:
    return Settings(
        database_url="postgres://sentinel:sentinel@localhost:5432/sentinel",
        kafka_brokers="localhost:9092",
        ai_api_key="",
        ai_model="sentinel-investigator",
    )


def test_unknown_tool_denied():
    try:
        authorize("shell", list(ALLOWED_TOOLS))
        raise AssertionError("expected deny")
    except ToolDenied as exc:
        assert exc.code == "unknown_tool"


def test_disallowed_tool_denied():
    try:
        authorize("get_recent_logs", ["get_metrics"])
        raise AssertionError("expected deny")
    except ToolDenied as exc:
        assert exc.code == "tool_not_allowed"


def test_validate_args_requires_service():
    try:
        validate_args("get_service_health", {})
        raise AssertionError("expected")
    except ToolDenied:
        pass
    cleaned = validate_args("get_service_health", {"service_id": str(uuid4())})
    assert cleaned["service_id"]


def test_query_truncated():
    cleaned = validate_args("search_runbooks", {"query": "x" * 300})
    assert len(cleaned["query"]) == 256


def test_confidence_bands():
    assert confidence_label(0.9) == "high"
    assert confidence_label(0.5) == "medium"
    assert confidence_label(0.2) == "low"


def test_extract_json_fenced():
    raw = '```json\n{"a": 1}\n```'
    assert extract_json(raw)["a"] == 1


def test_heuristic_payments_context():
    s = _settings()
    p = HeuristicProvider(s)
    ctx = "db_connection_saturation high_latency high_error_rate deployment 1.18.0 payments-rollback"
    inv, inc, svc = uuid4(), uuid4(), uuid4()
    raw = p.complete(ctx, inv, inc, svc)
    assert "database" in raw["root_cause_hypothesis"].lower() or "connection" in raw["root_cause_hypothesis"].lower()
    assert 0 <= raw["confidence"] <= 1
    result = validate_result(raw, inv, inc, [], [], s)
    assert result.recommendations
    assert "OBSERVED EVIDENCE" in result.reasoning_summary
    assert "INFERENCE" in result.reasoning_summary


def test_normalize_no_evidence():
    inv = uuid4()
    rows = normalize_tool_result(inv, "get_recent_logs", {"code": "no_evidence", "message": "none", "available": False})
    assert rows[0]["metadata"]["kind"] == "no_evidence"
    again = normalize_tool_result(inv, "get_recent_logs", {"code": "no_evidence", "message": "none", "available": False})
    assert rows[0]["id"] == again[0]["id"]


def test_context_truncated():
    text = build_context({"incident": {"title": "x"}}, [], [], max_chars=80)
    assert "truncated" in text or len(text) <= 80


def test_evaluation_case_keywords():
    case = load_case(ROOT / "evaluation" / "cases" / "payments_api_1180.json")
    good = {
        "root_cause_hypothesis": "Database connection issue after deploy 1.18.0",
        "confidence": 0.7,
        "recommendations": [{"action_type": "rollback_deployment"}],
        "evidence_refs": [
            {"source_type": "metrics"},
            {"source_type": "logs"},
            {"source_type": "deployment"},
            {"source_type": "runbook"},
        ],
    }
    scored = evaluate_result(case, good)
    assert scored["passed"]
    bad = evaluate_result(case, {"root_cause_hypothesis": "unknown", "confidence": 0.1, "recommendations": []})
    assert not bad["passed"]
