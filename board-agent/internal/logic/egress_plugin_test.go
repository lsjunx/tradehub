package logic_test

import (
	"testing"

	"github.com/local/board-agent/internal/egress"
	"github.com/local/board-agent/internal/logic"
	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

func TestRouterEgressApply(t *testing.T) {
	s := egress.NewStore()
	r := logic.NewRouter(logic.EchoPlugin{}, logic.NewEgressPlugin(s))
	args := `{"account_id":"a1","egress_id":"e1","proxy_url":"socks5://127.0.0.1:1080"}`
	res := r.Dispatch(&boardv1.Command{
		DeviceId: "d1",
		CmdId:    "c1",
		Action:   "egress.apply",
		Args:     args,
	})
	if !res.Ok {
		t.Fatalf("%+v", res)
	}
	got, ok := s.Get("a1")
	if !ok || got.ProxyURL != "socks5://127.0.0.1:1080" {
		t.Fatalf("%+v", got)
	}
}

func TestRouterEgressClear(t *testing.T) {
	s := egress.NewStore()
	r := logic.NewRouter(logic.NewEgressPlugin(s))
	_ = s.Apply(egress.Config{AccountID: "a1", EgressID: "e1", ProxyURL: "socks5://x"})
	res := r.Dispatch(&boardv1.Command{
		DeviceId: "d1",
		CmdId:    "c2",
		Action:   "egress.clear",
		Args:     `{"account_id":"a1"}`,
	})
	if !res.Ok {
		t.Fatalf("%+v", res)
	}
	if _, ok := s.Get("a1"); ok {
		t.Fatal("expected cleared")
	}
}

func TestEgressApplyInvalidJSONEmitsUnhealthy(t *testing.T) {
	s := egress.NewStore()
	r := logic.NewRouter(logic.NewEgressPlugin(s))
	logic.TakePendingEvents()
	res := r.Dispatch(&boardv1.Command{
		DeviceId: "d1",
		CmdId:    "c3",
		Action:   "egress.apply",
		Args:     `{bad`,
	})
	if res.Ok {
		t.Fatal("expected fail")
	}
	evs := logic.TakePendingEvents()
	if len(evs) != 1 || evs[0].GetName() != "egress.unhealthy" || evs[0].GetDeviceId() != "d1" {
		t.Fatalf("%+v", evs)
	}
}
