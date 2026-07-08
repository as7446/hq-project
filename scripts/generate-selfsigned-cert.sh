#!/usr/bin/env sh

### 自签证书脚本
set -eu

OUT_DIR="${1:-deploy/infra/nginx/certs}"
mkdir -p "${OUT_DIR}"

openssl req -x509 -nodes -newkey rsa:4096 -sha256 -days 365 \
  -keyout "${OUT_DIR}/privkey.pem" \
  -out "${OUT_DIR}/fullchain.pem" \
  -subj "/CN=gowellgo.top" \
  -addext "subjectAltName=DNS:jenkins.gowellgo.top,DNS:grafana.gowellgo.top"

echo "Generated ${OUT_DIR}/fullchain.pem and ${OUT_DIR}/privkey.pem"

