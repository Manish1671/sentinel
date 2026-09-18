#!/usr/bin/env python3
"""Run the Phase 11 evaluation suite (offline tests are separate via pytest)."""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from evaluation.chaos.run import main

if __name__ == "__main__":
    raise SystemExit(main())
