# Group-Control Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the transport + Board plugin router + SaaS-managed egress/policy foundation from `docs/superpowers/specs/2026-09-23-group-control-risk-design.md` (§9 steps 1–3), so commands/events stay stable while SaaS controls per-account proxy binding.

**Architecture:** Extend board/cloud envelopes with `event` and `capability`; Gateway remains a transparent forwarder; Board gains a plugin router (`echo`, `egress.apply`); SaaS adds in-memory account/egress stores and a Policy gate before `SendJSON` command dispatch.

**Tech Stack:** Go 1.22+, existing Gin/WSS stack, protobuf + length-prefix frames, PowerShell `scripts/gen-proto.ps1`.

## Global Constraints

- Gateway MUST NOT parse `action`/`args` business meaning or carry TG/WA data-plane traffic.
- Chat egress MUST be SaaS-assigned proxy (`egress_id`); board LAN IP MUST NOT be used as business egress when unbound.
- Wire `action` names use `{app}.{verb}` (e.g. `echo`, `egress.apply`).
- Line-wire cloud `type` strings derive from `MsgType` via existing `TypeName`/`ParseType` (`MSG_TYPE_FOO` → `foo`).
- Do not implement real Telegram/WhatsApp plugins, cold-outreach queue UI, or production proxy health probes in this plan.
- Prefer TDD; commit after each task; run gen-proto after every proto change.

## Out of Scope (follow-on plans)

- Full group-control web UI (account wall / CS desk).
- Cold-outreach channel and approvals.
- Real App automation / per-app proxy injection mechanics beyond storing applied egress config on device.
- Proxy vendor integration and live IP probing.

## File Structure

| Path | Responsibility |
|------|----------------|
| `proto/board/v1/messages.proto` | Add `Event`, `Capability`, optional command fields |
| `proto/cloud/v1/messages.proto` | Add `MSG_TYPE_EVENT`, `MSG_TYPE_CAPABILITY` + message shapes |
| `gateway/internal/logic/bridge.go` | Forward board Event/Capability uplink |
| `gateway/internal/cloud/msgtype.go` | Auto-picks new types after regen (verify vars if added) |
| `board-agent/internal/logic/plugin.go` | Plugin interface + router |
| `board-agent/internal/logic/plugins/echo.go` | `echo` handler |
| `board-agent/internal/logic/plugins/egress.go` | `egress.apply` / `egress.clear` |
| `board-agent/internal/egress/store.go` | Per-device applied proxy config (memory/file) |
| `board-agent/internal/logic/command.go` | Delegate to router; keep thin |
| `board-agent/internal/handle/agent.go` | After register: send Capability; route commands via router |
| `saas-server/internal/store/account.go` | Account records + egress bindings |
| `saas-server/internal/store/egress.go` | Proxy pool entries |
| `saas-server/internal/policy/policy.go` | Allow/Deny/Delay for actions |
| `saas-server/internal/logic/gateway.go` | Handle `event` / `capability` |
| `saas-server/internal/logic/device.go` | Policy gate + optional account resolution |
| `saas-server/internal/logic/egress.go` | Bind/apply egress → push `egress.apply` command |
| `saas-server/internal/handle/egress.go` | REST for pool + bind |
| `saas-server/internal/router/router.go` | Register new REST routes |
| `saas-server/internal/common/uievent/types.go` | UI event names for risk/egress if pushed |

---

### Task 1: Proto — Event, Capability, cloud MsgTypes

**Files:**
- Modify: `proto/board/v1/messages.proto`
- Modify: `proto/cloud/v1/messages.proto`
- Modify: (generated) `*/internal/pb/**` via `scripts/gen-proto.ps1`
- Test: `gateway/internal/cloud/envelope_test.go`

**Interfaces:**
- Consumes: existing `Envelope` / `MsgType`
- Produces: board `Event`, `Capability`, `CapabilityEntry`; cloud enum values `MSG_TYPE_EVENT=7`, `MSG_TYPE_CAPABILITY=8`; optional `Command.risk_class`, `Command.flow_id`

- [ ] **Step 1: Extend board proto**

Replace `proto/board/v1/messages.proto` payload/command sections with:

