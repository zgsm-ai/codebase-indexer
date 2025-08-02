package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/test/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var tempDir = "/tmp/"

var visitPattern = types.VisitPattern{ExcludeDirs: []string{".git", ".idea"}, IncludeExts: []string{".go"}}

// TODO 性能（内存、cpu）监控；各种路径、项目名（中文、符号）测试；索引数量统计；大仓库测试

func cleanIndexStoreTestHelper(ctx context.Context, projects []*workspace.Project, storage store.GraphStorage) error {
	for _, p := range projects {
		if err := storage.DeleteAll(ctx, p.Uuid); err != nil {
			return err
		}
		if storage.Size(ctx, p.Uuid) > 0 {
			return fmt.Errorf("clean workspace index failed, size not equal 0")
		}
	}
	return nil
}

func TestIndexer_IndexWorkspace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()

	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)

	// 测试 IndexWorkspace
	err = indexer.IndexWorkspace(context.Background(), workspaceDir)
	assert.NoError(t, err)
	for _, p := range projects {
		t.Logf("=> storage size: %d", storage.Size(ctx, p.Uuid))
	}
	// 统计项目的go文件数量，确保索引数和文件数一致。
	var goCount int
	err = workspaceReader.Walk(ctx, workspaceDir, func(walkCtx *types.WalkContext, reader io.ReadCloser) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if strings.HasSuffix(walkCtx.Path, ".go") {
			goCount++
		}
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})
	assert.NoError(t, err)

	var indexSize int
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if strings.HasPrefix(key, store.PathKeyPrefix) {
				indexSize++
			}
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
	assert.Equal(t, goCount, indexSize)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_IndexProjectFilesWhenProjectHasIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	newVisitPattern := types.VisitPattern{ExcludeDirs: []string{".git", ".idea", "mocks"}, IncludeExts: []string{".go"}}
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: newVisitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, newVisitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)

	// 先索引工作区，然后再索引文件
	err = indexer.IndexWorkspace(ctx, workspaceDir)
	assert.NoError(t, err)
	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListFiles(filePath)
	pathKeys := make(map[string]any)
	for _, f := range files {
		key, err := store.ElementPathKey{Language: lang.Go, Path: f}.Get()
		assert.NoError(t, err)
		pathKeys[key] = nil
	}
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !strings.HasPrefix(key, store.PathKeyPrefix) {
				continue
			}
			_, ok := pathKeys[key]
			assert.False(t, ok)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}

	// 测试 index files

	assert.NoError(t, err)
	filesByProject, err := indexer.groupFilesByProject(projects, files)
	assert.NoError(t, err)
	for k, v := range filesByProject {
		err = indexer.indexProjectFiles(context.Background(), k, v)
		assert.NoError(t, err)
	}

	assert.True(t, len(pathKeys) > 0)
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !strings.HasPrefix(key, store.PathKeyPrefix) {
				continue
			}
			t.Logf("key: %s", key)
			delete(pathKeys, key)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}
	assert.True(t, len(pathKeys) == 0)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_IndexProjectFilesWhenProjectHasNoIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)

	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListFiles(filePath)
	for _, p := range projects {
		size := storage.Size(ctx, p.Uuid)
		assert.Equal(t, size, 0)
	}
	// 测试 index files
	err = indexer.IndexFiles(context.Background(), workspaceDir, files)
	assert.NoError(t, err)

	// 统计项目的go文件数量，确保索引数和文件数一致。
	var goCount int
	err = workspaceReader.Walk(ctx, workspaceDir, func(walkCtx *types.WalkContext, reader io.ReadCloser) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if strings.HasSuffix(walkCtx.Path, ".go") {
			goCount++
		}
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})
	assert.NoError(t, err)

	var indexSize int
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if strings.HasPrefix(key, store.PathKeyPrefix) {
				indexSize++
			}
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
	assert.Equal(t, goCount, indexSize)

	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_RemoveIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	assert.NoError(t, err)

	storage, err := store.NewLevelDBStorage(storageDir, newLogger)
	assert.NoError(t, err)
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)
	sourceFileParser := parser.NewSourceFileParser(newLogger)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, workspaceReader, storage)
	assert.NoError(t, err)
	defer storage.Close()
	indexer := NewCodeIndexer(
		sourceFileParser,
		dependencyAnalyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		newLogger,
	)

	projects := workspaceReader.FindProjects(ctx, workspaceDir, visitPattern)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)

	// 校验没有索引mocks目录
	// 校验索引
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListFiles(filePath)
	for _, p := range projects {
		size := storage.Size(ctx, p.Uuid)
		assert.Equal(t, size, 0)
	}
	// 测试 index files
	err = indexer.IndexFiles(context.Background(), workspaceDir, files)
	assert.NoError(t, err)
	pathKeys := make(map[string]any)
	pathKeysBak := make(map[string]any)
	for _, f := range files {
		key, err := store.ElementPathKey{Language: lang.Go, Path: f}.Get()
		assert.NoError(t, err)
		pathKeys[key] = nil
		pathKeysBak[key] = nil
	}
	assert.True(t, len(pathKeys) > 0)
	assert.True(t, len(pathKeysBak) > 0)

	totalIndexBefore := 0
	// 测试存在
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			totalIndexBefore++
			key := iter.Key()
			delete(pathKeys, key)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}
	assert.True(t, len(pathKeys) == 0)
	// 统计项目的go文件数量，确保索引数和文件数一致。
	err = indexer.RemoveIndexes(ctx, workspaceDir, files)
	assert.NoError(t, err)
	totalIndexAfter := 0
	// 测试不存在
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			totalIndexAfter++
			key := iter.Key()
			delete(pathKeysBak, key)
		}
		err = iter.Close()
		assert.NoError(t, err)
	}
	assert.Equal(t, len(files), len(pathKeysBak))
	assert.True(t, totalIndexBefore-len(files) <= totalIndexAfter)
	err = cleanIndexStoreTestHelper(ctx, projects, storage)
	assert.NoError(t, err)
}

