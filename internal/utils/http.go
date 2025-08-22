package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// status code
const (
	StatusCodeUnauthorized       = "401" // HTTP 401 Unauthorized
	StatusCodeForbidden          = "403" // HTTP 403 Forbidden
	StatusCodePageNotFound       = "404" // HTTP 404 Not Found
	StatusCodeTooManyRequests    = "429" // HTTP 429 Too Many Requests
	StatusCodeServiceUnavailable = "503" // HTTP 503 Service Unavailable
)

const (
	BaseWriteTimeoutSeconds = 60
)

// IsAbortRetryError checks if the error indicates we should abort retrying
func IsAbortRetryError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	return strings.Contains(errorStr, StatusCodeUnauthorized) ||
		strings.Contains(errorStr, StatusCodePageNotFound) ||
		strings.Contains(errorStr, StatusCodeTooManyRequests) ||
		strings.Contains(errorStr, StatusCodeServiceUnavailable)
}

func IsUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodeUnauthorized)
}

func IsForbiddenError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodeForbidden)
}

func IsPageNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodePageNotFound)
}

func IsTooManyRequestsError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodeTooManyRequests)
}

func IsServiceUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodeServiceUnavailable)
}

// HTTPRequest 通用HTTP请求结构
type HTTPRequest struct {
	Method      string            // HTTP方法：GET, POST, PUT, DELETE
	URL         string            // 请求URL
	Headers     map[string]string // 请求头
	QueryParams map[string]string // 查询参数
	Body        interface{}       // 请求体
	ContentType string            // 内容类型
	Timeout     time.Duration     // 超时时间
}

// HTTPResponse 通用HTTP响应结构
type HTTPResponse struct {
	StatusCode int               // HTTP状态码
	Headers    map[string]string // 响应头
	Body       []byte            // 响应体
}

// MultipartFormData multipart表单数据结构
type MultipartFormData struct {
	Files  map[string]*MultipartFile // 文件字段
	Fields map[string]string         // 普通字段
}

// MultipartFile 文件字段结构
type MultipartFile struct {
	FileName string    // 文件名
	Reader   io.Reader // 文件读取器
}

// HTTPError HTTP错误结构
type HTTPError struct {
	StatusCode int    // HTTP状态码
	Message    string // 错误消息
	RequestID  string // 请求ID
	Timestamp  int64  // 错误时间戳
}

// Error 实现error接口
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: status=%d, message=%s", e.StatusCode, e.Message)
}

// NewHTTPError 创建HTTP错误
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Timestamp:  time.Now().Unix(),
	}
}

// HTTPClient HTTP客户端封装
type HTTPClient struct {
	httpClient *fasthttp.Client
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		httpClient: &fasthttp.Client{
			MaxIdleConnDuration: 90 * time.Second,
			ReadTimeout:         60 * time.Second,
			WriteTimeout:        BaseWriteTimeoutSeconds * time.Second,
			MaxConnsPerHost:     500,
		},
	}
}

// DoHTTPRequest 执行HTTP请求的公共方法
func (hc *HTTPClient) DoHTTPRequest(req *HTTPRequest, token string) (*HTTPResponse, error) {
	// 创建请求和响应对象
	httpReq := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(httpReq)
		fasthttp.ReleaseResponse(resp)
	}()

	// 设置请求URI和方法
	httpReq.SetRequestURI(req.URL)
	httpReq.Header.SetMethod(req.Method)

	// 设置请求头
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// 设置授权头
	httpReq.Header.Set("Authorization", "Bearer "+token)

	// 查询参数处理
	if req.QueryParams != nil {
		for key, value := range req.QueryParams {
			httpReq.URI().QueryArgs().Add(key, value)
		}
	}

	// 请求体处理
	if req.Body != nil {
		switch v := req.Body.(type) {
		case []byte:
			httpReq.SetBody(v)
		case string:
			httpReq.SetBody([]byte(v))
		default:
			// 尝试JSON序列化
			if req.ContentType == "application/json" {
				bodyBytes, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal request body: %v", err)
				}
				httpReq.SetBody(bodyBytes)
			} else {
				return nil, fmt.Errorf("unsupported body type for content type: %s", req.ContentType)
			}
		}
	}

	// 设置内容类型
	if req.ContentType != "" {
		httpReq.Header.SetContentType(req.ContentType)
	}

	// 设置超时（如果指定）
	if req.Timeout > 0 {
		mu := sync.Mutex{}
		mu.Lock()
		originalTimeout := hc.httpClient.WriteTimeout
		hc.httpClient.WriteTimeout = req.Timeout
		defer func() {
			hc.httpClient.WriteTimeout = originalTimeout
			mu.Unlock()
		}()
	}

	// 发送请求
	if err := hc.httpClient.Do(httpReq, resp); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// 处理响应
	response := &HTTPResponse{
		StatusCode: resp.StatusCode(),
		Body:       make([]byte, len(resp.Body())),
	}
	copy(response.Body, resp.Body())

	// 处理响应头
	response.Headers = make(map[string]string)
	resp.Header.VisitAll(func(key, value []byte) {
		response.Headers[string(key)] = string(value)
	})

	// 错误处理
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response, hc.handleHTTPError(resp)
	}

	return response, nil
}

