package cloudclient

import (
	"encoding/json"
	"fmt"
)

// Envelope Gateway↔Cloud WSS JSON 信封。
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

const (
	TypeGatewayHello    = "gateway_hello"
	TypeDeviceRegister  = "device_register"
	TypeDeviceHeartbeat = "device_heartbeat"
	TypeDeviceOffline   = "device_offline"
	TypeCommand         = "command"
	TypeCommandResult   = "command_result"
)

func MarshalEnvelope(typ string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{Type: typ, Payload: raw})
}

func UnmarshalEnvelope(data []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return env, err
	}
	if env.Type == "" {
		return env, fmt.Errorf("missing type")
	}
	return env, nil
}
