# SaaS / Gateway / Board-Agent Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现三个独立 Go 服务的最小闭环：Board 经 Gateway 向云端注册与心跳，云端经 Gateway 下发 echo 指令并回收执行结果。

**Architecture:** 中心会话路由。Board↔Gateway 使用 TCP+TLS+length-prefix+protobuf；Gateway↔Cloud 使用 WSS 文本 JSON；Browser↔Cloud 使用 HTTPS REST/页面与 `/ws/ui` JSON。`gateway_id` 由 Gateway 注入；Board 断连用 SessionID 防止旧连接误删。

**Tech Stack:** Go 1.22+、protobuf（`google.golang.org/protobuf`）、`crypto/tls`、`net/http` + `gorilla/websocket` 或 `coder/websocket`、自签证书（PowerShell/`openssl`）。

## Global Constraints

- 语言：全部 Go；单仓库三个独立 `go.mod`，仅共享 `proto/`
- Board `REGISTER` 不含 `gateway_id`；Gateway 转发时注入
- 前端不依赖 protobuf；只消费 JSON
- `insecure_skip_verify` 仅 `mode=development` 允许为 true；为 true 时启动必须打印 `WARNING: TLS certificate verification is disabled`；`production` 下为 true 则启动失败
- 第一版指令仅 `echo`；内存不落库
- 默认端口：saas `8443`，gateway board-listen `9443`
- 工作区当前无 git：首个任务执行 `git init` 后再按步骤 commit

---

## File Structure

```
test/
├── proto/
│   ├── board/v1/messages.proto
│   └── cloud/v1/messages.proto
├── scripts/
│   └── gen-proto.ps1          # 为三服务生成 internal/pb
├── certs/
│   ├── generate.ps1
│   ├── generate.sh
│   ├── ca.crt / server.crt / server.key   # 生成产物，可 gitignore key
├── saas-server/
│   ├── go.mod
│   ├── cmd/saas-server/main.go
│   ├── internal/config/config.go
│   ├── internal/pb/           # 生成自 cloud/v1
│   ├── internal/store/store.go
│   ├── internal/gatewayhub/hub.go
│   ├── internal/api/http.go
│   ├── internal/api/ws_gateway.go
│   ├── internal/api/ws_ui.go
│   └── web/index.html
├── gateway/
│   ├── go.mod
│   ├── cmd/gateway/main.go
│   ├── internal/config/config.go
│   ├── internal/identity/id.go
│   ├── internal/frame/frame.go
│   ├── internal/pb/board/...  # board/v1
│   ├── internal/pb/cloud/...  # cloud/v1
│   ├── internal/boardserver/server.go
│   ├── internal/boardsession/registry.go
│   └── internal/cloudclient/client.go
├── board-agent/
│   ├── go.mod
│   ├── cmd/board-agent/main.go
│   ├── internal/config/config.go
│   ├── internal/identity/id.go
│   ├── internal/frame/frame.go
│   ├── internal/pb/...
│   ├── internal/netinfo/ip.go
│   └── internal/agent/agent.go
├── docs/superpowers/specs/2026-09-22-saas-gateway-board-design.md
├── docs/superpowers/plans/2026-09-22-saas-gateway-board-plan.md
└── README.md
```

---

### Task 1: 仓库脚手架、证书脚本、README 骨架

**Files:**
- Create: `README.md`
- Create: `certs/generate.ps1`
- Create: `certs/generate.sh`
- Create: `saas-server/go.mod`
- Create: `gateway/go.mod`
- Create: `board-agent/go.mod`
- Create: `.gitignore`

**Interfaces:**
- Consumes: 无
- Produces: 三个空 module；`certs/generate.ps1` 生成 `ca.crt`、`server.crt`、`server.key`（SAN 含 `localhost`）

- [ ] **Step 1: 初始化 git 与 .gitignore**

```bash
cd C:\Users\jun\go\src\test
git init
```

`.gitignore` 内容：

```
**/data/
certs/*.key
certs/*.crt
certs/*.pem
certs/*.srl
!certs/generate.ps1
!certs/generate.sh
**/internal/pb/
*.exe
```

