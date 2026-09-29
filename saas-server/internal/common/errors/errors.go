// Package errors 提供业务错误类型。
package errors

import (
	"fmt"
	"net/http"
)

// AppError 带业务/HTTP 状态码的错误。
type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func As(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	ae, ok := err.(*AppError)
	return ae, ok
}

func Conflict(err error) *AppError {
	return &AppError{Code: http.StatusConflict, Message: fmt.Sprint(err)}
}

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, msg)
}

func NotFound(msg string) *AppError {
	return New(http.StatusNotFound, msg)
}
