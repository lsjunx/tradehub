package logic

import boardv1 "board-agent/internal/pb/board/v1"

type Router struct {
	byAction map[string]Plugin
}

func NewRouter(plugins ...Plugin) *Router {
	if len(plugins) == 0 {
		plugins = []Plugin{EchoPlugin{}}
	}
	r := &Router{byAction: map[string]Plugin{}}
	for _, p := range plugins {
		for _, a := range p.Actions() {
			r.byAction[a] = p
		}
	}
	return r
}

func (r *Router) Dispatch(cmd *boardv1.Command) *boardv1.CommandResult {
	p, ok := r.byAction[cmd.GetAction()]
	if !ok {
		return &boardv1.CommandResult{
			DeviceId: cmd.GetDeviceId(),
			CmdId:    cmd.GetCmdId(),
			Ok:       false,
			Message:  "unsupported action: " + cmd.GetAction(),
		}
	}
	return p.Handle(cmd)
}

func (r *Router) CapabilityEntries() []*boardv1.CapabilityEntry {
	var out []*boardv1.CapabilityEntry
	seen := map[string]bool{}
	for action := range r.byAction {
		if seen[action] {
			continue
		}
		seen[action] = true
		hint := "safe"
		if action != "echo" && action != "egress.apply" && action != "egress.clear" {
			hint = "sensitive"
		}
		out = append(out, &boardv1.CapabilityEntry{Action: action, RiskHint: hint})
	}
	return out
}
