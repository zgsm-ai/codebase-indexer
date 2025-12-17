//go:build integration
// +build integration

package codegraph

import (
	"codebase-indexer/pkg/codegraph/types"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestIndexer_QueryCallGraph_BySymbolName 测试基于符号名的调用链查询
// 注意：这是端到端集成测试，需要真实的项目环境，因此在单元测试中跳过
func TestIndexer_QueryCallGraph_BySymbolName(t *testing.T) {
	t.Skip("这是端到端测试，需要真实的项目环境，跳过单元测试")

	// CPU profiling
	cpuFile, err := os.Create("cpu.pprof")
	if err != nil {
		panic(err)
	}
	defer cpuFile.Close()

	// Memory profiling
	memFile, err := os.Create("mem.pprof")
	if err != nil {
		panic(err)
	}
	defer memFile.Close()
	defer pprof.WriteHeapProfile(memFile)

	assert.NoError(t, err)
	pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()

	// 步骤2: 测试基于符号名的调用链查询
	testCases := []struct {
		name         string
		filePath     string
		symbolName   string
		maxLayer     int
		desc         string
		project      string
		workspaceDir string
		IncludeExts  []string
	}{
		{
			name:         "setupTestEnvironment方法调用链",
			filePath:     "internal/service/indexer_integration_test.go",
			symbolName:   "setupTestEnvironment",
			maxLayer:     20,
			desc:         "查询setupTestEnvironment方法的调用链",
			project:      "codebase-indexer",
			workspaceDir: "", // 使用默认工作区
			IncludeExts:  []string{".go"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置测试环境
			env, err := setupTestEnvironment()
			assert.NoError(t, err)
			defer teardownTestEnvironment(t, env)

			workspaceDir := tc.workspaceDir
			if workspaceDir == "" {
				workspaceDir, err = filepath.Abs("../../")
				assert.NoError(t, err)
			}

			testVisitPattern.IncludeExts = tc.IncludeExts

			err = initWorkspaceModel(env, workspaceDir)
			assert.NoError(t, err)

			// 创建测试索引器
			testIndexer := createTestIndexer(env, testVisitPattern)

			// 查找工作区中的项目
			projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, testVisitPattern)

			// 清理索引存储
			err = cleanIndexStoreTest(env.ctx, projects, env.storage)
			assert.NoError(t, err)

			indexStart := time.Now()

			// 步骤1: 索引整个工作区
			metrics, err := testIndexer.IndexWorkspace(env.ctx, workspaceDir)
			assert.NoError(t, err)
			indexEnd := time.Now()

			// 构建完整文件路径
			start := time.Now()
			fullPath := filepath.Join(workspaceDir, tc.filePath)

			// 查询调用链
			opts := &types.QueryCallGraphOptions{
				Workspace:  workspaceDir,
				FilePath:   fullPath,
				SymbolName: tc.symbolName,
				MaxLayer:   tc.maxLayer,
			}

			nodes, err := testIndexer.QueryCallGraph(env.ctx, opts)
			assert.NoError(t, err)

			// 验证结果
			assert.NotNil(t, nodes, "调用链结果不应为空")
			fmt.Printf("符号 %s 的调用链包含 %d 个根节点\n", tc.symbolName, len(nodes))
			fmt.Printf("查询调用链时间: %s\n", time.Since(start))
			fmt.Printf("索引项目 %s 时间: %s, 索引 %d 个文件\n", tc.project, indexEnd.Sub(indexStart), metrics.TotalFiles)

			// 将结果输出到文件
			outputFile := filepath.Join(t.TempDir(), fmt.Sprintf("callgraph_%s_%s_symbol.txt", tc.symbolName, tc.project))
			printCallGraphToFile(t, nodes, outputFile)
			fmt.Printf("调用链输出到文件: %s\n", outputFile)

			// 基本验证
			if len(nodes) > 0 {
				for _, node := range nodes {
					assert.NotEmpty(t, node.SymbolName, "符号名不应为空")
					assert.NotEmpty(t, node.FilePath, "文件路径不应为空")
					assert.Equal(t, tc.symbolName, node.SymbolName, "根节点符号名应该匹配")
				}
			}
		})
	}
}

