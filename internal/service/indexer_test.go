package service

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
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
	"slices"
	"strings"
	"testing"
	"time"

	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"

	"github.com/stretchr/testify/assert"
)

var tempDir = "/tmp/"

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
}

// setupTestEnvironment 设置测试环境，创建所需的目录和组件
func setupTestEnvironment(t *testing.T) *testEnvironment {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// 创建存储目录
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "debug"
	}
	// 创建日志器
	newLogger, err := logger.NewLogger("/tmp/logs", logLevel, "codebase-indexer")
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

	// 获取测试工作区目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	// repository
	// Initialize database manager
	dbConfig := config.DefaultDatabaseConfig()
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
	filesByProject, err := codeIndexer.(*indexer).groupFilesByProject(projects, files)
	assert.NoError(t, err)
	for _, projectFiles := range filesByProject {
		err = codeIndexer.IndexFiles(context.Background(), env.workspaceDir, projectFiles)
		assert.NoError(t, err)
	}

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
	elementTables, err := codeIndexer.(*indexer).queryElements(env.ctx, env.workspaceDir, files)
	assert.NoError(t, err)
	assert.True(t, len(elementTables) > 0)

	// 步骤4: 验证查询结果
	for _, table := range elementTables {
		assert.NotEmpty(t, table.Path)
		assert.NotEmpty(t, table.Elements)
	}
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
	filePath := filepath.Join(env.workspaceDir, "test", "mocks", "mock_graph_store.go")
	symbolNames := []string{"MockGraphStorage", "BatchSave"}

	// 步骤3: 查询符号
	symbols, err := testIndexer.(*indexer).querySymbols(env.ctx, env.workspaceDir, filePath, symbolNames)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(symbols))

	// 步骤4: 验证符号信息
	for _, s := range symbols {
		assert.True(t, slices.Contains(symbolNames, s.Name))
		assert.Equal(t, s.Language, string(lang.Go))
		assert.True(t, len(s.Occurrences) > 0)
		foundPaths := make([]string, 0)
		for _, d := range s.Occurrences {
			assert.Equal(t, len(d.Range), 4)
			foundPaths = append(foundPaths, d.Path)
		}
		assert.True(t, slices.Contains(foundPaths, filePath))
	}
}

func TestIndexer_IndexWorkspace_NotExists(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 步骤1: 使用不存在的工作区路径
	nonExistentWorkspace := filepath.Join(tempDir, "non_existent_workspace")

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
	nonExistentWorkspace := filepath.Join(tempDir, "non_existent_workspace")
	testFiles := []string{"test.go"}

	// 步骤2: 尝试索引不存在的项目中的文件，应该返回错误
	err := indexer.IndexFiles(env.ctx, nonExistentWorkspace, testFiles)
	assert.ErrorContains(t, err, "not exists")
}

