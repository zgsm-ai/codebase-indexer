//go:build integration
// +build integration

package codegraph

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"

	"github.com/stretchr/testify/assert"
)

var testVisitPattern = &types.VisitPattern{ExcludeDirs: []string{".git", ".idea"}, IncludeExts: []string{".go"}}

// TODO 性能（内存、cpu）监控；各种路径、项目名（中文、符号）测试；索引数量统计；大仓库测试

// testEnvironment 包含测试所需的环境组件
type testEnvironment struct {
	ctx                context.Context
	cancel             context.CancelFunc
	storageDir         string
	logger             logger.Logger
	storage            store.GraphStorage
	repository         repository.WorkspaceRepository
	workspaceReader    workspace.WorkspaceReader
	sourceFileParser   *parser.SourceFileParser
	dependencyAnalyzer *analyzer.DependencyAnalyzer
	workspaceDir       string
	scanner            repository.ScannerInterface
	tempDir            string
}

// setupTestEnvironment 设置测试环境，创建所需的目录和组件
func setupTestEnvironment(t *testing.T) *testEnvironment {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// 创建临时目录
	tempDir := t.TempDir()

	// 创建存储目录
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)
	// 创建日志目录
	logDir := filepath.Join(tempDir, "logs")
	err = os.MkdirAll(logDir, 0755)
	assert.NoError(t, err)

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "debug"
	}
	// 创建日志器
	newLogger, err := logger.NewLogger(logDir, logLevel, "codebase-indexer-test")
	assert.NoError(t, err)

	// 创建存储
	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)

	// 创建工作区读取器
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)

	// 创建源文件解析器
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	packageClassifier := packageclassifier.NewPackageClassifier()
	// 创建依赖分析器
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, packageClassifier, workspaceReader, storage)
	assert.NoError(t, err)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	// 创建临时数据库目录
	dbDir := filepath.Join(tempDir, "db")
	err = os.MkdirAll(dbDir, 0755)
	assert.NoError(t, err)

	// repository
	// Initialize database manager
	dbConfig := config.DefaultDatabaseConfig()
	dbConfig.Path = filepath.Join(dbDir, "test.db")
	dbManager := database.NewSQLiteManager(dbConfig, newLogger)
	err = dbManager.Initialize()
	if err != nil {
		panic(err)
	}
	// Initialize repositories
	workspaceRepo := repository.NewWorkspaceRepository(dbManager, newLogger)

	return &testEnvironment{
		ctx:                ctx,
		cancel:             cancel,
		storageDir:         storageDir,
		logger:             newLogger,
		storage:            storage,
		repository:         workspaceRepo,
		workspaceReader:    workspaceReader,
		sourceFileParser:   sourceFileParser,
		dependencyAnalyzer: dependencyAnalyzer,
		workspaceDir:       workspaceDir,
		scanner:            repository.NewFileScanner(newLogger),
		tempDir:            tempDir,
	}
}

// teardownTestEnvironment 清理测试环境，关闭连接和删除临时文件
func teardownTestEnvironment(t *testing.T, env *testEnvironment, projects []*workspace.Project) {
	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 关闭存储连接
	env.storage.Close()

	// 取消上下文
	env.cancel()
}

// createTestIndexer 创建测试用的索引器
func createTestIndexer(env *testEnvironment, visitPattern *types.VisitPattern) Indexer {
	return NewCodeIndexer(
		env.scanner,
		env.sourceFileParser,
		env.dependencyAnalyzer,
		env.workspaceReader,
		env.storage,
		env.repository,
		IndexerConfig{VisitPattern: visitPattern},
		env.logger,
	)
}

// countGoFiles 统计工作区中的 Go 文件数量
func countGoFiles(ctx context.Context, workspaceReader workspace.WorkspaceReader, workspaceDir string, visitPattern *types.VisitPattern) (int, error) {
	var goCount int
	err := workspaceReader.WalkFile(ctx, workspaceDir, func(walkCtx *types.WalkContext) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if strings.HasSuffix(walkCtx.Path, ".go") {
			goCount++
		}
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})
	return goCount, err
}

