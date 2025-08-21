package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type IndexEventIntegrationTestSuite struct {
	BaseIntegrationTestSuite
}

type indexEventTestCase struct {
	name            string
	workspace       string
	data            []map[string]interface{}
	wantProcessTime time.Duration
	validateIndex   func(t *testing.T, indexes []map[string]interface{})
}

func (s *IndexEventIntegrationTestSuite) TestPublishEvent() {
	// 定义测试用例表
	testCases := []indexEventTestCase{
		{
			name:      "打开工作区",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType": "open_workspace",
					"eventTime": "2025-07-28 20:47:00",
				},
			},
			wantProcessTime: time.Second * 15,
			validateIndex:   func(t *testing.T, indexes []map[string]interface{}) {},
		},

		{
			name: "删除文件",
			data: []map[string]interface{}{
				{
					"eventType":  "delete_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": s.workspacePath,
					"targetPath": s.workspacePath,
				},
			},
			wantProcessTime: time.Second * 15,
			validateIndex:   func(t *testing.T, indexes []map[string]interface{}) {},
		},
		{
			name:      "删除文件夹",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "delete_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": s.workspacePath,
					"targetPath": s.workspacePath,
				},
			},
			wantProcessTime: time.Second * 15,
			validateIndex:   func(t *testing.T, indexes []map[string]interface{}) {},
		},
		{
			name:      "新增文件",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "add_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": s.workspacePath,
					"targetPath": s.workspacePath,
				},
			},
			wantProcessTime: time.Second * 15,
			validateIndex:   func(t *testing.T, indexes []map[string]interface{}) {},
		},
		{
			name:      "修改文件",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "modify_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": s.workspacePath,
					"targetPath": s.workspacePath,
				},
			},
			wantProcessTime: time.Second * 15,
			validateIndex:   func(t *testing.T, indexes []map[string]interface{}) {},
		},
		{
			name:      "重命名文件",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "rename_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": s.workspacePath,
					"targetPath": s.workspacePath,
				},
			},
			wantProcessTime: time.Second * 15,
			validateIndex:   func(t *testing.T, indexes []map[string]interface{}) {},
		},
		{
			name:      "重命名文件夹",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "rename_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": s.workspacePath,
					"targetPath": s.workspacePath,
				},
			},
			wantProcessTime: time.Second * 15,
			validateIndex:   func(t *testing.T, indexes []map[string]interface{}) {},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 发布事件

			// 等待一定事件

			// 查询索引，达到期望状态

		})
	}
}

func TestIndexEventIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IndexEventIntegrationTestSuite))
}

// publishEvent 发布索引事件
func (s *IndexEventIntegrationTestSuite) publishEvent(workspace string, data any) error {
	var resp *http.Response
	var err error

	// 准备请求体
	reqBody := map[string]interface{}{
		"workspace": workspace,
		"data":      data,
	}

	// 序列化请求体
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	// 创建HTTP请求
	req, err := s.CreatePOSTRequest(s.baseURL+"/codebase-indexer/api/v1/events", jsonData)
	if err != nil {
		return err
	}

	// 发送请求
	resp, err = s.SendRequest(req)

	s.Require().NoError(err)
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	// 解析响应JSON
	var response map[string]interface{}
	if err = json.Unmarshal(body, &response); err != nil {
		return err
	}

	s.Equal("ok", response["message"])
	if response["message"] != "ok" {
		return fmt.Errorf("expected message `ok` but got `%s`", response["message"])
	}
	return nil
}

// 获取全部索引
func (s *IndexEventIntegrationTestSuite) dumpIndex() ([]map[string]interface{}, error) {
	// 构建请求URL
	reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/index/export")
	s.Require().NoError(err)

	// 添加查询参数
	q := reqURL.Query()
	q.Add("codebasePath", s.workspacePath)

	reqURL.RawQuery = q.Encode()

	// 创建HTTP请求
	req, err := s.CreateGETRequest(reqURL.String())
	s.Require().NoError(err)

	// 发送请求
	resp, err := s.SendRequest(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	// 先将读取到的body转换为字符串
	bodyStr := string(body)

	// 按行分割内容
	lines := strings.Split(bodyStr, "\n")
	var indexes []map[string]interface{}
	for _, line := range lines {
		var indexLine map[string]interface{}
		err := json.Unmarshal([]byte(line), &indexLine)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, indexLine)
	}
	return indexes, nil
}