```protobuf
message Envelope {
  string msg_id = 1;
  oneof payload {
    Register register = 10;
    Heartbeat heartbeat = 11;
    Command command = 12;
    CommandResult command_result = 13;
    Event event = 14;
    Capability capability = 15;
  }
}

message Command {
  string device_id = 1;
  string cmd_id = 2;
  string action = 3;
  string args = 4;
  string risk_class = 5; // safe|sensitive|high; optional
  string flow_id = 6;    // required for ui.* when enforced later
}

message Event {
  string event_id = 1;
  string device_id = 2;
  string name = 3;
  string payload_json = 4;
  int64 ts_unix_ms = 5;
}

message CapabilityEntry {
  string action = 1;
  string risk_hint = 2; // safe|sensitive|high
}

message Capability {
  string device_id = 1;
  repeated CapabilityEntry entries = 2;
}
```

Keep existing `Register`, `Heartbeat`, `CommandResult` unchanged.

- [ ] **Step 2: Extend cloud proto MsgType + messages**

In `proto/cloud/v1/messages.proto`, append to enum (do not renumber existing):

```protobuf
  MSG_TYPE_EVENT = 7;
  MSG_TYPE_CAPABILITY = 8;
```

Add:

```protobuf
message DeviceEvent {
  string event_id = 1;
  string device_id = 2;
  string name = 3;
  string payload_json = 4;
  int64 ts_unix_ms = 5;
  string gateway_id = 6;
}

message DeviceCapability {
  string device_id = 1;
  string gateway_id = 2;
  repeated CapabilityEntry entries = 3;
}

message CapabilityEntry {
  string action = 1;
  string risk_hint = 2;
}
```

- [ ] **Step 3: Regenerate**

Run: `powershell -File scripts/gen-proto.ps1`  
Expected: completes with `proto generation complete` and lists `messages.pb.go` under board-agent/gateway/saas-service.

- [ ] **Step 4: Extend wire-name test**

In `gateway/internal/cloud/envelope_test.go` add:

```go
func TestTypeNameEventCapability(t *testing.T) {
	if TypeName(cloudv1.MsgType_MSG_TYPE_EVENT) != "event" {
		t.Fatalf("got %q", TypeName(cloudv1.MsgType_MSG_TYPE_EVENT))
	}
	if ParseType("capability") != cloudv1.MsgType_MSG_TYPE_CAPABILITY {
		t.Fatal("parse capability")
	}
}
```

- [ ] **Step 5: Run tests**

Run: `cd gateway; go test ./internal/cloud/...`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add proto scripts gateway/internal/cloud/envelope_test.go
git add board-agent/internal/pb gateway/internal/pb saas-server/internal/pb
git commit -m "feat(proto): add event and capability envelope types"
```

---

### Task 2: Gateway — forward Event and Capability uplink

**Files:**
- Modify: `gateway/internal/logic/bridge.go`
- Modify: `gateway/internal/cloud/msgtype.go` (ensure `TypeEvent` / `TypeCapability` vars exist after regen)
- Test: `gateway/internal/logic/bridge_forward_test.go` (create)

**Interfaces:**
- Consumes: `boardv1.Envelope_Event`, `boardv1.Envelope_Capability`
- Produces: `Cloud.Send(TypeEvent|TypeCapability, map...)` with `gateway_id` injected

- [ ] **Step 1: Add TypeEvent / TypeCapability aliases**

In `gateway/internal/cloud/msgtype.go` var block add:

```go
TypeEvent      = TypeName(cloudv1.MsgType_MSG_TYPE_EVENT)
TypeCapability = TypeName(cloudv1.MsgType_MSG_TYPE_CAPABILITY)
```

Same two lines in `saas-server/internal/common/cloudwire/msgtype.go`.

- [ ] **Step 2: Write failing forward test**

Create `gateway/internal/logic/bridge_forward_test.go`:

```go
package logic

import (
	"testing"

	boardv1 "gateway/internal/pb/board/v1"
)

type fakeCloud struct {
	lastType string
	lastPayload map[string]any
}

func (f *fakeCloud) Send(typ string, payload any) error {
	f.lastType = typ
	if m, ok := payload.(map[string]any); ok {
		f.lastPayload = m
	}
	return nil
}

