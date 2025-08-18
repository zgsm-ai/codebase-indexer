package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PublishEventIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type publishEventTestCase struct {
	name           string
	workspace      string
	data           []map[string]interface{}
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *PublishEventIntegrationTestSuite) TestPublishEvent() {
	// 定义测试用例表
	testCases := []publishEventTestCase{
		{
			name:      "成功发布事件",
			workspace: "g:\\tmp\\projects\\go\\kubernetes",
			data: []map[string]interface{}{
				{
					"eventType":  "open_workspace",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": "g:\\tmp\\projects\\go\\kubernetes",
					"targetPath": "g:\\tmp\\projects\\go\\kubernetes",
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"]
				assert.NotNil(t, data)
				// 验证返回的数据是数字类型（事件ID）
				assert.IsType(t, float64(1), data)
			},
		},
		{
			name:      "发布多个事件",
			workspace: "g:\\tmp\\projects\\go\\kubernetes",
			data: []map[string]interface{}{
				{
					"eventType":  "open_workspace",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": "g:\\tmp\\projects\\go\\kubernetes",
					"targetPath": "g:\\tmp\\projects\\go\\kubernetes",
				},
				{
					"eventType":  "file_changed",
					"eventTime":  "2025-07-28 20:48:00",
					"sourcePath": "g:\\tmp\\projects\\go\\kubernetes\\main.go",
					"targetPath": "g:\\tmp\\projects\\go\\kubernetes\\main.go",
				},
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"]
				assert.NotNil(t, data)
				assert.IsType(t, float64(1), data)
			},
		},
		{
			name: "缺少workspace参数",
			data: []map[string]interface{}{
				{
					"eventType":  "open_workspace",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": "g:\\tmp\\projects\\go\\kubernetes",
					"targetPath": "g:\\tmp\\projects\\go\\kubernetes",
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "空data数组",
			workspace:      "g:\\tmp\\projects\\go\\kubernetes",
			data:           []map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:      "缺少必要字段的事件",
			workspace: "g:\\tmp\\projects\\go\\kubernetes",
			data: []map[string]interface{}{
				{
					"eventType": "open_workspace",
					// 缺少eventTime, sourcePath, targetPath
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:      "无效的事件时间格式",
			workspace: "g:\\tmp\\projects\\go\\kubernetes",
			data: []map[string]interface{}{
				{
					"eventType":  "open_workspace",
					"eventTime":  "invalid-time-format",
					"sourcePath": "g:\\tmp\\projects\\go\\kubernetes",
					"targetPath": "g:\\tmp\\projects\\go\\kubernetes",
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:      "无效的JSON请求体",
			workspace: "g:\\tmp\\projects\\go\\kubernetes",
			data: []map[string]interface{}{
				{
					"eventType":  "open_workspace",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": "g:\\tmp\\projects\\go\\kubernetes",
					"targetPath": "g:\\tmp\\projects\\go\\kubernetes",
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				// 这个测试用例会在测试代码中特殊处理，发送无效的JSON
				assert.False(t, response["success"].(bool))
			},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			// 特殊处理无效JSON的情况
			if tc.name == "无效的JSON请求体" {
				// 创建HTTP请求
				req, err := http.NewRequest("POST", s.baseURL+"/codebase-indexer/api/v1/events", bytes.NewBuffer([]byte("invalid json")))
				s.Require().NoError(err)

				// 添加认证头
				for key, value := range s.extraHeaders {
					req.Header.Add(key, value)
				}

				// 发送请求
				resp, err = s.SendRequest(req)
			} else {
				// 准备请求体
				reqBody := map[string]interface{}{
					"workspace": tc.workspace,
					"data":      tc.data,
				}

				// 序列化请求体
				jsonData, err := json.Marshal(reqBody)
				s.Require().NoError(err)

				// 创建HTTP请求
				req, err := s.CreatePOSTRequest(s.baseURL+"/codebase-indexer/api/v1/events", jsonData)
				s.Require().NoError(err)

				// 发送请求
				resp, err = s.SendRequest(req)
			}

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

func TestPublishEventIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PublishEventIntegrationTestSuite))
}
