package codegraph

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/codegraph/analyzer"
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	_ "net/http/pprof" // 自动注册 pprof 接口
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
	Scanner            repository.ScannerInterface
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

	// 监控

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
		Scanner:            repository.NewFileScanner(newLogger),
	}, nil
}

// createTestIndexer 创建测试用的索引器
func createTestIndexer(env *testEnvironment, visitPattern *types.VisitPattern) *service.Indexer {
	return service.NewCodeIndexer(
		env.Scanner,
		env.sourceFileParser,
		env.dependencyAnalyzer,
		env.workspaceReader,
		env.storage,
		env.repository,
		service.IndexerConfig{VisitPattern: visitPattern, MaxBatchSize: 50, MaxConcurrency: 1},
		// 2,2, 300s， 20% cpu ,500MB内存占用；
		// 2

		// 100,10 156s ,  70% cpu , 500MB内存占用；
		env.logger,
	)
}

func setupPprof() {
	// 启动 pprof HTTP 服务（端口自定义，如 6060）
	// go tool pprof http://localhost:6060/debug/pprof/heap
	// top；  list 函数名；web；heapcheck
	go func() {
		_ = http.ListenAndServe("localhost:6060", nil)
	}()
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

// ParseProjectFiles 解析项目中的所有文件
func ParseProjectFiles(ctx context.Context, env *testEnvironment, p *workspace.Project) ([]*parser.FileElementTable, types.IndexTaskMetrics, error) {
	fileElementTables := make([]*parser.FileElementTable, 0)
	projectTaskMetrics := types.IndexTaskMetrics{}

	// TODO walk 目录收集列表， 并发构建，批量保存结果
	if err := env.workspaceReader.WalkFile(ctx, p.Path, func(walkCtx *types.WalkContext) error {
		projectTaskMetrics.TotalFiles++
		language, err := lang.InferLanguage(walkCtx.Path)
		if err != nil || language == types.EmptyString {
			// not supported language or not source file
			return nil
		}

		content, err := env.workspaceReader.ReadFile(ctx, walkCtx.Path, types.ReadOptions{})
		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			return err
		}
		fileElementTable, err := env.sourceFileParser.Parse(ctx, &types.SourceFile{
			Path:    walkCtx.Path,
			Content: content,
		})

		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			return err
		}
		fileElementTables = append(fileElementTables, fileElementTable)
		return nil
	}, types.WalkOptions{
		IgnoreError:  true,
		VisitPattern: workspace.DefaultVisitPattern,
	}); err != nil {
		return nil, types.IndexTaskMetrics{}, err
	}

	return fileElementTables, projectTaskMetrics, nil
}
