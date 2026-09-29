package egress_test

import (
	"testing"

	"github.com/tradehub/board-agent/internal/egress"
)

func TestApplyAndGet(t *testing.T) {
	s := egress.NewStore()
	cfg := egress.Config{AccountID: "a1", EgressID: "e1", ProxyURL: "socks5://127.0.0.1:1080"}
	if err := s.Apply(cfg); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get("a1")
	if !ok || got.EgressID != "e1" {
		t.Fatalf("%+v", got)
	}
	s.Clear("a1")
	if _, ok := s.Get("a1"); ok {
		t.Fatal("cleared")
	}
}
