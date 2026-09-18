"""Test-only fault flags. Disabled unless SENTINEL_FAULT_INJECTION=true."""

from __future__ import annotations

import os


def _flag(name: str) -> bool:
    return os.getenv(name, "").strip().lower() == "true"


def injection_enabled() -> bool:
    return _flag("SENTINEL_FAULT_INJECTION")


def fail_investigation() -> bool:
    return injection_enabled() and _flag("AI_FAULT_FAIL_INVESTIGATION")


class InjectedInvestigationFailure(RuntimeError):
    def __init__(self) -> None:
        super().__init__("sentinel fault injection: AI investigation failed (test-only)")
