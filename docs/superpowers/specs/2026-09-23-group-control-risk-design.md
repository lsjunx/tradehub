# 群控面板与账号风控设计

日期：2026-09-23  
状态：待用户审阅  
前置：`2026-09-22-saas-gateway-board-design.md`（SaaS / Gateway / Board 运输骨架）

## 1. 目标与约束

### 1.1 目标

- SaaS 作为 WhatsApp / Telegram 等聊天软件的**群控面板**。
- 板子安装多款 App 后，通过稳定控制面下发命令、回收结果与风险事件。
- **Board 与 Gateway 不因业务功能频繁改动**；新 App / 新动作主要落在「板子插件 + SaaS 动作目录与策略」。

### 1.2 业务前提（已确认）

- 执行模式 **C**：常见操作用板子**意图插件**（如 `tg.reply_text`）；复杂流程由 SaaS 编排板子**原子 UI**（如 `ui.tap`）。
- 账号用途 **C**：以客服 / 社群运营为主；冷启动触达陌生人允许存在，但**严格限流、独立通道**。
- 防封控 / 防死号的决策中心在 **SaaS**；板子只做执行、本地硬拒兜底、上报风险事件。

### 1.3 非目标

- 不为每个 App 单独拆 Gateway 协议或独立云通道。
- Gateway 不解析业务 `action` / `args`，不做风控判决。
- 不在本文展开具体 UI 自动化实现或各 App 逆向细节。

## 2. 总体方案

采用：**稳定运输层 + Board 插件路由 + SaaS 风控中枢**。

```text
浏览器群控面板
    ↓ HTTPS / WSS
SaaS（账号画像 · 动作目录 · 配额 · 熔断 · 冷触达队列）
    ↓ WSS JSON 信封（透明业务字段）
Gateway（按 device_id 转发，不解析 action）
    ↓ TCP+TLS+长度前缀+protobuf Envelope
Board-agent（路由）→ plugin/tg | plugin/wa | plugin/ui
    ↓ 本机网络 / 每账号代理出口
TG / WA 服务器（对外 IP = 板子或代理出口，≠ SaaS/Gateway）
```

## 3. 运输层协议

### 3.1 四类消息

| 类型 | 方向 | 用途 |
|------|------|------|
| `command` | SaaS → Board | 下发意图或原子动作 |
| `command_result` | Board → SaaS | 单次命令成败回执 |
| `event` | Board → SaaS | 异步业务或风控信号 |
| `capability` | Board → SaaS | 上报当前支持的 `app.verb` |

信封字段稳定；业务差异只出现在 `action` 名与 JSON `args` / `payload`。

### 3.2 命令形状

- 必填：`cmd_id`、`device_id`、`action`、`args`（JSON 字符串）
- 可选：`risk_class`（`safe | sensitive | high`）、`deadline_ms`、编排用 `flow_id`
- `action` 命名：`{app}.{verb}`，例如 `tg.reply_text`、`wa.send_text`、`ui.tap`、`echo`

Gateway 将 `args` 视为不透明字符串，原样转发。

### 3.3 事件形状

- 字段：`event_id`、`device_id`、`name`、`payload`（JSON）、`ts`
- 风控相关 `name`（首批固定）：
  - `account.login_required`
  - `account.challenge`
  - `account.rate_limited`
  - `account.session_dead`
- 业务事件（如 `tg.message_in`）可后续追加，**不改信封类型**。

### 3.4 能力声明

- 板子连上及插件热更新后上报：`[{ "action": "tg.reply_text", "risk_hint": "safe" }, ...]`
- SaaS 仅允许对已声明 capability 的下发；面板按 capability 显隐按钮。

### 3.5 Gateway 职责边界

- 维持现有：注册注入 `gateway_id`、会话表、`command`/`command_result` 转发。
- 新增：对 `event`、`capability` **同样透明转发**。
- 禁止：按 App 分支、解析 `args`、实现配额或熔断。

## 4. Board 插件边界

### 4.1 进程结构

```text
board-agent（稳定：连接、心跳、收发信封）
  └─ Plugin Router（按 action 前缀分发）
        ├─ plugin/tg/
        ├─ plugin/wa/
        └─ plugin/ui/     # tap / type / swipe / screenshot
```

新 App = 新插件包 + 更新 capability；**不改** Envelope 外壳，**不改** Gateway。

### 4.2 意图 vs 原子

| 档 | 示例 | 实现位置 | 使用场景 |
|----|------|----------|----------|
| 意图 | `tg.reply_text`、`tg.list_dialogs` | App 插件 | 客服、已知会话、日常运营 |
| 原子 | `ui.tap`、`ui.type`、`ui.screenshot` | 通用 UI 插件 | SaaS 编排复杂/未覆盖流程 |

规则：

1. 同一业务若已有意图 action，面板默认禁止裸下原子序列。
2. `ui.*` 默认 `sensitive`，须带 `flow_id`；板子可拒绝无编排上下文的裸 `ui.*`。
3. 插件只执行**单次**已授权命令；批量、重试、错峰全在 SaaS。
4. 本地兜底：challenge / session_dead 时发 `event`，并拒绝后续高危 action，直至 SaaS 解禁。

