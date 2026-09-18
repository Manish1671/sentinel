"""Keyword checks against an InvestigationResult. Not a published benchmark.

Live chaos scoring is `python -m evaluation.chaos.run`.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any


def load_case(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def evaluate_result(case: dict[str, Any], result: dict[str, Any]) -> dict[str, Any]:
    hypothesis = (result.get("root_cause_hypothesis") or "").lower()
    missing_terms = [t for t in case.get("expected_hypothesis_terms") or [] if t.lower() not in hypothesis]
    refs = result.get("evidence_refs") or []
    sources = {r.get("source_type") for r in refs}
    missing_cats = [c for c in case.get("expected_evidence_categories") or [] if c not in sources]
    return {
        "case_id": case.get("id"),
        "hypothesis_terms_ok": not missing_terms,
        "missing_hypothesis_terms": missing_terms,
        "evidence_categories_ok": not missing_cats,
        "missing_evidence_categories": missing_cats,
        "has_recommendation": bool(result.get("recommendations")),
        "has_confidence": result.get("confidence") is not None,
        "passed": not missing_terms and bool(result.get("recommendations")) and result.get("confidence") is not None,
    }


def main() -> None:
    root = Path(__file__).resolve().parent / "cases"
    print("evaluation harness: provide an InvestigationResult JSON to score a case")
    print("cases:", [p.name for p in root.glob("*.json")])


if __name__ == "__main__":
    main()
