# SaaS / Gateway / Board-Agent 系统设计

日期：2026-09-22  
状态：待用户审阅

## 1. 目标

在仓库下用 Go 实现三个独立服务，打通最小闭环：

1. Board 启动后持久化设备 ID，经 Gateway 向云端注册（上报 IP、端口等）
2. Board 经 Gateway 向云端发送心跳，云端可展示在线状态
3. 云端接收浏览器或 REST 指令，经 Gateway 下发到指定 Board，并回收执行结果

## 2. 约束与决策

| 项 | 决策 |
|----|------|
| 语言 | 全部 Go |
| 仓库 | 单仓库；三个独立 Go module；仅共享 `proto/` |
| 云端接入 | HTTPS 简易页面 + REST API + WebSocket |
| TLS | 仓库内自签证书脚本生成 `certs/` |
| 设备身份 | 启动时读本地文件；无则生成 UUID 并持久化，有则复用 |
| 架构模式 | 中心会话路由（云端维护 gateway / device 路由表） |

## 3. 总体架构

```
浏览器 ──HTTPS/WSS──► saas-service（云端）
                          ▲
                          │ WSS（Gateway 主动连云）
                          │
                     gateway（本地）
                          ▲
                          │ TCP + TLS + Protobuf
                          │
              board-agent × N（板子）
```

端到端载荷形态：

```
Board  ──protobuf──►  Gateway  ──JSON──►  Cloud  ──JSON──►  Browser
```

| 目录 | 职责 |
|------|------|
| `saas-server/` | HTTPS Web + REST；WSS 接 Gateway / UI；设备与网关路由；指令下发 |
| `gateway/` | 连云端 WSS；TLS 服务端接 Board；设备会话管理；上下行转发 |
| `board-agent/` | 设备 ID 持久化；连 Gateway；注册与心跳；执行指令并回执 |
| `proto/` | 共享 `.proto` 定义（Board 链路 + Gateway↔Cloud 模型） |
| `certs/` | 自签证书生成脚本与输出 |

## 4. 端口与协议

### 4.1 默认端口（均可配置）

| 服务 | 端口 | 协议 |
|------|------|------|
| saas-service | `8443` | HTTPS + WSS（同端口） |
| gateway → 云端 | 客户端 | `wss://<saas>:8443/ws/gateway` |
| gateway ← Board | `9443` | TCP + TLS，帧内 Protobuf |
| board-agent | 无对外业务监听 | TLS 客户端；REGISTER 中上报本机 IP 与配置端口（可为 `0`） |

### 4.2 Gateway ↔ Board 帧格式

- TLS 之上：`4 字节大端 length` + protobuf 字节
- 顶层消息：`Envelope`，含 `msg_id`、`type`、`payload`（oneof）

### 4.3 消息类型（第一版）

| 方向 | 类型 | 内容 |
|------|------|------|
| Board → Gateway | `REGISTER` | `device_id`, `ip`, `port`（**不含** `gateway_id`） |
| Gateway → Cloud | `REGISTER` | `device_id`, `ip`, `port`, **`gateway_id`（由 Gateway 注入）** |
| Board → Gateway → Cloud | `HEARTBEAT` | `device_id`, `ts`, 简单状态；Gateway 转发时同样附带 `gateway_id` |
| Cloud → Gateway → Board | `COMMAND` | `device_id`, `cmd_id`, `action`, `args` |
| Board → Gateway → Cloud | `COMMAND_RESULT` | `device_id`, `cmd_id`, `ok`, `message` |
| Gateway → Cloud | `GATEWAY_HELLO` | `gateway_id` |
| Gateway → Cloud | `DEVICE_OFFLINE` | `device_id`, `gateway_id`（仅当断开的是当前会话时发送） |

注册路径：

```
Board
  │ REGISTER(device_id, ip, port)
  ▼
Gateway
  │ REGISTER(device_id, ip, port, gateway_id)   ← Gateway 补自己的身份
  ▼
Cloud
```

Cloud 据此信任「该 Board 是从哪个 Gateway 连上来的」，而不是相信 Board 自报的网关身份。

### 4.4 云端 HTTP / WS

- `GET /` — 简易页面：设备列表、发指令、看状态
- `GET /api/devices` — 设备列表（在线、IP、所属 gateway、last_seen）
- `POST /api/devices/{id}/commands` — body：`{ "action": "echo", "args": "hello" }`
- `WSS /ws/gateway` — Gateway 长连接
- `WSS /ws/ui` — 浏览器订阅上下线、心跳刷新、指令结果（**纯 JSON**，不依赖 protobuf）

### 4.5 Proto 与消息边界

- `proto/board/v1/` — **仅** Gateway ↔ Board：`Envelope` 与 payload；线上为 **TLS + length-prefix + 二进制 protobuf**
- `proto/cloud/v1/` — **仅** Gateway ↔ Cloud 的消息模型（Go 侧可生成 struct，再 JSON 序列化到 WSS）

明确分层：

```
UI / Browser
  ↓ JSON（HTTP REST 或 /ws/ui）
Cloud 内部模型（手写 DTO 或复用 cloud struct，前端不依赖 .proto）
  ↓ JSON（/ws/gateway）
Gateway
  ↓ protobuf（board/v1）
Board
```

前端与 `/ws/ui` **不得**直接依赖 protobuf 定义或生成代码；Cloud 对浏览器暴露稳定 JSON 字段即可。

第一版指令 `action` 仅实现 `echo`（原样返回）；`status` / `upgrade` / `log` 仅预留字符串，不实现处理逻辑。

