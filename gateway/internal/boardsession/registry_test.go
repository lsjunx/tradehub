package boardsession

import "testing"

func TestRemoveIfSame_IgnoresStaleDisconnect(t *testing.T) {
	r := NewRegistry()
	old := &BoardConnection{DeviceID: "d1", SessionID: "A", Conn: nil}
	neu := &BoardConnection{DeviceID: "d1", SessionID: "B", Conn: nil}
	r.Put(old)
	r.Put(neu)
	if r.RemoveIfSame("d1", "A") {
		t.Fatal("stale should not remove")
	}
	got, ok := r.Get("d1")
	if !ok || got.SessionID != "B" {
		t.Fatalf("want B, got %+v ok=%v", got, ok)
	}
	if !r.RemoveIfSame("d1", "B") {
		t.Fatal("current should remove")
	}
}
