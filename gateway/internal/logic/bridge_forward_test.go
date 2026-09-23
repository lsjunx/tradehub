package logic

import (
	"testing"

	"github.com/local/gateway/internal/cloud"
	boardv1 "github.com/local/gateway/internal/pb/board/v1"
	"github.com/local/gateway/internal/session"
)

type fakeCloud struct {
	lastType    string
	lastPayload map[string]any
}

func (f *fakeCloud) Send(typ string, payload any) error {
	f.lastType = typ
	if m, ok := payload.(map[string]any); ok {
		f.lastPayload = m
	}
	return nil
}

func TestHandleBoardEnvelope_EventForwards(t *testing.T) {
	fc := &fakeCloud{}
	b := &Bridge{
		GatewayID: "g1",
		Registry:  session.NewRegistry(),
		Cloud:     fc,
	}
	var dev string
	env := &boardv1.Envelope{
		Payload: &boardv1.Envelope_Event{Event: &boardv1.Event{
			EventId:     "e1",
			DeviceId:    "d1",
			Name:        "egress.unhealthy",
			PayloadJson: `{"reason":"timeout"}`,
			TsUnixMs:    12345,
		}},
	}
	if err := b.HandleBoardEnvelope(nil, "sess", &dev, env); err != nil {
		t.Fatal(err)
	}
	if fc.lastType != cloud.TypeEvent {
		t.Fatalf("type %q want %q", fc.lastType, cloud.TypeEvent)
	}
	if fc.lastPayload["gateway_id"] != "g1" {
		t.Fatalf("gateway_id %v", fc.lastPayload["gateway_id"])
	}
	if fc.lastPayload["name"] != "egress.unhealthy" || fc.lastPayload["device_id"] != "d1" {
		t.Fatalf("%+v", fc.lastPayload)
	}
	if fc.lastPayload["event_id"] != "e1" || fc.lastPayload["payload_json"] != `{"reason":"timeout"}` {
		t.Fatalf("%+v", fc.lastPayload)
	}
	if fc.lastPayload["ts_unix_ms"] != int64(12345) {
		t.Fatalf("ts %v", fc.lastPayload["ts_unix_ms"])
	}
}

func TestHandleBoardEnvelope_CapabilityForwards(t *testing.T) {
	fc := &fakeCloud{}
	b := &Bridge{
		GatewayID: "gw-2",
		Registry:  session.NewRegistry(),
		Cloud:     fc,
	}
	var dev string
	env := &boardv1.Envelope{
		Payload: &boardv1.Envelope_Capability{Capability: &boardv1.Capability{
			DeviceId: "d2",
			Entries: []*boardv1.CapabilityEntry{
				{Action: "echo", RiskHint: "safe"},
				{Action: "egress.apply", RiskHint: "safe"},
			},
		}},
	}
	if err := b.HandleBoardEnvelope(nil, "sess", &dev, env); err != nil {
		t.Fatal(err)
	}
	if fc.lastType != cloud.TypeCapability {
		t.Fatalf("type %q want %q", fc.lastType, cloud.TypeCapability)
	}
	if fc.lastPayload["gateway_id"] != "gw-2" || fc.lastPayload["device_id"] != "d2" {
		t.Fatalf("%+v", fc.lastPayload)
	}
	entries, ok := fc.lastPayload["entries"].([]map[string]string)
	if !ok || len(entries) != 2 {
		t.Fatalf("entries %T %+v", fc.lastPayload["entries"], fc.lastPayload["entries"])
	}
	if entries[0]["action"] != "echo" || entries[1]["risk_hint"] != "safe" {
		t.Fatalf("%+v", entries)
	}
}
