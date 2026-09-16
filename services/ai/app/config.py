from __future__ import annotations

import os
from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


ALLOWED_TOOLS = (
    "get_recent_logs",
    "get_metrics",
    "get_service_health",
    "get_recent_deployments",
    "get_trace",
    "search_runbooks",
    "search_previous_incidents",
    "get_deployment_diff",
)


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=None, extra="ignore", populate_by_name=True)

    port: int = Field(default=8000, alias="PORT")
    environment: str = Field(default="development", alias="ENVIRONMENT")
    log_level: str = Field(default="info", alias="LOG_LEVEL")
    database_url: str = Field(alias="DATABASE_URL")
    kafka_brokers: str = Field(alias="KAFKA_BROKERS")
    kafka_group: str = Field(default="sentinel-ai-v1", alias="KAFKA_CONSUMER_GROUP")
    kafka_client_id: str = Field(default="sentinel-ai", alias="KAFKA_CLIENT_ID")
    migrations_path: str = Field(default="", alias="MIGRATIONS_PATH")
    ai_api_key: str = Field(default="", alias="AI_API_KEY")
    ai_model: str = Field(default="sentinel-investigator", alias="AI_MODEL")
    ai_base_url: str = Field(default="https://api.openai.com/v1", alias="AI_BASE_URL")
    ai_model_version: str = Field(default="phase6", alias="AI_MODEL_VERSION")
    max_investigation_steps: int = Field(default=8, alias="AI_MAX_STEPS")
    max_tool_calls: int = Field(default=16, alias="AI_MAX_TOOL_CALLS")
    tool_timeout_seconds: float = Field(default=10.0, alias="AI_TOOL_TIMEOUT_SECONDS")
    investigation_timeout_seconds: float = Field(default=90.0, alias="AI_INVESTIGATION_TIMEOUT_SECONDS")
    max_context_chars: int = Field(default=12000, alias="AI_MAX_CONTEXT_CHARS")
    llm_timeout_seconds: float = Field(default=30.0, alias="AI_LLM_TIMEOUT_SECONDS")

    @property
    def brokers(self) -> list[str]:
        return [p.strip() for p in self.kafka_brokers.split(",") if p.strip()]

    @property
    def use_remote_llm(self) -> bool:
        return bool(self.ai_api_key.strip())


@lru_cache
def get_settings() -> Settings:
    return Settings()  # type: ignore[call-arg]


def reset_settings() -> None:
    get_settings.cache_clear()


def default_migrations_path() -> str:
    env = os.getenv("MIGRATIONS_PATH", "").strip()
    if env:
        return env
    here = os.path.abspath(os.path.dirname(__file__))
    candidates = [
        os.path.join(here, "..", "..", "..", "database", "migrations"),
        os.path.join("database", "migrations"),
        "/app/database/migrations",
    ]
    for c in candidates:
        if os.path.isdir(c):
            return os.path.abspath(c)
    return os.path.abspath(candidates[0])
