package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type QueryCallGraphIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type queryCallGraphTestCase struct {
	name           string
	clientId       string
	codebasePath   string
	filePath       string
	startLine      int
	endLine        int
	symbolName     string
	maxLayer       int
	expectedStatus int
	expectedCode   string
	validateResp   func(t *testing.T, response map[string]interface{})
}

func (s *QueryCallGraphIntegrationTestSuite) TestQueryCallGraph() {
	// 定义测试用例表
	testCases := []queryCallGraphTestCase{
		{
			name:           "成功查询调用图信息",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       filepath.Join(s.workspacePath, "test", "api", "query_reference_test.go"),
			startLine:      1,
			endLine:        1000,
			symbolName:     "",
			maxLayer:       3,
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.GreaterOrEqual(t, len(list), 0)

				// 如果有数据，验证第一个元素的结构
				if len(list) > 0 {
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
					validNodeTypes := []string{"definition", "reference"}
					assert.Contains(t, validNodeTypes, firstItem["nodeType"])

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
			name:           "指定符号名查询调用图",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       filepath.Join(s.workspacePath, "test", "api", "query_reference_test.go"),
			startLine:      1,
			endLine:        1000,
			symbolName:     "TestQueryReference",
			maxLayer:       2,
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.GreaterOrEqual(t, len(list), 0)
			},
		},
		{
			name:           "缺少codebasePath参数",
			clientId:       "123",
			filePath:       filepath.Join(s.workspacePath, "test", "api", "query_reference_test.go"),
			startLine:      1,
			endLine:        1000,
			symbolName:     "",
			maxLayer:       3,
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
			symbolName:     "",
			maxLayer:       3,
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
			symbolName:     "",
			maxLayer:       3,
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
			symbolName:     "",
			maxLayer:       0,
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool))
			},
		},
		{
			name:           "测试最大层数限制",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       filepath.Join(s.workspacePath, "test", "api", "query_reference_test.go"),
			startLine:      1,
			endLine:        1000,
			symbolName:     "",
			maxLayer:       10, // 测试最大层数
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.GreaterOrEqual(t, len(list), 0)
			},
		},
		{
			name:           "测试行号范围",
			clientId:       "123",
			codebasePath:   s.workspacePath,
			filePath:       filepath.Join(s.workspacePath, "test", "api", "query_reference_test.go"),
			startLine:      30,
			endLine:        50,
			symbolName:     "",
			maxLayer:       2,
			expectedStatus: http.StatusOK,
			expectedCode:   "0",
			validateResp: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				assert.Equal(t, "ok", response["message"])

				data := response["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				assert.GreaterOrEqual(t, len(list), 0)
			},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 构建请求URL
			reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/callgraph")
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
			if tc.maxLayer > 0 {
				q.Add("maxLayer", fmt.Sprintf("%d", tc.maxLayer))
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

func TestQueryCallGraphIntegrationTestSuite(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			fmt.Println("开始执行Task", i)
			suite.Run(t, new(QueryCallGraphIntegrationTestSuite))
			fmt.Println("Task", i, "执行完成")
		}()
	}
	wg.Wait()

}
