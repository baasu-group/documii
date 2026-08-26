from documi_ai.api.health import health
from documi_ai.config import Settings


def test_health_has_only_public_configuration() -> None:
    result = health(
        Settings(
            environment="testing",
            log_level="debug",
            provider="disabled",
            provider_base_url=None,
            temporal_address="temporal:7233",
            temporal_namespace="default",
            air_gapped=True,
        )
    )
    assert result["status"] == "ok"
    assert result["config"]["air_gapped"] is True
    assert "log_level" not in result["config"]
