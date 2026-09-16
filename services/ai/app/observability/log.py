from __future__ import annotations

import logging
import os

_configured = False


def setup_logging(level: str = "info") -> None:
    global _configured
    if _configured:
        return
    logging.basicConfig(
        level=getattr(logging, level.upper(), logging.INFO),
        format="%(message)s",
    )
    _configured = True


def get_logger(name: str) -> logging.Logger:
    if not _configured:
        setup_logging(os.getenv("LOG_LEVEL", "info"))
    return logging.getLogger(name)
