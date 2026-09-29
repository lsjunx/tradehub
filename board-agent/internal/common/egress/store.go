package egress

import (
	"errors"
	"sync"
)

var errIncomplete = errors.New("incomplete egress config")

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
		return errIncomplete
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
