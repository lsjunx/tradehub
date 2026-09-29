#!/usr/bin/env bash
# 从共享 proto/ 为三个独立 module 生成 internal/pb
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="$(go env GOPATH)/bin:$PATH"
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

mkdir -p board-agent/internal/pb/board/v1
protoc --proto_path=proto \
  --go_out=board-agent --go_opt=module=github.com/tradehub/board-agent \
  --go_opt=Mboard/v1/messages.proto=github.com/tradehub/board-agent/internal/pb/board/v1 \
  board/v1/messages.proto

mkdir -p gateway/internal/pb/board/v1 gateway/internal/pb/cloud/v1
protoc --proto_path=proto \
  --go_out=gateway --go_opt=module=github.com/tradehub/gateway \
  --go_opt=Mboard/v1/messages.proto=github.com/tradehub/gateway/internal/pb/board/v1 \
  board/v1/messages.proto
protoc --proto_path=proto \
  --go_out=gateway --go_opt=module=github.com/tradehub/gateway \
  --go_opt=Mcloud/v1/messages.proto=github.com/tradehub/gateway/internal/pb/cloud/v1 \
  cloud/v1/messages.proto
rm -f gateway/internal/pb/cloud/v1/wire_types.go

mkdir -p saas-service/internal/pb/cloud/v1
protoc --proto_path=proto \
  --go_out=saas-service --go_opt=module=github.com/tradehub/saas-service \
  --go_opt=Mcloud/v1/messages.proto=github.com/tradehub/saas-service/internal/pb/cloud/v1 \
  cloud/v1/messages.proto
rm -f saas-service/internal/pb/cloud/v1/wire_types.go

echo "proto generation complete"
