package indexer

import (
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"os"
	"strconv"
	"sync"
	"time"
)

// Indexer 代码索引器实现
type Indexer struct {
	ignoreScanner       repository.ScannerInterface
	parser              *parser.SourceFileParser
	analyzer            *analyzer.DependencyAnalyzer
	workspaceReader     workspace.WorkspaceReader
	storage             store.GraphStorage
	workspaceRepository repository.WorkspaceRepository
	config              *Config
	logger              logger.Logger
	mu                  sync.Mutex
}

// NewIndexer 创建新的代码索引器
func NewIndexer(
	ignoreScanner repository.ScannerInterface,
	parser *parser.SourceFileParser,
	analyzer *analyzer.DependencyAnalyzer,
	workspaceReader workspace.WorkspaceReader,
	storage store.GraphStorage,
	workspaceRepository repository.WorkspaceRepository,
	config Config,
	logger logger.Logger,
) *Indexer {
	initConfig(&config)
	return &Indexer{
		ignoreScanner:       ignoreScanner,
		parser:              parser,
		analyzer:            analyzer,
		workspaceReader:     workspaceReader,
		storage:             storage,
		workspaceRepository: workspaceRepository,
		config:              &config,
		logger:              logger,
	}
}

// initConfig 初始化配置，增加环境变量读取逻辑
func initConfig(config *Config) {
	// 从环境变量获取MaxConcurrency（环境变量名：MAX_CONCURRENCY）
	if envVal, ok := os.LookupEnv("MAX_CONCURRENCY"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			config.MaxConcurrency = val
		}
	}
	// 若环境变量未设置或无效，使用默认值（原逻辑）
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = DefaultConcurrency
	}

	// 从环境变量获取MaxBatchSize（环境变量名：MAX_BATCH_SIZE）
	if envVal, ok := os.LookupEnv("MAX_BATCH_SIZE"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			config.MaxBatchSize = val
		}
	}
	if config.MaxBatchSize <= 0 {
		config.MaxBatchSize = DefaultBatchSize
	}

	// 从环境变量获取MaxFiles（环境变量名：MAX_FILES）
	if envVal, ok := os.LookupEnv("MAX_FILES"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			config.MaxFiles = val
		}
	}

	// 从环境变量获取MaxProjects（环境变量名：MAX_PROJECTS）
	if envVal, ok := os.LookupEnv("MAX_PROJECTS"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			config.MaxProjects = val
		}
	}
	if config.MaxProjects <= 0 {
		config.MaxProjects = DefaultMaxProjects
	}

	// 从环境变量获取CacheCapacity（环境变量名：CACHE_CAPACITY）
	if envVal, ok := os.LookupEnv("CACHE_CAPACITY"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			config.CacheCapacity = val
		}
	}
	if config.CacheCapacity <= 0 {
		config.CacheCapacity = DefaultCacheCapacity
	}
}

// IndexIter 获取索引迭代器
func (idx *Indexer) IndexIter(ctx context.Context, projectUuid string) store.Iterator {
	return idx.storage.Iter(ctx, projectUuid)
}

// GetSummary 获取代码图摘要信息
func (idx *Indexer) GetSummary(ctx context.Context, workspacePath string) (*types.CodeGraphSummary, error) {
	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, nil
	}
	summary := new(types.CodeGraphSummary)
	for _, p := range projects {
		summary.TotalFiles += idx.storage.Size(ctx, p.Uuid, store.PathKeySystemPrefix)
	}
	return summary, nil
}

// updateProgress 更新进度
func (idx *Indexer) updateProgress(ctx context.Context, progress *ProgressInfo) error {

	if err := idx.workspaceRepository.UpdateCodegraphInfo(progress.WorkspacePath,
		progress.Processed+progress.PreviousNum, time.Now().Unix()); err != nil {
		idx.logger.Error("update workspace %s codegraph successful file num %d/%d, err:%v",
			progress.WorkspacePath, progress.Processed+progress.PreviousNum, progress.Total, err)
		return err
	}

	return nil
}