// countIndexedFiles 统计已索引的文件数量
func countIndexedFiles(ctx context.Context, storage store.GraphStorage, projects []*workspace.Project) (int, error) {
	var indexSize int
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if store.IsElementPathKey(key) {
				indexSize++
			}
		}
		err := iter.Close()
		if err != nil {
			return 0, err
		}
	}
	return indexSize, nil
}

// validateStorageState 验证存储状态，确保索引数量与文件数量一致
func validateStorageState(t *testing.T, ctx context.Context, workspaceReader workspace.WorkspaceReader,
	storage store.GraphStorage, workspaceDir string,
	projects []*workspace.Project, visitPattern *types.VisitPattern) {
	// 统计 Go 文件数量
	goCount, err := countGoFiles(ctx, workspaceReader, workspaceDir, visitPattern)
	assert.NoError(t, err)

	// 统计索引数量
	indexSize, err := countIndexedFiles(ctx, storage, projects)
	assert.NoError(t, err)

	// 验证 80% 解析成功
	assert.True(t, float64(indexSize) > float64(goCount)*0.8)

	// 记录存储大小
	for _, p := range projects {
		t.Logf("=> storage size: %d", storage.Size(ctx, p.Uuid, store.PathKeySystemPrefix))
	}
}

func cleanIndexStoreTest(ctx context.Context, projects []*workspace.Project, storage store.GraphStorage) error {
	for _, p := range projects {
		if err := storage.DeleteAll(ctx, p.Uuid); err != nil {
			return err
		}
		if storage.Size(ctx, p.Uuid, types.EmptyString) > 0 {
			return fmt.Errorf("clean workspace index failed, size not equal 0")
		}
	}
	return nil
}

// getTestFiles 获取测试用的文件列表
func getTestFiles(t *testing.T, workspaceDir string) []string {
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListOnlyFiles(filePath)
	assert.NoError(t, err)
	return files
}

// createPathKeyMap 创建文件路径键的映射
func createPathKeyMap(t *testing.T, files []string) map[string]any {
	pathKeys := make(map[string]any)
	for _, f := range files {
		key, err := store.ElementPathKey{Language: lang.Go, Path: f}.Get()
		assert.NoError(t, err)
		pathKeys[key] = nil
	}
	return pathKeys
}

// validateFilesNotIndexed 验证指定文件没有被索引
func validateFilesNotIndexed(t *testing.T, ctx context.Context, storage store.GraphStorage, projects []*workspace.Project, pathKeys map[string]any) {
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !store.IsElementPathKey(key) {
				continue
			}
			_, ok := pathKeys[key]
			assert.False(t, ok, "File should not be indexed: %s", key)
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
}

// validateFilesIndexed 验证指定文件已经被索引
func validateFilesIndexed(t *testing.T, ctx context.Context, storage store.GraphStorage,
	projects []*workspace.Project, pathKeys map[string]any) {
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !store.IsElementPathKey(key) {
				continue
			}
			delete(pathKeys, key)
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
	assert.True(t, len(pathKeys) == 0, "All files should be indexed")
}

// validateStorageEmpty 验证存储为空
func validateStorageEmpty(t *testing.T, ctx context.Context, storage store.GraphStorage, projects []*workspace.Project) {
	for _, p := range projects {
		size := storage.Size(ctx, p.Uuid, types.EmptyString)
		assert.Equal(t, 0, size, "Storage should be empty for project: %s", p.Uuid)
	}
}

// TestIndexer_IndexWorkspace 测试索引器的 IndexWorkspace 方法
// 该测试验证索引器能够正确地索引整个工作区，并确保索引的文件数量与实际的文件数量一致
func TestIndexer_IndexWorkspace(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)

	if err := initWorkspaceModel(env); err != nil {
		panic(err)
	}

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 测试 IndexWorkspace - 索引整个工作区
	_, err := indexer.IndexWorkspace(context.Background(), env.workspaceDir)
	assert.NoError(t, err)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

	// 验证存储状态 - 确保索引数量与文件数量一致
	validateStorageState(t, env.ctx, env.workspaceReader, env.storage, env.workspaceDir, projects, testVisitPattern)

	// 清理测试环境
	teardownTestEnvironment(t, env, projects)
}

