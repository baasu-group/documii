from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol


@dataclass(frozen=True, slots=True)
class GenerationRequest:
    prompt: str
    model: str | None = None


@dataclass(frozen=True, slots=True)
class EmbeddingRequest:
    texts: tuple[str, ...]
    model: str | None = None


class AIProvider(Protocol):
    @property
    def name(self) -> str: ...

    def is_available(self) -> bool: ...

    def generate(self, request: GenerationRequest) -> str: ...

    def embed(self, request: EmbeddingRequest) -> tuple[tuple[float, ...], ...]: ...