// TestInitConfig 测试配置初始化函数
// 该测试验证 initConfig() 函数能够正确读取环境变量并设置配置值
func TestInitConfig(t *testing.T) {
	// 测试用例结构
	testCases := []struct {
		name           string
		envVars        map[string]string
		expectedConfig IndexerConfig
		description    string
	}{
		{
			name: "默认值测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "",
				"MAX_BATCH_SIZE":  "",
				"MAX_FILES":       "",
				"MAX_PROJECTS":    "",
				"CACHE_CAPACITY":  "",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: defaultConcurrency,
				MaxBatchSize:   defaultBatchSize,
				MaxFiles:       -1, // 后面使用的地方会处理。
				MaxProjects:    defaultMaxProjects,
				CacheCapacity:  defaultCacheCapacity,
			},
			description: "当所有环境变量都未设置时，应该使用默认值",
		},
		{
			name: "有效环境变量测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "4",
				"MAX_BATCH_SIZE":  "100",
				"MAX_FILES":       "5000",
				"MAX_PROJECTS":    "5",
				"CACHE_CAPACITY":  "10000",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: 4,
				MaxBatchSize:   100,
				MaxFiles:       5000,
				MaxProjects:    5,
				CacheCapacity:  10000,
			},
			description: "当所有环境变量都设置为有效值时，应该使用环境变量值",
		},
		{
			name: "无效环境变量测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "-1",
				"MAX_BATCH_SIZE":  "0",
				"MAX_FILES":       "invalid",
				"MAX_PROJECTS":    "-5",
				"CACHE_CAPACITY":  "not_a_number",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: defaultConcurrency,
				MaxBatchSize:   defaultBatchSize,
				MaxFiles:       -1, // 后面使用的地方会处理。
				MaxProjects:    defaultMaxProjects,
				CacheCapacity:  defaultCacheCapacity,
			},
			description: "当环境变量设置为无效值时，应该使用默认值",
		},
		{
			name: "部分环境变量测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "8",
				"MAX_BATCH_SIZE":  "",
				"MAX_FILES":       "2000",
				"MAX_PROJECTS":    "",
				"CACHE_CAPACITY":  "5000",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: 8,
				MaxBatchSize:   defaultBatchSize,
				MaxFiles:       2000,
				MaxProjects:    defaultMaxProjects,
				CacheCapacity:  5000,
			},
			description: "当部分环境变量设置时，设置的使用环境变量值，未设置的使用默认值",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 保存原始环境变量
			originalEnvVars := make(map[string]string)
			for key := range tc.envVars {
				originalEnvVars[key] = os.Getenv(key)
			}

			// 清理环境变量
			for key := range tc.envVars {
				os.Unsetenv(key)
			}

			// 设置测试环境变量
			for key, value := range tc.envVars {
				if value != "" {
					os.Setenv(key, value)
				}
			}

			// 创建测试配置
			config := IndexerConfig{
				MaxConcurrency: -1, // 设置为无效值，测试是否会使用环境变量或默认值
				MaxBatchSize:   -1,
				MaxFiles:       -1,
				MaxProjects:    -1,
				CacheCapacity:  -1,
			}
			// TODO 待确认是否缺少默认值
			// 调用初始化函数
			initConfig(&config)

			// 验证配置值
			assert.Equal(t, tc.expectedConfig.MaxConcurrency, config.MaxConcurrency, "MaxConcurrency 不匹配")
			assert.Equal(t, tc.expectedConfig.MaxBatchSize, config.MaxBatchSize, "MaxBatchSize 不匹配")
			assert.Equal(t, tc.expectedConfig.MaxFiles, config.MaxFiles, "MaxFiles 不匹配")
			assert.Equal(t, tc.expectedConfig.MaxProjects, config.MaxProjects, "MaxProjects 不匹配")
			assert.Equal(t, tc.expectedConfig.CacheCapacity, config.CacheCapacity, "CacheCapacity 不匹配")

			// 恢复原始环境变量
			for key, value := range originalEnvVars {
				if value != "" {
					os.Setenv(key, value)
				} else {
					os.Unsetenv(key)
				}
			}
		})
	}
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
func TestIndexer_QueryCallGraph_BySymbolName(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

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

	// env.workspaceDir = "/home/kcx/codeWorkspace/codebase-indexer/pkg/codegraph/parser/testdata/test"
	// env.workspaceDir = "/home/kcx/codeWorkspace/testProjects/python/django"
	// env.workspaceDir = "/home/kcx/codeWorkspace/testProjects/java/mall"
	// env.workspaceDir = "/home/kcx/codeWorkspace/testProjects/java/hadoop"
	// env.workspaceDir = "/home/kcx/codeWorkspace/testProjects/cpp/grpc"
	err = initWorkspaceModel(env)
	assert.NoError(t, err)
	// testVisitPattern.IncludeExts = []string{".py"}
	// testVisitPattern.IncludeExts = []string{".java"}
	// testVisitPattern.IncludeExts = []string{".cpp", ".cc", ".cxx", ".hpp", ".h"}
	// 创建测试索引器
	testIndexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	_, err = testIndexer.IndexWorkspace(env.ctx, env.workspaceDir)
	
	assert.NoError(t, err)
	pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()
	// 步骤2: 测试基于符号名的调用链查询
	testCases := []struct {
		name       string
		filePath   string
		symbolName string
		maxLayer   int
		desc       string
		project    string
	}{
		// {
		// 	name:       "IndexWorkspace方法调用链",
		// 	filePath:   "internal/service/indexer.go",
		// 	symbolName: "IndexWorkspace",
		// 	maxLayer:   20,
		// 	desc:       "查询IndexWorkspace方法的调用链",
		// 	project:    "codebase-indexer",
		// },
		{
			name:       "测试递归",
			filePath:   "internal/service/codebase.go",
			symbolName: "fillContent",
			maxLayer:   20,
			desc:       "查询fillContent方法的调用链",
			project:    "codebase-indexer",
		},
		// {
		// 	name:       "authenticate方法调用链",
		// 	filePath:   "hadoop-common-project/hadoop-auth/src/main/java/org/apache/hadoop/security/authentication/client/KerberosAuthenticator.java",
		// 	symbolName: "authenticate",
		// 	maxLayer:   1,
		// 	desc:       "查询authenticate方法的调用链",
		// 	project:    "hadoop",
		// },
		// {
		// 	name:       "listBrand方法调用链",
		// 	filePath:   "mall-demo/src/main/java/com/macro/mall/demo/service/impl/DemoServiceImpl.java",
		// 	symbolName: "listBrand",
		// 	maxLayer:   3,
		// 	desc:       "查询listBrand方法的调用链",
		// 	project:    "mall",
		// },
		// {
		// 	name:       "parse方法调用链",
		// 	filePath:   "django/http/multipartparser.py",
		// 	symbolName: "parse",
		// 	maxLayer:   2,
		// 	desc:       "查询parse方法的调用链",
		// 	project:    "django",
		// },
		// {
		// 	name:       "parse方法调用链",
		// 	filePath:   "multipartparser.py",
		// 	symbolName: "parse",
		// 	maxLayer:   3,
		// 	desc:       "查询parse方法的调用链",
		// 	project:    "test",
		// },
		// {
		// 	name:       "Get方法调用链",
		// 	filePath:   "staging/src/k8s.io/component-base/version/version.go",
		// 	symbolName: "Get",
		// 	maxLayer:   20,
		// 	desc:       "查询Get方法的调用链",
		// 	project:    "kubernetes",
		// },

		// {
		// 	name:"RunServer",
		// 	filePath: "examples/cpp/deadline/server.cc",
		// 	symbolName: "RunServer",
		// 	maxLayer: 3,
		// 	desc: "查询server.cc文件的调用链",
		// 	project: "grpc",
		// },
		// {
		// 	name:"SayHello",
		// 	filePath: "examples/cpp/error_details/greeter_client.cc",
		// 	symbolName: "SayHello",
		// 	maxLayer: 2,
		// 	desc: "查询greeter_client.cc文件的调用链",
		// 	project: "grpc",
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 构建完整文件路径
			start := time.Now()
			fullPath := filepath.Join(env.workspaceDir, tc.filePath)
			t.Log("fullPath", fullPath)
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

			// 将结果输出到文件
			outputFile := filepath.Join(tempDir, fmt.Sprintf("callgraph_%s_%s_symbol.txt", tc.symbolName,tc.project))
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
			fmt.Printf("查询调用链时间: %s\n", time.Since(start))
		})
	}
}

