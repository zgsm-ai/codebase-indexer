package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"codebase-indexer/internal/utils"

	"github.com/stretchr/testify/suite"
)

// BaseIntegrationTestSuite 基础集成测试套件
type BaseIntegrationTestSuite struct {
	suite.Suite
	baseURL       string
	workspacePath string
	extraHeaders  map[string]string
}

// SetupSuite 设置测试套件
func (s *BaseIntegrationTestSuite) SetupSuite() {
	// 设置API基础URL
	s.baseURL = "http://localhost:11380"

	// 设置工作目录路径
	s.workspacePath = "g:\\tmp\\projects\\go\\kubernetes"
	s.extraHeaders = make(map[string]string)

	// 读取认证配置
	s.setupAuthHeaders()
}

// setupAuthHeaders 设置认证头
func (s *BaseIntegrationTestSuite) setupAuthHeaders() {
	rootDir, err := utils.GetRootDir("codebase_indexer_test")
	if err != nil {
		panic(err)
	}
	authJsonPath := filepath.Join(rootDir, "share", "auth.json")
	file, err := os.ReadFile(authJsonPath)
	if err != nil {
		panic(err)
	}
	authConfig := make(map[string]string)
	if err = json.Unmarshal(file, &authConfig); err != nil {
		panic(err)
	}
	s.extraHeaders["Client-ID"] = authConfig["machine_id"]
	s.extraHeaders["Server-Endpoint"] = authConfig["base_url"]
	s.extraHeaders["Authorization"] = fmt.Sprintf("Bearer %s", authConfig["access_token"])
}

// CreateGETRequest 创建GET请求
func (s *BaseIntegrationTestSuite) CreateGETRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 添加认证头
	for key, value := range s.extraHeaders {
		req.Header.Add(key, value)
	}

	return req, nil
}

// CreatePOSTRequest 创建POST请求
func (s *BaseIntegrationTestSuite) CreatePOSTRequest(url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 添加认证头
	for key, value := range s.extraHeaders {
		req.Header.Add(key, value)
	}

	return req, nil
}

// SendRequest 发送HTTP请求
func (s *BaseIntegrationTestSuite) SendRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	return client.Do(req)
}

// ValidateCommonResponse 验证通用响应格式
func (s *BaseIntegrationTestSuite) ValidateCommonResponse(t *testing.T, response map[string]interface{}, expectedCode string) {
	if expectedCode != "" {
		s.Equal(expectedCode, response["code"])
	}

	// 验证响应包含必要字段
	s.Contains(response, "code")
	s.Contains(response, "message")
	s.Contains(response, "success")

	// 验证success字段类型
	if success, ok := response["success"].(bool); ok {
		// 如果响应包含data字段，且success为true，则验证data存在
		if success && response["data"] != nil {
			s.Contains(response, "data")
		}
	}
}

// AssertHTTPStatus 断言HTTP状态码
func (s *BaseIntegrationTestSuite) AssertHTTPStatus(t *testing.T, expected, actual int) {
	s.Equal(expected, actual)
}

// AssertResponseField 断言响应字段
func (s *BaseIntegrationTestSuite) AssertResponseField(t *testing.T, response map[string]interface{}, field string, expected interface{}) {
	s.Equal(expected, response[field])
}

// AssertResponseContains 断言响应包含字段
func (s *BaseIntegrationTestSuite) AssertResponseContains(t *testing.T, response map[string]interface{}, field string) {
	s.Contains(response, field)
}