func initWorkspaceModel(env *testEnvironment) error {
	workspaceModel, _ := env.repository.GetWorkspaceByPath(env.workspaceDir)
	if workspaceModel == nil {
		// 初始化workspace
		err := env.repository.CreateWorkspace(&model.Workspace{
			WorkspaceName: "codebase-indexer",
			WorkspacePath: env.workspaceDir,
			Active:        "true",
			FileNum:       100,
		})
		if err != nil {
			return err
		}
	} else {
		// 置为 0
		err := env.repository.UpdateCodegraphInfo(env.workspaceDir, 0, time.Now().Unix())
		if err != nil {
			return err
		}
	}
	return nil
}

func TestIndexer_IndexProjectFilesWhenProjectHasIndex(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)
	// 创建测试索引器，使用排除 mocks 目录的访问模式
	newVisitPattern := &types.VisitPattern{ExcludeDirs: []string{".git", ".idea", ".vscode", "mocks"}, IncludeExts: []string{".go"}}
	codeIndexer := createTestIndexer(env, newVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, newVisitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 先索引工作区（排除 mocks 目录）
	// TODO 过滤mocks目录 有问题，待进一步排查
	_, err = codeIndexer.IndexWorkspace(env.ctx, env.workspaceDir)
	assert.NoError(t, err)
	summary, err := codeIndexer.GetSummary(context.Background(), env.workspaceDir)
	assert.NoError(t, err)
	assert.True(t, summary.TotalFiles > 0)
	// 步骤2: 获取测试文件并创建路径键映射
	files := getTestFiles(t, env.workspaceDir)
	pathKeys := createPathKeyMap(t, files)
	// 步骤3: 验证 mocks 目录文件没有被索引
	validateFilesNotIndexed(t, env.ctx, env.storage, projects, pathKeys)

	// 步骤4: 测试索引特定文件
	// 直接索引文件，不再调用内部方法
	err = codeIndexer.IndexFiles(context.Background(), env.workspaceDir, files)
	assert.NoError(t, err)

	// 步骤5: 验证 mocks 目录文件现在已经被索引
	validateFilesIndexed(t, env.ctx, env.storage, projects, pathKeys)
}

func TestIndexer_IndexProjectFilesWhenProjectHasNoIndex(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)
	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 验证存储初始状态为空
	validateStorageEmpty(t, env.ctx, env.storage, projects)

	// 步骤2: 获取测试文件
	files := getTestFiles(t, env.workspaceDir)

	// 步骤3: 测试索引特定文件（当项目没有索引时）
	// TODO 这里有问题，待进一步排查
	err = indexer.IndexFiles(context.Background(), env.workspaceDir, files)
	assert.NoError(t, err)

	// 步骤4: 创建路径键映射
	pathKeys := createPathKeyMap(t, files)

	// 步骤5: 验证 mocks 目录文件已经被索引
	validateFilesIndexed(t, env.ctx, env.storage, projects, pathKeys)

	// 步骤6: 验证存储状态 - 确保索引数量与文件数量一致
	validateStorageState(t, env.ctx, env.workspaceReader, env.storage, env.workspaceDir, projects, testVisitPattern)
}

