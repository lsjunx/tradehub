package boardsession

import (
	"net"
	"sync"
)

// BoardConnection 表示一块板子的当前 TLS 会话。
type BoardConnection struct {
	DeviceID  string
	SessionID string
	Conn      net.Conn
}

// Registry 按 device_id 维护当前会话，断连时用 SessionID 防止误删新连接。
type Registry struct {
	mu   sync.Mutex
	byID map[string]*BoardConnection
}

func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]*BoardConnection)}
}

// Put 写入或替换设备连接；若替换则关闭旧 Conn。
func (r *Registry) Put(c *BoardConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.byID[c.DeviceID]; ok && old != nil && old.Conn != nil && old.SessionID != c.SessionID {
		_ = old.Conn.Close()
	}
	r.byID[c.DeviceID] = c
}

func (r *Registry) Get(deviceID string) (*BoardConnection, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byID[deviceID]
	return c, ok
}

// RemoveIfSame 仅当 map 中会话与 sessionID 一致时删除，返回是否删除。
func (r *Registry) RemoveIfSame(deviceID, sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.byID[deviceID]
	if !ok || cur.SessionID != sessionID {
		return false
	}
	delete(r.byID, deviceID)
	return true
}
