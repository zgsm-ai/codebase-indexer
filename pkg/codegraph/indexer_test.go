package codegraph

import (
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
func TestIndexer_IndexWorkspace(t *testing.T) {

	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// 测试本项目
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)
	newLogger, err := logger.NewLogger(utils.LogsDir, "info")
	if err != nil {
		fmt.Printf("failed to initialize logging system: %v\n", err)
		return
	}

	storage, err := store.NewBBoltStorage(storageDir, newLogger)
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

	// 测试 IndexWorkspace
	err = indexer.IndexWorkspace(context.Background(), workspaceDir)
	assert.NoError(t, err)

}

func TestIndexer_IndexFiles(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-files")
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

func main() {
	fmt.Println("Hello, World!")
}`), 0644)
	assert.NoError(t, err)

	// 创建 .git 目录使其成为项目
	err = os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	assert.NoError(t, err)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	parser := parser.NewSourceFileParser(mockLogger)
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, nil)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)
	storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
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

	// 测试 IndexFiles
	err = indexer.IndexFiles(context.Background(), tempDir, []string{testFile})
	assert.NoError(t, err)

	// 验证日志调用
	mockLogger.AssertCalled(t, "Info", mock.Anything, mock.Anything)
}

func TestIndexer_RemoveIndexes(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-remove")
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

func main() {
	fmt.Println("Hello, World!")
}`), 0644)
	assert.NoError(t, err)

	// 创建 .git 目录使其成为项目
	err = os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	assert.NoError(t, err)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	parser := parser.NewSourceFileParser(mockLogger)
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, nil)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)
	storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
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

	// 测试 RemoveIndexes
	err = indexer.RemoveIndexes(context.Background(), tempDir, []string{testFile})
	assert.NoError(t, err)

	// 验证日志调用
	mockLogger.AssertCalled(t, "Info", mock.Anything, mock.Anything)
}

func TestIndexer_QueryElements(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-query")
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

func main() {
	fmt.Println("Hello, World!")
}`), 0644)
	assert.NoError(t, err)

	// 创建 .git 目录使其成为项目
	err = os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	assert.NoError(t, err)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()

	parser := parser.NewSourceFileParser(mockLogger)
	storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
	assert.NoError(t, err)
	defer storage.Close()
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, storage)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)

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

	// 测试 QueryElements
	elements, err := indexer.QueryElements(tempDir, []string{testFile})
	assert.NoError(t, err)
	assert.NotEmpty(t, elements)

	// 验证日志调用
	mockLogger.AssertCalled(t, "Info", mock.Anything, mock.Anything)
}

func TestIndexer_QuerySymbols(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-symbols")
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

func main() {
	fmt.Println("Hello, World!")
}`), 0644)
	assert.NoError(t, err)

	// 创建 .git 目录使其成为项目
	err = os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	assert.NoError(t, err)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	parser := parser.NewSourceFileParser(mockLogger)
	storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
	assert.NoError(t, err)
	defer storage.Close()
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, storage)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)

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

	// 测试 QuerySymbols
	symbols, err := indexer.QuerySymbols(tempDir, testFile, []string{"main"})
	assert.NoError(t, err)
	assert.NotEmpty(t, symbols)

	// 验证日志调用
	mockLogger.AssertCalled(t, "Info", mock.Anything, mock.Anything)
}

func TestIndexer_IndexWorkspace_NoProjects(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-no-projects")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	parser := parser.NewSourceFileParser(mockLogger)
	storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
	assert.NoError(t, err)
	defer storage.Close()
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, storage)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)

	indexer := NewCodeIndexer(
		parser,
		analyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		mockLogger,
	)

	// 测试 IndexWorkspace - 应该返回错误，因为没有找到项目
	err = indexer.IndexWorkspace(context.Background(), tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find no projects")
}

func TestIndexer_IndexFiles_NoProject(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "test-no-project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 准备依赖
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	parser := parser.NewSourceFileParser(mockLogger)
	storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
	assert.NoError(t, err)
	defer storage.Close()
	analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, storage)
	workspaceReader := workspace.NewWorkSpaceReader(mockLogger)

	indexer := NewCodeIndexer(
		parser,
		analyzer,
		workspaceReader,
		storage,
		IndexerConfig{VisitPattern: visitPattern},
		mockLogger,
	)

	// 测试 IndexFiles - 应该返回错误，因为没有找到项目
	err = indexer.IndexFiles(context.Background(), tempDir, []string{"nonexistent.go"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

// TestIndexer_indexProject 测试 indexProject 方法的各种场景
func TestIndexer_indexProject(t *testing.T) {
	tests := []struct {
		name           string
		setupProject   func(t *testing.T) (*workspace.Project, string)
		expectedError  bool
		expectedFiles  int
		expectedSource int
	}{
		{
			name: "正常处理项目文件",
			setupProject: func(t *testing.T) (*workspace.Project, string) {
				tempDir, err := os.MkdirTemp("", "test-normal-project")
				assert.NoError(t, err)

				projectDir := filepath.Join(tempDir, "project")
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

				return &workspace.Project{
					Path: projectDir,
					Uuid: "test-uuid",
					Name: "test-project",
				}, tempDir
			},
			expectedError:  false,
			expectedFiles:  1,
			expectedSource: 1,
		},
		{
			name: "处理空项目（无源文件）",
			setupProject: func(t *testing.T) (*workspace.Project, string) {
				tempDir, err := os.MkdirTemp("", "test-empty-project")
				assert.NoError(t, err)

				projectDir := filepath.Join(tempDir, "project")
				err = os.MkdirAll(projectDir, 0755)
				assert.NoError(t, err)

				// 创建 .git 目录
				err = os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
				assert.NoError(t, err)

				// 创建非源文件
				err = os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test"), 0644)
				assert.NoError(t, err)

				return &workspace.Project{
					Path: projectDir,
					Uuid: "test-uuid",
					Name: "test-project",
				}, tempDir
			},
			expectedError:  true,
			expectedFiles:  1,
			expectedSource: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, tempDir := tt.setupProject(t)
			defer os.RemoveAll(tempDir)

			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Info", mock.Anything, mock.Anything).Return()
			mockLogger.On("Error", mock.Anything, mock.Anything).Return()

			parser := parser.NewSourceFileParser(mockLogger)
			analyzer := analyzer.NewDependencyAnalyzer(mockLogger, nil, nil)
			workspaceReader := workspace.NewWorkSpaceReader(mockLogger)
			storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
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

			metrics, errs := indexer.indexProject(context.Background(), project)

			if tt.expectedError {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
				assert.Equal(t, tt.expectedFiles, metrics.TotalFiles)
				assert.Equal(t, tt.expectedSource, metrics.TotalSourceFiles)
			}
		})
	}
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
			storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
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
	storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
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
			storage, err := store.NewBBoltStorage(filepath.Join(tempDir, "storage"), mockLogger)
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
			symbols, err := indexer.QuerySymbols(tempDir, testFile, tt.symbolNames)
			assert.NoError(t, err)
			assert.Len(t, symbols, tt.expectedCount)
		})
	}
}
