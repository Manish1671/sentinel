from __future__ import annotations

import asyncio
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.agent.processor import Processor
from app.api.routes import router
from app.config import default_migrations_path, get_settings
from app.kafka.client import KafkaIO, consume_loop
from app.observability.log import setup_logging
from app.storage.db import connect, migrate


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    setup_logging(settings.log_level)
    mig = settings.migrations_path or default_migrations_path()
    migrate(settings.database_url, mig)
    conn = connect(settings.database_url)
    kafka = KafkaIO(settings)
    await kafka.start_producer()
    await kafka.start_consumer()
    processor = Processor(conn, kafka, settings)
    app.state.settings = settings
    app.state.conn = conn
    app.state.kafka = kafka
    app.state.processor = processor
    task = asyncio.create_task(consume_loop(kafka, processor.handle_kafka_message))
    try:
        yield
    finally:
        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            pass
        await kafka.stop()
        conn.close()


app = FastAPI(title="sentinel-ai", lifespan=lifespan)
app.include_router(router)
