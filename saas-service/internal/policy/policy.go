// Package policy 在指令真正发往 Gateway 之前做 Allow/Deny。
// 目录见 catalog.go；账号档与出口健康来自 store。
package policy

import (
	"time"

	"github.com/tradehub/saas-service/internal/store"
)

// Decision 策略结果；Delay 预留错峰排队（foundation 阶段尚未使用）。
type Decision struct {
	Allow  bool
	Delay  time.Duration
	Reason string // 拒绝时的原因码，原样返回给 API（如 egress_required）
}

// Policy 无状态校验器；具体规则在 Check / catalog 中。
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
