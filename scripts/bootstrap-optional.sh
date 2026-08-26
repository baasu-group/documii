#!/bin/sh
set -eu

if command -v go >/dev/null 2>&1; then
  (cd services/core && go mod download)
else
  echo "bootstrap: Go unavailable; skipped Go dependency download"
fi

if command -v uv >/dev/null 2>&1; then
  (cd services/ai && uv sync --frozen)
else
  echo "bootstrap: uv unavailable; skipped Python environment creation"
fi

echo "bootstrap: complete for installed toolchains"
