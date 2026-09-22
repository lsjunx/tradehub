// Package cloud 负责 Gateway ↔ SaaS 的 WSS 客户端与 JSON 信封。
package cloud

import (
	"encoding/json"
	"fmt"
)

// Envelope SaaS↔Gateway 文本帧 JSON 信封。
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

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
