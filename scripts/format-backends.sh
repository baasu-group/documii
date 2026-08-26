#!/bin/sh
set -eu

if command -v gofmt >/dev/null 2>&1; then
  find services/core -name '*.go' -type f -exec gofmt -w {} +
else
  echo "format: Go unavailable; skipped gofmt"
fi

if command -v uv >/dev/null 2>&1; then
  (cd services/ai && uv run ruff format .)
else
  echo "format: uv unavailable; skipped Ruff format"
fi
