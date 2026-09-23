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

群控相关改动（`event` / `capability` 等 MsgType）依赖生成代码，改 `proto/` 后须重新执行。

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

## gateway 分层

```
cmd/gateway     → 组装依赖、启动 SaaS 读循环 + Board TLS
handle          → Board TLS Accept / 读帧（I/O 边界）
logic           → 上行 Board→SaaS、下行 SaaS→Board 转发
cloud           → SaaS WSS 客户端（收 command、发 register/heartbeat/result）
session         → Board 会话表（SessionID 防误删）
frame           → Board 二进制帧编解码
identity/config → gateway_id 与启动参数
```

关键入口：
- 收 Board：`handle.BoardServer.serveConn`
- 收 SaaS 指令：`cloud.Client.RunRead` → `logic.Bridge.HandleCloudCommand`
- Gateway **无查询 API**（查询在 SaaS REST）

## saas-service 分层与前端契约

```
cmd/saas-service → 组装启动
router           → Gin 路由
handle           → HTTP / WS 入口
logic            → 设备、异步指令、Gateway 上行
gatewayhub       → Gateway 连接与指令等待
store            → 设备 / 指令内存表
common/response · errors · uievent
```

Gateway↔SaaS 的 `type` 定义在 `proto/cloud/v1` 的 `MsgType` 枚举；线网字符串由生成代码的
`MsgType_name`/`MsgType_value` 推导（`MSG_TYPE_GATEWAY_HELLO` → `gateway_hello`），
业务侧用各服务的 `TypeName`/`ParseType`，勿手写字面量。

## board-agent 分层

```
cmd/board-agent → 组装启动
handle          → TLS 连 Gateway、收发帧
logic           → 指令执行（新增 action 主要改这里）
frame / identity / netinfo / config
```

**REST（真前端）**
- `GET /api/devices` / `GET /api/devices/:id`
- `POST /api/devices/:id/commands` → `data: {cmd_id, device_id, status:"accepted"}`
- `GET /api/commands/:cmd_id` → 查询指令状态
- **群控基础**：`POST/GET /api/egress`（出口池 `{id, proxy_url, region}`）；`POST/GET /api/accounts`（`{id, app, device_id, tier?}`）；`POST /api/accounts/:id/egress`（`{egress_id}` 绑定，在线则推送 `egress.apply`）

统一返回：

```json
{ "code": 0, "message": "ok", "data": {}, "timestamp": 1710000000000 }
```

- 成功：`code = 0`，`message = "ok"`，业务在 `data`
- 失败：`code` 为业务/HTTP 错误码，`data = null`，`timestamp` 为毫秒时间戳

**WS `/ws/ui`**（与 REST 同结构，多 `type` 字段）

```json
{
  "type": "command_result",
  "code": 0,
  "message": "ok",
  "data": { "device_id": "...", "cmd_id": "...", "ok": true, "message": "hello" },
  "timestamp": 1710000000000
}
```

- `device_updated`：`data` 为设备快照
- `command_result`：`data` 为执行结果
- `account_risk`：Board 风险 `event` 转发（如 `account.challenge`）

**Board 指令 action**（Plugin Router）：`echo`（args 字符串）；`egress.apply` / `egress.clear`（args JSON，含 `account_id` 等，与 SaaS 绑定一致）。

`POST .../commands` body 可选 `account_id`：除 `echo` 与 `egress.*` 外须带已注册账号并通过 Policy，否则 `account_required` / 策略拒绝。

**Gateway↔SaaS 线网 type**（`proto/cloud/v1` → `cloudwire`，Gateway 透明转发）：除原有 register/heartbeat/command 外，新增 `capability`（板端能力列表，SaaS 内存存）、`event`（`name` + `payload_json`，如 `egress.unhealthy`、`account.session_dead` → 出口/ tier 侧效应）。

指令为异步：HTTP 只受理，结果靠 WS（或轮询 commands API）。

设计文档：`docs/superpowers/specs/2026-09-22-saas-gateway-board-design.md`
