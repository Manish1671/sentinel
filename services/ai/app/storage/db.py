from __future__ import annotations

import hashlib
import os
import time
from pathlib import Path

import psycopg
from psycopg import ClientCursor
from psycopg.rows import dict_row
from psycopg.types.json import Json

from app.observability import metrics as m


def connect(database_url: str) -> psycopg.Connection:
    return psycopg.connect(database_url, row_factory=dict_row, autocommit=True)


def ping(conn: psycopg.Connection) -> None:
    env = os.getenv("ENVIRONMENT", "development")
    start = time.perf_counter()
    try:
        conn.execute("SELECT 1")
    except Exception:
        m.DB_ERRORS.labels(service=m.SERVICE, environment=env, operation="ping").inc()
        raise
    finally:
        m.DB_DURATION.labels(service=m.SERVICE, environment=env, operation="ping").observe(time.perf_counter() - start)


def migrate(database_url: str, migrations_dir: str) -> None:
    path = Path(migrations_dir)
    if not path.is_dir():
        raise FileNotFoundError(f"migrations path {migrations_dir}")
    with psycopg.connect(database_url, cursor_factory=ClientCursor, autocommit=True) as conn:
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                filename text PRIMARY KEY,
                checksum text NOT NULL,
                applied_at timestamptz NOT NULL DEFAULT now()
            )
            """
        )
        applied = {
            row[0]: row[1]
            for row in conn.execute("SELECT filename, checksum FROM schema_migrations")
        }
        files = sorted(p for p in os.listdir(path) if p.endswith(".sql"))
        for name in files:
            body = (path / name).read_bytes()
            checksum = hashlib.sha256(body).hexdigest()
            if name in applied:
                if applied[name] != checksum:
                    raise RuntimeError(f"migration {name} was already applied with a different checksum")
                continue
            conn.execute("BEGIN")
            try:
                conn.execute(body.decode("utf-8"))
                conn.execute(
                    "INSERT INTO schema_migrations (filename, checksum) VALUES (%s, %s)",
                    (name, checksum),
                )
                conn.execute("COMMIT")
            except Exception:
                conn.execute("ROLLBACK")
                raise


def as_json(value: object) -> Json:
    return Json(value)
