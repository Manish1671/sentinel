from __future__ import annotations

from pathlib import Path

from evaluation.chaos.client import load_scenarios

REQUIRED = {
    "id",
    "name",
    "inject",
    "expected",
    "cleanup",
    "pass_fail",
}


def test_scenarios_load():
    root = Path(__file__).resolve().parents[1]
    specs = load_scenarios(root)
    ids = [s["id"] for s in specs]
    assert "CHAOS-001" in ids
    assert "CHAOS-005" in ids
    for spec in specs:
        missing = REQUIRED - spec.keys()
        assert not missing, spec["id"]
        assert spec["inject"]["kind"] in ("telemetry", "telemetry_replay", "none")