// TestIndexer_QueryCallGraph_ByLineRange 测试基于行范围的调用链查询
// 注意：这是端到端集成测试，需要真实的项目环境，因此在单元测试中跳过
func TestIndexer_QueryCallGraph_ByLineRange(t *testing.T) {
	t.Skip("这是端到端测试，需要真实的项目环境，跳过单元测试")

	// 步骤1: 测试基于行范围的调用链查询
	testCases := []struct {
		name         string
		filePath     string
		startLine    int
		endLine      int
		maxLayer     int
		desc         string
		project      string
		workspaceDir string
		IncludeExts  []string
	}{
		{
			name:         "setupTestEnvironment函数范围",
			filePath:     "internal/service/indexer_integration_test.go",
			startLine:    52, // setupTestEnvironment函数开始行
			endLine:      70, // 函数部分范围
			maxLayer:     2,
			desc:         "查询setupTestEnvironment函数范围内的调用链",
			project:      "codebase-indexer",
			workspaceDir: "",
			IncludeExts:  []string{".go"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置测试环境
			env, err := setupTestEnvironment()
			assert.NoError(t, err)
			defer teardownTestEnvironment(t, env)

			workspaceDir := tc.workspaceDir
			if workspaceDir == "" {
				workspaceDir, err = filepath.Abs("../../")
				assert.NoError(t, err)
			}
			testVisitPattern.IncludeExts = tc.IncludeExts

			err = initWorkspaceModel(env, workspaceDir)
			assert.NoError(t, err)

			// 创建测试索引器
			testIndexer := createTestIndexer(env, testVisitPattern)

			// 查找工作区中的项目
			projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, testVisitPattern)

			// 清理索引存储
			err = cleanIndexStoreTest(env.ctx, projects, env.storage)
			assert.NoError(t, err)

			indexStart := time.Now()

			// 步骤1: 索引整个工作区
			metrics, err := testIndexer.IndexWorkspace(env.ctx, workspaceDir)
			assert.NoError(t, err)
			indexEnd := time.Now()

			start := time.Now()

			// 构建完整文件路径
			fullPath := filepath.Join(workspaceDir, tc.filePath)

			// 查询调用链
			opts := &types.QueryCallGraphOptions{
				Workspace: workspaceDir,
				FilePath:  fullPath,
				LineRange: fmt.Sprintf("%d-%d", tc.startLine, tc.endLine),
				MaxLayer:  tc.maxLayer,
			}

			nodes, err := testIndexer.QueryCallGraph(env.ctx, opts)
			assert.NoError(t, err)

			// 验证结果
			assert.NotNil(t, nodes, "调用链结果不应为空")
			fmt.Printf("行范围 %d-%d 的调用链包含 %d 个根节点\n", tc.startLine, tc.endLine, len(nodes))
			fmt.Printf("查询调用链时间: %s\n", time.Since(start))
			fmt.Printf("索引项目 %s 时间: %s, 索引 %d 个文件\n", tc.project, indexEnd.Sub(indexStart), metrics.TotalFiles)

			// 将结果输出到文件
			outputFile := filepath.Join(t.TempDir(), fmt.Sprintf("callgraph_lines_%d_%d_%s.txt", tc.startLine, tc.endLine, tc.project))
			printCallGraphToFile(t, nodes, outputFile)
			fmt.Printf("调用链输出到文件: %s\n", outputFile)

			// 基本验证
			if len(nodes) > 0 {
				for _, node := range nodes {
					assert.NotEmpty(t, node.SymbolName, "符号名不应为空")
					assert.NotEmpty(t, node.FilePath, "文件路径不应为空")
					assert.Equal(t, string(types.NodeTypeDefinition), node.NodeType, "根节点应该是定义类型")
				}
			}
		})
	}
}

// TestIndexer_QueryCallGraph_InvalidOptions 测试无效参数的情况
func TestIndexer_QueryCallGraph_InvalidOptions(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	err = initWorkspaceModel(env, workspaceDir)
	assert.NoError(t, err)

	// 创建测试索引器
	testIndexer := createTestIndexer(env, testVisitPattern)

	// 测试无效选项
	testCases := []struct {
		name        string
		opts        *types.QueryCallGraphOptions
		expectErr   bool
		expectNodes bool
		desc        string
	}{
		{
			name: "无符号名且无行范围",
			opts: &types.QueryCallGraphOptions{
				Workspace: workspaceDir,
				FilePath:  filepath.Join(workspaceDir, "internal/service/indexer.go"),
				MaxLayer:  3,
			},
			expectErr:   true,
			expectNodes: false,
			desc:        "既没有符号名也没有行范围应该返回错误",
		},
		{
			name: "不存在的文件",
			opts: &types.QueryCallGraphOptions{
				Workspace:  workspaceDir,
				FilePath:   filepath.Join(workspaceDir, "non_existent_file.go"),
				SymbolName: "SomeFunction",
				MaxLayer:   3,
			},
			expectErr:   true,
			expectNodes: false,
			desc:        "不存在的文件应该返回错误",
		},
		{
			name: "无效的行范围",
			opts: &types.QueryCallGraphOptions{
				Workspace: workspaceDir,
				FilePath:  filepath.Join(workspaceDir, "internal/service/indexer.go"),
				LineRange: "100-50",
				MaxLayer:  3,
			},
			expectErr:   false, // 应该会被NormalizeLineRange处理
			expectNodes: false,
			desc:        "无效的行范围会被自动修正",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := testIndexer.QueryCallGraph(env.ctx, tc.opts)

			if tc.expectErr {
				assert.Error(t, err, tc.desc)
			} else {
				assert.NoError(t, err, tc.desc)
				if tc.expectNodes {
					assert.NotNil(t, nodes, "调用链结果不应为空")
				} else {
					assert.Nil(t, nodes, "调用链结果应为空")
				}
			}
		})
	}
}

// TestIndexer_QueryDefinitionsBySymbolName 测试基于符号名的定义查询
// 注意：这是端到端集成测试，需要真实的项目环境，因此在单元测试中跳过
func TestIndexer_QueryDefinitionsBySymbolName(t *testing.T) {
	t.Skip("这是端到端测试，需要真实的项目环境，跳过单元测试")
}
