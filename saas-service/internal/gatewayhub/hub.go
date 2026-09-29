package gatewayhub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/tradehub/saas-service/internal/common/uievent"
)

// Hub 管理 Gateway WSS 与待完成指令。
type Hub struct {
	mu       sync.Mutex
	gateways map[string]*websocket.Conn          // gateway_id → 当前连接
	pending  map[string]chan CommandResult       // cmd_id → 等待板子回执
	OnEvent  func(eventType string, payload any) // 通常接到 UIHub.Broadcast（如 command_result）
}

type CommandResult struct {
	DeviceID string `json:"device_id"`
	CmdID    string `json:"cmd_id"`
	Ok       bool   `json:"ok"`
	Message  string `json:"message"`
}

func New() *Hub {
	return &Hub{
		gateways: make(map[string]*websocket.Conn),
		pending:  make(map[string]chan CommandResult),
	}
}

func (h *Hub) SetGateway(id string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.gateways[id]; ok && old != conn {
		_ = old.Close(websocket.StatusNormalClosure, "replaced")
	}
	h.gateways[id] = conn
}

func (h *Hub) RemoveGateway(id string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cur, ok := h.gateways[id]; ok && cur == conn {
		delete(h.gateways, id)
	}
}

func (h *Hub) HasGateway(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.gateways[id]
	return ok
}

func (h *Hub) SendJSON(gatewayID string, typ string, payload any) error {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env, err := json.Marshal(map[string]any{
		"type":    typ,
		"payload": json.RawMessage(rawPayload),
	})
	if err != nil {
		return err
	}
	h.mu.Lock()
	conn, ok := h.gateways[gatewayID]
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("gateway %s not connected", gatewayID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, env)
}

func (h *Hub) RegisterPending(cmdID string) <-chan CommandResult {
	ch := make(chan CommandResult, 1)
	h.mu.Lock()
	h.pending[cmdID] = ch
	h.mu.Unlock()
	return ch
}

func (h *Hub) Complete(res CommandResult) {
	h.mu.Lock()
	ch, ok := h.pending[res.CmdID]
	if ok {
		delete(h.pending, res.CmdID)
	}
	h.mu.Unlock()
	if ok {
		ch <- res
		close(ch)
	}
	if h.OnEvent != nil {
		h.OnEvent(uievent.CommandResult, res)
	}
}

func (h *Hub) CancelPending(cmdID string) bool {
	h.mu.Lock()
	ch, ok := h.pending[cmdID]
	if ok {
		delete(h.pending, cmdID)
	}
	h.mu.Unlock()
	if ok {
		close(ch)
	}
	return ok
}

func (h *Hub) Emit(eventType string, payload any) {
	if h.OnEvent != nil {
		h.OnEvent(eventType, payload)
	}
}
