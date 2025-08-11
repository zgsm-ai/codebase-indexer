package codegraph

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/codegraph"
	"codebase-indexer/pkg/codegraph/analyzer"
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var tempDir = "/tmp/"
var defaultExportDir = "/tmp/export"

var defaultVisitPattern = types.VisitPattern{ExcludeDirs: []string{".git", ".idea", ".vscode"}}

// TODO 性能（内存、cpu）监控；各种路径、项目名（中文、符号）测试；索引数量统计；大仓库测试

// testEnvironment 包含测试所需的环境组件
type testEnvironment struct {
	ctx                context.Context
	cancel             context.CancelFunc
	storageDir         string
	logger             logger.Logger
	storage            store.GraphStorage
	repository         repository.WorkspaceRepository
	workspaceReader    *workspace.WorkspaceReader
	sourceFileParser   *parser.SourceFileParser
	dependencyAnalyzer *analyzer.DependencyAnalyzer
}

// setupTestEnvironment 设置测试环境，创建所需的目录和组件
func setupTestEnvironment() (*testEnvironment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	// 创建存储目录
	storageDir := filepath.Join(tempDir, "index")
	err := os.MkdirAll(storageDir, 0755)
	if err != nil {
		cancel()
		return nil, err
	}
	// 创建日志器
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")

	// 创建存储
	storage, err := store.NewLevelDBStorage(storageDir, newLogger)

	// 创建工作区读取器
	workspaceReader := workspace.NewWorkSpaceReader(newLogger)

	// 创建源文件解析器
	sourceFileParser := parser.NewSourceFileParser(newLogger)

	packageClassifier := packageclassifier.NewPackageClassifier()

	// 创建依赖分析器
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(newLogger, packageClassifier, workspaceReader, storage)

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
	}, nil
}

// createTestIndexer 创建测试用的索引器
func createTestIndexer(env *testEnvironment, visitPattern *types.VisitPattern) *codegraph.Indexer {
	return codegraph.NewCodeIndexer(
		env.sourceFileParser,
		env.dependencyAnalyzer,
		env.workspaceReader,
		env.storage,
		env.repository,
		codegraph.IndexerConfig{VisitPattern: visitPattern, MaxBatchSize: 100, MaxConcurrency: 10},
		// 2,2, 300s， 20% cpu ,500MB内存占用；
		// 2

		// 100,10 156s ,  70% cpu , 500MB内存占用；
		env.logger,
	)
}

// teardownTestEnvironment 清理测试环境，关闭连接和删除临时文件
func teardownTestEnvironment(t *testing.T, env *testEnvironment) {

	// 关闭存储连接
	err := env.storage.Close()
	assert.NoError(t, err)
	// 取消上下文
	env.cancel()
}
func cleanTestIndexStore(ctx context.Context, projects []*workspace.Project, storage store.GraphStorage) error {
	for _, p := range projects {
		if err := storage.DeleteAll(ctx, p.Uuid); err != nil {
			return err
		}
		if storage.Size(ctx, p.Uuid, store.PathKeySystemPrefix) > 0 {
			return fmt.Errorf("clean workspace index failed, size not equal 0")
		}
	}
	return nil
}

func NewTestProject(path string, logger logger.Logger) *workspace.Project {
	project := workspace.NewProject(filepath.Base(path), path)
	return project
}

func exportFileElements(path string, project string, elements []*parser.FileElementTable) error {
	file := filepath.Join(path, project+"_file_elements.json")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ") // 第一个参数是前缀，第二个参数是缩进（这里使用两个空格）
	if err := encoder.Encode(elements); err != nil {
		return err
	}
	return nil
}

func initWorkspaceModel(env *testEnvironment, workspaceDir string) error {
	workspaceModel, err := env.repository.GetWorkspaceByPath(workspaceDir)
	if workspaceModel == nil {
		// 初始化workspace
		err := env.repository.CreateWorkspace(&model.Workspace{
			WorkspaceName: "codebase-indexer",
			WorkspacePath: workspaceDir,
			Active:        "true",
			FileNum:       100,
		})
		if err != nil {
			return err
		}
	} else {
		// 置为 0
		err := env.repository.UpdateCodegraphInfo(workspaceDir, 0, time.Now().Unix())
		if err != nil {
			return err
		}
	}
	return err
}