// Adapt Bridge.Cloud type: if Client is concrete, extract an interface in bridge.go:
// type CloudSender interface { Send(typ string, payload any) error }
// and change Bridge.Cloud to CloudSender for testability.

func TestHandleBoardEnvelope_EventForwards(t *testing.T) {
	// After introducing CloudSender, construct Bridge with fakeCloud.
	// Call HandleBoardEnvelope with Envelope_Event{Name: "egress.unhealthy", DeviceId: "d1"}.
	// Assert fake.lastType == "event" and gateway_id present.
	t.Fatal("implement after CloudSender refactor")
}
```

- [ ] **Step 3: Refactor Bridge.Cloud to interface**

In `bridge.go`:

```go
type CloudSender interface {
	Send(typ string, payload any) error
}

type Bridge struct {
	GatewayID string
	Registry  *session.Registry
	Cloud     CloudSender
}
```

Ensure `*cloud.Client` still satisfies it. Update `main.go` if types break (should not).

- [ ] **Step 4: Implement Event/Capability cases**

In `HandleBoardEnvelope` switch add:

```go
case *boardv1.Envelope_Event:
	ev := p.Event
	_ = b.Cloud.Send(cloud.TypeEvent, map[string]any{
		"event_id":     ev.GetEventId(),
		"device_id":    ev.GetDeviceId(),
		"name":         ev.GetName(),
		"payload_json": ev.GetPayloadJson(),
		"ts_unix_ms":   ev.GetTsUnixMs(),
		"gateway_id":   b.GatewayID,
	})

case *boardv1.Envelope_Capability:
	cap := p.Capability
	entries := make([]map[string]string, 0, len(cap.GetEntries()))
	for _, e := range cap.GetEntries() {
		entries = append(entries, map[string]string{
			"action":     e.GetAction(),
			"risk_hint":  e.GetRiskHint(),
		})
	}
	_ = b.Cloud.Send(cloud.TypeCapability, map[string]any{
		"device_id":  cap.GetDeviceId(),
		"gateway_id": b.GatewayID,
		"entries":    entries,
	})
```

Do **not** add App-specific branches.

- [ ] **Step 5: Finish test without `t.Fatal` stub**

Implement the test using `fakeCloud` and a real `Bridge{Cloud: fake, GatewayID: "g1", Registry: session.New()}`. Assert types and fields.

- [ ] **Step 6: Run tests**

Run: `cd gateway; go test ./internal/logic/... ./internal/cloud/...`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add gateway/internal/logic gateway/internal/cloud saas-server/internal/common/cloudwire/msgtype.go
git commit -m "feat(gateway): forward event and capability to SaaS"
```

---

### Task 3: Board — plugin router, echo, capability report

**Files:**
- Create: `board-agent/internal/logic/plugin.go`
- Create: `board-agent/internal/logic/plugins/echo.go`
- Create: `board-agent/internal/logic/router.go`
- Modify: `board-agent/internal/logic/command.go`
- Modify: `board-agent/internal/handle/agent.go`
- Test: `board-agent/internal/logic/router_test.go`

**Interfaces:**
- Consumes: `*boardv1.Command`
- Produces: `Plugin.Handle(cmd) *boardv1.CommandResult`; `Router.Dispatch`; `DefaultCapabilities() []CapabilityEntry`

- [ ] **Step 1: Write failing router test**

```go
package logic_test

import (
	"testing"

	"board-agent/internal/logic"
	boardv1 "board-agent/internal/pb/board/v1"
)

func TestRouterEcho(t *testing.T) {
	r := logic.NewRouter()
	res := r.Dispatch(&boardv1.Command{DeviceId: "d", CmdId: "c1", Action: "echo", Args: "hi"})
	if !res.Ok || res.Message != "hi" {
		t.Fatalf("%+v", res)
	}
}

func TestRouterUnknown(t *testing.T) {
	r := logic.NewRouter()
	res := r.Dispatch(&boardv1.Command{Action: "tg.send_text", CmdId: "c2", DeviceId: "d"})
	if res.Ok {
		t.Fatal("expected fail")
	}
}
```

- [ ] **Step 2: Run test — expect fail**

Run: `cd board-agent; go test ./internal/logic/...`  
Expected: FAIL (`NewRouter` undefined)

- [ ] **Step 3: Implement plugin + router**

