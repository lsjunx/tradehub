package logic

import (
	"encoding/json"
	"testing"

	"github.com/local/saas-service/internal/common/cloudwire"
	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/store"
)

func TestHandleMessage_CapabilityStoresEntries(t *testing.T) {
	caps := store.NewCapabilityStore()
	l := NewGatewayLogic(store.New(), gatewayhub.New(), NewUIHub(), caps, store.NewAccountStore(), store.NewEgressStore())
	payload, _ := json.Marshal(map[string]any{
		"device_id": "d1",
		"entries": []map[string]string{
			{"action": "echo", "risk_hint": "safe"},
		},
	})
	l.HandleMessage(cloudwire.TypeCapability, payload)
	got, ok := caps.Get("d1")
	if !ok || len(got) != 1 || got[0].Action != "echo" || got[0].RiskHint != "safe" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestHandleMessage_SessionDeadSetsTierDead(t *testing.T) {
	accounts := store.NewAccountStore()
	accounts.Upsert(store.Account{ID: "a1", App: "tg", DeviceID: "d1", Tier: store.TierNormal})
	l := NewGatewayLogic(store.New(), gatewayhub.New(), NewUIHub(), store.NewCapabilityStore(), accounts, store.NewEgressStore())
	payload, _ := json.Marshal(map[string]any{
		"name":         "account.session_dead",
		"device_id":    "d1",
		"payload_json": `{"account_id":"a1"}`,
	})
	l.HandleMessage(cloudwire.TypeEvent, payload)
	acc, ok := accounts.Get("a1")
	if !ok || acc.Tier != store.TierDead {
		t.Fatalf("tier %+v ok=%v", acc, ok)
	}
}
