// Package session 维护 Board 设备的 TLS 会话表。
package session

import (
	"net"
	"sync"
)

// BoardConn 一块板子当前的 TLS 连接会话。
type BoardConn struct {
	DeviceID  string
	SessionID string // 每次 Accept 生成，用于区分新旧连接，避免断连误删
	Conn      net.Conn
}

// Registry 按 device_id 索引当前在线 Board。
type Registry struct {
	mu   sync.Mutex
	byID map[string]*BoardConn
}

func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]*BoardConn)}
}

// Put 登记或替换设备连接；若存在旧会话则关闭旧连接。
func (r *Registry) Put(c *BoardConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.byID[c.DeviceID]; ok && old != nil && old.Conn != nil && old.SessionID != c.SessionID {
		_ = old.Conn.Close()
	}
	r.byID[c.DeviceID] = c
}

// Get 查找设备当前会话。
func (r *Registry) Get(deviceID string) (*BoardConn, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byID[deviceID]
	return c, ok
}

// RemoveIfSame 仅当 map 中仍是同一 SessionID 时删除。
// 用于处理「旧连接延迟断开」：重连后新会话已就位时，旧断开回调不得删掉新连接。
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