- [ ] **Step 2: 创建三个 go.mod**

```bash
mkdir saas-service gateway board-agent
cd saas-server && go mod init saas-server && cd ..
cd gateway && go mod init gateway && cd ..
cd board-agent && go mod init board-agent && cd ..
```

- [ ] **Step 3: 编写 certs/generate.ps1**

用 Go 一次性小工具或 openssl。推荐在 `certs/generate.ps1` 内调用：

```powershell
# 若系统无 openssl，改用: go run ../scripts/gencert/main.go
openssl req -x509 -newkey rsa:2048 -nodes `
  -keyout server.key -out server.crt -days 365 `
  -subj "/CN=localhost" `
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

若环境无 openssl：Create `scripts/gencert/main.go`，用 `crypto/x509` 生成自签证书写入 `certs/`。实现时优先走 Go 生成器以保证 Windows 可用。

- [ ] **Step 4: 写 README 骨架**（生成证书、生成 proto、启动三服务的命令占位）

- [ ] **Step 5: Commit**

```bash
git add .gitignore README.md certs/generate.ps1 certs/generate.sh scripts saas-server/go.mod gateway/go.mod board-agent/go.mod
git commit -m "$(cat <<'EOF'
chore: scaffold three Go modules and cert scripts

EOF
)"
```

Windows PowerShell 若无 HEREDOC，改用：

```powershell
git commit -m "chore: scaffold three Go modules and cert scripts"
```

---

### Task 2: Proto 定义与代码生成

**Files:**
- Create: `proto/board/v1/messages.proto`
- Create: `proto/cloud/v1/messages.proto`
- Create: `scripts/gen-proto.ps1`
- Create: `scripts/gen-proto.sh`

**Interfaces:**
- Consumes: 无
- Produces:
  - board: `Envelope` with oneof payload `Register`/`Heartbeat`/`Command`/`CommandResult`
  - cloud JSON 模型消息：`GatewayHello`/`DeviceRegister`/`DeviceHeartbeat`/`DeviceOffline`/`Command`/`CommandResult`（字段名 JSON camelCase 或 proto json_name）

- [ ] **Step 1: 编写 board proto**

`proto/board/v1/messages.proto`：

```protobuf
syntax = "proto3";
package board.v1;
option go_package = "board-agent/internal/pb/board/v1;boardv1";

message Envelope {
  string msg_id = 1;
  oneof payload {
    Register register = 10;
    Heartbeat heartbeat = 11;
    Command command = 12;
    CommandResult command_result = 13;
  }
}

message Register {
  string device_id = 1;
  string ip = 2;
  int32 port = 3;
}

message Heartbeat {
  string device_id = 1;
  int64 ts_unix_ms = 2;
  string status = 3;
}

message Command {
  string device_id = 1;
  string cmd_id = 2;
  string action = 3;
  string args = 4;
}

message CommandResult {
  string device_id = 1;
  string cmd_id = 2;
  bool ok = 3;
  string message = 4;
}
```

注意：`Register` **没有** `gateway_id`。

- [ ] **Step 2: 编写 cloud proto**

`proto/cloud/v1/messages.proto`：

```protobuf
syntax = "proto3";
package cloud.v1;
option go_package = "saas-server/internal/pb/cloud/v1;cloudv1";

// WSS JSON 信封：{"type":"...","payload":{...}}
// type 取值与下列消息名对应（snake）：gateway_hello, device_register, ...

message GatewayHello {
  string gateway_id = 1;
}

message DeviceRegister {
  string device_id = 1;
  string ip = 2;
  int32 port = 3;
  string gateway_id = 4;
}

message DeviceHeartbeat {
  string device_id = 1;
  int64 ts_unix_ms = 2;
  string status = 3;
  string gateway_id = 4;
}

message DeviceOffline {
  string device_id = 1;
  string gateway_id = 2;
}

message Command {
  string device_id = 1;
  string cmd_id = 2;
  string action = 3;
  string args = 4;
}

message CommandResult {
  string device_id = 1;
  string cmd_id = 2;
  bool ok = 3;
  string message = 4;
}
```

