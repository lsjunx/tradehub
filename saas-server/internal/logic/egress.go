package logic

import (
	"encoding/json"

	"github.com/google/uuid"

	"saas-server/internal/common/cloudwire"
	bizerr "saas-server/internal/common/errors"
	"saas-server/internal/store"
)

type egressCommandHub interface {
	HasGateway(id string) bool
	SendJSON(gatewayID string, typ string, payload any) error
}

// EgressLogic 管理代理池、账号绑定出口，并在设备在线时向板子推送 egress.apply。
// 聊天业务流量不经本服务；此处只下发「用哪条代理」的配置。
type EgressLogic struct {
	Store    *store.Store        // 查账号对应设备是否在线、Gateway 是谁
	Hub      egressCommandHub    // 向 Gateway 发 type=command、action=egress.apply
	Accounts *store.AccountStore // 账号 ↔ egress_id 绑定
	Egress   *store.EgressStore  // 代理池（proxy_url、healthy）
}

func NewEgressLogic(
	st *store.Store,
	hub egressCommandHub,
	accounts *store.AccountStore,
	egress *store.EgressStore,
) *EgressLogic {
	return &EgressLogic{
		Store:    st,
		Hub:      hub,
		Accounts: accounts,
		Egress:   egress,
	}
}

type EgressUpsertRequest struct {
	ID       string `json:"id"`
	ProxyURL string `json:"proxy_url"`
	Region   string `json:"region"`
}

func (l *EgressLogic) UpsertEgress(req EgressUpsertRequest) error {
	if req.ID == "" || req.ProxyURL == "" {
		return bizerr.BadRequest("id and proxy_url required")
	}
	l.Egress.Put(store.Egress{
		ID: req.ID, ProxyURL: req.ProxyURL, Region: req.Region, Healthy: true,
	})
	return nil
}

type AccountUpsertRequest struct {
	ID       string `json:"id"`
	App      string `json:"app"`
	DeviceID string `json:"device_id"`
	Tier     string `json:"tier"`
}

func (l *EgressLogic) UpsertAccount(req AccountUpsertRequest) error {
	if req.ID == "" {
		return bizerr.BadRequest("id required")
	}
	if req.Tier == "" {
		req.Tier = store.TierNew
	}
	l.Accounts.Upsert(store.Account{
		ID: req.ID, App: req.App, DeviceID: req.DeviceID, Tier: req.Tier,
	})
	return nil
}

type BindEgressRequest struct {
	EgressID string `json:"egress_id"`
}

func (l *EgressLogic) BindAndPush(accountID, egressID string) error {
	if err := l.Accounts.BindEgress(l.Egress, accountID, egressID); err != nil {
		return bizerr.BadRequest(err.Error())
	}
	acc, _ := l.Accounts.Get(accountID)
	eg, _ := l.Egress.Get(egressID)
	dev, ok := l.Store.Get(acc.DeviceID)
	if !ok || !dev.Online || !l.Hub.HasGateway(dev.GatewayID) {
		return nil
	}
	args, _ := json.Marshal(map[string]string{
		"account_id": accountID,
		"egress_id":  egressID,
		"proxy_url":  eg.ProxyURL,
	})
	return l.Hub.SendJSON(dev.GatewayID, cloudwire.TypeCommand, map[string]string{
		"device_id": acc.DeviceID,
		"cmd_id":    uuid.NewString(),
		"action":    "egress.apply",
		"args":      string(args),
	})
}

func (l *EgressLogic) ListAccounts() []store.Account {
	return l.Accounts.List()
}

func (l *EgressLogic) ListEgress() []store.Egress {
	return l.Egress.List()
}