`plugin.go`:

```go
package logic

import boardv1 "board-agent/internal/pb/board/v1"

type Plugin interface {
	Actions() []string
	Handle(cmd *boardv1.Command) *boardv1.CommandResult
}

type Router struct {
	byAction map[string]Plugin
}

func NewRouter(plugins ...Plugin) *Router {
	r := &Router{byAction: map[string]Plugin{}}
	for _, p := range plugins {
		for _, a := range p.Actions() {
			r.byAction[a] = p
		}
	}
	return r
}

func (r *Router) Dispatch(cmd *boardv1.Command) *boardv1.CommandResult {
	p, ok := r.byAction[cmd.GetAction()]
	if !ok {
		return &boardv1.CommandResult{
			DeviceId: cmd.GetDeviceId(),
			CmdId:    cmd.GetCmdId(),
			Ok:       false,
			Message:  "unsupported action: " + cmd.GetAction(),
		}
	}
	return p.Handle(cmd)
}

func (r *Router) CapabilityEntries() []*boardv1.CapabilityEntry {
	var out []*boardv1.CapabilityEntry
	seen := map[string]bool{}
	for action := range r.byAction {
		if seen[action] {
			continue
		}
		seen[action] = true
		hint := "safe"
		if action != "echo" && action != "egress.apply" && action != "egress.clear" {
			hint = "sensitive"
		}
		out = append(out, &boardv1.CapabilityEntry{Action: action, RiskHint: hint})
	}
	return out
}
```

`plugins/echo.go` — put echo in `logic` package first to avoid import cycles, as `internal/logic/echo_plugin.go`:

```go
package logic

import boardv1 "board-agent/internal/pb/board/v1"

type EchoPlugin struct{}

func (EchoPlugin) Actions() []string { return []string{"echo"} }

func (EchoPlugin) Handle(cmd *boardv1.Command) *boardv1.CommandResult {
	return &boardv1.CommandResult{
		DeviceId: cmd.GetDeviceId(),
		CmdId:    cmd.GetCmdId(),
		Ok:       true,
		Message:  cmd.GetArgs(),
	}
}
```

Replace `HandleCommand` body to use package-level router:

```go
var defaultRouter = NewRouter(EchoPlugin{})

func HandleCommand(cmd *boardv1.Command) *boardv1.CommandResult {
	return defaultRouter.Dispatch(cmd)
}

func DefaultRouter() *Router { return defaultRouter }
```

- [ ] **Step 4: Send Capability after Register in agent**

After successful Register send in `serveSession`, send:

```go
entries := logic.DefaultRouter().CapabilityEntries()
_ = send(&boardv1.Envelope{
	MsgId: uuid.NewString(),
	Payload: &boardv1.Envelope_Capability{Capability: &boardv1.Capability{
		DeviceId: a.DeviceID,
		Entries:  entries,
	}},
})
```

- [ ] **Step 5: Run tests**

Run: `cd board-agent; go test ./internal/logic/...`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add board-agent/internal/logic board-agent/internal/handle/agent.go
git commit -m "feat(board): plugin router with echo and capability report"
```

---

### Task 4: Board — egress store + `egress.apply` / `egress.clear`

**Files:**
- Create: `board-agent/internal/egress/store.go`
- Create: `board-agent/internal/logic/egress_plugin.go`
- Modify: `board-agent/internal/logic/command.go` (register plugin; wire store)
- Test: `board-agent/internal/egress/store_test.go`, `board-agent/internal/logic/egress_plugin_test.go`

**Interfaces:**
- Consumes: `args` JSON `{"account_id":"...","egress_id":"...","proxy_url":"socks5://user:pass@host:port"}`
- Produces: `Store.Apply(accountID, cfg)`; events name `egress.unhealthy` only when apply JSON invalid (later probes out of scope)

- [ ] **Step 1: Failing store test**

```go
package egress_test

import (
	"testing"

	"board-agent/internal/egress"
)

func TestApplyAndGet(t *testing.T) {
	s := egress.NewStore()
	cfg := egress.Config{AccountID: "a1", EgressID: "e1", ProxyURL: "socks5://127.0.0.1:1080"}
	if err := s.Apply(cfg); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get("a1")
	if !ok || got.EgressID != "e1" {
		t.Fatalf("%+v", got)
	}
	s.Clear("a1")
	if _, ok := s.Get("a1"); ok {
		t.Fatal("cleared")
	}
}
```

- [ ] **Step 2: Implement store**

```go
package egress