- [ ] **Step 3: 编写 gen-proto.ps1**

对每个目标 module 分别 `protoc`，`go_package` 路径按目标调整（gateway 需要 board+cloud 两份；board-agent 只要 board；saas 只要 cloud）。可用 `protoc --go_out` + 生成后复制，或为 gateway 单独一份 `option` 用相对 generate 配置。

推荐做法：proto 里 `go_package` 用中性路径，脚本为每个 module 指定 `--go_opt=module=` 或生成到临时目录再拷贝到：

- `board-agent/internal/pb/board/v1/`
- `gateway/internal/pb/board/v1/` 与 `gateway/internal/pb/cloud/v1/`
- `saas-server/internal/pb/cloud/v1/`

- [ ] **Step 4: 安装插件并生成**

```powershell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# 确保 protoc 在 PATH
.\scripts\gen-proto.ps1
```

Expected: 上述 `internal/pb` 目录出现 `*.pb.go`。

- [ ] **Step 5: Commit**

```powershell
git add proto scripts
git commit -m "feat: add board and cloud protobuf schemas"
```

（`internal/pb` 若在 gitignore，则各 module 构建前必须跑 gen；README 写明。若希望 CI 简单，可改为不 ignore pb 并提交生成代码——本计划选择 **gitignore + 脚本生成**。）

---

### Task 3: 公共原语 — identity、frame、TLS 配置校验

因 module 不共享 Go 代码，在 `gateway` 与 `board-agent` **各复制一份** 同名逻辑（允许小幅重复，YAGNI）。

**Files:**
- Create: `board-agent/internal/identity/id.go`
- Create: `board-agent/internal/identity/id_test.go`
- Create: `board-agent/internal/frame/frame.go`
- Create: `board-agent/internal/frame/frame_test.go`
- Create: `gateway/internal/identity/id.go`（同逻辑）
- Create: `gateway/internal/frame/frame.go`（同逻辑）
- Create: `gateway/internal/config/tls_mode.go`
- Create: `gateway/internal/config/tls_mode_test.go`
- Create: `saas-server/internal/config/tls_mode.go`（同逻辑）
- Create: `board-agent/internal/config/tls_mode.go`（同逻辑）

**Interfaces:**
- Produces:
  - `func LoadOrCreate(path string) (string, error)` — 文件不存在则写 UUID
  - `func WriteFrame(w io.Writer, payload []byte) error`
  - `func ReadFrame(r io.Reader) ([]byte, error)` — 4 字节大端长度 + body；单帧上限 1MiB
  - `func ValidateTLSMode(mode string, insecureSkipVerify bool) error`
  - `func WarnIfInsecure(insecureSkipVerify bool)` — 打印 WARNING 行

- [ ] **Step 1: 写 identity 失败测试**

```go
func TestLoadOrCreate_CreatesAndReuses(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "device_id")
    id1, err := LoadOrCreate(path)
    if err != nil || id1 == "" {
        t.Fatalf("first: %v %q", err, id1)
    }
    id2, err := LoadOrCreate(path)
    if err != nil || id2 != id1 {
        t.Fatalf("reuse: got %q want %q err=%v", id2, id1, err)
    }
}
```

- [ ] **Step 2: 实现 LoadOrCreate 并跑通测试**

```go
func LoadOrCreate(path string) (string, error) {
    if b, err := os.ReadFile(path); err == nil {
        id := strings.TrimSpace(string(b))
        if id != "" {
            return id, nil
        }
    }
    id := uuid.NewString() // github.com/google/uuid
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return "", err
    }
    if err := os.WriteFile(path, []byte(id+"\n"), 0o644); err != nil {
        return "", err
    }
    return id, nil
}
```

Run: `go test ./internal/identity/ -v`（在 board-agent 目录）  
Expected: PASS

- [ ] **Step 3: frame 编解码测试 + 实现**

```go
func TestFrameRoundTrip(t *testing.T) {
    var buf bytes.Buffer
    want := []byte("hello-proto")
    if err := WriteFrame(&buf, want); err != nil {
        t.Fatal(err)
    }
    got, err := ReadFrame(&buf)
    if err != nil || !bytes.Equal(got, want) {
        t.Fatalf("got %q err=%v", got, err)
    }
}
```

