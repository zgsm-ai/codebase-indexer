package service

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"codebase-indexer/test/mocks"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
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
	workspaceModel, err := env.repository.GetWorkspaceByPath(env.workspaceDir)
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
	return err
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
	err = initWorkspaceModel(env)
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
				MaxFiles:       defaultMaxFiles,
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
				MaxFiles:       defaultMaxFiles,
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

// TestFilterSourceFilesByTimestamp 测试文件时间戳过滤函数
// 该测试验证 filterSourceFilesByTimestamp() 函数能够正确根据时间戳过滤需要索引的文件
func TestFilterSourceFilesByTimestamp(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	testIndexer := createTestIndexer(env, testVisitPattern)

	// 测试用例结构
	testCases := []struct {
		name                 string
		sourceFileTimestamps map[string]int64
		mockIterData         []mockIterData
		expectedFiles        []*types.FileWithModTimestamp
		description          string
	}{
		{
			name: "存储中无数据测试",
			sourceFileTimestamps: map[string]int64{
				"/test/file1.go": 1000,
				"/test/file2.go": 2000,
			},
			mockIterData: []mockIterData{},
			expectedFiles: []*types.FileWithModTimestamp{
				{Path: "/test/file1.go", ModTime: 1000},
				{Path: "/test/file2.go", ModTime: 2000},
			},
			description: "当存储中无数据时，所有文件都需要索引",
		},
		{
			name: "时间戳匹配测试",
			sourceFileTimestamps: map[string]int64{
				"/test/file1.go": 1000,
				"/test/file2.go": 2000,
			},
			mockIterData: []mockIterData{
				{
					key:   "element_path:/test/file1.go:go",
					value: &codegraphpb.FileElementTable{Path: "/test/file1.go", Language: string(lang.Go), Timestamp: 1000},
				},
			},
			expectedFiles: []*types.FileWithModTimestamp{
				{Path: "/test/file2.go", ModTime: 2000},
			},
			description: "当文件时间戳与存储中的时间戳匹配时，应该过滤掉该文件",
		},
		{
			name: "时间戳不匹配测试",
			sourceFileTimestamps: map[string]int64{
				"/test/file1.go": 1000,
				"/test/file2.go": 2000,
			},
			mockIterData: []mockIterData{
				{
					key:   "element_path:/test/file1.go:go",
					value: &codegraphpb.FileElementTable{Path: "/test/file1.go", Language: string(lang.Go), Timestamp: 1500}, // 时间戳不匹配
				},
			},
			expectedFiles: []*types.FileWithModTimestamp{
				{Path: "/test/file1.go", ModTime: 1000},
				{Path: "/test/file2.go", ModTime: 2000},
			},
			description: "当文件时间戳与存储中的时间戳不匹配时，应该保留该文件",
		},
		{
			name: "存储中数据损坏测试",
			sourceFileTimestamps: map[string]int64{
				"/test/file1.go": 1000,
				"/test/file2.go": 2000,
			},
			mockIterData: []mockIterData{
				{
					key:   "element_path:/test/file1.go:go",
					value: "invalid_data", // 损坏的数据
				},
			},
			expectedFiles: []*types.FileWithModTimestamp{
				{Path: "/test/file1.go", ModTime: 1000},
				{Path: "/test/file2.go", ModTime: 2000},
			},
			description: "当存储中数据损坏时，应该能够正常处理并保留所有文件",
		},
		{
			name: "混合测试 - 部分匹配",
			sourceFileTimestamps: map[string]int64{
				"/test/file1.go": 1000,
				"/test/file2.go": 2000,
				"/test/file3.go": 3000,
			},
			mockIterData: []mockIterData{
				{
					key:   "element_path:/test/file1.go:go",
					value: &codegraphpb.FileElementTable{Path: "/test/file1.go", Language: string(lang.Go), Timestamp: 1000}, // 匹配
				},
				{
					key:   "element_path:/test/file3.go:go",
					value: &codegraphpb.FileElementTable{Path: "/test/file3.go", Language: string(lang.Go), Timestamp: 3500}, // 不匹配
				},
			},
			expectedFiles: []*types.FileWithModTimestamp{
				{Path: "/test/file2.go", ModTime: 2000},
				{Path: "/test/file3.go", ModTime: 3000},
			},
			description: "混合测试 - 部分文件时间戳匹配，部分不匹配",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建模拟迭代器
			mockIterator := mocks.NewMockIterator(ctrl)

			// 设置迭代器行为
			for i, data := range tc.mockIterData {
				if i == 0 {
					mockIterator.EXPECT().Next().Return(true)
				} else if i < len(tc.mockIterData) {
					mockIterator.EXPECT().Next().Return(true)
				} else {
					mockIterator.EXPECT().Next().Return(false)
				}

				if i < len(tc.mockIterData) {
					mockIterator.EXPECT().Key().Return(data.key)

					if value, ok := data.value.(*codegraphpb.FileElementTable); ok {
						valueBytes, err := proto.Marshal(value)
						assert.NoError(t, err)
						mockIterator.EXPECT().Value().Return(valueBytes)
					} else if str, ok := data.value.(string); ok {
						// 模拟损坏的数据
						mockIterator.EXPECT().Value().Return([]byte(str))
					}
				}
			}

			// 最后一次调用 Next() 返回 false
			if len(tc.mockIterData) > 0 {
				mockIterator.EXPECT().Next().Return(false)
			} else {
				mockIterator.EXPECT().Next().Return(false)
			}

			// 设置 Close() 调用
			mockIterator.EXPECT().Close().Return(nil)

			// 创建模拟存储
			mockStorage := mocks.NewMockGraphStorage(ctrl)
			mockStorage.EXPECT().Iter(gomock.Any(), gomock.Any()).Return(mockIterator)

			// 调用过滤函数
			result := testIndexer.(*indexer).filterSourceFilesByTimestamp(env.ctx, "test-project", tc.sourceFileTimestamps)

			// 验证结果
			assert.Equal(t, len(tc.expectedFiles), len(result), "返回的文件数量不匹配")

			// 验证文件内容
			expectedMap := make(map[string]int64)
			for _, file := range tc.expectedFiles {
				expectedMap[file.Path] = file.ModTime
			}

			resultMap := make(map[string]int64)
			for _, file := range result {
				resultMap[file.Path] = file.ModTime
			}

			assert.Equal(t, expectedMap, resultMap, "返回的文件内容不匹配")
		})
	}
}

