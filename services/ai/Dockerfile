# syntax=docker/dockerfile:1.7
FROM ghcr.io/astral-sh/uv:0.12.5-python3.12-bookworm-slim
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy
WORKDIR /app
RUN useradd --create-home --uid 10001 documi
COPY services/ai/pyproject.toml services/ai/uv.lock ./
RUN uv sync --frozen --no-dev
COPY services/ai/src ./src
RUN uv sync --frozen --no-dev
USER documi
ENTRYPOINT ["uv", "run", "--no-sync", "document-worker"]