`WriteFrame`：`binary.BigEndian.PutUint32` + write。  
`ReadFrame`：读 4 字节；若 length==0 或 >1<<20 返回 error。

- [ ] **Step 4: TLS mode 测试**

```go
func TestValidateTLSMode(t *testing.T) {
    if err := ValidateTLSMode("development", true); err != nil {
        t.Fatal(err)
    }
    if err := ValidateTLSMode("production", true); err == nil {
        t.Fatal("expected error")
    }
    if err := ValidateTLSMode("production", false); err != nil {
        t.Fatal(err)
    }
}
```

`WarnIfInsecure(true)` 必须向 stderr 打印精确前缀：`WARNING: TLS certificate verification is disabled`

- [ ] **Step 5: 将相同 identity/frame/tls_mode 复制到 gateway（及 saas 的 tls_mode），跑测试后 Commit**

```powershell
git add board-agent gateway saas-service
git commit -m "feat: add identity, frame codec, and TLS mode guards"
```

---

### Task 4: Gateway Board 会话表（SessionID 防误删）

**Files:**
- Create: `gateway/internal/boardsession/registry.go`
- Create: `gateway/internal/boardsession/registry_test.go`

**Interfaces:**
- Produces:
  - `type BoardConnection struct { DeviceID, SessionID string; Conn net.Conn }`
  - `type Registry struct` with mutex map
  - `func (r *Registry) Put(c *BoardConnection)` — 替换旧连接（可选关闭旧 Conn）
  - `func (r *Registry) Get(deviceID string) (*BoardConnection, bool)`
  - `func (r *Registry) RemoveIfSame(deviceID, sessionID string) bool` — 仅匹配才删，返回是否删除

- [ ] **Step 1: 写防误删测试**

```go
func TestRemoveIfSame_IgnoresStaleDisconnect(t *testing.T) {
    r := NewRegistry()
    old := &BoardConnection{DeviceID: "d1", SessionID: "A", Conn: nil}
    neu := &BoardConnection{DeviceID: "d1", SessionID: "B", Conn: nil}
    r.Put(old)
    r.Put(neu)
    if r.RemoveIfSame("d1", "A") {
        t.Fatal("stale should not remove")
    }
    got, ok := r.Get("d1")
    if !ok || got.SessionID != "B" {
        t.Fatalf("want B, got %+v ok=%v", got, ok)
    }
    if !r.RemoveIfSame("d1", "B") {
        t.Fatal("current should remove")
    }
}
```

- [ ] **Step 2: 实现 Registry，测试 PASS，Commit**

```powershell
git commit -m "feat(gateway): session registry guards stale disconnects"
```

---

### Task 5: board-agent 连接、注册、心跳、echo

**Files:**
- Create: `board-agent/internal/config/config.go`
- Create: `board-agent/internal/netinfo/ip.go`
- Create: `board-agent/internal/agent/agent.go`
- Create: `board-agent/internal/agent/agent_test.go`（echo 处理单测）
- Create: `board-agent/cmd/board-agent/main.go`

**Interfaces:**
- Consumes: identity、frame、boardv1 protobuf、tls_mode
- Produces: 可运行二进制；连 `gateway_addr`；发 Register/Heartbeat；处理 Command action=`echo`

- [ ] **Step 1: echo handler 单测**

```go
func TestHandleCommand_Echo(t *testing.T) {
    res := HandleCommand(&boardv1.Command{DeviceId: "d", CmdId: "c1", Action: "echo", Args: "hi"})
    if !res.Ok || res.Message != "hi" || res.CmdId != "c1" {
        t.Fatalf("%+v", res)
    }
    res = HandleCommand(&boardv1.Command{Action: "upgrade"})
    if res.Ok {
        t.Fatal("upgrade should fail in v1")
    }
}
```

- [ ] **Step 2: 实现 Agent 主循环（伪代码级完整逻辑）**

