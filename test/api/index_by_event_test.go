package api

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	preProcess      func(*IndexEventIntegrationTestSuite) error
	postProcess     func(*IndexEventIntegrationTestSuite) error
	wantProcessTime time.Duration
	validateIndex   func(s *IndexEventIntegrationTestSuite) // 索引数和文件数、数据库数据一致
}

func (s *IndexEventIntegrationTestSuite) TestIndexByEvent() {
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
			preProcess: func(suite *IndexEventIntegrationTestSuite) error {
				err := s.deleteAllIndex()
				if err != nil {
					return err
				}
				// 校验索引存储和数据库都已清空
				s.validateIndexCount(0, 0) // 期望数量为0，不检查容差
				return nil
			},
			wantProcessTime: time.Second * 15,
			validateIndex: func(s *IndexEventIntegrationTestSuite) {
				// 验证索引数量大于0且数据库和图存储索引数量一致（10%容差）
				s.validateIndexCount(1, 0.1)
			},
		},

		{
			name:      "删除文件",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "delete_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
					"targetPath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
				},
			},
			preProcess: func(testSuite *IndexEventIntegrationTestSuite) error {
				if err := s.indexWorkspace(); err != nil {
					s.T().Fatal(err)
				}

				time.Sleep(time.Second * 10)
				// 验证索引数量和文件存在性
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, true) // 期望文件存在
				return nil
			},
			wantProcessTime: time.Second * 2,
			validateIndex: func(s *IndexEventIntegrationTestSuite) {
				// 验证索引数量和文件已被删除
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, false) // 期望文件不存在
			},
		},
		{
			name:      "删除文件夹",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "delete_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": filepath.Join(s.workspacePath, "test", "api"),
					"targetPath": filepath.Join(s.workspacePath, "test", "api"),
				},
			},
			wantProcessTime: time.Second * 2,
			preProcess: func(s *IndexEventIntegrationTestSuite) error {
				if err := s.indexWorkspace(); err != nil {
					s.T().Fatal(err)
				}

				time.Sleep(time.Second * 10)
				// 验证索引数量和文件存在性
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, true) // 期望文件存在
				return nil
			},
			validateIndex: func(s *IndexEventIntegrationTestSuite) {
				// 验证索引数量
				s.validateIndexCount(1, 0.1)

				// 校验该文件夹下面文件的索引已经被删除
				deletedPath, err := utils.ListOnlyFiles(filepath.Join(s.workspacePath, "test", "api"))
				if err != nil {
					s.T().Fatal(err)
				}
				if len(deletedPath) == 0 {
					s.T().Fatalf("no files found in %s", filepath.Join(s.workspacePath, "test", "api"))
				}

				// 验证这些文件都不存在于索引中
				s.checkFilesInIndex(deletedPath, false) // 期望文件不存在
			},
		},
		{
			name:      "新增文件",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "add_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
					"targetPath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
				},
			},
			wantProcessTime: time.Second * 2,
			preProcess: func(s *IndexEventIntegrationTestSuite) error {
				if err := s.indexWorkspace(); err != nil {
					s.T().Fatal(err)
				}

				time.Sleep(time.Second * 10)
				// 删除文件
				if err := s.publishEvent(s.workspacePath, []map[string]interface{}{
					{
						"eventType":  "delete_file",
						"eventTime":  "2025-07-28 20:47:00",
						"sourcePath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
						"targetPath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
					},
				}); err != nil {
					s.T().Fatal(err)
				}
				time.Sleep(time.Second * 5)
				// 验证索引数量和文件已被删除
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, false) // 期望文件不存在
				return nil
			},
			validateIndex: func(s *IndexEventIntegrationTestSuite) {
				// 验证索引数量和文件已新增
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, true) // 期望文件存在
			},
		},
		{
			name:      "修改文件",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "modify_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
					"targetPath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
				},
			},
			wantProcessTime: time.Second * 2,
			preProcess: func(s *IndexEventIntegrationTestSuite) error {
				if err := s.indexWorkspace(); err != nil {
					s.T().Fatal(err)
				}

				time.Sleep(time.Second * 10)
				// 删除文件
				if err := s.publishEvent(s.workspacePath, []map[string]interface{}{
					{
						"eventType":  "delete_file",
						"eventTime":  "2025-07-28 20:47:00",
						"sourcePath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
						"targetPath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
					},
				}); err != nil {
					s.T().Fatal(err)
				}
				time.Sleep(time.Second * 5)
				// 验证索引数量和文件已被删除
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, false) // 期望文件不存在
				return nil
			},
			validateIndex: func(s *IndexEventIntegrationTestSuite) {
				// 验证索引数量和文件已修改（重新索引）
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, true) // 期望文件存在
			},
		},
		{
			name:      "重命名文件",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "rename_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go"),
					"targetPath": filepath.Join(s.workspacePath, "test", "api", "new_get_file_structure_test.go"),
				},
			},
			wantProcessTime: time.Second * 2,
			preProcess: func(s *IndexEventIntegrationTestSuite) error {
				if err := s.deleteAllIndex(); err != nil {
					s.T().Fatal(err)
				}
				if err := s.indexWorkspace(); err != nil {
					s.T().Fatal(err)
				}

				time.Sleep(time.Second * 10)
				// 验证索引数量和文件存在性
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, true) // 期望文件存在
				return nil
			},
			validateIndex: func(s *IndexEventIntegrationTestSuite) {
				// 重命名：删除老的，添加新的
				s.validateIndexCount(1, 0.1)

				// 验证旧文件不存在，新文件存在
				oldFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				newFile := filepath.Join(s.workspacePath, "test", "api", "new_get_file_structure_test.go")

				s.checkFileInIndex(oldFile, false) // 期望旧文件不存在
				s.checkFileInIndex(newFile, true)  // 期望新文件存在
			},
		},
		{
			name:      "重命名文件夹",
			workspace: s.workspacePath,
			data: []map[string]interface{}{
				{
					"eventType":  "rename_file",
					"eventTime":  "2025-07-28 20:47:00",
					"sourcePath": filepath.Join(s.workspacePath, "test", "api"),
					"targetPath": filepath.Join(s.workspacePath, "new_test", "api"),
				},
			},
			wantProcessTime: time.Second * 10,
			preProcess: func(s *IndexEventIntegrationTestSuite) error {
				if err := os.MkdirAll(filepath.Join(s.workspacePath, "new_test", "api"), 0755); err != nil {
					s.T().Fatal(err)
				}

				if err := s.deleteAllIndex(); err != nil {
					s.T().Fatal(err)
				}
				if err := s.indexWorkspace(); err != nil {
					s.T().Fatal(err)
				}
				time.Sleep(time.Second * 15)
				// 验证索引数量和文件存在性
				s.validateIndexCount(1, 0.1)
				testFile := filepath.Join(s.workspacePath, "test", "api", "get_file_structure_test.go")
				s.checkFileInIndex(testFile, true) // 期望文件存在
				return nil
			},
			validateIndex: func(s *IndexEventIntegrationTestSuite) {
				// 重命名：删除老的，添加新的
				s.validateIndexCount(1, 0.1)

				oldPaths, err := utils.ListOnlyFiles(filepath.Join(s.workspacePath, "test", "api"))
				if err != nil {
					s.T().Fatal(err)
				}
				if len(oldPaths) == 0 {
					s.T().Fatalf("no files found in %s", filepath.Join(s.workspacePath, "test", "api"))
				}
				newPaths := make([]string, 0)
				for _, path := range oldPaths {
					newPaths = append(newPaths, strings.Replace(path, string(filepath.Separator)+"test"+
						string(filepath.Separator), string(filepath.Separator)+
						"new_test"+string(filepath.Separator), 1))
				}

				// 验证旧路径文件不存在，新路径文件存在
				s.checkFilesInIndex(oldPaths, false) // 期望旧文件不存在
				s.checkFilesInIndex(newPaths, true)  // 期望新文件存在
			},
			postProcess: func(testSuite *IndexEventIntegrationTestSuite) error {
				if err := os.RemoveAll(filepath.Join(s.workspacePath, "new_test")); err != nil {
					s.T().Fatal(err)
				}
				return nil
			},
		},
	}

	// 执行表格驱动测试
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {

			if tc.preProcess != nil {
				if err := tc.preProcess(s); err != nil {
					t.Fatal(err)
				}
			}
			// 发布事件
			err := s.publishEvent(tc.workspace, tc.data)
			if err != nil {
				panic(err)
			}
			// 等待一定时间
			t.Logf("waiting for %f second for %s event completed", tc.wantProcessTime.Seconds(), tc.name)
			time.Sleep(tc.wantProcessTime)
			// 查询索引，检查达到期望状态
			tc.validateIndex(s)

			if tc.postProcess != nil {
				if err := tc.postProcess(s); err != nil {
					t.Fatal(err)
				}
			}
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
	// 读取响应体
	body, _ := io.ReadAll(resp.Body)

	// 验证响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body:%s", resp.StatusCode, string(body))
	}

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

