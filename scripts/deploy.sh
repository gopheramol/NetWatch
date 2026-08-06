#!/usr/bin/env bash
# Deploys NetWatch to a remote Ubuntu home server over SSH: rsyncs the repo,
# then builds and (re)starts the Docker Compose stack there.
#
# Usage:
#   ./scripts/deploy.sh user@host
#   NETWATCH_DEPLOY_HOST=user@host ./scripts/deploy.sh
#
# Requires on the target host: Docker + Docker Compose v2, and SSH key-based
# access (no interactive password prompts). The remote ~/NetWatch/.env is
# never touched by this script — see "First-time setup" below.
set -euo pipefail

HOST="${1:-${NETWATCH_DEPLOY_HOST:-}}"
REMOTE_DIR="${NETWATCH_DEPLOY_DIR:-NetWatch}"

if [[ -z "$HOST" ]]; then
  echo "usage: $0 user@host   (or set NETWATCH_DEPLOY_HOST)" >&2
  exit 1
fi

cd "$(dirname "$0")/.."

echo "==> Syncing repo to ${HOST}:${REMOTE_DIR}"
rsync -avz --delete \
  --exclude='.git' \
  --exclude='node_modules' \
  --exclude='web/node_modules' \
  --exclude='.next' \
  --exclude='web/.next' \
  --exclude='data' \
  --exclude='*.db' \
  --exclude='*.log' \
  --exclude='.env' \
  --exclude='.env.local' \
  --exclude='web/.env.local' \
  --exclude='bin' \
  ./ "${HOST}:${REMOTE_DIR}/"

echo "==> Checking remote .env"
if ! ssh "$HOST" "test -f ${REMOTE_DIR}/.env"; then
  echo "    No .env found on ${HOST} — copying .env.example as a starting point."
  echo "    Edit ${REMOTE_DIR}/.env on the server (NEXT_PUBLIC_API_BASE_URL, TELEGRAM_*) before relying on it."
  ssh "$HOST" "cp ${REMOTE_DIR}/.env.example ${REMOTE_DIR}/.env && chmod 600 ${REMOTE_DIR}/.env"
fi

echo "==> Building images on ${HOST}"
ssh "$HOST" "cd ${REMOTE_DIR} && docker compose build"

echo "==> Starting stack on ${HOST}"
ssh "$HOST" "cd ${REMOTE_DIR} && docker compose up -d"

echo "==> Verifying health"
ssh "$HOST" "sleep 3 && curl -sf http://localhost:8080/healthz && echo && curl -s -o /dev/null -w 'frontend: %{http_code}\n' http://localhost:3000"

echo "==> Deploy complete."
