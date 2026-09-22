package logic

import (
	"fmt"
	"net/http"
)

// AppError 带 HTTP 状态码的业务错误。
type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func AsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	ae, ok := err.(*AppError)
	return ae, ok
}

func WrapConflict(err error) *AppError {
	return &AppError{Code: http.StatusConflict, Message: fmt.Sprint(err)}
}
