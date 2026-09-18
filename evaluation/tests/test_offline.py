from __future__ import annotations

from evaluation.assertions import ai_properties
from evaluation.model import compute_durations, new_record
from evaluation.runner import evaluate_result, load_case


def test_keyword_eval_still_works():
    case = {
        "id": "x",
        "expected_hypothesis_terms": ["database"],
        "expected_evidence_categories": ["metrics"],
    }
    result = {
        "root_cause_hypothesis": "Database saturation after deploy",
        "confidence": 0.6,
        "recommendations": [{"action_type": "rollback_deployment"}],
        "evidence_refs": [{"source_type": "metrics"}],
    }
    scored = evaluate_result(case, result)
    assert scored["passed"] is True


def test_durations_omit_missing():
    rec = new_record("CHAOS-001")
    rec["timestamps"]["telemetry_emitted_at"] = "2026-09-18T01:00:00Z"
    rec["timestamps"]["alert_created_at"] = "2026-09-18T01:00:02Z"
    d = compute_durations(rec)
    assert d["detection_latency_ms"] == 2000
    assert "end_to_end_recovery_ms" not in d


def test_ai_properties_reject_bad_confidence():
    props = ai_properties({"root_cause_hypothesis": "x", "confidence": 1.5, "recommendations": [{"action_type": "rollback_deployment"}], "reasoning_summary": "OBSERVED EVIDENCE. INFERENCE. RECOMMENDATION.", "evidence_refs": []})
    assert props["confidence_in_range"] is False
    assert props["passed"] is False


def test_ai_properties_unknown_evidence():
    props = ai_properties(
        {
            "root_cause_hypothesis": "x",
            "confidence": 0.5,
            "recommendations": [{"action_type": "rollback_deployment"}],
            "reasoning_summary": "OBSERVED EVIDENCE. INFERENCE / HYPOTHESIS. RECOMMENDATION.",
            "evidence_refs": [{"evidence_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}],
        },
        evidence=[{"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}],
    )
    assert props["evidence_grounded"] is False
