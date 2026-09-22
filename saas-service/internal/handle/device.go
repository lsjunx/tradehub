package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/local/saas-service/internal/logic"
)

// APIError 统一 REST 错误体。
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeErr(c *gin.Context, err error) {
	if ae, ok := logic.AsAppError(err); ok {
		c.JSON(ae.Code, APIError{Code: ae.Code, Message: ae.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, APIError{Code: http.StatusInternalServerError, Message: err.Error()})
}

// DeviceHandle 设备相关 HTTP 接口。
type DeviceHandle struct {
	Logic *logic.DeviceLogic
}

func NewDeviceHandle(l *logic.DeviceLogic) *DeviceHandle {
	return &DeviceHandle{Logic: l}
}

func (h *DeviceHandle) ListDevices(c *gin.Context) {
	c.JSON(http.StatusOK, h.Logic.ListDevices())
}

func (h *DeviceHandle) GetDevice(c *gin.Context) {
	dev, err := h.Logic.GetDevice(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dev)
}

func (h *DeviceHandle) SendCommand(c *gin.Context) {
	deviceID := c.Param("id")
	var req logic.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIError{Code: http.StatusBadRequest, Message: "bad json"})
		return
	}
	res, err := h.Logic.AcceptCommand(deviceID, req)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusAccepted, res)
}

func (h *DeviceHandle) GetCommand(c *gin.Context) {
	rec, err := h.Logic.GetCommand(c.Param("cmd_id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, rec)
}
