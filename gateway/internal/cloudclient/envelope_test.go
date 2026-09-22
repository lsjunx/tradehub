package cloudclient

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	payload := map[string]string{"gateway_id": "g1"}
	raw, err := MarshalEnvelope(TypeGatewayHello, payload)
	if err != nil {
		t.Fatal(err)
	}
	env, err := UnmarshalEnvelope(raw)
	if err != nil || env.Type != TypeGatewayHello {
		t.Fatalf("%+v %v", env, err)
	}
	var got map[string]string
	if err := json.Unmarshal(env.Payload, &got); err != nil || got["gateway_id"] != "g1" {
		t.Fatalf("%v %v", got, err)
	}
}