import "sync"

type Config struct {
	AccountID string `json:"account_id"`
	EgressID  string `json:"egress_id"`
	ProxyURL  string `json:"proxy_url"`
}

type Store struct {
	mu   sync.RWMutex
	byAc map[string]Config
}

func NewStore() *Store { return &Store{byAc: map[string]Config{}} }

func (s *Store) Apply(cfg Config) error {
	if cfg.AccountID == "" || cfg.EgressID == "" || cfg.ProxyURL == "" {
		return errIncomplete // define var errors.New("incomplete egress config")
	}
	s.mu.Lock()
	s.byAc[cfg.AccountID] = cfg
	s.mu.Unlock()
	return nil
}

func (s *Store) Clear(accountID string) {
	s.mu.Lock()
	delete(s.byAc, accountID)
	s.mu.Unlock()
}

func (s *Store) Get(accountID string) (Config, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byAc[accountID]
	return c, ok
}

func (s *Store) HasAny() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byAc) > 0
}
```

- [ ] **Step 3: Egress plugin + register on default router**

`egress_plugin.go` parses `args` JSON into `egress.Config`.  
`egress.apply` → `Apply`; `egress.clear` → `Clear` (args `{"account_id":"..."}`).

Package-level `var EgressStore = egress.NewStore()` for MVP; construct router as:

```go
var defaultRouter = NewRouter(EchoPlugin{}, NewEgressPlugin(EgressStore))
```

Business plugins (future TG) MUST call `EgressStore.Get(accountID)` and refuse if missing — document in plugin comment; not required for echo.

- [ ] **Step 4: Tests for apply/clear actions via Router**

Assert `Dispatch` action `egress.apply` with valid JSON → `Ok=true` and store Get works.

- [ ] **Step 5: Run tests + commit**

```bash
cd board-agent; go test ./internal/egress/... ./internal/logic/...
git add board-agent/internal/egress board-agent/internal/logic
git commit -m "feat(board): egress.apply stores SaaS-assigned proxy config"
```

---

### Task 5: SaaS — egress pool + account binding store

**Files:**
- Create: `saas-server/internal/store/egress.go`
- Create: `saas-server/internal/store/account.go`
- Test: `saas-server/internal/store/egress_test.go`, `account_test.go`

**Interfaces:**
- Produces:
  - `EgressStore.Put(Egress{ID, ProxyURL, Region, Healthy bool})`
  - `AccountStore.Upsert(Account{ID, App, DeviceID, EgressID, Tier})`
  - `AccountStore.BindEgress(accountID, egressID) error` — fails if egress missing/unhealthy

- [ ] **Step 1: Failing tests for Put/Bind**

```go
func TestBindRequiresHealthyEgress(t *testing.T) {
	es := store.NewEgressStore()
	as := store.NewAccountStore()
	es.Put(store.Egress{ID: "e1", ProxyURL: "socks5://x", Healthy: false})
	as.Upsert(store.Account{ID: "a1", App: "tg", DeviceID: "d1", Tier: store.TierNew})
	if err := as.BindEgress(es, "a1", "e1"); err == nil {
		t.Fatal("expected error")
	}
	es.Put(store.Egress{ID: "e1", ProxyURL: "socks5://x", Healthy: true})
	if err := as.BindEgress(es, "a1", "e1"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Implement stores**

Tiers as string constants: `new`, `warming`, `normal`, `restricted`, `dead`.  
`BindEgress` sets `Account.EgressID` only when `Healthy`.

- [ ] **Step 3: Run tests + commit**

```bash
cd saas-server; go test ./internal/store/...
git add saas-server/internal/store
git commit -m "feat(saas): in-memory account and egress pool stores"
```

---

### Task 6: SaaS — Policy Allow/Deny + wire into AcceptCommand

**Files:**
- Create: `saas-server/internal/policy/policy.go`
- Create: `saas-server/internal/policy/policy_test.go`
- Create: `saas-server/internal/policy/catalog.go` (action → risk class)
- Modify: `saas-server/internal/logic/device.go`
- Modify: `saas-server/cmd/saas-server/main.go` / `router` to inject stores+policy

**Interfaces:**
- Produces: `type Decision struct { Allow bool; Delay time.Duration; Reason string }`
- Produces: `func (p *Policy) Check(account *store.Account, egress *store.Egress, action string) Decision`
- Rules for MVP:
  - unknown action → Deny `unknown_action`
  - account `dead`/`restricted` → Deny
  - action risk `high` or `sensitive` requires healthy egress binding
  - `echo` is `safe` and allowed without egress (dev only)
  - `egress.apply` / `egress.clear` are platform actions, allowed without chat egress

- [ ] **Step 1: Catalog + failing policy tests**

```go
func TestDenyDeadAccount(t *testing.T) {
	p := policy.New()
	d := p.Check(&store.Account{Tier: store.TierDead}, nil, "echo")
	if d.Allow {
		t.Fatal(d.Reason)
	}
}

func TestSensitiveRequiresEgress(t *testing.T) {
	p := policy.New()
	// register tg.reply_text as safe in catalog for later; for test use a sensitive action
	d := p.Check(&store.Account{Tier: store.TierNormal, EgressID: ""}, nil, "ui.tap")
	if d.Allow {
		t.Fatal("ui.tap needs egress/flow later; MVP deny without binding")
	}
}
```

Define catalog map in `catalog.go`:

```go
var DefaultRisk = map[string]string{
	"echo":          "safe",
	"egress.apply":  "safe",
	"egress.clear":  "safe",
	"ui.tap":        "sensitive",
	"tg.reply_text": "safe",
}
```

- [ ] **Step 2: Implement `Check`**

```go
func (p *Policy) Check(acc *store.Account, eg *store.Egress, action string) Decision {
	risk, ok := DefaultRisk[action]
	if !ok {
		return Decision{Allow: false, Reason: "unknown_action"}
	}
	if acc != nil && (acc.Tier == store.TierDead || acc.Tier == store.TierRestricted) {
		return Decision{Allow: false, Reason: "account_blocked"}
	}
	if risk != "safe" {
		if acc == nil || acc.EgressID == "" || eg == nil || !eg.Healthy {
			return Decision{Allow: false, Reason: "egress_required"}
		}
	}
	return Decision{Allow: true}
}
```

- [ ] **Step 3: Gate `AcceptCommand`**

Before `Hub.SendJSON`, if request includes `account_id` (extend `CommandRequest`):

```go
type CommandRequest struct {
	Action    string `json:"action"`
	Args      string `json:"args"`
	AccountID string `json:"account_id"`
}
```

Load account/egress; `Check`; on deny return `bizerr.New(http.StatusConflict, decision.Reason)`.  
If `account_id` empty and action is `echo`, allow (backward compatible).  
If `account_id` empty and action not platform/echo → Deny `account_required`.

Inject `AccountStore`, `EgressStore`, `*policy.Policy` into `DeviceLogic` via router `Deps`.

- [ ] **Step 4: Run tests**

```bash
cd saas-server; go test ./internal/policy/... ./internal/logic/... ./internal/store/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add saas-server/internal/policy saas-server/internal/logic saas-server/internal/router saas-server/cmd
git commit -m "feat(saas): policy gate on commands with action catalog"
```

---

### Task 7: SaaS — REST for egress pool, bind, and push `egress.apply`

**Files:**
- Create: `saas-server/internal/logic/egress.go`
- Create: `saas-server/internal/handle/egress.go`
- Modify: `saas-server/internal/router/router.go`
- Test: `saas-server/internal/logic/egress_test.go` (table test with fake hub)

**Interfaces:**
- REST:
  - `POST /api/egress` body `{id, proxy_url, region}` → upsert healthy=true
  - `POST /api/accounts` body `{id, app, device_id, tier}`
  - `POST /api/accounts/:id/egress` body `{egress_id}` → bind + if device online send command
- Produces command action `egress.apply` args JSON matching board Task 4

- [ ] **Step 1: Implement `EgressLogic.BindAndPush`**

```go
func (l *EgressLogic) BindAndPush(accountID, egressID string) error {
	if err := l.Accounts.BindEgress(l.Egress, accountID, egressID); err != nil {
		return bizerr.BadRequest(err.Error())
	}
	acc, _ := l.Accounts.Get(accountID)
	eg, _ := l.Egress.Get(egressID)
	dev, ok := l.Store.Get(acc.DeviceID)
	if !ok || !dev.Online || !l.Hub.HasGateway(dev.GatewayID) {
		return nil // bound; push when device connects — MVP return ok without push
	}
	args, _ := json.Marshal(map[string]string{
		"account_id": accountID,
		"egress_id":  egressID,
		"proxy_url":  eg.ProxyURL,
	})
	return l.Hub.SendJSON(dev.GatewayID, cloudwire.TypeCommand, map[string]string{
		"device_id": acc.DeviceID,
		"cmd_id":    uuid.NewString(),
		"action":    "egress.apply",
		"args":      string(args),
	})
}
```

- [ ] **Step 2: Handlers + routes**

```go
r.POST("/api/egress", egressH.Upsert)
r.POST("/api/accounts", egressH.UpsertAccount)
r.POST("/api/accounts/:id/egress", egressH.BindEgress)
r.GET("/api/accounts", egressH.ListAccounts)
r.GET("/api/egress", egressH.ListEgress)
```

Use `response.OK` / `response.Error`.

- [ ] **Step 3: Manual smoke (document in commit body)**

With three services running: create egress → create account → bind → confirm board log shows `egress.apply` and store Get works.  
If full E2E not available in CI, unit-test `BindAndPush` with fake `Hub` that records `SendJSON` calls.

- [ ] **Step 4: Commit**

```bash
git add saas-server/internal/logic/egress.go saas-server/internal/handle/egress.go saas-server/internal/router
git commit -m "feat(saas): egress pool REST and push egress.apply to boards"
```

---

### Task 8: SaaS — handle uplink `event` and `capability`

**Files:**
- Modify: `saas-server/internal/logic/gateway.go`
- Create: `saas-server/internal/store/capability.go` (device → entries map)
- Modify: `saas-server/internal/common/uievent/types.go`
- Test: `saas-server/internal/logic/gateway_event_test.go`

**Interfaces:**
- On `capability`: store entries for `device_id`
- On `event` name `egress.unhealthy` / `account.session_dead`: mark egress unhealthy or account tier `dead`; `UI.Broadcast`
- On `account.challenge` / `account.rate_limited`: set tier `restricted`

- [ ] **Step 1: Add cases in `HandleMessage`**

```go
case cloudv1.MsgType_MSG_TYPE_CAPABILITY:
	// unmarshal device_id, entries; Caps.Put(deviceID, entries)

case cloudv1.MsgType_MSG_TYPE_EVENT:
	// unmarshal name; switch name { ... Policy side effects ...; UI.Broadcast(uievent.AccountRisk, payload) }
```

Add `uievent.AccountRisk = "account_risk"`.

- [ ] **Step 2: Unit test parsing capability + session_dead → tier dead**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(saas): ingest capability and risk events from gateways"
```

---

### Task 9: Docs + spec status alignment

**Files:**
- Modify: `README.md` (short section on group-control foundation APIs)
- Spec already marked 已确认

- [ ] **Step 1: Document new REST endpoints and action names in README**
- [ ] **Step 2: Commit**

```bash
git commit -m "docs: document group-control foundation APIs"
```

---

## Spec coverage checklist (self-review)

| Spec item | Task |
|-----------|------|
| `event` / `capability` transport | 1, 2, 8 |
| Gateway transparent forward | 2 |
| Plugin router + echo | 3 |
| SaaS proxy pool model A + `egress.apply` | 4, 5, 7 |
| Policy Allow/Deny + catalog | 6 |
| Risk events affect account/egress | 8 |
| Board/Gateway infrequent change path | 3 plugins / SaaS catalog |
| Full panel / cold outreach / real TG | Out of scope |

## Placeholder / consistency scan

- CloudSender interface introduced in Task 2 for testability — used consistently.
- `egress.apply` args JSON fields aligned Task 4 ↔ Task 7 (`account_id`, `egress_id`, `proxy_url`).
- Wire types `event` / `capability` derived from enum names Task 1.
- No TBD steps remain in-scope.
