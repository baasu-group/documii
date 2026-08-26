#!/bin/sh
set -eu

tool=$1
directory=$2
shift 2

if ! command -v "$tool" >/dev/null 2>&1; then
  echo "error: '$tool' is required for this command" >&2
  exit 127
fi

cd "$directory"
exec "$@"
