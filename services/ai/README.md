# Documi AI and document workers

This one Python project owns document processing and private-AI adapters. Bootstrap intentionally installs no model framework and downloads no weights.

```sh
uv sync --frozen
uv run document-worker --once
uv run ai-worker --once
uv run pytest
uv run ruff check .
```

The long-running entry points currently wait cleanly for shutdown. Temporal polling arrives with the first real ingestion activity so no fake queue behavior becomes a contract.
