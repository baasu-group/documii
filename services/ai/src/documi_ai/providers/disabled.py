from .base import EmbeddingRequest, GenerationRequest


class DisabledProvider:
    @property
    def name(self) -> str:
        return "disabled"

    def is_available(self) -> bool:
        return False

    def generate(self, request: GenerationRequest) -> str:
        del request
        raise RuntimeError("AI is disabled for this Documi deployment")

    def embed(self, request: EmbeddingRequest) -> tuple[tuple[float, ...], ...]:
        del request
        raise RuntimeError("AI is disabled for this Documi deployment")
