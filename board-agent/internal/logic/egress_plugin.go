package logic

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/local/board-agent/internal/egress"
	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

// EgressPlugin handles SaaS-pushed proxy bindings (egress.apply / egress.clear).
// Future business plugins (e.g. TG) MUST call EgressStore.Get(accountID) and refuse if missing.
type EgressPlugin struct {
	store *egress.Store
}

func NewEgressPlugin(store *egress.Store) EgressPlugin {
	return EgressPlugin{store: store}
}

func (EgressPlugin) Actions() []string { return []string{"egress.apply", "egress.clear"} }

func (p EgressPlugin) Handle(cmd *boardv1.Command) *boardv1.CommandResult {
	base := &boardv1.CommandResult{
		DeviceId: cmd.GetDeviceId(),
		CmdId:    cmd.GetCmdId(),
	}
	switch cmd.GetAction() {
	case "egress.apply":
		var cfg egress.Config
		if err := json.Unmarshal([]byte(cmd.GetArgs()), &cfg); err != nil {
			emitEgressUnhealthy(cmd.GetDeviceId(), err.Error())
			base.Ok = false
			base.Message = "invalid egress.apply args: " + err.Error()
			return base
		}
		if err := p.store.Apply(cfg); err != nil {
			base.Ok = false
			base.Message = err.Error()
			return base
		}
		base.Ok = true
		base.Message = "applied"
		return base
	case "egress.clear":
		var body struct {
			AccountID string `json:"account_id"`
		}
		if err := json.Unmarshal([]byte(cmd.GetArgs()), &body); err != nil {
			base.Ok = false
			base.Message = "invalid egress.clear args: " + err.Error()
			return base
		}
		p.store.Clear(body.AccountID)
		base.Ok = true
		base.Message = "cleared"
		return base
	default:
		base.Ok = false
		base.Message = "internal: unknown egress action"
		return base
	}
}

func emitEgressUnhealthy(deviceID, detail string) {
	payload, _ := json.Marshal(map[string]string{"detail": detail})
	enqueueEvent(&boardv1.Event{
		EventId:     uuid.NewString(),
		DeviceId:    deviceID,
		Name:        "egress.unhealthy",
		PayloadJson: string(payload),
		TsUnixMs:    time.Now().UnixMilli(),
	})
}
