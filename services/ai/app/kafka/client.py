from __future__ import annotations

import asyncio
import json
import os
from typing import Any

from aiokafka import AIOKafkaConsumer, AIOKafkaProducer
from opentelemetry import context as otel_context
from opentelemetry.propagate import extract, inject

from app.config import Settings
from app.kafka.envelope import TOPIC_COMPLETED, TOPIC_REQUESTED
from app.observability import metrics as m
from app.observability.otel import tracer

_ENV = os.getenv("ENVIRONMENT", "development")


def _headers_to_carrier(headers) -> dict[str, str]:
    carrier: dict[str, str] = {}
    if not headers:
        return carrier
    for key, value in headers:
        if isinstance(key, bytes):
            key = key.decode("utf-8", "replace")
        if isinstance(value, bytes):
            value = value.decode("utf-8", "replace")
        if key and value is not None:
            carrier[str(key)] = str(value)
    return carrier


def _carrier_to_headers(carrier: dict[str, str]) -> list[tuple[str, bytes]]:
    return [(k, v.encode("utf-8")) for k, v in carrier.items()]


class KafkaIO:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.producer: AIOKafkaProducer | None = None
        self.consumer: AIOKafkaConsumer | None = None

    async def start_producer(self) -> None:
        self.producer = AIOKafkaProducer(
            bootstrap_servers=self.settings.brokers,
            client_id=self.settings.kafka_client_id,
            acks="all",
        )
        await self.producer.start()

    async def start_consumer(self) -> None:
        self.consumer = AIOKafkaConsumer(
            TOPIC_REQUESTED,
            bootstrap_servers=self.settings.brokers,
            client_id=self.settings.kafka_client_id,
            group_id=self.settings.kafka_group,
            enable_auto_commit=False,
            auto_offset_reset="earliest",
        )
        await self.consumer.start()

    async def publish(self, topic: str, key: str, value: dict[str, Any]) -> None:
        if not self.producer:
            raise RuntimeError("kafka producer is not started")
        carrier: dict[str, str] = {}
        inject(carrier)
        with tracer().start_as_current_span("sentinel.kafka.publish") as span:
            span.set_attribute("messaging.destination", topic)
            try:
                await self.producer.send_and_wait(
                    topic,
                    json.dumps(value).encode("utf-8"),
                    key=key.encode("utf-8"),
                    headers=_carrier_to_headers(carrier),
                )
                m.KAFKA_PUBLISHED.labels(service=m.SERVICE, environment=_ENV, topic=topic, operation="publish").inc()
            except Exception:
                m.KAFKA_FAILURES.labels(service=m.SERVICE, environment=_ENV, operation="publish", topic=topic).inc()
                raise

    async def ready(self) -> None:
        if not self.producer:
            raise RuntimeError("kafka producer is not started")
        await self.producer.partitions_for("investigations.requested")

    async def stop(self) -> None:
        if self.consumer:
            await self.consumer.stop()
        if self.producer:
            await self.producer.stop()


async def consume_loop(kafka: KafkaIO, handler) -> None:
    assert kafka.consumer is not None
    async for msg in kafka.consumer:
        ctx = extract(_headers_to_carrier(msg.headers))
        token = otel_context.attach(ctx)
        try:
            with tracer().start_as_current_span("sentinel.kafka.consume") as span:
                span.set_attribute("messaging.destination", msg.topic)
                m.KAFKA_CONSUMED.labels(service=m.SERVICE, environment=_ENV, topic=msg.topic, operation="consume").inc()
                while True:
                    try:
                        await handler(msg)
                        await kafka.consumer.commit()
                        break
                    except asyncio.CancelledError:
                        raise
                    except Exception:
                        m.KAFKA_FAILURES.labels(
                            service=m.SERVICE, environment=_ENV, operation="consume", topic=msg.topic
                        ).inc()
                        await asyncio.sleep(1)
        finally:
            otel_context.detach(token)
