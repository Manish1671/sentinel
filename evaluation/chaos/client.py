from __future__ import annotations

import json
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


class HTTPClient:
    def __init__(self, timeout: float = 15.0) -> None:
        self.timeout = timeout

    def request(
        self,
        method: str,
        url: str,
        body: dict[str, Any] | None = None,
        headers: dict[str, str] | None = None,
    ) -> tuple[int, dict[str, Any] | list | str]:
        raw = None if body is None else json.dumps(body).encode("utf-8")
        hdrs = {"Accept": "application/json"}
        if body is not None:
            hdrs["Content-Type"] = "application/json"
        if headers:
            hdrs.update(headers)
        req = Request(url, data=raw, headers=hdrs, method=method)
        try:
            with urlopen(req, timeout=self.timeout) as resp:
                payload = resp.read().decode("utf-8")
                code = resp.getcode()
        except HTTPError as exc:
            payload = exc.read().decode("utf-8")
            code = exc.code
        except URLError as exc:
            raise RuntimeError(f"{method} {url}: {exc}") from exc
        try:
            parsed: dict[str, Any] | list | str = json.loads(payload) if payload else {}
        except json.JSONDecodeError:
            parsed = payload
        return code, parsed


def load_scenarios(root: Path) -> list[dict[str, Any]]:
    folder = root / "chaos" / "scenarios"
    out = []
    for path in sorted(folder.glob("*.json")):
        out.append(json.loads(path.read_text(encoding="utf-8")))
    return out
