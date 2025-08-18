package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type QueryStatusIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type queryStatusTestCase struct {
	name           string
	endpoint       string // "status" or "summary"
	clientId       string
	codebasePath   string
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *QueryStatusIntegrationTestSuite) TestQueryStatus() {
	// 定义查询索引状态的测试用例
	statusTestCases := []queryStatusTestCase{
		{
			name:           "成功查询索引状态",
			endpoint:       "status",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "Status")
				assert.Contains(t, data, "Message")
				assert.Contains(t, data, "Timestamp")

				// 验证Status字段的有效值
				validStatuses := []string{"completed", "in_progress", "failed", "not_started"}
				assert.Contains(t, validStatuses, data["Status"])

				// 验证Timestamp是字符串
				assert.IsType(t, "", data["Timestamp"])
			},
		},
		{
			name:           "缺少clientId参数查询状态",
			endpoint:       "status",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少codebasePath参数查询状态",
			endpoint:       "status",
			clientId:       "123",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "查询不存在的代码库状态",
			endpoint:       "status",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\nonexistent",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "Status")
				assert.Contains(t, data, "Message")
				assert.Contains(t, data, "Timestamp")
			},
		},
		{
			name:           "空参数值查询状态",
			endpoint:       "status",
			clientId:       "",
			codebasePath:   "",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
	}

	// 执行索引状态表格驱动测试
	for _, tc := range statusTestCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 构建请求URL
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/index/status")
			s.Require().NoError(err)

			// 添加查询参数
			q := reqURL.Query()
			if tc.clientId != "" {
				q.Add("clientId", tc.clientId)
			}
			if tc.codebasePath != "" {
				q.Add("codebasePath", tc.codebasePath)
			}
			reqURL.RawQuery = q.Encode()

			// 创建HTTP请求
			req, err := s.CreateGETRequest(reqURL.String())
			s.Require().NoError(err)

			// 发送请求
			resp, err := s.SendRequest(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			// 验证响应状态码
			s.AssertHTTPStatus(t, tc.expectedStatus, resp.StatusCode)

			// 读取响应体
			body, err := io.ReadAll(resp.Body)
			s.Require().NoError(err)

			// 解析响应JSON
			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			s.Require().NoError(err)

			// 验证通用响应格式
			s.ValidateCommonResponse(t, response, tc.expectedCode)

			// 执行自定义验证
			if tc.validateResp != nil {
				tc.validateResp(t, response)
			}
		})
	}
}

func (s *QueryStatusIntegrationTestSuite) TestQuerySummary() {
	// 定义查询索引摘要的测试用例
	summaryTestCases := []queryStatusTestCase{
		{
			name:           "成功查询索引摘要",
			endpoint:       "summary",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "TotalFiles")
				assert.Contains(t, data, "TotalSize")
				assert.Contains(t, data, "IndexedFiles")
				assert.Contains(t, data, "IndexedSize")
				assert.Contains(t, data, "LastIndexed")

				// 验证数值类型
				assert.IsType(t, 0.0, data["TotalFiles"])
				assert.IsType(t, 0.0, data["TotalSize"])
				assert.IsType(t, 0.0, data["IndexedFiles"])
				assert.IsType(t, 0.0, data["IndexedSize"])

				// 验证LastIndexed是字符串
				assert.IsType(t, "", data["LastIndexed"])

				// 验证数值的合理性
				totalFiles := data["TotalFiles"].(float64)
				indexedFiles := data["IndexedFiles"].(float64)
				assert.GreaterOrEqual(t, totalFiles, indexedFiles)
				assert.GreaterOrEqual(t, totalFiles, 0.0)
				assert.GreaterOrEqual(t, indexedFiles, 0.0)

				totalSize := data["TotalSize"].(float64)
				indexedSize := data["IndexedSize"].(float64)
				assert.GreaterOrEqual(t, totalSize, indexedSize)
				assert.GreaterOrEqual(t, totalSize, 0.0)
				assert.GreaterOrEqual(t, indexedSize, 0.0)
			},
		},
		{
			name:           "缺少clientId参数查询摘要",
			endpoint:       "summary",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少codebasePath参数查询摘要",
			endpoint:       "summary",
			clientId:       "123",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "查询不存在的代码库摘要",
			endpoint:       "summary",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\nonexistent",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "TotalFiles")
				assert.Contains(t, data, "TotalSize")
				assert.Contains(t, data, "IndexedFiles")
				assert.Contains(t, data, "IndexedSize")
				assert.Contains(t, data, "LastIndexed")

				// 不存在的代码库应该返回0值
				assert.Equal(t, 0.0, data["TotalFiles"])
				assert.Equal(t, 0.0, data["TotalSize"])
				assert.Equal(t, 0.0, data["IndexedFiles"])
				assert.Equal(t, 0.0, data["IndexedSize"])
			},
		},
		{
			name:           "空参数值查询摘要",
			endpoint:       "summary",
			clientId:       "",
			codebasePath:   "",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
	}

	// 执行索引摘要表格驱动测试
	for _, tc := range summaryTestCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 构建请求URL
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/index/summary")
			s.Require().NoError(err)

			// 添加查询参数
			q := reqURL.Query()
			if tc.clientId != "" {
				q.Add("clientId", tc.clientId)
			}
			if tc.codebasePath != "" {
				q.Add("codebasePath", tc.codebasePath)
			}
			reqURL.RawQuery = q.Encode()

			// 创建HTTP请求
			req, err := s.CreateGETRequest(reqURL.String())
			s.Require().NoError(err)

			// 发送请求
			resp, err := s.SendRequest(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			// 验证响应状态码
			s.AssertHTTPStatus(t, tc.expectedStatus, resp.StatusCode)

			// 读取响应体
			body, err := io.ReadAll(resp.Body)
			s.Require().NoError(err)

			// 解析响应JSON
			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			s.Require().NoError(err)

			// 验证通用响应格式
			s.ValidateCommonResponse(t, response, tc.expectedCode)

			// 执行自定义验证
			if tc.validateResp != nil {
				tc.validateResp(t, response)
			}
		})
	}
}

func TestQueryStatusIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueryStatusIntegrationTestSuite))
}
