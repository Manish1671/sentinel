from __future__ import annotations

from datetime import datetime, timezone
from typing import Any
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field


def utcnow() -> datetime:
    return datetime.now(timezone.utc)


class ServiceRef(BaseModel):
    model_config = ConfigDict(extra="ignore")
    id: UUID
    slug: str
    name: str | None = None
    environment: str
    health_status: str


class IncidentRef(BaseModel):
    model_config = ConfigDict(extra="ignore")
    reference: str
    title: str
    summary: str = ""
    severity: str
    status: str
    detected_at: datetime


class AlertRef(BaseModel):
    model_config = ConfigDict(extra="ignore")
    id: UUID
    detector_id: str
    severity: str
    title: str
    summary: str = ""
    started_at: datetime


class DeploymentRef(BaseModel):
    model_config = ConfigDict(extra="ignore")
    id: UUID
    version: str
    git_sha: str | None = None
    status: str
    started_at: datetime


class InvestigationRequest(BaseModel):
    model_config = ConfigDict(extra="ignore", populate_by_name=True)
    investigation_id: UUID
    incident_id: UUID
    correlation_id: UUID
    requested_by_user_id: UUID | None = None
    service: ServiceRef
    incident: IncidentRef
    alerts: list[AlertRef] = Field(default_factory=list)
    recent_deployments: list[DeploymentRef] = Field(default_factory=list)
    time_window: dict[str, Any]
    allowed_tools: list[str]


class EvidenceRef(BaseModel):
    evidence_id: UUID | None = None
    tool_name: str
    source_type: str
    summary: str
    artifact_uri: str | None = None
    source_ref: str | None = None


class RecommendationOut(BaseModel):
    action_type: str
    title: str
    rationale: str
    target_service_id: UUID | None = None
    parameters: dict[str, Any] = Field(default_factory=dict)
    confidence: float
    risk_level: str
    required_approval_role: str = "approver"


class ToolUsage(BaseModel):
    tool: str
    call_count: int
    duration_ms: int = 0
    status: str


class ModelInfo(BaseModel):
    name: str
    version: str


class InvestigationResult(BaseModel):
    model_config = ConfigDict(extra="forbid")
    investigation_id: UUID
    incident_id: UUID
    root_cause_hypothesis: str
    confidence: float
    reasoning_summary: str
    risk_level: str
    evidence_refs: list[EvidenceRef]
    recommendations: list[RecommendationOut]
    tool_usage: list[ToolUsage]
    model: ModelInfo


ACTION_TYPES = {
    "rollback_deployment",
    "restart_service",
    "scale_replicas",
    "disable_feature_flag",
    "run_runbook",
    "page_owner",
    "other",
}

SOURCE_TYPES = {
    "logs",
    "metrics",
    "trace",
    "deployment",
    "runbook",
    "previous_incident",
    "health",
}

TOOL_SOURCE = {
    "get_recent_logs": "logs",
    "get_metrics": "metrics",
    "get_service_health": "health",
    "get_recent_deployments": "deployment",
    "get_trace": "trace",
    "search_runbooks": "runbook",
    "search_previous_incidents": "previous_incident",
    "get_deployment_diff": "deployment",
}


def confidence_label(value: float) -> str:
    if value >= 0.75:
        return "high"
    if value >= 0.45:
        return "medium"
    return "low"
