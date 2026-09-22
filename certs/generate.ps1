# 生成本地开发自签证书到本目录（优先 Go 工具，无需 openssl）
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $Root "scripts\gencert")
try {
  go run . -out (Join-Path $Root "certs")
} finally {
  Pop-Location
}
