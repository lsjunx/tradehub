package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tradehub/saas-service/internal/common/response"
	"github.com/tradehub/saas-service/internal/logic"
)

// EgressHandle egress pool and account binding REST.
type EgressHandle struct {
	Logic *logic.EgressLogic
}

func NewEgressHandle(l *logic.EgressLogic) *EgressHandle {
	return &EgressHandle{Logic: l}
}

func (h *EgressHandle) Upsert(c *gin.Context) {
	var req logic.EgressUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "bad json")
		return
	}
	if err := h.Logic.UpsertEgress(req); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *EgressHandle) UpsertAccount(c *gin.Context) {
	var req logic.AccountUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "bad json")
		return
	}
	if err := h.Logic.UpsertAccount(req); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *EgressHandle) BindEgress(c *gin.Context) {
	var req logic.BindEgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "bad json")
		return
	}
	if err := h.Logic.BindAndPush(c.Param("id"), req.EgressID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *EgressHandle) ListAccounts(c *gin.Context) {
	response.OK(c, h.Logic.ListAccounts())
}

func (h *EgressHandle) ListEgress(c *gin.Context) {
	response.OK(c, h.Logic.ListEgress())
}
