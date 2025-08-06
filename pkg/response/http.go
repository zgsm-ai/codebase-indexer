package response

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	CodeOK = "0"

	MessageOk = "ok"

	CodeError = "-1"
)

type codeMsg struct {
	Code    string
	Message string
}

func (c *codeMsg) Error() string {
	return fmt.Sprintf("code: %s, message: %s", c.Code, c.Message)
}

// NewError creates a new codeMsg.
func NewError(code string, msg string) error {
	return &codeMsg{Code: code, Message: msg}
}

type Response[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
}

func Ok(c *gin.Context) {
	c.JSON(http.StatusOK, wrapResponse(nil))
}

func Error(c *gin.Context, httpStatusCode int, e error) {
	c.JSON(httpStatusCode, wrapResponse(e))
}

func Bytes(c *gin.Context, v []byte) {
	c.Header("Content-Type", "application/octet-stream")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(v)
}

func OkJson(c *gin.Context, v any) {
	c.JSON(http.StatusOK, wrapResponse(v))
}

func wrapResponse(v any) Response[any] {
	var resp Response[any]
	switch data := v.(type) {
	case *codeMsg:
		resp.Code = data.Code
		resp.Message = data.Message
	case codeMsg:
		resp.Code = data.Code
		resp.Message = data.Message
	case error:
		resp.Code = CodeError
		resp.Message = data.Error()
	default:
		resp.Code = CodeOK
		resp.Message = MessageOk
		resp.Success = true
		resp.Data = v
	}

	return resp
}
