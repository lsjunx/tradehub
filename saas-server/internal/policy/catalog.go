package policy

// DefaultRisk 动作风险目录（发令前 Policy.Check 查表）。
//
//	safe      — 客服类 / 平台动作；无绑定出口时也可过（业务 TG 日后应收紧）
//	sensitive — 须账号已绑健康 egress
//	high      — 预留给冷触达等高危（本阶段 catalog 尚未挂 high）
//
// 新增 App 动作：在此登记风险级，并在板子插件里实现同名 action。
var DefaultRisk = map[string]string{
	"echo":          "safe",
	"egress.apply":  "safe", // 平台：下发代理配置
	"egress.clear":  "safe",
	"ui.tap":        "sensitive",
	"tg.reply_text": "safe", // 占位；真实业务前建议改为须 egress
}

// AllowedWithoutAccount 允许请求不带 account_id 的动作（联调 echo、平台配出口）。
func AllowedWithoutAccount(action string) bool {
	switch action {
	case "echo", "egress.apply", "egress.clear":
		return true
	default:
		return false
	}
}
