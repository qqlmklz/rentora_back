#!/usr/bin/env sh
set -e
cd "$(dirname "$0")"
# Railway Railpack names the Go binary `out` in the app directory.
if [ -f ./out ]; then
  exec ./out
fi
go build -o /tmp/rentora-server ./cmd/server
exec /tmp/rentora-server
