// Package response 提供统一 REST 返回结构。
package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	bizerr "github.com/tradehub/saas-service/internal/common/errors"
)

// Body 统一 REST 返回结构。
type Body struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

func nowMS() int64 {
	return time.Now().UnixMilli()
}

// OK 成功响应：code=0，message=ok。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:      0,
		Message:   "ok",
		Data:      data,
		Timestamp: nowMS(),
	})
}

// Fail 失败响应：data 为 null。
func Fail(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, Body{
		Code:      code,
		Message:   message,
		Data:      nil,
		Timestamp: nowMS(),
	})
}

// Error 将 error 写成统一失败响应；识别 AppError，否则 500。
func Error(c *gin.Context, err error) {
	if ae, ok := bizerr.As(err); ok {
		Fail(c, ae.Code, ae.Code, ae.Message)
		return
	}
	Fail(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
}