func TestIndexer_RemoveIndexes(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)
	err := initWorkspaceModel(env)
	assert.NoError(t, err)
	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 获取测试文件
	files := getTestFiles(t, env.workspaceDir)

	// 步骤2: 索引测试文件
	err = indexer.IndexFiles(context.Background(), env.workspaceDir, files)
	assert.NoError(t, err)

	// 步骤3: 创建路径键映射
	pathKeys := createPathKeyMap(t, files)
	pathKeysBak := createPathKeyMap(t, files)
	assert.True(t, len(pathKeys) > 0)
	assert.True(t, len(pathKeysBak) > 0)

	// 步骤4: 验证文件已被索引
	validateFilesIndexed(t, env.ctx, env.storage, projects, pathKeys)

	// 步骤5: 统计索引前的总数
	totalIndexBefore, err := countIndexedFiles(env.ctx, env.storage, projects)
	assert.NoError(t, err)

	// 步骤6: 删除索引
	err = indexer.RemoveIndexes(env.ctx, env.workspaceDir, files)
	assert.NoError(t, err)

	// 步骤7: 统计索引后的总数
	totalIndexAfter, err := countIndexedFiles(env.ctx, env.storage, projects)
	assert.NoError(t, err)

	// 步骤8: 验证删除结果
	validateFilesNotIndexed(t, env.ctx, env.storage, projects, pathKeysBak)
	assert.Equal(t, len(files), len(pathKeysBak))
	assert.True(t, totalIndexBefore-len(files) <= totalIndexAfter)
}