// deleteAllIndex 清空索引
func (s *IndexEventIntegrationTestSuite) deleteAllIndex() error {
	var resp *http.Response
	var err error
	reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/index")
	s.Require().NoError(err)

	// 添加查询参数
	q := reqURL.Query()
	q.Add("clientId", s.workspacePath)
	q.Add("codebasePath", s.workspacePath)
	q.Add("indexType", "codegraph")
	reqURL.RawQuery = q.Encode()

	// 创建HTTP请求
	req, err := s.CreateDeleteRequest(reqURL.String())
	s.Require().NoError(err)

	// 发送请求
	resp, err = s.SendRequest(req)

	s.Require().NoError(err)
	defer resp.Body.Close()
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	// 验证响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body:%s", resp.StatusCode, string(body))
	}

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
	q.Add("clientId", s.clientId)

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
		if line == "" {
			continue
		}
		var indexLine map[string]interface{}
		err := json.Unmarshal([]byte(line), &indexLine)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, indexLine)
	}
	return indexes, nil
}

func (s *IndexEventIntegrationTestSuite) countFileElementTables(indexes []map[string]interface{}) int {
	count := 0
	for _, index := range indexes {
		if v, ok := index["path"]; ok && v.(string) != types.EmptyString {
			count++
		}
	}
	return count
}

