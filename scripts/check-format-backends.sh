#!/bin/sh
set -eu

if command -v gofmt >/dev/null 2>&1; then
  unformatted=$(find services/core -name '*.go' -type f -exec gofmt -l {} +)
  if [ -n "$unformatted" ]; then
    echo "$unformatted"
    echo "Go files require gofmt" >&2
    exit 1
  fi
else
  echo "format check: Go unavailable; skipped gofmt"
fi

if command -v uv >/dev/null 2>&1; then
  (cd services/ai && uv run ruff format --check .)
else
  echo "format check: uv unavailable; skipped Ruff format"
fi
