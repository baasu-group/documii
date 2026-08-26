from __future__ import annotations

import os
from dataclasses import dataclass


def _boolean(name: str, default: bool = False) -> bool:
    value = os.getenv(name, str(default)).lower()
    if value not in {"true", "false"}:
        raise ValueError(f"{name} must be true or false")
    return value == "true"


@dataclass(frozen=True, slots=True)
class Settings:
    environment: str
    log_level: str
    provider: str
    provider_base_url: str | None
    temporal_address: str
    temporal_namespace: str
    air_gapped: bool

    @classmethod
    def from_environment(cls) -> Settings:
        provider = os.getenv("DOCUMI_AI_PROVIDER", "disabled").strip().lower()
        if not provider:
            raise ValueError("DOCUMI_AI_PROVIDER must not be empty")
        return cls(
            environment=os.getenv("DOCUMI_ENV", "development"),
            log_level=os.getenv("DOCUMI_LOG_LEVEL", "info"),
            provider=provider,
            provider_base_url=os.getenv("DOCUMI_AI_BASE_URL") or None,
            temporal_address=os.getenv("DOCUMI_TEMPORAL_ADDRESS", "localhost:7233"),
            temporal_namespace=os.getenv("DOCUMI_TEMPORAL_NAMESPACE", "default"),
            air_gapped=_boolean("AIR_GAPPED"),
        )

    def public_dict(self) -> dict[str, str | bool | None]:
        return {
            "environment": self.environment,
            "provider": self.provider,
            "provider_base_url": self.provider_base_url,
            "temporal_address": self.temporal_address,
            "temporal_namespace": self.temporal_namespace,
            "air_gapped": self.air_gapped,
        }
