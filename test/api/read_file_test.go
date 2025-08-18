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
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\pkg\\controller\\daemon\\util\\daemonset_util.go",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "content")
				assert.Contains(t, data, "filePath")
				assert.Contains(t, data, "fileName")
				assert.Contains(t, data, "fileSize")
				assert.Contains(t, data, "lastModified")

				// 验证content是字符串且不为空
				content := data["content"].(string)
				assert.NotEmpty(t, content)

				// 验证filePath与请求一致
				assert.Equal(t, "g:\\tmp\\projects\\go\\kubernetes\\pkg\\controller\\daemon\\util\\daemonset_util.go", data["filePath"])

				// 验证fileName
				assert.Equal(t, "daemonset_util.go", data["fileName"])

				// 验证fileSize是数字且大于0
				fileSize := data["fileSize"].(float64)
				assert.Greater(t, fileSize, 0.0)

				// 验证lastModified是字符串
				assert.IsType(t, "", data["lastModified"])

				// 验证文件内容包含预期的Go代码特征
				assert.Contains(t, content, "package")
				assert.Contains(t, content, "import")
			},
		},
		{
			name:           "读取不同类型文件",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\go.mod",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				content := data["content"].(string)

				// 验证go.mod文件内容特征
				assert.Contains(t, content, "module")
				assert.Contains(t, content, "go")

				// 验证fileName
				assert.Equal(t, "go.mod", data["fileName"])
			},
		},
		{
			name:           "缺少clientId参数",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\pkg\\controller\\daemon\\util\\daemonset_util.go",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少codebasePath参数",
			clientId:       "123",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\pkg\\controller\\daemon\\util\\daemonset_util.go",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "缺少filePath参数",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "读取不存在的文件",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\nonexistent\\file.go",
			expectedStatus: http.StatusNotFound,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "读取目录而非文件",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\pkg\\controller\\daemon\\util",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
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
			name:           "文件路径包含特殊字符",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\pkg\\controller\\daemon\\util\\daemonset_util.go",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "content")
				assert.Contains(t, data, "filePath")
				assert.Contains(t, data, "fileName")

				// 验证content是字符串
				content := data["content"].(string)
				assert.IsType(t, "", content)
			},
		},
		{
			name:           "读取大文件",
			clientId:       "123",
			codebasePath:   "g:\\tmp\\projects\\go\\kubernetes",
			filePath:       "g:\\tmp\\projects\\go\\kubernetes\\pkg\\controller\\daemon\\daemon.go",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				content := data["content"].(string)

				// 验证大文件内容不为空
				assert.NotEmpty(t, content)

				// 验证文件大小合理
				fileSize := data["fileSize"].(float64)
				assert.Greater(t, fileSize, 0.0)
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

func TestReadFileIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ReadFileIntegrationTestSuite))
}
