package logic

import (
	"testing"
	"time"
)

type stubPayload struct {
	DeviceID string `json:"device_id"`
}

func TestBroadcastDevice_Throttle(t *testing.T) {
	u := NewUIHub()
	p1 := stubPayload{DeviceID: "d1"}
	// 无连接时也应更新 lastPush，否则测不到节流逻辑
	u.BroadcastDevice("d1", "device_updated", p1, 5*time.Second)
	u.mu.Lock()
	first := u.lastPush["d1"]
	u.mu.Unlock()
	if first.IsZero() {
		t.Fatal("expected lastPush set")
	}

	u.BroadcastDevice("d1", "device_updated", p1, 5*time.Second)
	u.mu.Lock()
	second := u.lastPush["d1"]
	u.mu.Unlock()
	if !second.Equal(first) {
		t.Fatal("throttled push should not update lastPush time window incorrectly; should skip")
	}

	time.Sleep(10 * time.Millisecond)
	u.BroadcastDevice("d1", "device_updated", p1, 0)
	u.mu.Lock()
	// minInterval=0 不走节流分支，lastPush 可不更新；再测强制过期
	u.lastPush["d1"] = time.Now().Add(-6 * time.Second)
	u.mu.Unlock()
	u.BroadcastDevice("d1", "device_updated", p1, 5*time.Second)
	u.mu.Lock()
	third := u.lastPush["d1"]
	u.mu.Unlock()
	if !third.After(first) {
		t.Fatal("after interval should push again")
	}
}
