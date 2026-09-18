"""Deterministic checks. No invented quality scores."""

from __future__ import annotations

from typing import Any

from evaluation.model import REMEDIATION_ACTIONS, REASONING_MARKERS


def add_assertion(record: dict[str, Any], name: str, passed: bool, detail: str = "") -> None:
    record.setdefault("assertions", []).append({"name": name, "passed": passed, "detail": detail})
    if not passed:
        if not record.get("failure_reason"):
            record["failure_reason"] = f"{name}: {detail}" if detail else name


def mark_category(record: dict[str, Any], name: str, passed: bool | None, detail: str | None = None) -> None:
    record.setdefault("categories", {}).setdefault(name, {})
    record["categories"][name] = {"passed": passed, "detail": detail}


def detectors_present(alerts: list[dict[str, Any]], expected: list[str]) -> tuple[bool, list[str]]:
    have = {a.get("detector_id") for a in alerts}
    missing = [d for d in expected if d not in have]
    return not missing, missing


def no_duplicate_detector_ids(alerts: list[dict[str, Any]], detector: str) -> bool:
    ids = [a.get("id") for a in alerts if a.get("detector_id") == detector]
    return len(ids) == len(set(ids))


def ai_properties(result: dict[str, Any], evidence: list[dict[str, Any]] | None = None) -> dict[str, Any]:
    evidence = evidence or []
    evidence_ids = {str(e.get("id") or e.get("evidence_id")) for e in evidence if e.get("id") or e.get("evidence_id")}
    refs = result.get("evidence_refs") or []
    cited = []
    uncited = []
    for ref in refs:
        eid = str(ref.get("evidence_id") or "")
        if not eid or eid == "None":
            continue
        if evidence_ids and eid not in evidence_ids:
            uncited.append(eid)
        else:
            cited.append(eid)
    reasoning = result.get("reasoning_summary") or ""
    recs = result.get("recommendations") or []
    actions = [r.get("action_type") for r in recs]
    conf = result.get("confidence")
    conf_ok = isinstance(conf, (int, float)) and 0 <= float(conf) <= 1
    markers_ok = all(m in reasoning for m in REASONING_MARKERS)
    schema_ok = bool(result.get("root_cause_hypothesis")) and recs and conf_ok
    return {
        "schema_ok": schema_ok,
        "confidence_in_range": conf_ok,
        "reasoning_separated": markers_ok,
        "has_recommendation": bool(recs),
        "actions": actions,
        "allowlisted_remediation_action": any(a in REMEDIATION_ACTIONS for a in actions),
        "cited_unknown_evidence": uncited,
        "evidence_grounded": not uncited,
        "passed": schema_ok and conf_ok and markers_ok and bool(recs) and not uncited,
    }