// handleHTTPError 统一HTTP错误处理
func (hc *HTTPClient) handleHTTPError(resp *fasthttp.Response) error {
	statusCode := resp.StatusCode()

	// 根据状态码分类处理错误
	switch {
	case statusCode >= 200 && statusCode < 300:
		return nil // 成功响应
	case statusCode == fasthttp.StatusUnauthorized:
		return NewHTTPError(statusCode, "unauthorized access")
	case statusCode == fasthttp.StatusForbidden:
		return NewHTTPError(statusCode, "access forbidden")
	case statusCode == fasthttp.StatusNotFound:
		return NewHTTPError(statusCode, "resource not found")
	case statusCode == fasthttp.StatusTooManyRequests:
		return NewHTTPError(statusCode, "too many requests")
	case statusCode >= 500:
		return NewHTTPError(statusCode, "server internal error")
	default:
		return NewHTTPError(statusCode, fmt.Sprintf("request failed with status %d", statusCode))
	}
}

// DoJSONRequest 执行JSON请求的专用方法
func (hc *HTTPClient) DoJSONRequest(method, url string, requestBody interface{}, token string, response interface{}) error {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	req := &HTTPRequest{
		Method:      method,
		URL:         url,
		Headers:     headers,
		Body:        requestBody,
		ContentType: "application/json",
	}

	resp, err := hc.DoHTTPRequest(req, token)
	if err != nil {
		return err
	}

	if response != nil {
		if err := json.Unmarshal(resp.Body, response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
	}

	return nil
}

// DoMultipartRequest 执行 multipart/form-data 请求的专用方法
func (hc *HTTPClient) DoMultipartRequest(url string, formData *MultipartFormData, token string, response interface{}) error {
	// 创建multipart表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件
	if formData.Files != nil {
		for fieldName, file := range formData.Files {
			part, err := writer.CreateFormFile(fieldName, file.FileName)
			if err != nil {
				return fmt.Errorf("failed to create form file: %v", err)
			}
			if _, err := io.Copy(part, file.Reader); err != nil {
				return fmt.Errorf("failed to copy file content: %v", err)
			}
		}
	}

	// 添加普通字段
	if formData.Fields != nil {
		for fieldName, value := range formData.Fields {
			if err := writer.WriteField(fieldName, value); err != nil {
				return fmt.Errorf("failed to write field: %v", err)
			}
		}
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	req := &HTTPRequest{
		Method:      "POST",
		URL:         url,
		Headers:     headers,
		Body:        body.Bytes(),
		ContentType: writer.FormDataContentType(),
	}

	resp, err := hc.DoHTTPRequest(req, token)
	if err != nil {
		return err
	}

	if response != nil {
		if err := json.Unmarshal(resp.Body, response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
	}

	return nil
}

// DoGetRequest 执行GET请求的专用方法
func (hc *HTTPClient) DoGetRequest(url string, queryParams map[string]string, token string, response interface{}) error {
	req := &HTTPRequest{
		Method:      "GET",
		URL:         url,
		QueryParams: queryParams,
		ContentType: "application/json",
	}

	httpResp, err := hc.DoHTTPRequest(req, token)
	if err != nil {
		return err
	}

	if response != nil {
		if err := json.Unmarshal(httpResp.Body, response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
	}

	return nil
}
