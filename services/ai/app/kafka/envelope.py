from __future__ import annotations

import json
import uuid
from datetime import datetime, timezone
from typing import Any
from uuid import UUID

SOURCE = "services.ai"
EVENT_VERSION = 1
TYPE_REQUESTED = "investigation.requested"
TYPE_COMPLETED = "investigation.completed"
TOPIC_REQUESTED = "investigations.requested"
TOPIC_COMPLETED = "investigations.completed"


def parse_envelope(raw: bytes) -> dict[str, Any]:
    data = json.loads(raw.decode("utf-8"))
    if not data.get("event_id") or not data.get("event_type"):
        raise ValueError("malformed envelope")
    return data


def completed_event_id(investigation_id: UUID, status: str) -> UUID:
    return uuid.uuid5(uuid.NAMESPACE_OID, f"investigation.completed:{investigation_id}:{status}")


def requested_event_id(investigation_id: UUID) -> UUID:
    return uuid.uuid5(uuid.NAMESPACE_OID, f"investigation.requested:{investigation_id}")


def build_envelope(
    event_type: str,
    event_id: UUID,
    correlation_id: UUID,
    causation_id: UUID | None,
    occurred: datetime,
    payload: dict[str, Any],
) -> dict[str, Any]:
    if occurred.tzinfo is None:
        occurred = occurred.replace(tzinfo=timezone.utc)
    return {
        "event_id": str(event_id),
        "event_type": event_type,
        "event_version": EVENT_VERSION,
        "occurred_at": occurred.astimezone(timezone.utc).isoformat(),
        "source": SOURCE,
        "correlation_id": str(correlation_id),
        "causation_id": str(causation_id) if causation_id else None,
        "tenant_id": "default",
        "idempotency_key": f"{event_type}:{event_id}",
        "payload": payload,
    }
