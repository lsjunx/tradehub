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
	TierNew        = "new"        // 新号：几乎只许安全动作
	TierWarming    = "warming"    // 养号中
	TierNormal     = "normal"     // 正常运营
	TierRestricted = "restricted" // 风控限制（challenge / 限流等）
	TierDead       = "dead"       // 会话失效等，禁止业务下发
)

// Account 聊天账号画像：挂在哪台设备、走哪条出口、当前风控档。
type Account struct {
	ID       string `json:"id"`
	App      string `json:"app"`       // 如 tg / wa
	DeviceID string `json:"device_id"` // 宿主板子
	EgressID string `json:"egress_id"` // 绑定的代理线路；空表示未绑出口
	Tier     string `json:"tier"`
}

// AccountStore 进程内账号表（绑定出口须经 BindEgress，且要求线路 healthy）。
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
	if prev, ok := s.byID[acc.ID]; ok {
		if acc.App == "" {
			acc.App = prev.App
		}
		if acc.DeviceID == "" {
			acc.DeviceID = prev.DeviceID
		}
		if acc.EgressID == "" {
			acc.EgressID = prev.EgressID
		}
		if acc.Tier == "" {
			acc.Tier = prev.Tier
		}
	}
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

func (s *AccountStore) SetTier(accountID, tier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.byID[accountID]
	if !ok {
		return ErrAccountNotFound
	}
	a.Tier = tier
	return nil
}

// ListByDevice returns accounts bound to deviceID.
func (s *AccountStore) ListByDevice(deviceID string) []Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Account
	for _, a := range s.byID {
		if a.DeviceID == deviceID {
			out = append(out, *a)
		}
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
