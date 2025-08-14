package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GetSnippetsIntegrationTestSuite struct {
	suite.Suite
	baseURL       string
	workspacePath string
}

func (s *GetSnippetsIntegrationTestSuite) SetupSuite() {
	// 设置API基础URL
	s.baseURL = "http://localhost:11380"

	// 设置工作目录路径
	s.workspacePath = "g:\\tmp\\projects\\go\\kubernetes"
}

type testCase struct {
	name           string
	clientId       string
	workspacePath  string
	codeSnippets   []map[string]interface{}
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *GetSnippetsIntegrationTestSuite) TestReadCodeSnippets() {
	// 定义测试用例表
	testCases := []testCase{
		{
			name:          "成功获取单个文件片段",
			clientId:      "test-client-success",
			workspacePath: s.workspacePath,
			codeSnippets: []map[string]interface{}{
				{
					"filePath":  filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"),
					"startLine": 15,
					"endLine":   25,
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.Len(t, list, 1)

				snippet := list[0].(map[string]interface{})
				assert.Equal(t, filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"), snippet["filePath"])
				assert.Equal(t, float64(15), snippet["startLine"])
				assert.Equal(t, float64(25), snippet["endLine"])
				assert.Contains(t, snippet["content"].(string), "package")
			},
		},
		{
			name:          "成功获取多个文件片段",
			clientId:      "test-client-multiple",
			workspacePath: s.workspacePath,
			codeSnippets: []map[string]interface{}{
				{
					"filePath":  filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"),
					"startLine": 1,
					"endLine":   5,
				},
				{
					"filePath":  filepath.Join(s.workspacePath, "cmd", "kube-apiserver", "app", "server.go"),
					"startLine": 1,
					"endLine":   5,
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.Len(t, list, 2)
			},
		},
		{
			name:          "超过500行限制应被截断",
			clientId:      "test-client-large",
			workspacePath: s.workspacePath,
			codeSnippets: []map[string]interface{}{
				{
					"filePath":  filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"),
					"startLine": 1,
					"endLine":   600, // 超过500行限制
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.Len(t, list, 1)

				snippet := list[0].(map[string]interface{})
				startLine := int(snippet["startLine"].(float64))
				endLine := int(snippet["endLine"].(float64))
				assert.Equal(t, 1, startLine)
				assert.Equal(t, 501, endLine) // 应该被截断为501行
			},
		},
		{
			name:          "超过200个片段限制应被忽略",
			clientId:      "test-client-too-many",
			workspacePath: s.workspacePath,
			codeSnippets: func() []map[string]interface{} {
				snippets := make([]map[string]interface{}, 201)
				for i := 0; i < 201; i++ {
					snippets[i] = map[string]interface{}{
						"filePath":  filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"),
						"startLine": 1,
						"endLine":   10,
					}
				}
				return snippets
			}(),
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.Len(t, list, 200) // 应该被限制为200个
			},
		},
		{
			name:          "无效JSON请求体",
			clientId:      "test-client",
			workspacePath: s.workspacePath,
			codeSnippets: []map[string]interface{}{
				{
					"filePath":  filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"),
					"startLine": 1,
					"endLine":   10,
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				// 对于无效JSON，响应格式可能不同，这里只验证状态码
			},
		},
		{
			name:          "缺少clientId",
			workspacePath: s.workspacePath,
			codeSnippets: []map[string]interface{}{
				{
					"filePath":  filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"),
					"startLine": 1,
					"endLine":   10,
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				// 验证返回错误
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:     "缺少workspacePath",
			clientId: "test-client",
			codeSnippets: []map[string]interface{}{
				{
					"filePath":  filepath.Join(s.workspacePath, "cmd", "kubelet", "app", "server.go"),
					"startLine": 1,
					"endLine":   10,
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				// 验证返回错误
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "空的codeSnippets数组",
			clientId:       "test-client",
			workspacePath:  s.workspacePath,
			codeSnippets:   []map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				// 验证返回错误
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:          "不存在的文件",
			clientId:      "test-client",
			workspacePath: s.workspacePath,
			codeSnippets: []map[string]interface{}{
				{
					"filePath":  filepath.Join(s.workspacePath, "nonexistent", "file.go"),
					"startLine": 1,
					"endLine":   10,
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				// 验证返回错误
				assert.True(t, response["success"].(bool))
			},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 准备请求体
			reqBody := map[string]interface{}{
				"clientId":      tc.clientId,
				"workspacePath": tc.workspacePath,
				"codeSnippets":  tc.codeSnippets,
			}

			var resp *http.Response
			var err error

			// 特殊处理无效JSON的情况
			if tc.name == "无效JSON请求体" {
				resp, err = http.Post(s.baseURL+"/codebase-indexer/api/v1/snippets/read", "application/json", bytes.NewBuffer([]byte("invalid json")))
			} else {
				// 发送正常请求
				jsonData, err := json.Marshal(reqBody)
				s.Require().NoError(err)
				resp, err = http.Post(s.baseURL+"/codebase-indexer/api/v1/snippets/read", "application/json", bytes.NewBuffer(jsonData))
			}

			s.Require().NoError(err)
			defer resp.Body.Close()

			// 验证响应状态码
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// 读取响应体
			body, err := io.ReadAll(resp.Body)
			s.Require().NoError(err)

			// 解析响应JSON
			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			s.Require().NoError(err)

			// 如果有期望的响应码，验证它
			if tc.expectedCode != "" {
				assert.Equal(t, tc.expectedCode, response["code"])
			}

			// 执行自定义验证
			if tc.validateResp != nil {
				tc.validateResp(t, response)
			}
		})
	}
}

func TestGetSnippetsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(GetSnippetsIntegrationTestSuite))
}
