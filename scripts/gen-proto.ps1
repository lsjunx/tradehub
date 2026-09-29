# 从共享 proto/ 为三个独立 module 生成 internal/pb
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$env:PATH = "$(go env GOPATH)\bin;$env:PATH"
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

function Ensure-Dir($p) {
  New-Item -ItemType Directory -Force -Path $p | Out-Null
}

# board-agent: board only
Ensure-Dir "board-agent/internal/pb/board/v1"
protoc `
  --proto_path=proto `
  --go_out=board-agent `
  --go_opt=module=board-agent `
  --go_opt=Mboard/v1/messages.proto=board-agent/internal/pb/board/v1 `
  board/v1/messages.proto

# gateway: board + cloud
Ensure-Dir "gateway/internal/pb/board/v1"
Ensure-Dir "gateway/internal/pb/cloud/v1"
protoc `
  --proto_path=proto `
  --go_out=gateway `
  --go_opt=module=gateway `
  --go_opt=Mboard/v1/messages.proto=gateway/internal/pb/board/v1 `
  board/v1/messages.proto
protoc `
  --proto_path=proto `
  --go_out=gateway `
  --go_opt=module=gateway `
  --go_opt=Mcloud/v1/messages.proto=gateway/internal/pb/cloud/v1 `
  cloud/v1/messages.proto
Remove-Item -Force -ErrorAction SilentlyContinue "gateway/internal/pb/cloud/v1/wire_types.go"

# saas-server: cloud only
Ensure-Dir "saas-server/internal/pb/cloud/v1"
protoc `
  --proto_path=proto `
  --go_out=saas-server `
  --go_opt=module=saas-server `
  --go_opt=Mcloud/v1/messages.proto=saas-server/internal/pb/cloud/v1 `
  cloud/v1/messages.proto
Remove-Item -Force -ErrorAction SilentlyContinue "saas-server/internal/pb/cloud/v1/wire_types.go"

Write-Host "proto generation complete"
Get-ChildItem -Recurse -Filter *.go board-agent/internal/pb,gateway/internal/pb,saas-server/internal/pb | ForEach-Object { $_.FullName }
