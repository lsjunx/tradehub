package agent

import (
	"testing"

	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

func TestHandleCommand_Echo(t *testing.T) {
	res := HandleCommand(&boardv1.Command{DeviceId: "d", CmdId: "c1", Action: "echo", Args: "hi"})
	if !res.Ok || res.Message != "hi" || res.CmdId != "c1" {
		t.Fatalf("%+v", res)
	}
	res = HandleCommand(&boardv1.Command{Action: "upgrade"})
	if res.Ok {
		t.Fatal("upgrade should fail in v1")
	}
}