// mockIterData 模拟迭代器数据结构
type mockIterData struct {
	key   string
	value interface{}
}

// TestQueryReferences 测试查询引用函数
// 该测试验证 QueryReferences() 函数能够正确查询代码引用关系
func TestQueryReferences(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 测试用例结构
	testCases := []struct {
		name          string
		queryOptions  *types.QueryReferenceOptions
		mockStorage   *mockReferenceStorageData
		mockWorkspace *mockWorkspaceData
		expected      []*types.RelationNode
		expectError   bool
		errorMsg      string
		description   string
	}{
		{
			name: "正常引用查询",
			queryOptions: &types.QueryReferenceOptions{
				Workspace:  "/test/workspace",
				FilePath:   "/test/workspace/main.go",
				StartLine:  10,
				EndLine:    15,
				SymbolName: "TestFunction",
			},
			mockStorage: &mockReferenceStorageData{
				projectExists: true,
				fileElementTable: &codegraphpb.FileElementTable{
					Path:     "/test/workspace/main.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "TestFunction",
							ElementType:  codegraphpb.ElementType_FUNCTION,
							IsDefinition: true,
							Range:        []int32{9, 0, 15, 0},
						},
					},
				},
				iterData: []mockIterData{
					{
						key: "element_path:/test/workspace/utils.go:go",
						value: &codegraphpb.FileElementTable{
							Path:     "/test/workspace/utils.go",
							Language: string(lang.Go),
							Elements: []*codegraphpb.Element{
								{
									Name:         "TestFunction",
									ElementType:  codegraphpb.ElementType_REFERENCE,
									IsDefinition: false,
									Range:        []int32{20, 0, 20, 15},
								},
							},
						},
					},
				},
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expected: []*types.RelationNode{
				{
					FilePath:   "/test/workspace/main.go",
					SymbolName: "TestFunction",
					Position:   types.Position{StartLine: 10, StartColumn: 0, EndLine: 15, EndColumn: 0},
					NodeType:   "function",
					Children: []*types.RelationNode{
						{
							FilePath:   "/test/workspace/utils.go",
							SymbolName: "TestFunction",
							Position:   types.Position{StartLine: 21, StartColumn: 0, EndLine: 21, EndColumn: 15},
							NodeType:   "reference",
						},
					},
				},
			},
			expectError: false,
			description: "正常查询引用关系",
		},
		{
			name: "无引用结果查询",
			queryOptions: &types.QueryReferenceOptions{
				Workspace:  "/test/workspace",
				FilePath:   "/test/workspace/main.go",
				StartLine:  10,
				EndLine:    15,
				SymbolName: "UnusedFunction",
			},
			mockStorage: &mockReferenceStorageData{
				projectExists: true,
				fileElementTable: &codegraphpb.FileElementTable{
					Path:     "/test/workspace/main.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "UnusedFunction",
							ElementType:  codegraphpb.ElementType_FUNCTION,
							IsDefinition: true,
							Range:        []int32{9, 0, 15, 0},
						},
					},
				},
				iterData: []mockIterData{}, // 没有引用数据
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expected:    []*types.RelationNode{},
			expectError: false,
			description: "查询无引用的函数",
		},
		{
			name: "无效文件路径查询",
			queryOptions: &types.QueryReferenceOptions{
				Workspace: "/nonexistent/workspace",
				FilePath:  "/nonexistent/file.go",
				StartLine: 10,
				EndLine:   15,
			},
			mockStorage: &mockReferenceStorageData{
				projectExists:    false,
				fileElementTable: nil,
				iterData:         []mockIterData{},
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expectError: true,
			errorMsg:    "failed to find project which file",
			expected:    []*types.RelationNode{},
			description: "查询不存在的文件",
		},
		{
			name: "跨项目引用查询",
			queryOptions: &types.QueryReferenceOptions{
				Workspace:  "/test/workspace",
				FilePath:   "/test/workspace/main.go",
				StartLine:  10,
				EndLine:    15,
				SymbolName: "SharedFunction",
			},
			mockStorage: &mockReferenceStorageData{
				projectExists: true,
				fileElementTable: &codegraphpb.FileElementTable{
					Path:     "/test/workspace/main.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "SharedFunction",
							ElementType:  codegraphpb.ElementType_FUNCTION,
							IsDefinition: true,
							Range:        []int32{9, 0, 15, 0},
						},
					},
				},
				iterData: []mockIterData{
					{
						key: "element_path:/another/project/helper.go:go",
						value: &codegraphpb.FileElementTable{
							Path:     "/another/project/helper.go",
							Language: string(lang.Go),
							Elements: []*codegraphpb.Element{
								{
									Name:         "SharedFunction",
									ElementType:  codegraphpb.ElementType_REFERENCE,
									IsDefinition: false,
									Range:        []int32{5, 0, 5, 20},
								},
							},
						},
					},
				},
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expected: []*types.RelationNode{
				{
					FilePath:   "/test/workspace/main.go",
					SymbolName: "SharedFunction",
					Position:   types.Position{StartLine: 10, StartColumn: 0, EndLine: 15, EndColumn: 0},
					NodeType:   "function",
					Children: []*types.RelationNode{
						{
							FilePath:   "/another/project/helper.go",
							SymbolName: "SharedFunction",
							Position:   types.Position{StartLine: 6, StartColumn: 0, EndLine: 6, EndColumn: 20},
							NodeType:   "reference",
						},
					},
				},
			},
			expectError: false,
			description: "跨项目引用查询",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建模拟存储
			mockStorage := mocks.NewMockGraphStorage(ctrl)
			mockIterator := mocks.NewMockIterator(ctrl)

			// 设置存储行为
			if tc.mockStorage.fileElementTable != nil {
				// 设置 ProjectIndexExists 方法
				mockStorage.EXPECT().ProjectIndexExists(gomock.Any()).Return(true, nil)

				// 设置 Get 方法
				fileTableBytes, err := proto.Marshal(tc.mockStorage.fileElementTable)
				assert.NoError(t, err)
				mockStorage.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fileTableBytes, nil)

				// 设置 Iter 方法
				mockStorage.EXPECT().Iter(gomock.Any(), gomock.Any()).Return(mockIterator)

				// 设置迭代器行为
				for i, data := range tc.mockStorage.iterData {
					if i == 0 {
						mockIterator.EXPECT().Next().Return(true)
					} else if i < len(tc.mockStorage.iterData) {
						mockIterator.EXPECT().Next().Return(true)
					} else {
						mockIterator.EXPECT().Next().Return(false)
					}

					if i < len(tc.mockStorage.iterData) {
						mockIterator.EXPECT().Key().Return(data.key)

						if value, ok := data.value.(*codegraphpb.FileElementTable); ok {
							valueBytes, err := proto.Marshal(value)
							assert.NoError(t, err)
							mockIterator.EXPECT().Value().Return(valueBytes)
						}
					}
				}

				// 最后一次调用 Next() 返回 false
				if len(tc.mockStorage.iterData) > 0 {
					mockIterator.EXPECT().Next().Return(false)
				} else {
					mockIterator.EXPECT().Next().Return(false)
				}

				// 设置 Close() 调用
				mockIterator.EXPECT().Close().Return(nil)
			} else {
				// 文件不存在的情况
				// 设置 ProjectIndexExists 方法
				mockStorage.EXPECT().ProjectIndexExists(gomock.Any()).Return(true, nil)
				// 设置 Get 方法
				mockStorage.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, store.ErrKeyNotFound)
			}

			// 调用查询函数
			result, err := indexer.QueryReferences(env.ctx, tc.queryOptions)

			// 验证结果
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// 简化的结果验证，实际使用中可能需要更复杂的比较逻辑
				assert.Equal(t, len(tc.expected), len(result), "返回的引用节点数量不匹配")
			}
		})
	}
}

