package logic_test

import (
	"testing"

	"board-agent/internal/logic"
	boardv1 "board-agent/internal/pb/board/v1"
)

func TestRouterEcho(t *testing.T) {
	r := logic.NewRouter()
	res := r.Dispatch(&boardv1.Command{DeviceId: "d", CmdId: "c1", Action: "echo", Args: "hi"})
	if !res.Ok || res.Message != "hi" {
		t.Fatalf("%+v", res)
	}
}

func TestRouterUnknown(t *testing.T) {
	r := logic.NewRouter()
	res := r.Dispatch(&boardv1.Command{Action: "tg.send_text", CmdId: "c2", DeviceId: "d"})
	if res.Ok {
		t.Fatal("expected fail")
	}
}
