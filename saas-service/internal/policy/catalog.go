package policy

// DefaultRisk maps command actions to risk classes for MVP policy.
var DefaultRisk = map[string]string{
	"echo":          "safe",
	"egress.apply":  "safe",
	"egress.clear":  "safe",
	"ui.tap":        "sensitive",
	"tg.reply_text": "safe",
}

// AllowedWithoutAccount is true for dev echo and platform egress actions.
func AllowedWithoutAccount(action string) bool {
	switch action {
	case "echo", "egress.apply", "egress.clear":
		return true
	default:
		return false
	}
}
