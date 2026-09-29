package store

import (
	"testing"
	"time"
)

func TestCommandStoreFinish(t *testing.T) {
	s := NewCommandStore()
	s.Put(&CommandRecord{
		CmdID:     "c1",
		DeviceID:  "d1",
		Action:    "echo",
		Status:    CmdAccepted,
		CreatedAt: time.Now(),
	})
	got := s.Finish("c1", CmdSucceeded, true, "hello")
	if got == nil || got.Status != CmdSucceeded || !got.Ok || got.Message != "hello" || got.FinishedAt == nil {
		t.Fatalf("%+v", got)
	}
	q, ok := s.Get("c1")
	if !ok || q.Status != CmdSucceeded {
		t.Fatalf("%v %v", q, ok)
	}
}
