package store

import "sync"

// Egress is a SaaS-managed proxy endpoint in the pool.
type Egress struct {
	ID       string
	ProxyURL string
	Region   string
	Healthy  bool
}

// EgressStore holds the in-memory egress pool (later DB/Redis).
type EgressStore struct {
	mu   sync.RWMutex
	byID map[string]Egress
}

func NewEgressStore() *EgressStore {
	return &EgressStore{byID: make(map[string]Egress)}
}

func (s *EgressStore) Put(eg Egress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[eg.ID] = eg
}

func (s *EgressStore) Get(id string) (Egress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	eg, ok := s.byID[id]
	return eg, ok
}
