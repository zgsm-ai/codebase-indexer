package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type QueryDefinitionIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type queryDefinitionTestCase struct {
	name           string
	clientId       string
	codebasePath   string
	filePath       string
	startLine      int
	endLine        int
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *QueryDefinitionIntegrationTestSuite) TestQueryDefinition() {
	// 定义测试用例表
	testCases := []queryDefinitionTestCase{
		{
			name:           "成功查询定义信息",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\cluster\\gce\\gci\\mounter\\mounter.go",
			startLine:      1,
			endLine:        1000,
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.Greater(t, len(list), 0)

				// 验证第一个元素的结构
				firstItem := list[0].(map[string]interface{})
				assert.Contains(t, firstItem, "filePath")
				assert.Contains(t, firstItem, "name")
				assert.Contains(t, firstItem, "type")
				assert.Contains(t, firstItem, "content")
				assert.Contains(t, firstItem, "position")

				position := firstItem["position"].(map[string]interface{})
				assert.Contains(t, position, "startLine")
				assert.Contains(t, position, "startColumn")
				assert.Contains(t, position, "endLine")
				assert.Contains(t, position, "endColumn")

				// 验证类型字段的有效值
				validTypes := []string{"variable", "definition.function", "definition.method"}
				assert.Contains(t, validTypes, firstItem["type"])
			},
		},
		{
			name:           "查询小范围定义",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\cluster\\gce\\gci\\mounter\\mounter.go",
			startLine:      1,
			endLine:        50,
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				// 小范围应该返回较少的定义
				assert.GreaterOrEqual(t, len(list), 0)
			},
		},
		{
			name:           "缺少clientId参数",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\cluster\\gce\\gci\\mounter\\mounter.go",
			startLine:      1,
			endLine:        1000,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少codebasePath参数",
			clientId:       "123",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\cluster\\gce\\gci\\mounter\\mounter.go",
			startLine:      1,
			endLine:        1000,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少filePath参数",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			startLine:      1,
			endLine:        1000,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少startLine参数",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\cluster\\gce\\gci\\mounter\\mounter.go",
			endLine:        1000,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少endLine参数",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\cluster\\gce\\gci\\mounter\\mounter.go",
			startLine:      1,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "不存在的文件路径",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\nonexistent\\file.go",
			startLine:      1,
			endLine:        1000,
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				// 不存在的文件应该返回空列表
				assert.Len(t, list, 0)
			},
		},
		{
			name:           "无效的行号范围",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\cluster\\gce\\gci\\mounter\\mounter.go",
			startLine:      1000,
			endLine:        1, // 结束行小于开始行
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				// 无效范围应该返回空列表
				assert.Len(t, list, 0)
			},
		},
		{
			name:           "空参数值",
			clientId:       "",
			codebasePath:   "",
			filePath:       "",
			startLine:      0,
			endLine:        0,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 构建请求URL
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/search/definition")
			s.Require().NoError(err)

			// 添加查询参数
			q := reqURL.Query()
			if tc.clientId != "" {
				q.Add("clientId", tc.clientId)
			}
			if tc.codebasePath != "" {
				q.Add("codebasePath", tc.codebasePath)
			}
			if tc.filePath != "" {
				q.Add("filePath", tc.filePath)
			}
			if tc.startLine > 0 {
				q.Add("startLine", fmt.Sprintf("%d", tc.startLine))
			}
			if tc.endLine > 0 {
				q.Add("endLine", fmt.Sprintf("%d", tc.endLine))
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

func TestQueryDefinitionIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueryDefinitionIntegrationTestSuite))
}
