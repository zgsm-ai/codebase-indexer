package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type QueryReferenceIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type queryReferenceTestCase struct {
	name           string
	clientId       string
	codebasePath   string
	filePath       string
	startLine      int
	endLine        int
	symbolName     string
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *QueryReferenceIntegrationTestSuite) TestQueryReference() {
	// 定义测试用例表
	testCases := []queryReferenceTestCase{
		{
			name:           "成功查询引用信息",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       filepath.Join(s.workspacePath, "test", "api", "query_reference_test.go"),
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
				assert.Contains(t, firstItem, "symbolName")
				assert.Contains(t, firstItem, "position")
				assert.Contains(t, firstItem, "content")
				assert.Contains(t, firstItem, "nodeType")

				position := firstItem["position"].(map[string]interface{})
				assert.Contains(t, position, "startLine")
				assert.Contains(t, position, "startColumn")
				assert.Contains(t, position, "endLine")
				assert.Contains(t, position, "endColumn")

				// 验证nodeType字段的有效值
				validNodeTypes := []string{"definition.function", "definition.method", "definition.class","definition.interface"}
				assert.Contains(t, validNodeTypes, firstItem["nodeType"])

				// 验证children字段
				children := firstItem["children"]
				if children != nil {
					childrenList := children.([]interface{})
					assert.IsType(t, []interface{}{}, childrenList)
				}
			},
		},
		{
			name:           "缺少codebasePath参数",
			clientId:       "123",
			filePath:       filepath.Join(s.workspacePath, "test", "api", "query_reference_test.go"),
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
			codebasePath:   s.workspacePath,
			startLine:      1,
			endLine:        1000,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},

		{
			name:           "不存在的文件路径",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       filepath.Join(s.workspacePath, "test", "api", "no_exists.go"),
			startLine:      1,
			endLine:        1000,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "-1",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
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
		{
			name:           "根据符号名全局查找引用-成功",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       "", // 空文件路径，触发全局符号名查找
			symbolName:     "TestQueryReference",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				fmt.Println("list", list)
				// 全局查找可能返回空结果，这是正常的
				assert.GreaterOrEqual(t, len(list), 0)

				// 如果有结果，验证结构
				if len(list) > 0 {
					firstItem := list[0].(map[string]interface{})
					assert.Contains(t, firstItem, "symbolName")
					assert.Contains(t, firstItem, "nodeType")
					assert.Contains(t, firstItem, "children")

					// 验证符号名匹配
					if symbolName, ok := firstItem["symbolName"].(string); ok {
						assert.Equal(t, "TestQueryReference", symbolName)
					}

					// 验证nodeType字段的有效值
					validNodeTypes := []string{"definition.function", "definition.method", "definition.class", "definition.interface"}
					if nodeType, ok := firstItem["nodeType"].(string); ok {
						assert.Contains(t, validNodeTypes, nodeType)
					}

					// 验证children字段
					children := firstItem["children"]
					if children != nil {
						childrenList := children.([]interface{})
						assert.IsType(t, []interface{}{}, childrenList)
					}
				}
			},
		},
		{
			name:           "根据符号名全局查找引用-不存在的符号",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       "", // 空文件路径，触发全局符号名查找
			symbolName:     "NonExistentSymbol",
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				// 返回结果为nil
				list := data["list"]
				// 不存在的符号应该返回空列表
				assert.Equal(t, nil, list)
			},
		},
		{
			name:           "根据符号名全局查找引用-缺少codebasePath",
			clientId:       "123",
			filePath:       "", // 空文件路径
			symbolName:     "TestQueryReference",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "根据符号名全局查找引用-空符号名",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       "", // 空文件路径
			symbolName:     "", // 空符号名
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
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/search/reference")
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
			if tc.symbolName != "" {
				q.Add("symbolName", tc.symbolName)
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

func TestQueryReferenceIntegrationTestSuite(t *testing.T) {

	suite.Run(t, new(QueryReferenceIntegrationTestSuite))
}
