package store

import (
	"errors"
	"sync"
)

var (
	ErrEgressNotFound  = errors.New("egress not found")
	ErrEgressUnhealthy = errors.New("egress unhealthy")
	ErrAccountNotFound = errors.New("account not found")
)

const (
	TierNew        = "new"
	TierWarming    = "warming"
	TierNormal     = "normal"
	TierRestricted = "restricted"
	TierDead       = "dead"
)

// Account binds a chat/app identity to a device and optional egress.
type Account struct {
	ID       string `json:"id"`
	App      string `json:"app"`
	DeviceID string `json:"device_id"`
	EgressID string `json:"egress_id"`
	Tier     string `json:"tier"`
}

// AccountStore holds in-memory account records.
type AccountStore struct {
	mu   sync.RWMutex
	byID map[string]*Account
}

func NewAccountStore() *AccountStore {
	return &AccountStore{byID: make(map[string]*Account)}
}

func (s *AccountStore) Upsert(acc Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := acc
	s.byID[acc.ID] = &cp
}

func (s *AccountStore) Get(id string) (Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.byID[id]
	if !ok {
		return Account{}, false
	}
	return *a, true
}

func (s *AccountStore) List() []Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Account, 0, len(s.byID))
	for _, a := range s.byID {
		out = append(out, *a)
	}
	return out
}

func (s *AccountStore) BindEgress(es *EgressStore, accountID, egressID string) error {
	eg, ok := es.Get(egressID)
	if !ok {
		return ErrEgressNotFound
	}
	if !eg.Healthy {
		return ErrEgressUnhealthy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.byID[accountID]
	if !ok {
		return ErrAccountNotFound
	}
	a.EgressID = egressID
	return nil
}
