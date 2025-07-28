package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse 统一API响应格式
type APIResponse struct {
	Success   bool        `json:"success"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success:   true,
		Code:      "0",
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Fail 返回失败响应
func Fail(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success:   false,
		Code:      "4001",
		Message:   message,
		Data:      nil,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// FailWithCode 返回指定错误码的失败响应
func FailWithCode(c *gin.Context, code string, message string, statusCode int) {
	c.JSON(statusCode, APIResponse{
		Success:   false,
		Code:      code,
		Message:   message,
		Data:      nil,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// NotFound 返回404响应
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	FailWithCode(c, "4041", message, http.StatusNotFound)
}

// InternalError 返回500响应
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	FailWithCode(c, "5001", message, http.StatusInternalServerError)
}

// ValidationError 返回参数验证错误
func ValidationError(c *gin.Context, message string) {
	if message == "" {
		message = "validation failed"
	}
	FailWithCode(c, "4001", message, http.StatusBadRequest)
}

// PaginatedResponse 生成分页响应
func PaginatedResponse(c *gin.Context, data interface{}, page, size, total int) {
	totalPage := (total + size - 1) / size
	response := map[string]interface{}{
		"data": data,
		"pagination": map[string]interface{}{
			"page":      page,
			"size":      size,
			"total":     total,
			"totalPage": totalPage,
		},
	}
	Success(c, response)
}
