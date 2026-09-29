package store

import "sync"

// CapabilityEntry 板子声明的一项可执行能力（action + 风险提示）。
type CapabilityEntry struct {
	Action   string `json:"action"`
	RiskHint string `json:"risk_hint"` // safe | sensitive | high（仅提示，门禁以 Policy 目录为准）
}

// CapabilityStore 按 device_id 保存板子最近一次上报的 capability。
// 用于面板显隐按钮、后续「只允许已声明 action」等；由 Gateway 上行写入。
type CapabilityStore struct {
	mu       sync.RWMutex
	byDevice map[string][]CapabilityEntry
}

func NewCapabilityStore() *CapabilityStore {
	return &CapabilityStore{byDevice: make(map[string][]CapabilityEntry)}
}

func (s *CapabilityStore) Put(deviceID string, entries []CapabilityEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]CapabilityEntry, len(entries))
	copy(cp, entries)
	s.byDevice[deviceID] = cp
}

func (s *CapabilityStore) Get(deviceID string) ([]CapabilityEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.byDevice[deviceID]
	if !ok {
		return nil, false
	}
	out := make([]CapabilityEntry, len(e))
	copy(out, e)
	return out, true
}
