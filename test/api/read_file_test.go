package api

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ReadFileIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type readFileTestCase struct {
	name           string
	clientId       string
	codebasePath   string
	filePath       string
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *ReadFileIntegrationTestSuite) TestReadFile() {
	// 定义测试用例表
	testCases := []readFileTestCase{
		{
			name:           "成功读取文件内容",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       filepath.Join(s.workspacePath, "test", "api", "read_file_test.go"),
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:           "缺少codebasePath参数",
			clientId:       "123",
			filePath:       filepath.Join(s.workspacePath, "test", "api", "read_file_test.go"),
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:           "缺少filePath参数",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:           "读取不存在的文件",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\nonexistent\\file.go",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:           "读取目录而非文件",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       s.workspacePath,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
			},
		},
		{
			name:           "空参数值",
			clientId:       "",
			codebasePath:   "",
			filePath:       "",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
			},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 构建请求URL
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/files/content")
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
			_, err = io.ReadAll(resp.Body)
			s.Require().NoError(err)

		})
	}
}

func TestReadFileIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ReadFileIntegrationTestSuite))
}
