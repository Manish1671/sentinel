from __future__ import annotations

import json
import re
from typing import Any, Protocol
from uuid import UUID

import httpx

from app.config import Settings
from app.models.contracts import (
    ACTION_TYPES,
    EvidenceRef,
    InvestigationResult,
    ModelInfo,
    RecommendationOut,
    ToolUsage,
)
from app.observability import metrics as m
from app.prompts.investigator import SYSTEM_PROMPT, user_prompt


class LLMProvider(Protocol):
    def complete(self, context: str, investigation_id: UUID, incident_id: UUID, service_id: UUID) -> dict[str, Any]:
        ...


class HeuristicProvider:
    """Deterministic investigator used when AI_API_KEY is unset and in tests."""

    def __init__(self, settings: Settings):
        self.settings = settings

    def complete(self, context: str, investigation_id: UUID, incident_id: UUID, service_id: UUID) -> dict[str, Any]:
        m.AI_MODEL_REQUESTS.labels(service=m.SERVICE, environment=self.settings.environment, provider="heuristic").inc()
        lower = context.lower()
        db = "db_connection" in lower or "saturation" in lower or "connection" in lower
        latency = "latency" in lower or "high_latency" in lower
        errors = "error_rate" in lower or "high_error_rate" in lower
        deploy = "1.18.0" in lower or "deployment" in lower
        runbook = "payments-rollback" in lower or "runbook" in lower
        observed = []
        if db:
            observed.append("DB connection saturation alert is present")
        if latency:
            observed.append("latency-related alert/metric is present")
        if errors:
            observed.append("error-rate related alert/metric is present")
        if deploy:
            observed.append("deployment 1.18.0 (or another recent deploy) is recorded")
        if "no evidence" in lower:
            observed.append("one or more tools returned no evidence")

        if db and deploy:
            hypothesis = (
                "Database connection saturation coinciding with deployment 1.18.0 "
                "is the leading hypothesis for payments-api degradation. "
                "The deployment is correlated in time, not proven as root cause."
            )
            confidence = 0.72 if (latency and errors) else 0.58
            risk = "high"
            rec_action = "rollback_deployment"
            rec_title = "Consider rolling back payments-api to last known good version"
            rec_rationale = (
                "Runbook and alerts support considering a rollback if error rate remains elevated after deploy. "
                "This is a recommendation only; no production change is executed."
            )
        elif deploy and (latency or errors):
            hypothesis = (
                "A recent deployment is temporally associated with latency/error symptoms. "
                "Root cause is not uniquely identified from available evidence."
            )
            confidence = 0.5
            risk = "medium"
            rec_action = "run_runbook"
            rec_title = "Follow the service error-rate runbook"
            rec_rationale = "Use published runbook steps; do not treat historical incidents as proof."
        else:
            hypothesis = (
                "Insufficient or mixed evidence for a specific root cause. "
                "Further observation is required before a high-confidence conclusion."
            )
            confidence = 0.28
            risk = "medium"
            rec_action = "page_owner"
            rec_title = "Page the service owner for additional context"
            rec_rationale = "Tool results did not establish a durable causal chain."

        reasoning = (
            "OBSERVED EVIDENCE: "
            + ("; ".join(observed) if observed else "limited tool output")
            + ". INFERENCE / HYPOTHESIS: "
            + hypothesis
            + " RECOMMENDATION: proposed action is not executed by this service."
        )
        if runbook:
            reasoning += " Runbook text influenced the recommendation and must be cited as guidance only."

        return {
            "investigation_id": str(investigation_id),
            "incident_id": str(incident_id),
            "root_cause_hypothesis": hypothesis,
            "confidence": confidence,
            "reasoning_summary": reasoning,
            "risk_level": risk,
            "evidence_refs": [],
            "recommendations": [
                {
                    "action_type": rec_action,
                    "title": rec_title,
                    "rationale": rec_rationale,
                    "target_service_id": str(service_id),
                    "parameters": {"source": "heuristic"} if rec_action != "rollback_deployment" else {"version_hint": "previous"},
                    "confidence": min(confidence, 0.7),
                    "risk_level": risk,
                    "required_approval_role": "approver" if rec_action != "rollback_deployment" else "admin",
                }
            ],
            "tool_usage": [],
            "model": {"name": self.settings.ai_model or "sentinel-investigator", "version": "heuristic"},
        }


