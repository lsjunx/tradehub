// Package handle Board/Gateway/UI 的 HTTP 与 WebSocket 入口。
package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"saas-server/internal/common/response"
	"saas-server/internal/logic"
)

// DeviceHandle 设备相关 HTTP 接口。
type DeviceHandle struct {
	Logic *logic.DeviceLogic
}

func NewDeviceHandle(l *logic.DeviceLogic) *DeviceHandle {
	return &DeviceHandle{Logic: l}
}

func (h *DeviceHandle) ListDevices(c *gin.Context) {
	response.OK(c, h.Logic.ListDevices())
}

func (h *DeviceHandle) GetDevice(c *gin.Context) {
	dev, err := h.Logic.GetDevice(c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dev)
}

func (h *DeviceHandle) SendCommand(c *gin.Context) {
	deviceID := c.Param("id")
	var req logic.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "bad json")
		return
	}
	res, err := h.Logic.AcceptCommand(deviceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

func (h *DeviceHandle) GetCommand(c *gin.Context) {
	rec, err := h.Logic.GetCommand(c.Param("cmd_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rec)
}
