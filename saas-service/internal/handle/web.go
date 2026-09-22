package handle

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed web/*
var webFS embed.FS

// WebHandle 静态页面。
type WebHandle struct{}

func NewWebHandle() *WebHandle {
	return &WebHandle{}
}

func (h *WebHandle) Index(c *gin.Context) {
	data, err := fs.ReadFile(webFS, "web/index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
