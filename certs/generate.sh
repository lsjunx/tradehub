#!/usr/bin/env bash
# 生成本地开发自签证书到本目录（优先 Go 工具）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/scripts/gencert"
go run . -out "$ROOT/certs"