func TestIndexer_QueryElements(t *testing.T) {

}

func TestIndexer_QuerySymbols(t *testing.T) {

}

func TestIndexer_IndexWorkspace_NoProjects(t *testing.T) {

}

func TestIndexer_IndexFiles_NoProject(t *testing.T) {

}

// TestIndexer_indexProject 测试 indexProject 方法的各种场景
func TestIndexer_indexProject(t *testing.T) {

}

// TestIndexer_processProjectFiles 测试 processProjectFiles 方法
func TestIndexer_processProjectFiles(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     func(t *testing.T) (*workspace.Project, string)
		expectedError  bool
		expectedFiles  int
		expectedSource int
	}{
		{
			name: "正常处理文件",
			setupFiles: func(t *testing.T) (*workspace.Project, string) {
				tempDir, err := os.MkdirTemp("", "test-normal-files")
				assert.NoError(t, err)

				projectDir := filepath.Join(tempDir, "project")
				err = os.MkdirAll(projectDir, 0755)
				assert.NoError(t, err)

				// 创建 Go 源文件
				err = os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(`package main`), 0644)
				assert.NoError(t, err)

				return &workspace.Project{
					Path: projectDir,
					Uuid: "test-uuid",
				}, tempDir
			},
			expectedError:  false,
			expectedFiles:  1,
			expectedSource: 1,
		},
		{
			name: "跳过非源文件",
			setupFiles: func(t *testing.T) (*workspace.Project, string) {
				tempDir, err := os.MkdirTemp("", "test-skip-files")
				assert.NoError(t, err)

				projectDir := filepath.Join(tempDir, "project")
				err = os.MkdirAll(projectDir, 0755)
				assert.NoError(t, err)

				// 创建非源文件
				err = os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# README"), 0644)
				assert.NoError(t, err)
				err = os.WriteFile(filepath.Join(projectDir, "config.json"), []byte("{}"), 0644)
				assert.NoError(t, err)

				return &workspace.Project{
					Path: projectDir,
					Uuid: "test-uuid",
				}, tempDir
			},
			expectedError:  false,
			expectedFiles:  2,
			expectedSource: 0,
		},
		{
			name: "处理文件读取错误",
			setupFiles: func(t *testing.T) (*workspace.Project, string) {
				tempDir, err := os.MkdirTemp("", "test-read-error")
				assert.NoError(t, err)

				projectDir := filepath.Join(tempDir, "project")
				err = os.MkdirAll(projectDir, 0755)
				assert.NoError(t, err)

				// 创建无权限文件
				filePath := filepath.Join(projectDir, "no-permission.go")
				err = os.WriteFile(filePath, []byte(`package main`), 0000)
				assert.NoError(t, err)

				return &workspace.Project{
					Path: projectDir,
					Uuid: "test-uuid",
				}, tempDir
			},
			expectedError:  false,
			expectedFiles:  1,
			expectedSource: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, tempDir := tt.setupFiles(t)
			defer os.RemoveAll(tempDir)

			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything).Return()
			mockLogger.On("Error", mock.Anything, mock.Anything).Return()

			parser := parser.NewSourceFileParser(mockLogger)
			analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, nil)
			workspaceReader := workspace.NewWorkSpaceReader(mockLogger)
			storage, err := store.NewLevelDBStorage(filepath.Join(tempDir, "storage"), mockLogger)
			assert.NoError(t, err)
			defer storage.Close()

			indexer := NewCodeIndexer(
				parser,
				analyzer,
				workspaceReader,
				storage,
				IndexerConfig{VisitPattern: visitPattern},
				mockLogger,
			)

			fileTables, metrics, err := indexer.processProjectFiles(context.Background(), project)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedFiles, metrics.TotalFiles)
			assert.Equal(t, tt.expectedSource, metrics.TotalSourceFiles)

			if tt.expectedSource > 0 {
				assert.NotEmpty(t, fileTables)
			}
		})
	}
}

