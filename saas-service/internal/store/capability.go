package store

import "sync"

// CapabilityEntry is one action the board advertises for a device.
type CapabilityEntry struct {
	Action   string `json:"action"`
	RiskHint string `json:"risk_hint"`
}

// CapabilityStore holds the latest capability snapshot per device.
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
