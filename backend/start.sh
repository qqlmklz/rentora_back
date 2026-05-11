#!/usr/bin/env sh
set -e
cd "$(dirname "$0")"
go build -o /tmp/rentora-server ./cmd/server
exec /tmp/rentora-server
