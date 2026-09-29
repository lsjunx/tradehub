package logic

import boardv1 "board-agent/internal/pb/board/v1"

type EchoPlugin struct{}

func (EchoPlugin) Actions() []string { return []string{"echo"} }

func (EchoPlugin) Handle(cmd *boardv1.Command) *boardv1.CommandResult {
	return &boardv1.CommandResult{
		DeviceId: cmd.GetDeviceId(),
		CmdId:    cmd.GetCmdId(),
		Ok:       true,
		Message:  cmd.GetArgs(),
	}
}
