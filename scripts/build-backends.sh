#!/bin/sh
set -eu

if command -v go >/dev/null 2>&1; then
  (cd services/core && go build ./...)
else
  echo "build: Go unavailable; skipped Go build"
fi

if command -v uv >/dev/null 2>&1; then
  (cd services/ai && uv run document-worker --once >/dev/null && uv run ai-worker --once >/dev/null)
else
  PYTHONPATH=services/ai/src python3 -m documi_ai.workers.document --once >/dev/null
  PYTHONPATH=services/ai/src python3 -m documi_ai.workers.ai --once >/dev/null
fi
