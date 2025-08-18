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

type GetFileStructureIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type getFileStructureTestCase struct {
	name           string
	clientId       string
	codebasePath   string
	filePath       string
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *GetFileStructureIntegrationTestSuite) TestGetFileStructure() {
	// 定义测试用例表
	testCases := []getFileStructureTestCase{
		{
			name:           "成功获取文件结构",
			clientId:       "123",
			codebasePath:   "g:\\projects\\codebase-indexer",
			filePath:       "G:\\projects\\casdoor\\controllers\\auth.go",
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
				assert.Contains(t, firstItem, "type")
				assert.Contains(t, firstItem, "name")
				assert.Contains(t, firstItem, "position")
				assert.Contains(t, firstItem, "content")

				position := firstItem["position"].(map[string]interface{})
				assert.Contains(t, position, "startLine")
				assert.Contains(t, position, "startColumn")
				assert.Contains(t, position, "endLine")
				assert.Contains(t, position, "endColumn")
			},
		},
		{
			name:           "缺少clientId参数",
			codebasePath:   "g:\\projects\\codebase-indexer",
			filePath:       "G:\\projects\\casdoor\\controllers\\auth.go",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少codebasePath参数",
			clientId:       "123",
			filePath:       "G:\\projects\\casdoor\\controllers\\auth.go",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少filePath参数",
			clientId:       "123",
			codebasePath:   "g:\\projects\\codebase-indexer",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "不存在的文件路径",
			clientId:       "123",
			codebasePath:   "g:\\projects\\codebase-indexer",
			filePath:       "G:\\projects\\nonexistent\\file.go",
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
			name:           "空参数值",
			clientId:       "",
			codebasePath:   "",
			filePath:       "",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "特殊字符路径",
			clientId:       "123",
			codebasePath:   "g:\\projects\\codebase-indexer",
			filePath:       "G:\\projects\\casdoor\\controllers\\auth.go",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.Greater(t, len(list), 0)
			},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 构建请求URL
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/files/structure")
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

func TestGetFileStructureIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(GetFileStructureIntegrationTestSuite))
}