```go
// Dial TLS -> loop: read envelope; on Command reply; ticker Heartbeat
// On start and reconnect: send Register{device_id, ip, port}  // NO gateway_id
```

配置字段：`GatewayAddr`、`CertFile`/`Key`（客户端可只校验证书）、`DataDir`、`ReportPort`、`HeartbeatInterval`、`Mode`、`InsecureSkipVerify`。

- [ ] **Step 3: `go build ./cmd/board-agent` 成功**

- [ ] **Step 4: Commit**

```powershell
git commit -m "feat(board-agent): register, heartbeat, and echo command"
```

---

### Task 6: Gateway TLS Board 服务端 + 云端 WSS 客户端

**Files:**
- Create: `gateway/internal/config/config.go`
- Create: `gateway/internal/boardserver/server.go`
- Create: `gateway/internal/cloudclient/client.go`
- Create: `gateway/cmd/gateway/main.go`

**Interfaces:**
- Consumes: boardsession、frame、boardv1、cloudv1（JSON）、identity
- Produces:
  - Board 接入后 `Put` 会话；解析 Register/Heartbeat/CommandResult
  - 转发 Cloud：`{"type":"device_register","payload":{...,"gateway_id":...}}`
  - 收 Cloud `command` → 按 device_id 写 Board frame
  - 断连：`RemoveIfSame` 成功才发 `device_offline`

- [ ] **Step 1: cloudclient JSON 信封编解码单测**

```go
type Envelope struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

`type` 常量：`gateway_hello`、`device_register`、`device_heartbeat`、`device_offline`、`command`、`command_result`。

- [ ] **Step 2: 实现 boardserver Accept 循环**

每个 conn：生成 `SessionID`（uuid）；读第一帧应为 Register；`Put`；循环读帧并上行；defer 中 `RemoveIfSame` + 条件离线通知。

- [ ] **Step 3: 实现 cloudclient**

- Dial `wss://.../ws/gateway`（development+insecure 时 `tls.Config{InsecureSkipVerify:true}` + WARNING）
- 连上发 `gateway_hello`
- 读写泵：上行 channel、下行 command 回调

- [ ] **Step 4: main 组装；`go build ./cmd/gateway`**

- [ ] **Step 5: Commit**

```powershell
git commit -m "feat(gateway): TLS board server and cloud WSS client"
```

---

### Task 7: saas-service 路由存储、Gateway WS、REST、指令等待

**Files:**
- Create: `saas-server/internal/config/config.go`
- Create: `saas-server/internal/store/store.go`
- Create: `saas-server/internal/store/store_test.go`
- Create: `saas-server/internal/gatewayhub/hub.go`
- Create: `saas-server/internal/api/http.go`
- Create: `saas-server/internal/api/ws_gateway.go`
- Create: `saas-server/internal/api/ws_ui.go`
- Create: `saas-server/cmd/saas-server/main.go`

**Interfaces:**
- Produces:
  - `Store`: UpsertDeviceRegister / TouchHeartbeat / MarkOffline / ListDevices / GetDevice
  - `Hub`: gateway_id → conn；`SendCommand(gatewayID, cmd) error`
  - `POST /api/devices/{id}/commands`：生成 `cmd_id`，等待 result 或 10s timeout
  - 设备不存在或离线：404；有设备但无 gateway 连接：409

- [ ] **Step 1: Store 单测**（注册、心跳、离线、列表）

- [ ] **Step 2: 实现 Hub + ws_gateway**

收到 `device_register` 写 Store 并 broadcast UI；`command_result` 完成 pending waiters。

Pending map：`cmd_id → chan CommandResult`。

- [ ] **Step 3: REST handlers**

```go
// GET /api/devices -> JSON array
// POST /api/devices/{id}/commands {"action":"echo","args":"..."}
```

- [ ] **Step 4: HTTPS ListenAndServeTLS(cert,key) 挂载路由；Mode/TLS 校验**

- [ ] **Step 5: `go build` + Commit**

```powershell
git commit -m "feat(saas): device store, gateway hub, REST and command wait"
```

---

### Task 8: 简易 Web UI + /ws/ui