## 5. 组件设计

### 5.1 board-agent

- 启动：读 `data/device_id`；不存在则生成 UUID 写入后继续
- 连接 Gateway TLS → 发 `REGISTER`（仅 `device_id`、本机 IP、配置 port）→ 每 N 秒 `HEARTBEAT`（默认 10s）
- 收 `COMMAND`：仅处理 `echo` → 回 `COMMAND_RESULT`
- 断线：指数退避重连；重连后重新 `REGISTER`

### 5.2 gateway

- 启动：读/生成 `data/gateway_id`（同 Board 持久化策略）
- 连云端 WSS → 发 `GATEWAY_HELLO`
- TLS 监听：维护 `device_id → BoardConnection`，其中：

```go
type BoardConnection struct {
    DeviceID  string
    Conn      net.Conn
    SessionID string // 每次接受连接时生成，用于区分新旧连接
}
```

- 上行：Board protobuf 转为云端 JSON；**由 Gateway 注入 `gateway_id`**，Board 不携带该字段
- 下行：按 `device_id` 转发 `COMMAND`；无连接则向云端回报失败
- Board 断开处理（避免旧连接误删新连接）：
  1. 断开回调拿到当时的 `BoardConnection`（含 `SessionID` / `Conn` 指针）
  2. **仅当** map 中当前条目与该连接为同一会话时，才 `delete` 并通知 Cloud `DEVICE_OFFLINE`
  3. 若 map 中已是更新的连接（重连后的新 Session），则忽略此次断开，不删 map、不报离线

### 5.3 saas-service

- 内存表：
  - `gateways[gateway_id]`：WS 连接
  - `devices[device_id]`：ip、port、online、last_seen、gateway_id
- 处理 Gateway 消息：更新路由与设备状态；向 `/ws/ui` 推送 **JSON** 事件
- REST/页面下发：查 `device → gateway` → 经该 Gateway WS 发 `COMMAND`；用 `cmd_id` 等待执行结果（默认超时 10s）
- 心跳超时：超过 3 个心跳周期未更新则标离线
- 第一版不落库；进程重启后需设备重新注册

## 6. 关键数据流

1. **注册**：Board `REGISTER(device_id, ip, port)` → Gateway 注入 `gateway_id` → Cloud → 写入设备表 → UI 可见  
2. **心跳**：同上路径刷新 `last_seen`（Gateway 转发时附带 `gateway_id`）  
3. **指令**：UI/REST → Cloud → 目标 Gateway → Board → `COMMAND_RESULT` → Cloud → UI  
4. **离线**：仅当前会话断开时 Gateway 发 `DEVICE_OFFLINE` → Cloud 标离线 → UI 更新  

## 7. 错误处理（最小）

| 场景 | 行为 |
|------|------|
| 设备不在线 / 无 Gateway | REST 返回 404 或 409 |
| 指令超时 | 结果标记 `timeout` |
| TLS / WSS 失败 | 打日志并重连，进程不退出 |

## 8. 目录结构

```
test/
├── proto/
│   ├── board/v1/
│   └── cloud/v1/
├── certs/
│   ├── generate.ps1
│   └── generate.sh
├── saas-server/
│   ├── go.mod
│   ├── cmd/saas-server/
│   ├── internal/
│   └── web/
├── gateway/
│   ├── go.mod
│   ├── cmd/gateway/
│   └── internal/
├── board-agent/
│   ├── go.mod
│   ├── cmd/board-agent/
│   └── internal/
├── docs/superpowers/specs/
└── README.md
```

### Module 约定

- 三个服务各自 `go.mod`，不互相 require 业务包
- `board-agent` / `gateway` 从 `proto/board/v1` 生成本地 `internal/pb/`
- `gateway` / `saas-server` 从 `proto/cloud/v1` 生成本地 `internal/pb/`（或等价 Go struct）；`saas-server/web` 只消费 REST/WS JSON
- 配置：命令行 flag 或简单 YAML（`saas_addr`、`listen`、`cert`、`key`、`heartbeat_interval`、`mode`）

## 9. 非目标（第一版不做）

- 真实升级 / 拉日志业务（仅接口占位）
- 数据库、用户登录与鉴权
- Gateway 集群、多租户
- 正式 CA、设备 mTLS 身份
- 生产级可观测（仅必要日志）

## 10. 本地验证

1. 生成自签证书  
2. 生成 protobuf 代码  
3. 启动 saas-service → gateway → 1~N 个 board-agent  
4. 浏览器打开 `https://localhost:8443`（信任自签）查看设备并发 `echo`  
5. 或用 `curl -k` 调用 REST  

成功标准：设备出现在列表、心跳刷新、echo 指令有执行结果返回。

## 11. 实现备注

### 11.1 TLS 与 `insecure_skip_verify`

| 模式 | `insecure_skip_verify` |
|------|------------------------|
| `development` | 允许为 `true`（本地自签联调） |
| `production` | **必须为** `false`；若配置为 true，启动直接失败 |

- 当 `insecure_skip_verify=true` 时，进程启动必须打印：`WARNING: TLS certificate verification is disabled`
- 禁止「默认静默开启」；必须通过显式 `mode=development`（或等价配置）才能关闭校验

### 11.2 其他

- 自签证书：HTTPS 与 Gateway↔Board TLS 可共用一套开发证书，或分别生成 server 证书
- `gateway_id` / `device_id` 持久化路径默认 `./data/`，可配置
- 本机 IP 探测：优先配置覆盖，否则取第一块非 loopback IPv4
