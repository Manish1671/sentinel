from __future__ import annotations

from typing import Any

NO_EVIDENCE = "no_evidence"


def no_evidence(message: str) -> dict[str, Any]:
    return {"code": NO_EVIDENCE, "message": message, "available": False}


def ok(payload: dict[str, Any]) -> dict[str, Any]:
    out = dict(payload)
    out["available"] = True
    return out


def parse_window(args: dict[str, Any], fallback: dict[str, Any] | None = None) -> tuple[str | None, str | None]:
    start = args.get("from") or (fallback or {}).get("from")
    end = args.get("to") or (fallback or {}).get("to")
    return start, end
