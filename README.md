# SaaS / Gateway / Board-Agent

三个独立 Go 服务：云端 `saas-service`、本地 `gateway`、板端 `board-agent`。共享仅 `proto/`。

## 前置

- Go 1.22+
- `protoc` + `protoc-gen-go`

## 生成证书

```powershell
.\certs\generate.ps1
# 或: Push-Location scripts\gencert; go run . -out ..\..\certs; Pop-Location
```

产出：`certs/server.crt`、`certs/server.key`、`certs/ca.crt`（gitignore）。

## 生成 Protobuf

```powershell
.\scripts\gen-proto.ps1
```

## 启动（三个终端）

```powershell
# 1) 云端
cd saas-service
go run ./cmd/saas-service -mode development -addr :8443 -cert ../certs/server.crt -key ../certs/server.key

# 2) Gateway
cd gateway
go run ./cmd/gateway -mode development -saas wss://127.0.0.1:8443/ws/gateway -listen :9443 -cert ../certs/server.crt -key ../certs/server.key -insecure-skip-verify

# 3) Board
cd board-agent
go run ./cmd/board-agent -mode development -gateway 127.0.0.1:9443 -insecure-skip-verify -report-port 0
```

浏览器打开：https://127.0.0.1:8443/ （信任自签证书）

```powershell
curl.exe -k https://127.0.0.1:8443/api/devices
curl.exe -k -X POST https://127.0.0.1:8443/api/devices/<device_id>/commands -H "Content-Type: application/json" -d "{\"action\":\"echo\",\"args\":\"hello\"}"
```

设计文档：`docs/superpowers/specs/2026-09-22-saas-gateway-board-design.md`