func TestIndexer_QueryElements(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	codeIndexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	_, err = codeIndexer.IndexWorkspace(env.ctx, env.workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 获取测试文件
	files := getTestFiles(t, env.workspaceDir)
	assert.True(t, len(files) > 0)

	// 步骤3: 查询元素
	// TODO: queryElements 已经是私有方法，需要通过其他方式测试
	// elementTables, err := codeIndexer.queryElements(env.ctx, env.workspaceDir, files)
	// assert.NoError(t, err)
	// assert.True(t, len(elementTables) > 0)

	// 步骤4: 验证查询结果
	// TODO: 需要通过公共接口测试
	// for _, table := range elementTables {
	// 	assert.NotEmpty(t, table.Path)
	// 	assert.NotEmpty(t, table.Elements)
	// }
}

func TestIndexer_QuerySymbols_WithExistFile(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	testIndexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	_, err = testIndexer.IndexWorkspace(env.ctx, env.workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 准备测试文件和符号名称
	_ = filepath.Join(env.workspaceDir, "test", "mocks", "mock_graph_store.go")
	_ = []string{"MockGraphStorage", "BatchSave"}

	// 步骤3: 查询符号
	// TODO: querySymbols 已经是私有方法，需要通过其他方式测试
	// symbols, err := testIndexer.querySymbols(env.ctx, env.workspaceDir, filePath, symbolNames)
	// assert.NoError(t, err)
	// assert.Equal(t, 2, len(symbols))

	// 步骤4: 验证符号信息
	// TODO: 需要通过公共接口测试
}

func TestIndexer_IndexWorkspace_NotExists(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 步骤1: 使用不存在的工作区路径
	nonExistentWorkspace := filepath.Join(t.TempDir(), "non_existent_workspace")

	// 步骤2: 尝试索引不存在的工作区，应该返回错误
	_, err := indexer.IndexWorkspace(env.ctx, nonExistentWorkspace)
	assert.ErrorContains(t, err, "not exists")
}

func TestIndexer_IndexFiles_NoProject(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 步骤1: 使用不存在的项目路径
	nonExistentWorkspace := filepath.Join(t.TempDir(), "non_existent_workspace")
	testFiles := []string{"test.go"}

	// 步骤2: 尝试索引不存在的项目中的文件，应该返回错误
	err := indexer.IndexFiles(env.ctx, nonExistentWorkspace, testFiles)
	assert.ErrorContains(t, err, "not exists")
}

// printCallGraphToFile 将调用链以层次结构打印到文件
func printCallGraphToFile(t *testing.T, nodes []*types.RelationNode, filename string) {
	output, err := os.Create(filename)
	assert.NoError(t, err)
	defer output.Close()

	fmt.Fprintf(output, "调用链分析结果 (生成时间: %s)\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(output, "===========================================\n\n")

	for i, node := range nodes {
		fmt.Fprintf(output, "根节点 %d:\n", i+1)
		printNodeRecursive(output, node, 0)
		fmt.Fprintf(output, "\n")
	}
}

// printNodeRecursive 递归打印节点及其子节点
func printNodeRecursive(output *os.File, node *types.RelationNode, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(output, "%s├─ %s [%s]\n", indent, node.SymbolName, node.NodeType)
	fmt.Fprintf(output, "%s   文件: %s\n", indent, node.FilePath)
	if node.Position.StartLine > 0 || node.Position.EndLine > 0 {
		fmt.Fprintf(output, "%s   位置: 行%d-%d, 列%d-%d\n", indent,
			node.Position.StartLine, node.Position.EndLine,
			node.Position.StartColumn, node.Position.EndColumn)
	}
	if node.Content != "" {
		// 只显示内容的前50个字符
		content := strings.ReplaceAll(node.Content, "\n", "\\n")
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		fmt.Fprintf(output, "%s   内容: %s\n", indent, content)
	}
	fmt.Fprintf(output, "%s   子调用数: %d\n", indent, len(node.Children))

	for _, child := range node.Children {
		printNodeRecursive(output, child, depth+1)
	}
}

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
			env := setupTestEnvironment(t)
			defer teardownTestEnvironment(t, env, nil)

			if tc.workspaceDir != "" {
				env.workspaceDir = tc.workspaceDir
			}
			testVisitPattern.IncludeExts = tc.IncludeExts

			err := initWorkspaceModel(env)
			assert.NoError(t, err)
			// 创建测试索引器
			testIndexer := createTestIndexer(env, testVisitPattern)

			// 查找工作区中的项目
			projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

			// 清理索引存储
			err = cleanIndexStoreTest(env.ctx, projects, env.storage)
			assert.NoError(t, err)
			indexStart := time.Now()
			// 步骤1: 索引整个工作区
			metrics, err := testIndexer.IndexWorkspace(env.ctx, env.workspaceDir)
			assert.NoError(t, err)
			indexEnd := time.Now()

			// 构建完整文件路径
			start := time.Now()
			fullPath := filepath.Join(env.workspaceDir, tc.filePath)
			// 查询调用链
			opts := &types.QueryCallGraphOptions{
				Workspace:  env.workspaceDir,
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
			env := setupTestEnvironment(t)
			defer teardownTestEnvironment(t, env, nil)

			if tc.workspaceDir != "" {
				env.workspaceDir = tc.workspaceDir
			}
			testVisitPattern.IncludeExts = tc.IncludeExts

			err := initWorkspaceModel(env)
			assert.NoError(t, err)
			// 创建测试索引器
			testIndexer := createTestIndexer(env, testVisitPattern)

			// 查找工作区中的项目
			projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

			// 清理索引存储
			err = cleanIndexStoreTest(env.ctx, projects, env.storage)
			assert.NoError(t, err)
			indexStart := time.Now()
			// 步骤1: 索引整个工作区
			metrics, err := testIndexer.IndexWorkspace(env.ctx, env.workspaceDir)
			assert.NoError(t, err)
			indexEnd := time.Now()

			start := time.Now()
			// 构建完整文件路径
			fullPath := filepath.Join(env.workspaceDir, tc.filePath)
			// 查询调用链
			opts := &types.QueryCallGraphOptions{
				Workspace: env.workspaceDir,
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
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	err := initWorkspaceModel(env)
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
				Workspace: env.workspaceDir,
				FilePath:  filepath.Join(env.workspaceDir, "internal/service/indexer.go"),
				MaxLayer:  3,
			},
			expectErr:   true,
			expectNodes: false,
			desc:        "既没有符号名也没有行范围应该返回错误",
		},
		{
			name: "不存在的文件",
			opts: &types.QueryCallGraphOptions{
				Workspace:  env.workspaceDir,
				FilePath:   filepath.Join(env.workspaceDir, "non_existent_file.go"),
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
				Workspace: env.workspaceDir,
				FilePath:  filepath.Join(env.workspaceDir, "internal/service/indexer.go"),
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