// validateIndexCount 验证索引数量和数据库索引数量的一致性
// expectedMinCount: 期望的最小索引数量，如果为0则只验证不为0
// tolerance: 容差比例，默认为0.1（10%）
func (s *IndexEventIntegrationTestSuite) validateIndexCount(expectedMinCount int, tolerance float32) {
	// 获取索引数量
	indexes, err := s.dumpIndex()
	if err != nil {
		s.T().Fatal(err)
	}
	graphIndexNum := s.countFileElementTables(indexes)

	// 验证最小数量
	if expectedMinCount > 0 && graphIndexNum < expectedMinCount {
		s.T().Fatalf("graph index num %d is less than expected min count %d", graphIndexNum, expectedMinCount)
	} else if expectedMinCount == 0 && graphIndexNum <= 0 {
		s.T().Fatal("graph index num is zero")
	}

	// 获取数据库索引数量
	databaseIndexNum, err := s.getDatabaseIndexNum()
	if err != nil {
		s.T().Fatal(err)
	}

	if expectedMinCount > 0 && databaseIndexNum < expectedMinCount {
		s.T().Fatalf("database index num %d is less than expected min count %d", databaseIndexNum, expectedMinCount)
	} else if expectedMinCount == 0 && databaseIndexNum <= 0 {
		s.T().Fatal("database index num is zero")
	}

	// 验证一致性
	if tolerance > 0 && graphIndexNum <= int(float32(databaseIndexNum)*(1-tolerance)) {
		s.T().Fatalf("database index num %d not equal to graph store index num %d (tolerance: %.2f)", databaseIndexNum, graphIndexNum, tolerance)
	}
}

