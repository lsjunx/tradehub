package store

import (
	"sync"
	"time"
)

// Device 内存中的设备视图。
type Device struct {
	DeviceID  string    `json:"device_id"`
	IP        string    `json:"ip"`
	Port      int32     `json:"port"`
	Online    bool      `json:"online"`
	LastSeen  time.Time `json:"last_seen"`
	GatewayID string    `json:"gateway_id"`
	Status    string    `json:"status"`
}

// Store 进程内设备表。
type Store struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

func New() *Store {
	return &Store{devices: make(map[string]*Device)}
}

func (s *Store) UpsertRegister(deviceID, ip string, port int32, gatewayID string) *Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.devices[deviceID]
	if d == nil {
		d = &Device{DeviceID: deviceID}
		s.devices[deviceID] = d
	}
	d.IP = ip
	d.Port = port
	d.GatewayID = gatewayID
	d.Online = true
	d.LastSeen = time.Now()
	d.Status = "registered"
	cp := *d
	return &cp
}

func (s *Store) TouchHeartbeat(deviceID, status, gatewayID string) *Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.devices[deviceID]
	if d == nil {
		d = &Device{DeviceID: deviceID}
		s.devices[deviceID] = d
	}
	d.Online = true
	d.LastSeen = time.Now()
	d.Status = status
	if gatewayID != "" {
		d.GatewayID = gatewayID
	}
	cp := *d
	return &cp
}

func (s *Store) MarkOffline(deviceID, gatewayID string) *Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.devices[deviceID]
	if d == nil {
		return nil
	}
	if gatewayID != "" && d.GatewayID != "" && d.GatewayID != gatewayID {
		return nil
	}
	d.Online = false
	cp := *d
	return &cp
}

func (s *Store) Get(deviceID string) (*Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.devices[deviceID]
	if !ok {
		return nil, false
	}
	cp := *d
	return &cp, true
}

func (s *Store) List() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, *d)
	}
	return out
}

// MarkStaleOffline 将超过 maxAge 未心跳的设备标离线。
func (s *Store) MarkStaleOffline(maxAge time.Duration) []Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var changed []Device
	for _, d := range s.devices {
		if d.Online && now.Sub(d.LastSeen) > maxAge {
			d.Online = false
			changed = append(changed, *d)
		}
	}
	return changed
}
