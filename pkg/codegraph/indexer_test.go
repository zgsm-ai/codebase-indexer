package codegraph

import (
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"io"
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
	"github.com/stretchr/testify/assert"
)

var tempDir = "/tmp/"

var visitPattern = types.VisitPattern{ExcludeDirs: []string{".git", ".idea"}, IncludeExts: []string{".go"}}

// TODO 性能（内存、cpu）监控；各种路径、项目名（中文、符号）测试；索引数量统计；大仓库测试

// testEnvironment 包含测试所需的环境组件
type testEnvironment struct {
	ctx                context.Context
	cancel             context.CancelFunc
	storageDir         string
	logger             logger.Logger
	storage            store.GraphStorage
	workspaceReader    *workspace.WorkspaceReader
	sourceFileParser   *parser.SourceFileParser
	dependencyAnalyzer *analyzer.DependencyAnalyzer
	workspaceDir       string
}

// setupTestEnvironment 设置测试环境，创建所需的目录和组件
func setupTestEnvironment(t *testing.T) *testEnvironment {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	// 创建存储目录
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 创建日志器
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
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

	return &testEnvironment{
		ctx:                ctx,
		cancel:             cancel,
		storageDir:         storageDir,
		logger:             newLogger,
		storage:            storage,
		workspaceReader:    workspaceReader,
		sourceFileParser:   sourceFileParser,
		dependencyAnalyzer: dependencyAnalyzer,
		workspaceDir:       workspaceDir,
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
func createTestIndexer(env *testEnvironment, visitPattern types.VisitPattern) *Indexer {
	return NewCodeIndexer(
		env.sourceFileParser,
		env.dependencyAnalyzer,
		env.workspaceReader,
		env.storage,
		IndexerConfig{VisitPattern: visitPattern},
		env.logger,
	)
}

// countGoFiles 统计工作区中的 Go 文件数量
func countGoFiles(ctx context.Context, workspaceReader *workspace.WorkspaceReader, workspaceDir string, visitPattern types.VisitPattern) (int, error) {
	var goCount int
	err := workspaceReader.WalkFile(ctx, workspaceDir, func(walkCtx *types.WalkContext, reader io.ReadCloser) error {
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
func validateStorageState(t *testing.T, ctx context.Context, workspaceReader *workspace.WorkspaceReader, storage store.GraphStorage, workspaceDir string, projects []*workspace.Project, visitPattern types.VisitPattern) {
	// 统计 Go 文件数量
	goCount, err := countGoFiles(ctx, workspaceReader, workspaceDir, visitPattern)
	assert.NoError(t, err)

	// 统计索引数量
	indexSize, err := countIndexedFiles(ctx, storage, projects)
	assert.NoError(t, err)

	// 验证数量一致
	assert.Equal(t, goCount, indexSize)

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
	files, err := utils.ListFiles(filePath)
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
			t.Logf("key: %s", key)
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

	// 创建测试索引器
	indexer := createTestIndexer(env, visitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, visitPattern)

	// 测试 IndexWorkspace - 索引整个工作区
	err := indexer.IndexWorkspace(context.Background(), env.workspaceDir)
	assert.NoError(t, err)

	// 验证存储状态 - 确保索引数量与文件数量一致
	validateStorageState(t, env.ctx, env.workspaceReader, env.storage, env.workspaceDir, projects, visitPattern)

	// 清理测试环境
	teardownTestEnvironment(t, env, projects)
}

func TestIndexer_IndexProjectFilesWhenProjectHasIndex(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器，使用排除 mocks 目录的访问模式
	newVisitPattern := types.VisitPattern{ExcludeDirs: []string{".git", ".idea", "mocks"}, IncludeExts: []string{".go"}}
	indexer := createTestIndexer(env, newVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, newVisitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 先索引工作区（排除 mocks 目录）
	err = indexer.IndexWorkspace(env.ctx, env.workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 获取测试文件并创建路径键映射
	files := getTestFiles(t, env.workspaceDir)
	pathKeys := createPathKeyMap(t, files)

	// 步骤3: 验证 mocks 目录文件没有被索引
	validateFilesNotIndexed(t, env.ctx, env.storage, projects, pathKeys)

	// 步骤4: 测试索引特定文件
	filesByProject, err := indexer.groupFilesByProject(projects, files)
	assert.NoError(t, err)
	for project, projectFiles := range filesByProject {
		err = indexer.indexProjectFiles(context.Background(), project, projectFiles)
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
	indexer := createTestIndexer(env, visitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, visitPattern)

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
	validateStorageState(t, env.ctx, env.workspaceReader, env.storage, env.workspaceDir, projects, visitPattern)
}

func TestIndexer_RemoveIndexes(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, visitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, visitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
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
	indexer := createTestIndexer(env, visitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, visitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	err = indexer.IndexWorkspace(env.ctx, env.workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 获取测试文件
	files := getTestFiles(t, env.workspaceDir)
	assert.True(t, len(files) > 0)

	// 步骤3: 查询元素
	elementTables, err := indexer.QueryElements(env.ctx, env.workspaceDir, files)
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
	indexer := createTestIndexer(env, visitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, env.workspaceDir, visitPattern)

	// 清理索引存储
	err := cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	err = indexer.IndexWorkspace(env.ctx, env.workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 准备测试文件和符号名称
	filePath := filepath.Join(env.workspaceDir, "test", "mocks", "mock_graph_store.go")
	symbolNames := []string{"MockGraphStorage", "BatchSave"}

	// 步骤3: 查询符号
	symbols, err := indexer.QuerySymbols(env.ctx, env.workspaceDir, filePath, symbolNames)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(symbols))

	// 步骤4: 验证符号信息
	for _, s := range symbols {
		assert.True(t, slices.Contains(symbolNames, s.Name))
		assert.Equal(t, s.Language, string(lang.Go))
		assert.True(t, len(s.Definitions) > 0)
		for _, d := range s.Definitions {
			assert.Equal(t, len(d.Range), 4)
			assert.True(t, d.Path == filePath)
		}
	}
}

func TestIndexer_IndexWorkspace_NotExists(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, visitPattern)

	// 步骤1: 使用不存在的工作区路径
	nonExistentWorkspace := filepath.Join(tempDir, "non_existent_workspace")

	// 步骤2: 尝试索引不存在的工作区，应该返回错误
	err := indexer.IndexWorkspace(env.ctx, nonExistentWorkspace)
	assert.ErrorContains(t, err, "not exists")
}

func TestIndexer_IndexFiles_NoProject(t *testing.T) {
	// 设置测试环境
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env, nil)

	// 创建测试索引器
	indexer := createTestIndexer(env, visitPattern)

	// 步骤1: 使用不存在的项目路径
	nonExistentWorkspace := filepath.Join(tempDir, "non_existent_workspace")
	testFiles := []string{"test.go"}

	// 步骤2: 尝试索引不存在的项目中的文件，应该返回错误
	err := indexer.IndexFiles(env.ctx, nonExistentWorkspace, testFiles)
	assert.ErrorContains(t, err, "not exists")
}
