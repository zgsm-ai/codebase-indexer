package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAbortRetryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "UnauthorizedError",
			err:      errors.New("401 Unauthorized"),
			expected: true,
		},
		{
			name:     "PageNotFoundError",
			err:      errors.New("404 Not Found"),
			expected: true,
		},
		{
			name:     "TooManyRequestsError",
			err:      errors.New("429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "ServiceUnavailableError",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAbortRetryError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsUnauthorizedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "UnauthorizedError",
			err:      errors.New("401 Unauthorized"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUnauthorizedError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPageNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "PageNotFoundError",
			err:      errors.New("404 Not Found"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPageNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTooManyRequestsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "TooManyRequestsError",
			err:      errors.New("429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTooManyRequestsError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsServiceUnavailableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "ServiceUnavailableError",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServiceUnavailableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// func TestNewHTTPError(t *testing.T) {
// 	statusCode := 404
// 	message := "resource not found"

// 	httpErr := NewHTTPError(statusCode, message)

// 	assert.NotNil(t, httpErr)
// 	assert.Equal(t, statusCode, httpErr.StatusCode)
// 	assert.Equal(t, message, httpErr.Message)
// 	assert.True(t, httpErr.Timestamp > 0)
// 	assert.Contains(t, httpErr.Error(), fmt.Sprintf("status=%d", statusCode))
// 	assert.Contains(t, httpErr.Error(), message)
// }

// func TestNewHTTPClient(t *testing.T) {
// 	mockLogger := &mocks.MockLogger{}
// 	client := NewHTTPClient(mockLogger)

// 	assert.NotNil(t, client)
// 	assert.NotNil(t, client.httpClient)
// 	assert.Equal(t, mockLogger, client.logger)
// 	assert.Equal(t, 90*time.Second, client.httpClient.MaxIdleConnDuration)
// 	assert.Equal(t, 60*time.Second, client.httpClient.ReadTimeout)
// 	assert.Equal(t, BaseWriteTimeoutSeconds*time.Second, client.httpClient.WriteTimeout)
// 	assert.Equal(t, 500, client.httpClient.MaxConnsPerHost)
// }

// func TestHTTPClient_DoHTTPRequest(t *testing.T) {
// 	// 创建测试HTTP服务器
// 	ln := fasthttputil.NewInmemoryListener()
// 	defer ln.Close()

// 	// 启动测试服务器
// 	go func() {
// 		if err := fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
// 			switch string(ctx.Path()) {
// 			case "/success":
// 				ctx.SetStatusCode(fasthttp.StatusOK)
// 				ctx.SetBodyString(`{"status": "success"}`)
// 			case "/unauthorized":
// 				ctx.SetStatusCode(fasthttp.StatusUnauthorized)
// 				ctx.SetBodyString(`{"error": "unauthorized"}`)
// 			case "/notfound":
// 				ctx.SetStatusCode(fasthttp.StatusNotFound)
// 				ctx.SetBodyString(`{"error": "not found"}`)
// 			case "/servererror":
// 				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
// 				ctx.SetBodyString(`{"error": "server error"}`)
// 			default:
// 				ctx.SetStatusCode(fasthttp.StatusBadRequest)
// 				ctx.SetBodyString(`{"error": "bad request"}`)
// 			}
// 		}); err != nil {
// 			t.Errorf("server error: %v", err)
// 		}
// 	}()

// 	// 等待服务器启动
// 	time.Sleep(10 * time.Millisecond)

// 	mockLogger := &mocks.MockLogger{}
// 	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()

// 	client := NewHTTPClient(mockLogger)
// 	baseURL := "http://" + ln.Addr().String()

// 	tests := []struct {
// 		name        string
// 		req         *HTTPRequest
// 		token       string
// 		expectError bool
// 		expectCode  int
// 	}{
// 		{
// 			name: "SuccessfulGETRequest",
// 			req: &HTTPRequest{
// 				Method:      "GET",
// 				URL:         baseURL + "/success",
// 				ContentType: "application/json",
// 			},
// 			token:       "test-token",
// 			expectError: false,
// 			expectCode:  fasthttp.StatusOK,
// 		},
// 		{
// 			name: "UnauthorizedRequest",
// 			req: &HTTPRequest{
// 				Method:      "GET",
// 				URL:         baseURL + "/unauthorized",
// 				ContentType: "application/json",
// 			},
// 			token:       "invalid-token",
// 			expectError: true,
// 			expectCode:  fasthttp.StatusUnauthorized,
// 		},
// 		{
// 			name: "NotFoundRequest",
// 			req: &HTTPRequest{
// 				Method:      "GET",
// 				URL:         baseURL + "/notfound",
// 				ContentType: "application/json",
// 			},
// 			token:       "test-token",
// 			expectError: true,
// 			expectCode:  fasthttp.StatusNotFound,
// 		},
// 		{
// 			name: "ServerRequest",
// 			req: &HTTPRequest{
// 				Method:      "GET",
// 				URL:         baseURL + "/servererror",
// 				ContentType: "application/json",
// 			},
// 			token:       "test-token",
// 			expectError: true,
// 			expectCode:  fasthttp.StatusInternalServerError,
// 		},
// 		{
// 			name: "POSTRequestWithJSONBody",
// 			req: &HTTPRequest{
// 				Method:      "POST",
// 				URL:         baseURL + "/success",
// 				ContentType: "application/json",
// 				Body:        map[string]string{"key": "value"},
// 			},
// 			token:       "test-token",
// 			expectError: false,
// 			expectCode:  fasthttp.StatusOK,
// 		},
// 		{
// 			name: "GETRequestWithQueryParams",
// 			req: &HTTPRequest{
// 				Method:      "GET",
// 				URL:         baseURL + "/success",
// 				ContentType: "application/json",
// 				QueryParams: map[string]string{"param1": "value1", "param2": "value2"},
// 			},
// 			token:       "test-token",
// 			expectError: false,
// 			expectCode:  fasthttp.StatusOK,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			resp, err := client.DoHTTPRequest(tt.req, tt.token)

// 			if tt.expectError {
// 				assert.Error(t, err)
// 				if resp != nil {
// 					assert.Equal(t, tt.expectCode, resp.StatusCode)
// 				}
// 			} else {
// 				assert.NoError(t, err)
// 				assert.NotNil(t, resp)
// 				assert.Equal(t, tt.expectCode, resp.StatusCode)
// 				assert.NotNil(t, resp.Body)
// 			}
// 		})
// 	}

// 	mockLogger.AssertExpectations(t)
// }

// func TestHTTPClient_DoHTTPRequest_InvalidURL(t *testing.T) {
// 	mockLogger := &mocks.MockLogger{}
// 	client := NewHTTPClient(mockLogger)

// 	req := &HTTPRequest{
// 		Method:      "GET",
// 		URL:         "http://invalid-url-that-does-not-exist.example.com",
// 		ContentType: "application/json",
// 	}

// 	resp, err := client.DoHTTPRequest(req, "test-token")

// 	assert.Error(t, err)
// 	assert.Nil(t, resp)
// }

// func TestHTTPClient_DoHTTPRequest_InvalidJSONBody(t *testing.T) {
// 	// 创建包含无效JSON的请求体
// 	invalidJSON := make(chan int) // channels cannot be marshaled to JSON

// 	mockLogger := &mocks.MockLogger{}
// 	client := NewHTTPClient(mockLogger)

// 	req := &HTTPRequest{
// 		Method:      "POST",
// 		URL:         "http://example.com",
// 		ContentType: "application/json",
// 		Body:        invalidJSON,
// 	}

// 	resp, err := client.DoHTTPRequest(req, "test-token")

// 	assert.Error(t, err)
// 	assert.Contains(t, err.Error(), "failed to marshal request body")
// 	assert.Nil(t, resp)
// }

// func TestHTTPClient_DoJSONRequest(t *testing.T) {
// 	// 创建测试HTTP服务器
// 	ln := fasthttputil.NewInmemoryListener()
// 	defer ln.Close()

// 	// 启动测试服务器
// 	go func() {
// 		if err := fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
// 			// 验证请求方法
// 			assert.Equal(t, "POST", string(ctx.Method()))

// 			// 验证请求头
// 			assert.Equal(t, "application/json", string(ctx.Request.Header.ContentType()))
// 			assert.Equal(t, "application/json", string(ctx.Request.Header.Peek("Accept")))
// 			assert.Equal(t, "Bearer test-token", string(ctx.Request.Header.Peek("Authorization")))

// 			// 验证请求体
// 			var reqBody map[string]string
// 			if err := json.Unmarshal(ctx.Request.Body(), &reqBody); err == nil {
// 				assert.Equal(t, "test-value", reqBody["test-key"])
// 			}

// 			// 返回响应
// 			ctx.SetStatusCode(fasthttp.StatusOK)
// 			ctx.SetBodyString(`{"result": "success", "value": "test-response"}`)
// 		}); err != nil {
// 			t.Errorf("server error: %v", err)
// 		}
// 	}()

// 	// 等待服务器启动
// 	time.Sleep(10 * time.Millisecond)

// 	mockLogger := &mocks.MockLogger{}
// 	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()

// 	client := NewHTTPClient(mockLogger)
// 	baseURL := "http://" + ln.Addr().String()

// 	// 测试成功请求
// 	t.Run("SuccessfulRequest", func(t *testing.T) {
// 		requestBody := map[string]string{"test-key": "test-value"}
// 		var responseBody map[string]string

// 		err := client.DoJSONRequest("POST", baseURL, requestBody, "test-token", &responseBody)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, responseBody)
// 		assert.Equal(t, "success", responseBody["result"])
// 		assert.Equal(t, "test-response", responseBody["value"])
// 	})

// 	// 测试不传递响应对象
// 	t.Run("NoResponseObject", func(t *testing.T) {
// 		requestBody := map[string]string{"test-key": "test-value"}

// 		err := client.DoJSONRequest("POST", baseURL, requestBody, "test-token", nil)

// 		assert.NoError(t, err)
// 	})

// 	mockLogger.AssertExpectations(t)
// }

// func TestHTTPClient_DoMultipartRequest(t *testing.T) {
// 	// 创建测试HTTP服务器
// 	ln := fasthttputil.NewInmemoryListener()
// 	defer ln.Close()

// 	// 启动测试服务器
// 	go func() {
// 		if err := fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
// 			// 验证请求方法
// 			assert.Equal(t, "POST", string(ctx.Method()))

// 			// 验证内容类型
// 			contentType := string(ctx.Request.Header.ContentType())
// 			assert.Contains(t, contentType, "multipart/form-data")

// 			// 验证Authorization头
// 			assert.Equal(t, "Bearer test-token", string(ctx.Request.Header.Peek("Authorization")))

// 			// 解析multipart表单
// 			form, err := multipart.NewReader(bytes.NewReader(ctx.Request.Body()), contentType[len("multipart/form-data; boundary="):]).ReadForm(100 << 20)
// 			if err != nil {
// 				t.Errorf("failed to parse multipart form: %v", err)
// 				ctx.SetStatusCode(fasthttp.StatusBadRequest)
// 				return
// 			}

// 			// 验证字段
// 			assert.Equal(t, "test-value", form.Value["field1"][0])
// 			assert.Equal(t, "test-value2", form.Value["field2"][0])

// 			// 验证文件
// 			file := form.File["file1"][0]
// 			assert.Equal(t, "test.txt", file.Filename)

// 			fileContent, err := file.Open()
// 			if err != nil {
// 				t.Errorf("failed to open file: %v", err)
// 				ctx.SetStatusCode(fasthttp.StatusBadRequest)
// 				return
// 			}

// 			content, err := io.ReadAll(fileContent)
// 			if err != nil {
// 				t.Errorf("failed to read file content: %v", err)
// 				ctx.SetStatusCode(fasthttp.StatusBadRequest)
// 				return
// 			}

// 			assert.Equal(t, "test file content", string(content))
// 			fileContent.Close()

// 			// 返回响应
// 			ctx.SetStatusCode(fasthttp.StatusOK)
// 			ctx.SetBodyString(`{"result": "success"}`)
// 		}); err != nil {
// 			t.Errorf("server error: %v", err)
// 		}
// 	}()

// 	// 等待服务器启动
// 	time.Sleep(10 * time.Millisecond)

// 	mockLogger := &mocks.MockLogger{}
// 	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()

// 	client := NewHTTPClient(mockLogger)
// 	baseURL := "http://" + ln.Addr().String()

// 	// 创建multipart表单数据
// 	formData := &MultipartFormData{
// 		Files: map[string]*MultipartFile{
// 			"file1": {
// 				FileName: "test.txt",
// 				Reader:   strings.NewReader("test file content"),
// 			},
// 		},
// 		Fields: map[string]string{
// 			"field1": "test-value",
// 			"field2": "test-value2",
// 		},
// 	}

// 	// 测试成功请求
// 	t.Run("SuccessfulRequest", func(t *testing.T) {
// 		var responseBody map[string]string

// 		err := client.DoMultipartRequest(baseURL, formData, "test-token", &responseBody)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, responseBody)
// 		assert.Equal(t, "success", responseBody["result"])
// 	})

// 	// 测试不传递响应对象
// 	t.Run("NoResponseObject", func(t *testing.T) {
// 		err := client.DoMultipartRequest(baseURL, formData, "test-token", nil)

// 		assert.NoError(t, err)
// 	})

// 	mockLogger.AssertExpectations(t)
// }

// func TestHTTPClient_DoGetRequest(t *testing.T) {
// 	// 创建测试HTTP服务器
// 	ln := fasthttputil.NewInmemoryListener()
// 	defer ln.Close()

// 	// 启动测试服务器
// 	go func() {
// 		if err := fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
// 			// 验证请求方法
// 			assert.Equal(t, "GET", string(ctx.Method()))

// 			// 验证查询参数
// 			queryParams := ctx.QueryArgs()
// 			assert.Equal(t, "value1", string(queryParams.Peek("param1")))
// 			assert.Equal(t, "value2", string(queryParams.Peek("param2")))

// 			// 验证Authorization头
// 			assert.Equal(t, "Bearer test-token", string(ctx.Request.Header.Peek("Authorization")))

// 			// 返回响应
// 			ctx.SetStatusCode(fasthttp.StatusOK)
// 			ctx.SetBodyString(`{"result": "success", "param1": "value1", "param2": "value2"}`)
// 		}); err != nil {
// 			t.Errorf("server error: %v", err)
// 		}
// 	}()

// 	// 等待服务器启动
// 	time.Sleep(10 * time.Millisecond)

// 	mockLogger := &mocks.MockLogger{}
// 	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()

// 	client := NewHTTPClient(mockLogger)
// 	baseURL := "http://" + ln.Addr().String()

// 	// 测试成功请求
// 	t.Run("SuccessfulRequestWithParams", func(t *testing.T) {
// 		queryParams := map[string]string{
// 			"param1": "value1",
// 			"param2": "value2",
// 		}
// 		var responseBody map[string]string

// 		err := client.DoGetRequest(baseURL, queryParams, "test-token", &responseBody)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, responseBody)
// 		assert.Equal(t, "success", responseBody["result"])
// 		assert.Equal(t, "value1", responseBody["param1"])
// 		assert.Equal(t, "value2", responseBody["param2"])
// 	})

// 	// 测试不带查询参数的请求
// 	t.Run("RequestWithoutParams", func(t *testing.T) {
// 		var responseBody map[string]string

// 		err := client.DoGetRequest(baseURL, nil, "test-token", &responseBody)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, responseBody)
// 		assert.Equal(t, "success", responseBody["result"])
// 	})

// 	// 测试不传递响应对象
// 	t.Run("NoResponseObject", func(t *testing.T) {
// 		queryParams := map[string]string{
// 			"param1": "value1",
// 		}

// 		err := client.DoGetRequest(baseURL, queryParams, "test-token", nil)

// 		assert.NoError(t, err)
// 	})

// 	mockLogger.AssertExpectations(t)
// }

// func TestHTTPClient_handleHTTPError(t *testing.T) {
// 	mockLogger := &mocks.MockLogger{}
// 	client := NewHTTPClient(mockLogger)

// 	tests := []struct {
// 		name           string
// 		statusCode     int
// 		expectError    bool
// 		expectedErrMsg string
// 	}{
// 		{
// 			name:        "SuccessStatus",
// 			statusCode:  fasthttp.StatusOK,
// 			expectError: false,
// 		},
// 		{
// 			name:           "UnauthorizedError",
// 			statusCode:     fasthttp.StatusUnauthorized,
// 			expectError:    true,
// 			expectedErrMsg: "unauthorized access",
// 		},
// 		{
// 			name:           "ForbiddenError",
// 			statusCode:     fasthttp.StatusForbidden,
// 			expectError:    true,
// 			expectedErrMsg: "access forbidden",
// 		},
// 		{
// 			name:           "NotFoundError",
// 			statusCode:     fasthttp.StatusNotFound,
// 			expectError:    true,
// 			expectedErrMsg: "resource not found",
// 		},
// 		{
// 			name:           "TooManyRequestsError",
// 			statusCode:     fasthttp.StatusTooManyRequests,
// 			expectError:    true,
// 			expectedErrMsg: "too many requests",
// 		},
// 		{
// 			name:           "InternalServerError",
// 			statusCode:     fasthttp.StatusInternalServerError,
// 			expectError:    true,
// 			expectedErrMsg: "server internal error",
// 		},
// 		{
// 			name:           "OtherClientError",
// 			statusCode:     fasthttp.StatusBadRequest,
// 			expectError:    true,
// 			expectedErrMsg: "request failed with status 400",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// 创建响应对象
// 			resp := fasthttp.AcquireResponse()
// 			defer fasthttp.ReleaseResponse(resp)

// 			resp.SetStatusCode(tt.statusCode)

// 			// 调用错误处理方法
// 			err := client.handleHTTPError(resp)

// 			if tt.expectError {
// 				assert.Error(t, err)
// 				assert.Contains(t, err.Error(), tt.expectedErrMsg)

// 				// 验证是否为HTTPError类型
// 				if httpErr, ok := err.(*HTTPError); ok {
// 					assert.Equal(t, tt.statusCode, httpErr.StatusCode)
// 					assert.Equal(t, tt.expectedErrMsg, httpErr.Message)
// 					assert.True(t, httpErr.Timestamp > 0)
// 				}
// 			} else {
// 				assert.NoError(t, err)
// 			}
// 		})
// 	}
// }
