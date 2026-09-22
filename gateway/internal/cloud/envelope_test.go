package cloud

import (
	"encoding/json"
	"testing"

	cloudv1 "github.com/local/gateway/internal/pb/cloud/v1"
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

func TestParseType(t *testing.T) {
	if ParseType(TypeCommand) != cloudv1.MsgType_MSG_TYPE_COMMAND {
		t.Fatal("parse command")
	}
	if TypeName(cloudv1.MsgType_MSG_TYPE_COMMAND_RESULT) != TypeCommandResult {
		t.Fatal("typename result")
	}
	if TypeGatewayHello != "gateway_hello" {
		t.Fatalf("wire name %q", TypeGatewayHello)
	}
}
