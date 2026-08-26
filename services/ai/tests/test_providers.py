import pytest

from documi_ai.providers import DisabledProvider, GenerationRequest


def test_disabled_provider_fails_closed() -> None:
    provider = DisabledProvider()
    assert provider.is_available() is False
    with pytest.raises(RuntimeError, match="disabled"):
        provider.generate(GenerationRequest(prompt="confidential"))
