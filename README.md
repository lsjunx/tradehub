# SaaS / Gateway / Board-Agent

三个独立 Go 服务：云端 `saas-service`、本地 `gateway`、板端 `board-agent`。共享仅 `proto/`。

## 生成证书

```powershell
.\certs\generate.ps1
# 或: go run ./scripts/gencert -out certs
```

产出：`certs/server.crt`、`certs/server.key`、`certs/ca.crt`（gitignore）。

## 生成 Protobuf

```powershell
.\scripts\gen-proto.ps1
```

（Task 2 落地后可用。）

## 启动（占位）

```powershell
# 1) saas-service  :8443 HTTPS/WSS
# 2) gateway       :9443 TLS ← board；WSS → saas
# 3) board-agent   连 gateway
```

详细步骤见 `docs/superpowers/specs/2026-09-22-saas-gateway-board-design.md`。
