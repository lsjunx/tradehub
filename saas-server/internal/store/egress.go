package store

import "sync"

// Egress 是 SaaS 代理池中的一条出口线路（板子上 App 应走此代理出海）。
// TG/WA 看见的是代理出口 IP，不是板子机房 IP，也不是本 SaaS 公网 IP。
type Egress struct {
	ID       string `json:"id"`
	ProxyURL string `json:"proxy_url"` // 下发给板子的代理地址（含鉴权时可写进 URL）
	Region   string `json:"region"`
	Healthy  bool   `json:"healthy"` // false 时禁止新绑定，Policy 也拒敏感动作
}

// EgressStore 进程内代理池。
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

func (s *EgressStore) SetHealthy(id string, healthy bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	eg, ok := s.byID[id]
	if !ok {
		return ErrEgressNotFound
	}
	eg.Healthy = healthy
	s.byID[id] = eg
	return nil
}

func (s *EgressStore) List() []Egress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Egress, 0, len(s.byID))
	for _, eg := range s.byID {
		out = append(out, eg)
	}
	return out
}
