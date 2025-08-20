package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	var currentWorkspace, err = filepath.Abs("../../")
	if err != nil {
		panic(err)
	}
	// 设置工作目录路径
	s.workspacePath = currentWorkspace
	s.extraHeaders = make(map[string]string)

	// 读取认证配置
	s.setupAuthHeaders()
	// 注册工作区
	s.registerWorkspace()
}

func (s *BaseIntegrationTestSuite) registerWorkspace() {
	var resp *http.Response
	var err error
	// 准备请求体
	reqBody := map[string]interface{}{
		"workspace": s.workspacePath,
		"data": []map[string]interface{}{{
			"eventType": "open_workspace",
			"eventTime": time.Now().UTC().Format(time.RFC3339),
		},
		},
	}
	// 序列化请求体
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		panic(err)
	}

	// 创建HTTP请求
	req, err := s.CreatePOSTRequest(s.baseURL+"/codebase-indexer/api/v1/events", jsonData)
	if err != nil {
		panic(err)
	}
	// 发送请求
	resp, err = s.SendRequest(req)

	s.Require().NoError(err)
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		panic("init workspace=> unexpected status code: " + resp.Status + ", body: " + string(body))
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	// 解析响应JSON
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		panic("init workspace=> unexpected body: " + string(body))

	}
	if response["message"] != "ok" {
		panic("init workspace=> unexpected body: " + string(body))
	}
	// sleep 2s，等待处理完成
	s.T().Log("wait 2 seconds for initializing workspace...")
	time.Sleep(2 * time.Second)
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
