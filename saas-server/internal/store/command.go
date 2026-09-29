package store

import (
	"sync"
	"time"
)

// 指令状态，供 REST 查询与前端轮询兜底。
const (
	CmdAccepted  = "accepted"
	CmdSucceeded = "succeeded"
	CmdFailed    = "failed"
	CmdTimeout   = "timeout"
)

// CommandRecord 一次指令的生命周期记录。
type CommandRecord struct {
	CmdID      string     `json:"cmd_id"`
	DeviceID   string     `json:"device_id"`
	Action     string     `json:"action"`
	Args       string     `json:"args"`
	Status     string     `json:"status"`
	Ok         bool       `json:"ok"`
	Message    string     `json:"message"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// CommandStore 进程内指令记录（后续可换 Redis/DB）。
type CommandStore struct {
	mu   sync.RWMutex
	byID map[string]*CommandRecord
}

func NewCommandStore() *CommandStore {
	return &CommandStore{byID: make(map[string]*CommandRecord)}
}

func (s *CommandStore) Put(rec *CommandRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *rec
	s.byID[rec.CmdID] = &cp
}

func (s *CommandStore) Get(cmdID string) (*CommandRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.byID[cmdID]
	if !ok {
		return nil, false
	}
	cp := *rec
	return &cp, true
}

func (s *CommandStore) Finish(cmdID string, status string, ok bool, message string) *CommandRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, exists := s.byID[cmdID]
	if !exists {
		return nil
	}
	now := time.Now()
	rec.Status = status
	rec.Ok = ok
	rec.Message = message
	rec.FinishedAt = &now
	cp := *rec
	return &cp
}

// FinishIfAccepted 仅当仍为 accepted 时结算，避免超时与成功结果互相覆盖。
func (s *CommandStore) FinishIfAccepted(cmdID string, status string, ok bool, message string) *CommandRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, exists := s.byID[cmdID]
	if !exists || rec.Status != CmdAccepted {
		return nil
	}
	now := time.Now()
	rec.Status = status
	rec.Ok = ok
	rec.Message = message
	rec.FinishedAt = &now
	cp := *rec
	return &cp
}