class OpenAICompatibleProvider:
    def __init__(self, settings: Settings):
        self.settings = settings

    def complete(self, context: str, investigation_id: UUID, incident_id: UUID, service_id: UUID) -> dict[str, Any]:
        env = self.settings.environment
        m.AI_MODEL_REQUESTS.labels(service=m.SERVICE, environment=env, provider="openai").inc()
        body = {
            "model": self.settings.ai_model,
            "temperature": 0.1,
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {
                    "role": "user",
                    "content": user_prompt(context, str(investigation_id), str(incident_id), str(service_id)),
                },
            ],
        }
        headers = {"Authorization": f"Bearer {self.settings.ai_api_key}", "Content-Type": "application/json"}
        url = self.settings.ai_base_url.rstrip("/") + "/chat/completions"
        try:
            with httpx.Client(timeout=self.settings.llm_timeout_seconds) as client:
                resp = client.post(url, headers=headers, json=body)
                resp.raise_for_status()
                data = resp.json()
        except Exception:
            m.AI_MODEL_FAILURES.labels(service=m.SERVICE, environment=env, provider="openai").inc()
            raise
        usage = data.get("usage") or {}
        prompt_tokens = usage.get("prompt_tokens")
        completion_tokens = usage.get("completion_tokens")
        if isinstance(prompt_tokens, int) and prompt_tokens > 0:
            m.AI_TOKENS.labels(service=m.SERVICE, environment=env, kind="prompt").inc(prompt_tokens)
        if isinstance(completion_tokens, int) and completion_tokens > 0:
            m.AI_TOKENS.labels(service=m.SERVICE, environment=env, kind="completion").inc(completion_tokens)
        content = data["choices"][0]["message"]["content"]
        return extract_json(content)


def extract_json(text: str) -> dict[str, Any]:
    text = text.strip()
    if text.startswith("```"):
        text = re.sub(r"^```(?:json)?", "", text).strip()
        text = re.sub(r"```$", "", text).strip()
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        start = text.find("{")
        end = text.rfind("}")
        if start >= 0 and end > start:
            return json.loads(text[start : end + 1])
        raise


def build_provider(settings: Settings) -> LLMProvider:
    if settings.use_remote_llm:
        return OpenAICompatibleProvider(settings)
    return HeuristicProvider(settings)


def validate_result(
    raw: dict[str, Any],
    investigation_id: UUID,
    incident_id: UUID,
    evidence_rows: list[dict[str, Any]],
    tool_usage: list[dict[str, Any]],
    settings: Settings,
) -> InvestigationResult:
    raw = dict(raw)
    raw["investigation_id"] = str(investigation_id)
    raw["incident_id"] = str(incident_id)
    raw.setdefault("model", {"name": settings.ai_model, "version": settings.ai_model_version})
    raw["tool_usage"] = [
        ToolUsage(
            tool=u["tool"],
            call_count=int(u.get("call_count") or 0),
            duration_ms=int(u.get("duration_ms") or 0),
            status=u.get("status") or "ok",
        ).model_dump()
        for u in tool_usage
    ]
    refs = raw.get("evidence_refs") or []
    if not refs:
        raw["evidence_refs"] = [
            EvidenceRef(
                evidence_id=ev["id"],
                tool_name=ev["tool_name"],
                source_type=ev["source_type"],
                summary=ev["summary"],
            ).model_dump()
            for ev in evidence_rows
            if (ev.get("metadata") or {}).get("kind") != "no_evidence"
        ][:12]
    recs = []
    for rec in raw.get("recommendations") or []:
        action = rec.get("action_type")
        if action not in ACTION_TYPES:
            rec["action_type"] = "other"
        recs.append(rec)
    if not recs:
        recs = [
            {
                "action_type": "page_owner",
                "title": "Review investigation with service owner",
                "rationale": "Model omitted recommendations; defaulting to human review.",
                "parameters": {},
                "confidence": 0.2,
                "risk_level": "low",
                "required_approval_role": "approver",
            }
        ]
    raw["recommendations"] = recs
    conf = float(raw.get("confidence") or 0)
    raw["confidence"] = max(0.0, min(1.0, conf))
    if raw.get("risk_level") not in ("low", "medium", "high"):
        raw["risk_level"] = "medium"
    # Fill evidence_id from stored rows when missing
    by_summary = {ev["summary"]: ev for ev in evidence_rows}
    filled = []
    for ref in raw["evidence_refs"]:
        if not ref.get("evidence_id"):
            match = by_summary.get(ref.get("summary") or "")
            if match:
                ref["evidence_id"] = str(match["id"])
                ref["source_type"] = match["source_type"]
                ref["tool_name"] = match["tool_name"]
        filled.append(ref)
    raw["evidence_refs"] = filled
    return InvestigationResult.model_validate(raw)
