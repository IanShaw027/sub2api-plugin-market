package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(httpStatusByCode(code), Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithStatus 带 HTTP 状态码的错误响应
func ErrorWithStatus(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

// 错误码定义
const (
	ErrCodeInvalidParam   = 1001
	ErrCodeNotFound       = 1002
	ErrCodeInternalError  = 1003
	ErrCodeDatabaseError  = 1004
	ErrCodeStorageError   = 1005
)

// httpStatusByCode 根据业务错误码映射 HTTP 状态码
func httpStatusByCode(code int) int {
	switch code {
	case ErrCodeInvalidParam:
		return http.StatusBadRequest
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeInternalError, ErrCodeDatabaseError, ErrCodeStorageError:
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}
