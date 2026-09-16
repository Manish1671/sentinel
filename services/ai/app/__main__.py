from __future__ import annotations

import uvicorn

from app.config import get_settings
from app.main import app


def run() -> None:
    settings = get_settings()
    uvicorn.run(app, host="0.0.0.0", port=settings.port)


if __name__ == "__main__":
    run()
