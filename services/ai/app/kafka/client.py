from __future__ import annotations

import asyncio
import json
from typing import Any

from aiokafka import AIOKafkaConsumer, AIOKafkaProducer

from app.config import Settings
from app.kafka.envelope import TOPIC_COMPLETED, TOPIC_REQUESTED


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
        await self.producer.send_and_wait(topic, json.dumps(value).encode("utf-8"), key=key.encode("utf-8"))

    async def ready(self) -> None:
        if not self.producer:
            raise RuntimeError("kafka producer is not started")
        # metadata fetch
        await self.producer.partitions_for("investigations.requested")

    async def stop(self) -> None:
        if self.consumer:
            await self.consumer.stop()
        if self.producer:
            await self.producer.stop()


async def consume_loop(kafka: KafkaIO, handler) -> None:
    assert kafka.consumer is not None
    async for msg in kafka.consumer:
        while True:
            try:
                await handler(msg)
                await kafka.consumer.commit()
                break
            except asyncio.CancelledError:
                raise
            except Exception:
                await asyncio.sleep(1)
