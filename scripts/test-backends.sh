#!/bin/sh
set -eu

if command -v go >/dev/null 2>&1; then
  (cd services/core && go test ./...)
else
  echo "test: Go unavailable; skipped Go tests"
fi

if command -v uv >/dev/null 2>&1; then
  (cd services/ai && uv run pytest)
else
  echo "test: uv unavailable; skipped Python tests"
fi