// checkFileInIndex 检查文件是否存在于索引中
// filePath: 要检查的文件路径
// expectExists: 期望文件是否存在
func (s *IndexEventIntegrationTestSuite) checkFileInIndex(filePath string, expectExists bool) (elementTableFound bool, symbolFound bool) {
	indexes, err := s.dumpIndex()
	if err != nil {
		s.T().Fatal(err)
	}

	for _, index := range indexes {
		if elementTableFound && symbolFound {
			break
		}

		// 检查元素表
		if path, ok := index["path"]; ok && path.(string) == filePath {
			elementTableFound = true
		}

		// 检查符号
		if _, ok := index["name"]; ok {
			if occs, ok := index["occurrences"]; ok {
				occSlice := occs.([]interface{})
				for _, v := range occSlice {
					vMap := v.(map[string]interface{})
					if path, ok := vMap["path"]; ok && path.(string) == filePath {
						symbolFound = true
					}
				}
			}
		}
	}

	// 验证结果是否符合期望
	if expectExists && (!elementTableFound || !symbolFound) {
		s.T().Fatalf("file %s should exist but element table found: %t, symbol found: %t", filePath, elementTableFound, symbolFound)
	} else if !expectExists && (elementTableFound || symbolFound) {
		s.T().Fatalf("file %s should not exist but element table found: %t, symbol found: %t", filePath, elementTableFound, symbolFound)
	}

	return elementTableFound, symbolFound
}

// checkFilesInIndex 批量检查文件是否存在于索引中
// filePaths: 要检查的文件路径列表
// expectExists: 期望文件是否存在
func (s *IndexEventIntegrationTestSuite) checkFilesInIndex(filePaths []string, expectExists bool) {
	indexes, err := s.dumpIndex()
	if err != nil {
		s.T().Fatal(err)
	}

	for _, filePath := range filePaths {
		elementTableFound := false
		symbolFound := false

		for _, index := range indexes {
			if elementTableFound && symbolFound {
				break
			}

			// 检查元素表
			if path, ok := index["path"]; ok && path.(string) == filePath {
				elementTableFound = true
			}

			// 检查符号
			if _, ok := index["name"]; ok {
				if occs, ok := index["occurrences"]; ok {
					occSlice := occs.([]interface{})
					for _, v := range occSlice {
						vMap := v.(map[string]interface{})
						if path, ok := vMap["path"]; ok && path.(string) == filePath {
							symbolFound = true
						}
					}
				}
			}
		}

		// 验证结果是否符合期望
		if expectExists && (!elementTableFound || !symbolFound) {
			s.T().Fatalf("file %s should exist but element table found: %t, symbol found: %t", filePath, elementTableFound, symbolFound)
		} else if !expectExists && (elementTableFound || symbolFound) {
			s.T().Fatalf("file %s should not exist but element table found: %t, symbol found: %t", filePath, elementTableFound, symbolFound)
		}
	}
}

func (s *IndexEventIntegrationTestSuite) indexWorkspace() error {
	err := s.publishEvent(s.workspacePath, []map[string]interface{}{
		{
			"eventType": "open_workspace",
			"eventTime": "2025-07-28 20:47:00",
		},
	})
	return err

}

func (s *IndexEventIntegrationTestSuite) getDatabaseIndexNum() (int, error) {
	reqURL, err := url.Parse(s.baseURL + "/codebase-indexer/api/v1/index/status")
	s.Require().NoError(err)

	// 添加查询参数
	q := reqURL.Query()
	q.Add("workspace", s.workspacePath)
	reqURL.RawQuery = q.Encode()

	// 创建HTTP请求
	req, err := s.CreateGETRequest(reqURL.String())
	s.Require().NoError(err)

	// 发送请求
	resp, err := s.SendRequest(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d, body:%s", resp.StatusCode, string(body))
	}

	// 解析响应JSON
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}
	if response["message"] != "ok" {
		return 0, fmt.Errorf("expected message `ok` but got `%s`", response["message"])
	}
	data := response["data"].(map[string]interface{})
	codegraph := data["codegraph"].(map[string]interface{})

	return int(codegraph["totalSucceed"].(float64)), nil
}