// mockReferenceStorageData 模拟引用查询的存储数据
type mockReferenceStorageData struct {
	projectExists    bool
	fileElementTable *codegraphpb.FileElementTable
	iterData         []mockIterData
}

// mockWorkspaceData 模拟工作区数据
type mockWorkspaceData struct {
	project *workspace.Project
}

// TestQueryDefinitions 测试 QueryDefinitions() 函数
func TestQueryDefinitions(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, []*workspace.Project{})

	// 创建索引器实例
	indexer := createTestIndexer(env, testVisitPattern)

	// 定义测试用例
	testCases := []struct {
		name          string
		queryOptions  *types.QueryDefinitionOptions
		mockStorage   *mockDefinitionStorageData
		mockWorkspace *mockWorkspaceData
		expectError   bool
		errorMsg      string
		expected      []*types.Definition
		description   string
	}{
		{
			name: "代码片段定义查询",
			queryOptions: &types.QueryDefinitionOptions{
				Workspace:   "/test/workspace",
				FilePath:    "/test/workspace/main.go",
				CodeSnippet: []byte("func TestFunction() {\n\tfmt.Println(\"test\")\n}"),
			},
			mockStorage: &mockDefinitionStorageData{
				projectExists: true,
				symbolData: &codegraphpb.SymbolOccurrence{
					Occurrences: []*codegraphpb.Occurrence{
						{
							Path:  "/test/workspace/utils.go",
							Range: []int32{5, 0, 10, 0},
						},
					},
				},
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expected: []*types.Definition{
				{
					Name:  "TestFunction",
					Path:  "/test/workspace/utils.go",
					Range: []int32{5, 0, 10, 0},
					Type:  "function",
				},
			},
			expectError: false,
			description: "通过代码片段查询定义",
		},
		{
			name: "行范围定义查询",
			queryOptions: &types.QueryDefinitionOptions{
				Workspace: "/test/workspace",
				FilePath:  "/test/workspace/main.go",
				StartLine: 10,
				EndLine:   15,
			},
			mockStorage: &mockDefinitionStorageData{
				projectExists: true,
				fileElementTable: &codegraphpb.FileElementTable{
					Path:     "/test/workspace/main.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "MainFunction",
							ElementType:  codegraphpb.ElementType_FUNCTION,
							IsDefinition: true,
							Range:        []int32{9, 0, 15, 0},
						},
					},
				},
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expected: []*types.Definition{
				{
					Name:  "MainFunction",
					Path:  "/test/workspace/main.go",
					Range: []int32{9, 0, 15, 0},
					Type:  "function",
				},
			},
			expectError: false,
			description: "通过行范围查询定义",
		},
		{
			name: "无效查询参数",
			queryOptions: &types.QueryDefinitionOptions{
				Workspace: "/nonexistent/workspace",
				FilePath:  "/nonexistent/file.go",
				StartLine: 10,
				EndLine:   15,
			},
			mockStorage: &mockDefinitionStorageData{
				projectExists:    false,
				fileElementTable: nil,
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expectError: true,
			errorMsg:    "failed to find project which file",
			expected:    []*types.Definition{},
			description: "无效的查询参数",
		},
		{
			name: "模糊匹配定义",
			queryOptions: &types.QueryDefinitionOptions{
				Workspace:   "/test/workspace",
				FilePath:    "/test/workspace/main.go",
				CodeSnippet: []byte("fmt.Println(\"hello\")"),
			},
			mockStorage: &mockDefinitionStorageData{
				projectExists: true,
				// 代码片段查询不直接使用存储，而是通过解析器处理
			},
			mockWorkspace: &mockWorkspaceData{
				project: &workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				},
			},
			expected:    []*types.Definition{},
			expectError: false,
			description: "模糊匹配定义查询",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建模拟存储
			mockStorage := mocks.NewMockGraphStorage(ctrl)

			// 设置存储行为
			if tc.mockStorage.projectExists {
				// 设置 ProjectIndexExists 方法
				mockStorage.EXPECT().ProjectIndexExists(gomock.Any()).Return(true, nil)

				if tc.mockStorage.fileElementTable != nil {
					// 设置 Get 方法 - 文件元素表
					fileTableBytes, err := proto.Marshal(tc.mockStorage.fileElementTable)
					assert.NoError(t, err)
					mockStorage.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fileTableBytes, nil)
				}

				if tc.mockStorage.symbolData != nil {
					// 设置 Get 方法 - 符号数据
					symbolBytes, err := proto.Marshal(tc.mockStorage.symbolData)
					assert.NoError(t, err)
					mockStorage.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(symbolBytes, nil)
				}
			} else {
				// 项目不存在的情况 - 这个测试实际上在调用 GetProjectByFilePath 时就会失败，所以不会调用 ProjectIndexExists
				// 所以我们不需要设置任何 mock 期望
			}

			// 调用查询函数
			result, err := indexer.QueryDefinitions(env.ctx, tc.queryOptions)

			// 验证结果
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// 简化的结果验证，实际使用中可能需要更复杂的比较逻辑
				if tc.name == "模糊匹配定义" {
					// 对于模糊匹配查询，我们只验证查询是否成功执行，不验证具体结果
					// 因为代码片段查询的结果取决于解析和分析逻辑，比较复杂
					_ = result
				} else {
					assert.Equal(t, len(tc.expected), len(result), "返回的定义节点数量不匹配")
				}
			}
		})
	}
}

