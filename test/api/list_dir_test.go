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

type ListDirIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type listDirTestCase struct {
	name           string
	clientId       string
	codebasePath   string
	subDir         string
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *ListDirIntegrationTestSuite) TestListDir() {
	// 定义测试用例表
	testCases := []listDirTestCase{
		{
			name:           "成功列出目录内容",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			subDir:         "cmd",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				assert.Equal(t, "g:\\tmp\\projects\\go\\kubernetes", data["rootPath"])

				directoryTree := data["directoryTree"].([]interface{})
				assert.Greater(t, len(directoryTree), 0)

				// 验证第一个元素的结构
				firstItem := directoryTree[0].(map[string]interface{})
				assert.Contains(t, firstItem, "name")
				assert.Contains(t, firstItem, "path")
				assert.Contains(t, firstItem, "IsDir")
				assert.IsType(t, true, firstItem["IsDir"])
			},
		},
		{
			name:           "列出根目录内容",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			subDir:         "",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				directoryTree := data["directoryTree"].([]interface{})
				assert.Greater(t, len(directoryTree), 0)
			},
		},
		{
			name:           "缺少clientId参数",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			subDir:         "cmd",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少codebasePath参数",
			clientId:       "123",
			subDir:         "cmd",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "不存在的子目录",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			subDir:         "nonexistent_dir",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				directoryTree := data["directoryTree"].([]interface{})
				// 不存在的目录应该返回空列表
				assert.Len(t, directoryTree, 0)
			},
		},
		{
			name:           "空参数值",
			clientId:       "",
			codebasePath:   "",
			subDir:         "",
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
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/codebases/directory")
			s.Require().NoError(err)

			// 添加查询参数
			q := reqURL.Query()
			if tc.clientId != "" {
				q.Add("clientId", tc.clientId)
			}
			if tc.codebasePath != "" {
				q.Add("codebasePath", tc.codebasePath)
			}
			if tc.subDir != "" {
				q.Add("subDir", tc.subDir)
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

func TestListDirIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ListDirIntegrationTestSuite))
}
