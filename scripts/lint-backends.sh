#!/bin/sh
set -eu

if command -v go >/dev/null 2>&1; then
  (cd services/core && go vet ./...)
else
  echo "lint: Go unavailable; skipped go vet"
fi

if command -v uv >/dev/null 2>&1; then
  (cd services/ai && uv run ruff check .)
else
  echo "lint: uv unavailable; skipped Ruff"
fi