// TestIndexer_QueryCallGraph_ByLineRange 测试基于行范围的调用链查询
func TestIndexer_QueryCallGraph_ByLineRange(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	err := initWorkspaceModel(env)
	assert.NoError(t, err)

	// 创建测试索引器
	testIndexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	_, err = testIndexer.IndexWorkspace(env.ctx, env.workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 测试基于行范围的调用链查询
	testCases := []struct {
		name      string
		filePath  string
		startLine int
		endLine   int
		maxLayer  int
		desc      string
	}{
		// {
		// 	name:      "NewCodeIndexer函数范围",
		// 	filePath:  "internal/service/indexer.go",
		// 	startLine: 153, // NewCodeIndexer函数开始行
		// 	endLine:   175, // 函数结束行
		// 	maxLayer:  3,
		// 	desc:      "查询NewCodeIndexer函数范围内的调用链",
		// },
		{
			name:      "test_utils.go文件范围",
			filePath:  "test/codegraph/test_utils.go",
			startLine: 180,
			endLine:   203,
			maxLayer:  1,
			desc:      "查询test_utils.go文件范围内的调用链",
		},

		{
			name:      "IndexWorkspace方法范围",
			filePath:  "internal/service/indexer.go",
			startLine: 228, // IndexWorkspace方法开始行
			endLine:   250, // 方法部分范围
			maxLayer:  2,
			desc:      "查询IndexWorkspace方法范围内的调用链",
		},
		{
			name:      "setupTestEnvironment函数范围",
			filePath:  "internal/service/indexer_test.go",
			startLine: 52, // setupTestEnvironment函数开始行
			endLine:   70, // 函数部分范围
			maxLayer:  2,
			desc:      "查询setupTestEnvironment函数范围内的调用链",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 构建完整文件路径
			fullPath := filepath.Join(env.workspaceDir, tc.filePath)

			// 查询调用链
			opts := &types.QueryCallGraphOptions{
				Workspace: env.workspaceDir,
				FilePath:  fullPath,
				StartLine: tc.startLine,
				EndLine:   tc.endLine,
				MaxLayer:  tc.maxLayer,
			}

			nodes, err := testIndexer.QueryCallGraph(env.ctx, opts)
			assert.NoError(t, err)

			// 验证结果
			assert.NotNil(t, nodes, "调用链结果不应为空")
			t.Logf("行范围 %d-%d 的调用链包含 %d 个根节点", tc.startLine, tc.endLine, len(nodes))

			// 将结果输出到文件
			outputFile := filepath.Join(tempDir, fmt.Sprintf("callgraph_lines_%d_%d.txt", tc.startLine, tc.endLine))
			printCallGraphToFile(t, nodes, outputFile)
			t.Logf("调用链输出到文件: %s", outputFile)

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
		name      string
		opts      *types.QueryCallGraphOptions
		expectErr bool
		desc      string
	}{
		{
			name: "无符号名且无行范围",
			opts: &types.QueryCallGraphOptions{
				Workspace: env.workspaceDir,
				FilePath:  filepath.Join(env.workspaceDir, "internal/service/indexer.go"),
				MaxLayer:  3,
			},
			expectErr: true,
			desc:      "既没有符号名也没有行范围应该返回错误",
		},
		{
			name: "不存在的文件",
			opts: &types.QueryCallGraphOptions{
				Workspace:  env.workspaceDir,
				FilePath:   filepath.Join(env.workspaceDir, "non_existent_file.go"),
				SymbolName: "SomeFunction",
				MaxLayer:   3,
			},
			expectErr: true,
			desc:      "不存在的文件应该返回错误",
		},
		{
			name: "无效的行范围",
			opts: &types.QueryCallGraphOptions{
				Workspace: env.workspaceDir,
				FilePath:  filepath.Join(env.workspaceDir, "internal/service/indexer.go"),
				StartLine: 100,
				EndLine:   50, // 结束行小于开始行
				MaxLayer:  3,
			},
			expectErr: false, // 应该会被NormalizeLineRange处理
			desc:      "无效的行范围会被自动修正",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := testIndexer.QueryCallGraph(env.ctx, tc.opts)

			if tc.expectErr {
				assert.Error(t, err, tc.desc)
			} else {
				assert.NoError(t, err, tc.desc)
				assert.NotNil(t, nodes, "调用链结果不应为空")
			}
		})
	}
}
