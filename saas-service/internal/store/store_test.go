package store

import (
	"testing"
	"time"
)

func TestUpsertAndOffline(t *testing.T) {
	s := New()
	d := s.UpsertRegister("d1", "10.0.0.1", 0, "g1")
	if !d.Online || d.IP != "10.0.0.1" {
		t.Fatalf("%+v", d)
	}
	s.TouchHeartbeat("d1", "ok", "g1")
	got, ok := s.Get("d1")
	if !ok || !got.Online {
		t.Fatal(got)
	}
	off := s.MarkOffline("d1", "g1")
	if off == nil || off.Online {
		t.Fatal(off)
	}
}

func TestMarkOfflineIgnoresOtherGateway(t *testing.T) {
	s := New()
	s.UpsertRegister("d1", "1.1.1.1", 0, "g2")
	if s.MarkOffline("d1", "g1") != nil {
		t.Fatal("should ignore mismatched gateway")
	}
}

func TestMarkStaleOffline(t *testing.T) {
	s := New()
	s.UpsertRegister("d1", "1.1.1.1", 0, "g1")
	s.mu.Lock()
	s.devices["d1"].LastSeen = time.Now().Add(-time.Minute)
	s.mu.Unlock()
	changed := s.MarkStaleOffline(30 * time.Second)
	if len(changed) != 1 || changed[0].Online {
		t.Fatalf("%+v", changed)
	}
}
