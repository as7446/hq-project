#!/usr/bin/env sh
set -eu
### 部署脚本

APP_DIR="${APP_DIR:-/opt/hq-project-demo}"

mkdir -p "${APP_DIR}"
cd "${APP_DIR}"
docker compose pull
docker compose up -d --remove-orphans
docker compose ps