## 5. 对外 IP 与出口隔离

- TG/WA 服务器看到的出口 IP 来自**板子本机出口或账号绑定代理**，不是 SaaS，也不是 Gateway。
- 默认策略：**一账号一 `egress_id`（独立代理/线路）**。
- 多号共一出口仅可用于少量、低敏客服号；禁止与冷触达号共用出口。
- SaaS 账号画像记录 `account_id → device_id → egress_id`；变更 device 或 egress 进入观察期（降档、削减敏感配额）。

## 6. SaaS 风控中枢

### 6.1 账号画像

每条聊天账号包含：

- 身份：`account_id`、`app`、`device_id`、`egress_id`
- 状态档：`new` → `warming` → `normal` → `restricted` → `dead`
- `trust`：是否仅服务已知会话（客服号）
- 配额余量、最近风险事件摘要

### 6.2 动作分级

| 级 | 示例 | 策略倾向 |
|----|------|----------|
| `safe` | 已知会话回复、拉会话列表 | 客服主路径，配额较宽 |
| `sensitive` | 加好友、进群、改资料、`ui.*` | 须编排 + 间隔 |
| `high` | 陌生人群发、批量邀请 | 冷触达专用通道，默认关或极低配额 |

未在动作目录登记的 `action` → 拒绝下发。

### 6.3 配额与节奏

- 维度：`account × action_class × 滑动窗口（小时/天）`
- `new` / `warming`：几乎仅 `safe`
- 发令前 Policy：`Allow` / `Deny` / `Delay`；原因码返回面板；Delay 由 SaaS 排队
- 同 `egress_id` 多号共享出口级上限

### 6.4 熔断

| 事件 | 行为 |
|------|------|
| `account.challenge` | 暂停该号全部下发，待人工 |
| `account.rate_limited` | 降档 + 冷却窗 |
| `account.session_dead` | `tier=dead`，仅允许重新登录类流程 |
| 短时大量 command 失败 | 账号或整机临时熔断 |

禁止失败后自动高频重试；解禁须显式操作或观察期到期。

### 6.5 冷触达通道

- 与客服下发分队列、分配额、分审批权限。
- 前置：`tier` 至少 `warming`、健康 `egress_id`、无未处理 challenge。
- 单任务总量上限；不与 `safe` 回复抢同一突发窗口。
- 面板入口默认折叠或需权限；默认策略可整体关闭冷触达。

### 6.6 与运输层衔接

```text
API/面板 → Policy.Allow(account, action) → 通过才 Send command
Board event(risk) → Policy.Trip(account) → 后续短路
capability 变更 → 刷新可操作按钮（不改 Gateway）
```

## 7. 群控面板能力

1. **账号墙**：App、登录态、`tier`、`egress`、配额、熔断原因  
2. **客服台**：账号 → 会话 → 回复（`safe` 意图）  
3. **编排台**：敏感 / 原子动作，展示 Deny/Delay 原因  
4. **冷触达台**：独立权限与进度、一键停  
5. **告警**：风险事件推 `/ws/ui`  

运营主对象为**账号**；设备是宿主与连通性视图。

## 8. 相对现状的改造范围

| 层 | 保持稳定 | 需要演进 |
|----|----------|----------|
| Gateway | 会话、帧、command/result 转发 | 透传 `event`、`capability` |
| Board | TLS、帧、Envelope 外壳 | Plugin Router；插件；上报 event/capability；本地高危硬拒 |
| board proto | Register / Heartbeat / Command / Result | Envelope 增加 event、capability；Command 可选 risk_class / flow_id |
| cloud 线网 | hello / register / heartbeat / offline / command / result | 增加 event、capability 类型；command 载荷可扩字段 |
| SaaS | Hub、异步 cmd_id、设备视图 | Policy、账号模型、动作目录、冷触达队列、账号维面板 |

## 9. 建议落地顺序

1. Proto / 线网增加 `event`、`capability`；Gateway 透传。  
2. Board Plugin Router；保留 `echo` 验证通路。  
3. SaaS Policy 最小版（分级 + 日配额 + 熔断）+ 账号绑定 `egress_id`。  
4. 面板账号墙 + 客服回复。  
5. 冷触达通道最后开启（默认关）。

## 10. 成功标准

- 新增一种聊天 App 或一批 verb：**无需改 Gateway 核心**，仅插件 + SaaS 目录/配额。  
- 客服回复不经过冷触达配额；冷触达无法绕过 Policy。  
- 出现 challenge / session_dead 后，该账号自动停止高危与普通下发直至解禁。  
- 同 egress 多号受到出口级上限约束；换出口进入观察期。

## 11. 开放问题（实现阶段再定，不阻塞本设计）

- `args` / event `payload` 的 JSON schema 登记方式（SaaS 配置 vs 独立 schema 仓）。  
- 代理/`egress_id` 的具体供应商与健康检查探针。  
- 意图插件调用官方 API、协议库或 UI 自动化的技术选型（按 App 分插件决策）。
