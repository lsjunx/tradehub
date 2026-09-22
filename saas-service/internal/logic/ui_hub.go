package logic

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// UIHub 向浏览器 /ws/ui 推送事件。
type UIHub struct {
	mu       sync.Mutex
	conn     map[*websocket.Conn]struct{}
	lastPush map[string]time.Time // device_id -> 上次推送时间（用于心跳降频）
}

func NewUIHub() *UIHub {
	return &UIHub{
		conn:     make(map[*websocket.Conn]struct{}),
		lastPush: make(map[string]time.Time),
	}
}

func (u *UIHub) Add(c *websocket.Conn) {
	u.mu.Lock()
	u.conn[c] = struct{}{}
	u.mu.Unlock()
}

func (u *UIHub) Remove(c *websocket.Conn) {
	u.mu.Lock()
	delete(u.conn, c)
	u.mu.Unlock()
}

func (u *UIHub) Broadcast(eventType string, payload any) {
	u.broadcastLocked(eventType, payload, "", 0)
}

// BroadcastDevice 按 device 推送；minInterval>0 时同一设备在窗口内只推一次（心跳降频）。
func (u *UIHub) BroadcastDevice(deviceID, eventType string, payload any, minInterval time.Duration) {
	u.broadcastLocked(eventType, payload, deviceID, minInterval)
}

func (u *UIHub) broadcastLocked(eventType string, payload any, deviceID string, minInterval time.Duration) {
	msg, err := json.Marshal(map[string]any{"type": eventType, "payload": payload})
	if err != nil {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if minInterval > 0 && deviceID != "" {
		if t, ok := u.lastPush[deviceID]; ok && time.Since(t) < minInterval {
			return
		}
		u.lastPush[deviceID] = time.Now()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for c := range u.conn {
		_ = c.Write(ctx, websocket.MessageText, msg)
	}
}