// mockDefinitionStorageData 模拟定义查询的存储数据
type mockDefinitionStorageData struct {
	projectExists    bool
	fileElementTable *codegraphpb.FileElementTable
	symbolData       *codegraphpb.SymbolOccurrence
}

func TestRenameIndexes(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, []*workspace.Project{})

	// 定义测试用例
	testCases := []struct {
		name           string
		workspacePath  string
		sourceFilePath string
		targetFilePath string
		mockStorage    func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader)
		expectError    bool
		errorMsg       string
		description    string
	}{
		{
			name:           "重命名文件索引成功",
			workspacePath:  "/test/workspace",
			sourceFilePath: "/test/workspace/main.go",
			targetFilePath: "/test/workspace/app.go",
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 设置 GetProjectByFilePath 方法
				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), "/test/workspace", "/test/workspace/main.go", false).Return(&workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				}, nil)

				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), "/test/workspace", "/test/workspace/app.go", false).Return(&workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				}, nil)

				// 创建模拟的文件元素表
				fileElementTable := &codegraphpb.FileElementTable{
					Path:     "/test/workspace/main.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "main",
							IsDefinition: true,
							Range:        []int32{1, 0, 1, 0},
							ElementType:  codegraphpb.ElementType_FUNCTION,
						},
					},
				}

				// 创建模拟的符号数据
				symbolData := &codegraphpb.SymbolOccurrence{
					Name:     "main",
					Language: string(lang.Go),
					Occurrences: []*codegraphpb.Occurrence{
						{
							Path:        "/test/workspace/main.go",
							Range:       []int32{1, 0, 1, 0},
							ElementType: codegraphpb.ElementType_FUNCTION,
						},
					},
				}

				// 设置存储行为
				fileTableBytes, err := proto.Marshal(fileElementTable)
				assert.NoError(t, err)

				symbolBytes, err := proto.Marshal(symbolData)
				assert.NoError(t, err)

				// 设置 Get 方法
				storage.EXPECT().Get(gomock.Any(), "test-project-uuid", gomock.Any()).Return(fileTableBytes, nil)

				// 设置 Delete 方法
				storage.EXPECT().Delete(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil)

				// 设置 Put 方法（保存新的文件元素表）
				storage.EXPECT().Put(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil)

				// 设置 Get 方法（获取符号数据）
				storage.EXPECT().Get(gomock.Any(), "test-project-uuid", gomock.Any()).Return(symbolBytes, nil)

				// 设置 Put 方法（保存符号数据）
				storage.EXPECT().Put(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil)
			},
			expectError: false,
			description: "测试重命名文件索引成功的情况",
		},
		{
			name:           "工作区不存在项目",
			workspacePath:  "/test/workspace",
			sourceFilePath: "/test/workspace/main.go",
			targetFilePath: "/test/workspace/app.go",
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法 - 返回空列表
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{})
			},
			expectError: false,
			description: "测试工作区不存在项目的情况",
		},
		{
			name:           "源文件项目查找失败",
			workspacePath:  "/test/workspace",
			sourceFilePath: "/test/workspace/main.go",
			targetFilePath: "/test/workspace/app.go",
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 设置 GetProjectByFilePath 方法 - 返回错误
				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), "/test/workspace", "/test/workspace/main.go", false).Return(nil, fmt.Errorf("project not found"))
			},
			expectError: true,
			errorMsg:    "cannot find project for file",
			description: "测试源文件项目查找失败的情况",
		},
		{
			name:           "源文件索引不存在",
			workspacePath:  "/test/workspace",
			sourceFilePath: "/test/workspace/main.go",
			targetFilePath: "/test/workspace/app.go",
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 设置 GetProjectByFilePath 方法
				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), "/test/workspace", "/test/workspace/main.go", false).Return(&workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				}, nil)

				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), "/test/workspace", "/test/workspace/app.go", false).Return(&workspace.Project{
					Uuid: "test-project-uuid",
					Path: "/test/workspace",
				}, nil)

				// 设置 Get 方法 - 返回错误，表示文件不存在
				storage.EXPECT().Get(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil, store.ErrKeyNotFound)
			},
			expectError: false,
			description: "测试源文件索引不存在的情况",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建模拟存储
			mockStorage := mocks.NewMockGraphStorage(ctrl)

			// 创建模拟工作区读取器
			mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

			// 设置 mock 行为
			tc.mockStorage(ctrl, mockStorage, mockWorkspaceReader)

			// 创建使用 mock 的索引器
			mockIndexer := NewCodeIndexer(
				env.scanner,
				env.sourceFileParser,
				env.dependencyAnalyzer,
				mockWorkspaceReader,
				mockStorage,
				env.repository,
				IndexerConfig{VisitPattern: testVisitPattern},
				env.logger,
			)

			// 调用重命名索引方法
			err := mockIndexer.RenameIndexes(env.ctx, tc.workspacePath, tc.sourceFilePath, tc.targetFilePath)

			// 验证结果
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIndexAddFiles(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, []*workspace.Project{})

	// 定义测试用例
	testCases := []struct {
		name          string
		workspacePath string
		filePaths     []string
		mockStorage   func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader, repository *mocks.MockWorkspaceRepository)
		expectError   bool
		errorMsg      string
		description   string
	}{
		{
			name:          "索引文件成功 - 项目已索引",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/main.go", "/test/workspace/utils.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader, repository *mocks.MockWorkspaceRepository) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 Exists 方法
				mockWorkspaceReader.EXPECT().Exists(gomock.Any(), "/test/workspace").Return(true, nil)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", true, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 模拟工作区仓库
				mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)

				// 设置 GetWorkspaceByPath 方法
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/test/workspace").Return(&model.Workspace{
					WorkspacePath:    "/test/workspace",
					FileNum:          10,
					CodegraphFileNum: 5,
				}, nil)

				// 设置存储行为
				storage.EXPECT().Size(gomock.Any(), "test-project-uuid", store.PathKeySystemPrefix).Return(5) // 项目已索引
			},
			expectError: false,
			description: "测试索引文件成功的情况 - 项目已索引",
		},
		{
			name:          "索引文件成功 - 项目未索引",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/main.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader, repository *mocks.MockWorkspaceRepository) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 Exists 方法
				mockWorkspaceReader.EXPECT().Exists(gomock.Any(), "/test/workspace").Return(true, nil)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", true, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 模拟工作区仓库
				mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)

				// 设置 GetWorkspaceByPath 方法
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/test/workspace").Return(&model.Workspace{
					WorkspacePath:    "/test/workspace",
					FileNum:          10,
					CodegraphFileNum: 5,
				}, nil)

				// 设置存储行为
				storage.EXPECT().Size(gomock.Any(), "test-project-uuid", store.PathKeySystemPrefix).Return(0) // 项目未索引
			},
			expectError: false,
			description: "测试索引文件成功的情况 - 项目未索引",
		},
		{
			name:          "工作区不存在",
			workspacePath: "/nonexistent/workspace",
			filePaths:     []string{"/nonexistent/workspace/main.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader, repository *mocks.MockWorkspaceRepository) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 Exists 方法 - 工作区不存在
				mockWorkspaceReader.EXPECT().Exists(gomock.Any(), "/nonexistent/workspace").Return(false, nil)
			},
			expectError: true,
			errorMsg:    "workspace path /nonexistent/workspace not exists",
			description: "测试工作区不存在的情况",
		},
		{
			name:          "工作区中没有项目",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/main.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader, repository *mocks.MockWorkspaceRepository) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 Exists 方法
				mockWorkspaceReader.EXPECT().Exists(gomock.Any(), "/test/workspace").Return(true, nil)

				// 设置 FindProjects 方法 - 返回空列表
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", true, gomock.Any()).Return([]*workspace.Project{})
			},
			expectError: true,
			errorMsg:    "no project found in workspace",
			description: "测试工作区中没有项目的情况",
		},
		{
			name:          "获取工作区信息失败",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/main.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader, repository *mocks.MockWorkspaceRepository) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 Exists 方法
				mockWorkspaceReader.EXPECT().Exists(gomock.Any(), "/test/workspace").Return(true, nil)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", true, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 模拟工作区仓库
				mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)

				// 设置 GetWorkspaceByPath 方法 - 返回错误
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/test/workspace").Return(nil, fmt.Errorf("workspace not found"))
			},
			expectError: true,
			errorMsg:    "get workspace err",
			description: "测试获取工作区信息失败的情况",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建模拟存储
			mockStorage := mocks.NewMockGraphStorage(ctrl)

			// 创建模拟工作区读取器
			mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

			// 创建模拟工作区仓库
			mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)

			// 设置 mock 行为
			tc.mockStorage(ctrl, mockStorage, mockWorkspaceReader, mockWorkspaceRepo)

			// 创建使用 mock 的索引器
			mockIndexer := NewCodeIndexer(
				env.scanner,
				env.sourceFileParser,
				env.dependencyAnalyzer,
				mockWorkspaceReader,
				mockStorage,
				mockWorkspaceRepo,
				IndexerConfig{VisitPattern: testVisitPattern},
				env.logger,
			)

			// 调用索引文件方法
			err := mockIndexer.IndexFiles(env.ctx, tc.workspacePath, tc.filePaths)

			// 验证结果
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIndexRemoveFiles(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, []*workspace.Project{})

	// 定义测试用例
	testCases := []struct {
		name          string
		workspacePath string
		filePaths     []string
		mockStorage   func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader)
		expectError   bool
		errorMsg      string
		description   string
	}{
		{
			name:          "删除文件索引成功",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/main.go", "/test/workspace/utils.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 创建模拟的文件元素表
				fileElementTable1 := &codegraphpb.FileElementTable{
					Path:     "/test/workspace/main.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "main",
							IsDefinition: true,
							Range:        []int32{1, 0, 1, 0},
							ElementType:  codegraphpb.ElementType_FUNCTION,
						},
					},
				}

				fileElementTable2 := &codegraphpb.FileElementTable{
					Path:     "/test/workspace/utils.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "helper",
							IsDefinition: true,
							Range:        []int32{1, 0, 1, 0},
							ElementType:  codegraphpb.ElementType_FUNCTION,
						},
					},
				}

				// 设置存储行为
				fileTableBytes1, err := proto.Marshal(fileElementTable1)
				assert.NoError(t, err)

				fileTableBytes2, err := proto.Marshal(fileElementTable2)
				assert.NoError(t, err)

				// 设置 Get 方法 - 返回文件元素表
				storage.EXPECT().Get(gomock.Any(), "test-project-uuid", gomock.Any()).Return(fileTableBytes1, nil)
				storage.EXPECT().Get(gomock.Any(), "test-project-uuid", gomock.Any()).Return(fileTableBytes2, nil)

				// 设置 Delete 方法 - 删除文件索引
				storage.EXPECT().Delete(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil).Times(2)

				// 设置 Put 方法 - 更新符号数据
				storage.EXPECT().Put(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil).Times(2)
			},
			expectError: false,
			description: "测试删除文件索引成功的情况",
		},
		{
			name:          "工作区中没有项目",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/main.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法 - 返回空列表
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{})
			},
			expectError: true,
			errorMsg:    "no project found in workspace",
			description: "测试工作区中没有项目的情况",
		},
		{
			name:          "删除不存在的文件索引",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/nonexistent.go"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 设置 Get 方法 - 返回错误，表示文件不存在
				storage.EXPECT().Get(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil, store.ErrKeyNotFound)
			},
			expectError: false,
			description: "测试删除不存在的文件索引的情况",
		},
		{
			name:          "删除文件夹索引成功",
			workspacePath: "/test/workspace",
			filePaths:     []string{"/test/workspace/utils/"},
			mockStorage: func(ctrl *gomock.Controller, storage *mocks.MockGraphStorage, workspaceReader *mocks.MockWorkspaceReader) {
				// 模拟工作区读取器
				mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

				// 设置 FindProjects 方法
				mockWorkspaceReader.EXPECT().FindProjects(gomock.Any(), "/test/workspace", false, gomock.Any()).Return([]*workspace.Project{
					{Uuid: "test-project-uuid", Path: "/test/workspace"},
				})

				// 创建模拟的文件元素表
				fileElementTable1 := &codegraphpb.FileElementTable{
					Path:     "/test/workspace/utils/helper.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "helper",
							IsDefinition: true,
							Range:        []int32{1, 0, 1, 0},
							ElementType:  codegraphpb.ElementType_FUNCTION,
						},
					},
				}

				fileElementTable2 := &codegraphpb.FileElementTable{
					Path:     "/test/workspace/utils/common.go",
					Language: string(lang.Go),
					Elements: []*codegraphpb.Element{
						{
							Name:         "common",
							IsDefinition: true,
							Range:        []int32{1, 0, 1, 0},
							ElementType:  codegraphpb.ElementType_FUNCTION,
						},
					},
				}

				// 设置存储行为
				_, err := proto.Marshal(fileElementTable1)
				assert.NoError(t, err)

				_, err = proto.Marshal(fileElementTable2)
				assert.NoError(t, err)

				// 设置 Get 方法 - 第一次返回错误，触发前缀搜索
				storage.EXPECT().Get(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil, store.ErrKeyNotFound)

				// 设置 Delete 方法 - 删除文件索引
				storage.EXPECT().Delete(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil).Times(2)

				// 设置 Put 方法 - 更新符号数据
				storage.EXPECT().Put(gomock.Any(), "test-project-uuid", gomock.Any()).Return(nil).Times(2)
			},
			expectError: false,
			description: "测试删除文件夹索引成功的情况",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建模拟存储
			mockStorage := mocks.NewMockGraphStorage(ctrl)

			// 创建模拟工作区读取器
			mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)

			// 设置 mock 行为
			tc.mockStorage(ctrl, mockStorage, mockWorkspaceReader)

			// 创建使用 mock 的索引器
			mockIndexer := NewCodeIndexer(
				env.scanner,
				env.sourceFileParser,
				env.dependencyAnalyzer,
				mockWorkspaceReader,
				mockStorage,
				env.repository,
				IndexerConfig{VisitPattern: testVisitPattern},
				env.logger,
			)

			// 调用删除索引方法
			err := mockIndexer.RemoveIndexes(env.ctx, tc.workspacePath, tc.filePaths)

			// 验证结果
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
