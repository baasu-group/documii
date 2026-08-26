#!/bin/sh
set -eu

if command -v go >/dev/null 2>&1; then
  (cd services/core && go test -run '^$' ./...)
else
  echo "typecheck: Go unavailable; skipped compile check"
fi

if command -v uv >/dev/null 2>&1; then
  (cd services/ai && uv run python -m compileall -q src tests)
else
  python3 -m compileall -q services/ai/src services/ai/tests
fi