// TestIndexer_groupFilesByProject 通过公开方法间接测试文件分组功能
func TestIndexer_groupFilesByProject(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-group-files")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建多个项目
	project1Dir := filepath.Join(tempDir, "project1")
	project2Dir := filepath.Join(tempDir, "project2")
	err = os.MkdirAll(project1Dir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(project2Dir, 0755)
	assert.NoError(t, err)

	// 创建 .git 目录
	err = os.MkdirAll(filepath.Join(project1Dir, ".git"), 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(filepath.Join(project2Dir, ".git"), 0755)
	assert.NoError(t, err)

	// 创建测试文件
	file1 := filepath.Join(project1Dir, "main.go")
	file2 := filepath.Join(project2Dir, "lib.go")
	err = os.WriteFile(file1, []byte(`package main`), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(file2, []byte(`package lib`), 0644)
	assert.NoError(t, err)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	parser := parser.NewSourceFileParser(mockLogger)
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, nil)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)
	storage, err := store.NewLevelDBStorage(filepath.Join(tempDir, "storage"), mockLogger)
	assert.NoError(t, err)
	defer storage.Close()

	indexer := NewCodeIndexer(
		parser,
		analyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		mockLogger,
	)

	// 测试文件分组 - 通过 IndexFiles 方法间接测试
	err = indexer.IndexFiles(context.Background(), tempDir, []string{file1, file2})
	assert.NoError(t, err)
}

// TestIndexer_StorageErrors 测试存储层错误处理
func TestIndexer_StorageErrors(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-storage-errors")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试项目结构
	projectDir := filepath.Join(tempDir, "test-project")
	err = os.MkdirAll(projectDir, 0755)
	assert.NoError(t, err)

	// 创建测试文件
	testFile := filepath.Join(projectDir, "main.go")
	err = os.WriteFile(testFile, []byte(`package main
import "fmt"
func main() { fmt.Println("Hello") }`), 0644)
	assert.NoError(t, err)

	// 创建 .git 目录
	err = os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	assert.NoError(t, err)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	// 使用 mock 存储来模拟错误
	mockStorage := &mocks.MockGraphStorage{}
	mockStorage.EXPECT().Size(mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(0, nil)

	parser := parser.NewSourceFileParser(mockLogger)
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, nil)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)

	indexer := NewCodeIndexer(
		parser,
		analyzer,
		workspaceReader,
		mockStorage,
		IndexerConfig{VisitPattern: visitPattern},
		mockLogger,
	)

	// 测试存储错误
	err = indexer.IndexFiles(context.Background(), tempDir, []string{testFile})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// TestIndexer_QuerySymbols_BoundaryConditions 测试符号查询边界条件
func TestIndexer_QuerySymbols_BoundaryConditions(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name          string
		symbolNames   []string
		expectedCount int
	}{
		{
			name:          "查询存在的符号",
			symbolNames:   []string{"main"},
			expectedCount: 1,
		},
		{
			name:          "查询不存在的符号",
			symbolNames:   []string{"nonexistent"},
			expectedCount: 0,
		},
		{
			name:          "查询空符号列表",
			symbolNames:   []string{},
			expectedCount: 0,
		},
		{
			name:          "查询多个符号",
			symbolNames:   []string{"main", "nonexistent"},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时测试目录
			tempDir, err := os.MkdirTemp("", "test-symbols-boundary")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// 创建测试项目结构
			projectDir := filepath.Join(tempDir, "test-project")
			err = os.MkdirAll(projectDir, 0755)
			assert.NoError(t, err)

			// 创建测试文件
			testFile := filepath.Join(projectDir, "main.go")
			err = os.WriteFile(testFile, []byte(`package main
import "fmt"
func main() { fmt.Println("Hello") }`), 0644)
			assert.NoError(t, err)

			// 创建 .git 目录
			err = os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
			assert.NoError(t, err)

			// 准备依赖
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything).Return()
			mockLogger.On("Error", mock.Anything, mock.Anything).Return()

			parser := parser.NewSourceFileParser(mockLogger)
			analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, nil)
			workspaceReader := workspace.NewWorkSpaceReader(mockLogger)
			storage, err := store.NewLevelDBStorage(filepath.Join(tempDir, "storage"), mockLogger)
			assert.NoError(t, err)
			defer storage.Close()

			indexer := NewCodeIndexer(
				parser,
				analyzer,
				workspaceReader,
				storage,
				IndexerConfig{VisitPattern: visitPattern},
				mockLogger,
			)

			// 先索引文件
			err = indexer.IndexFiles(context.Background(), tempDir, []string{testFile})
			assert.NoError(t, err)

			// 测试符号查询
			symbols, err := indexer.QuerySymbols(ctx, tempDir, testFile, tt.symbolNames)
			assert.NoError(t, err)
			assert.Len(t, symbols, tt.expectedCount)
		})
	}
}