**Files:**
- Create: `saas-server/web/index.html`
- Modify: `saas-server/internal/api/http.go`（静态文件 `/`）
- Modify: `saas-server/internal/api/ws_ui.go`

**Interfaces:**
- UI：轮询或 WS 刷新设备表；输入 device_id / args，点发送调用 REST 或显示结果
- `/ws/ui` 推送：`device_updated`、`command_result` 的 JSON（手写结构，不 import pb 到前端）

- [ ] **Step 1: index.html** — 单页：表格 + 表单；`fetch('/api/devices')`；`WebSocket` 连 `/ws/ui`

- [ ] **Step 2: 静态托管 `embed` 或 `http.FileServer`**

推荐：

```go
//go:embed ../web/*
var webFS embed.FS
```

注意 embed 路径相对包目录；可将 `web` 放到 `internal/api/web` 或从 main embed。

- [ ] **Step 3: Commit**

```powershell
git commit -m "feat(saas): simple device UI and ui websocket"
```

---

### Task 9: 端到端联调与 README 完善

**Files:**
- Modify: `README.md`
- Create: `scripts/run-all.ps1`（可选）

- [ ] **Step 1: 生成证书与 proto**

```powershell
go run ./scripts/gencert   # 或 certs/generate.ps1
.\scripts\gen-proto.ps1
```

- [ ] **Step 2: 三终端启动**

```powershell
# terminal 1
cd saas-server
go run ./cmd/saas-server -mode development -addr :8443 -cert ../certs/server.crt -key ../certs/server.key

# terminal 2
cd gateway
go run ./cmd/gateway -mode development -saas wss://127.0.0.1:8443/ws/gateway -listen :9443 -cert ../certs/server.crt -key ../certs/server.key -insecure-skip-verify

# terminal 3
cd board-agent
go run ./cmd/board-agent -mode development -gateway 127.0.0.1:9443 -insecure-skip-verify -report-port 0
```

Expected 日志：saas 收到 gateway_hello；board register；心跳刷新。

- [ ] **Step 3: 浏览器或 curl 验证**

```powershell
curl -k https://127.0.0.1:8443/api/devices
curl -k -X POST https://127.0.0.1:8443/api/devices/<id>/commands -H "Content-Type: application/json" -d "{\"action\":\"echo\",\"args\":\"hello\"}"
```

Expected: devices 非空；POST 返回 `{"ok":true,"message":"hello",...}`。

- [ ] **Step 4: 重连防误删手工验证** — 重启 board-agent，旧连接断开日志不得把新会话标离线超过一瞬间；最终设备仍 online。

- [ ] **Step 5: 更新 README 完整步骤；Commit**

```powershell
git commit -m "docs: add local run instructions for three services"
```

---

## Spec Coverage Self-Review

| Spec 要求 | Task |
|-----------|------|
| 三独立 module + 共享 proto | 1, 2 |
| 自签证书 | 1, 9 |
| Board UUID 持久化 | 3, 5 |
| REGISTER 无 gateway_id，Gateway 注入 | 2, 5, 6 |
| TCP+TLS+length+protobuf | 3, 5, 6 |
| WSS JSON Gateway↔Cloud | 6, 7 |
| HTTPS + REST + Web UI + /ws/ui | 7, 8 |
| SessionID 防误删 | 4, 6 |
| insecure 仅 development + WARNING | 3, 5–7 |
| echo 指令与执行结果 | 5, 7, 9 |
| 心跳与离线 | 5–7, 9 |

## Placeholder / Consistency Check

- 消息 type 字符串统一用 snake：`gateway_hello`、`device_register`、`device_heartbeat`、`device_offline`、`command`、`command_result`
- `BoardConnection.SessionID` 与 `RemoveIfSame` 命名在 Task 4/6 一致
- 无 TBD；生成证书在 Windows 优先 `scripts/gencert` Go 实现

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-09-22-saas-gateway-board-plan.md`.

**两种执行方式：**

1. **Subagent-Driven（推荐）** — 每个 Task 派一个新子代理，任务间我做审查，迭代快  
2. **Inline Execution** — 本会话按 executing-plans 连续执行，设检查点

要哪种？
