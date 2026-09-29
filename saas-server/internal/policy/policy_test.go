package policy_test

import (
	"testing"

	"saas-server/internal/policy"
	"saas-server/internal/store"
)

func TestDenyDeadAccount(t *testing.T) {
	p := policy.New()
	d := p.Check(&store.Account{Tier: store.TierDead}, nil, "echo")
	if d.Allow {
		t.Fatal(d.Reason)
	}
}

func TestSensitiveRequiresEgress(t *testing.T) {
	p := policy.New()
	d := p.Check(&store.Account{Tier: store.TierNormal, EgressID: ""}, nil, "ui.tap")
	if d.Allow {
		t.Fatal("ui.tap needs egress/flow later; MVP deny without binding")
	}
}

func TestUnknownActionDenied(t *testing.T) {
	p := policy.New()
	d := p.Check(&store.Account{Tier: store.TierNormal}, nil, "nope.action")
	if d.Allow || d.Reason != "unknown_action" {
		t.Fatalf("got allow=%v reason=%q", d.Allow, d.Reason)
	}
}
