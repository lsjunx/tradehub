package logic

import (
	"testing"
	"time"

	"saas-server/internal/common/uievent"
)

type stubPayload struct {
	DeviceID string `json:"device_id"`
}

func TestBroadcastDevice_Throttle(t *testing.T) {
	u := NewUIHub()
	p1 := stubPayload{DeviceID: "d1"}
	u.BroadcastDevice("d1", uievent.DeviceUpdated, p1, 5*time.Second)
	u.mu.Lock()
	first := u.lastPush["d1"]
	u.mu.Unlock()
	if first.IsZero() {
		t.Fatal("expected lastPush set")
	}

	u.BroadcastDevice("d1", uievent.DeviceUpdated, p1, 5*time.Second)
	u.mu.Lock()
	second := u.lastPush["d1"]
	u.mu.Unlock()
	if !second.Equal(first) {
		t.Fatal("throttled push should not update lastPush")
	}

	time.Sleep(10 * time.Millisecond)
	u.BroadcastDevice("d1", uievent.DeviceUpdated, p1, 0)
	u.mu.Lock()
	u.lastPush["d1"] = time.Now().Add(-6 * time.Second)
	u.mu.Unlock()
	u.BroadcastDevice("d1", uievent.DeviceUpdated, p1, 5*time.Second)
	u.mu.Lock()
	third := u.lastPush["d1"]
	u.mu.Unlock()
	if !third.After(first) {
		t.Fatal("after interval should push again")
	}
}
