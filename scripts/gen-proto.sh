#!/usr/bin/env bash
# 从共享 proto/ 为三个独立 module 生成 internal/pb
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="$(go env GOPATH)/bin:$PATH"
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

mkdir -p board-agent/internal/pb/board/v1
protoc --proto_path=proto \
  --go_out=board-agent --go_opt=module=board-agent \
  --go_opt=Mboard/v1/messages.proto=board-agent/internal/pb/board/v1 \
  board/v1/messages.proto

mkdir -p gateway/internal/pb/board/v1 gateway/internal/pb/cloud/v1
protoc --proto_path=proto \
  --go_out=gateway --go_opt=module=gateway \
  --go_opt=Mboard/v1/messages.proto=gateway/internal/pb/board/v1 \
  board/v1/messages.proto
protoc --proto_path=proto \
  --go_out=gateway --go_opt=module=gateway \
  --go_opt=Mcloud/v1/messages.proto=gateway/internal/pb/cloud/v1 \
  cloud/v1/messages.proto
rm -f gateway/internal/pb/cloud/v1/wire_types.go

mkdir -p saas-server/internal/pb/cloud/v1
protoc --proto_path=proto \
  --go_out=saas-server --go_opt=module=saas-server \
  --go_opt=Mcloud/v1/messages.proto=saas-server/internal/pb/cloud/v1 \
  cloud/v1/messages.proto
rm -f saas-server/internal/pb/cloud/v1/wire_types.go

echo "proto generation complete"
