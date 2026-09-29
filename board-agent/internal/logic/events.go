package logic

import (
	"sync"

	boardv1 "github.com/tradehub/board-agent/internal/pb/board/v1"
)

var (
	pendingEvents []*boardv1.Event
	pendingMu     sync.Mutex
)

func enqueueEvent(ev *boardv1.Event) {
	pendingMu.Lock()
	pendingEvents = append(pendingEvents, ev)
	pendingMu.Unlock()
}

// TakePendingEvents drains events produced while handling commands (e.g. egress.unhealthy).
func TakePendingEvents() []*boardv1.Event {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	out := pendingEvents
	pendingEvents = nil
	return out
}
