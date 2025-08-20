package api

// curl --location 'http://localhost:11380/codebase-indexer/api/v1/index' \
//--header 'content-type: application/json' \
//--header 'Client-ID: 123' \
//--header 'Server-Endpoint: https://zgsm.sangfor.com' \
//--header 'Authorization: ••••••' \
//--data '{
//    "workspace": "g:\\tmp\\projects\\javascript\\react",
//    "type": "codegraph"
//}'
// response:
// {
//    "code": "0",
//    "success": true,
//    "message": "ok",
//    "data": 1
//}

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// IndexIntegrationTestSuite 索引接口集成测试套件
type IndexIntegrationTestSuite struct {
	BaseIntegrationTestSuite // 继承基础测试套件（假设已实现）
}

// indexTestCase 索引接口测试用例结构
type indexTestCase struct {
	name           string
	workspace      string
	indexType      string
	headers        map[string]string // 额外请求头
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
	isInvalidJSON  bool // 是否发送无效JSON
}

// TestIndex 测试索引接口
func (s *IndexIntegrationTestSuite) TestIndex() {

	// 定义测试用例表
	testCases := []indexTestCase{
		{
			name:      "成功创建索引",
			workspace: s.workspacePath,
			indexType: "codegraph",

			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])
				assert.IsType(t, float64(0), response["data"]) // 验证data为数字类型（索引ID）
			},
		},
		{
			name:      "缺少workspace参数",
			indexType: "codegraph",

			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400", // 假设错误码为1
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
				assert.NotEmpty(t, response["message"])
			},
		},
		{
			name:      "缺少type参数",
			workspace: s.workspacePath,

			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:      "无效的type值",
			workspace: s.workspacePath,
			indexType: "invalid-type", // 无效的索引类型

			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400", // 假设无效类型错误码为2
		},
		{
			name:      "无效的JSON请求体",
			workspace: s.workspacePath,
			indexType: "codegraph",

			expectedStatus: http.StatusBadRequest,
			expectedCode:   "400",
			isInvalidJSON:  true,
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			// 创建请求
			reqURL := s.baseURL + "/codebase-indexer/api/v1/index"
			var req *http.Request

			if tc.isInvalidJSON {
				// 发送无效JSON
				req, err = s.CreatePOSTRequest(reqURL, []byte("{invalid json}"))
			} else {
				// 构建请求体
				reqBody := map[string]interface{}{
					"workspace": tc.workspace,
					"type":      tc.indexType,
				}

				// 序列化请求体
				jsonData, err := json.Marshal(reqBody)
				s.Require().NoError(err, "JSON序列化失败")

				// 创建POST请求
				req, err = s.CreatePOSTRequest(reqURL, jsonData)
			}

			s.Require().NoError(err, "创建请求失败")

			// 设置请求头
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			// 发送请求
			resp, err = s.SendRequest(req)
			s.Require().NoError(err, "发送请求失败")
			defer resp.Body.Close()

			// 验证状态码
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// 读取响应体
			body, err := io.ReadAll(resp.Body)
			s.Require().NoError(err, "读取响应体失败")

			// 解析响应（无效JSON场景可能解析失败，需要特殊处理）
			var response map[string]interface{}
			if !tc.isInvalidJSON {
				err = json.Unmarshal(body, &response)
				s.Require().NoError(err, "解析响应JSON失败")

				// 验证通用响应格式
				assert.Contains(t, response, "code")
				assert.Contains(t, response, "success")
				assert.Contains(t, response, "message")

				// 验证预期code
				if tc.expectedCode != "" {
					assert.Equal(t, tc.expectedCode, response["code"].(string))
				}

				// 执行自定义验证
				if tc.validateResp != nil {
					tc.validateResp(t, response)
				}
			}
		})
	}
}

// TestIndexIntegrationTestSuite 运行索引接口测试套件
func TestIndexIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IndexIntegrationTestSuite))
}
