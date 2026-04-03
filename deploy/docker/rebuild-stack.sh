#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git -C "$(dirname "$0")/../.." rev-parse --show-toplevel)"
cd "$ROOT/deploy/docker"

export ENX_BUILD_REVISION="${ENX_BUILD_REVISION:-$(git -C "$ROOT" rev-parse HEAD)}"
echo "Building app images with ENX_BUILD_REVISION=${ENX_BUILD_REVISION}"

docker compose build platform-api session-gateway job-runner console-web
docker compose up -d platform-api session-gateway job-runner console-web

echo "Done. Compare /readyz and /healthz revision fields with: git -C \"$ROOT\" rev-parse HEAD"
