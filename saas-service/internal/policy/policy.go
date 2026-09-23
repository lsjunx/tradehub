package policy

import (
	"time"

	"github.com/local/saas-service/internal/store"
)

// Decision is the outcome of a policy check.
type Decision struct {
	Allow  bool
	Delay  time.Duration
	Reason string
}

// Policy evaluates command actions against account and egress state.
type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (p *Policy) Check(acc *store.Account, eg *store.Egress, action string) Decision {
	risk, ok := DefaultRisk[action]
	if !ok {
		return Decision{Allow: false, Reason: "unknown_action"}
	}
	if acc != nil && (acc.Tier == store.TierDead || acc.Tier == store.TierRestricted) {
		return Decision{Allow: false, Reason: "account_blocked"}
	}
	if risk != "safe" {
		if acc == nil || acc.EgressID == "" || eg == nil || !eg.Healthy {
			return Decision{Allow: false, Reason: "egress_required"}
		}
	}
	return Decision{Allow: true}
}
