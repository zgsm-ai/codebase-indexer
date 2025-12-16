package service

import (
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service/indexer"
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
)

// Indexer 定义代码索引器的接口，便于mock测试
// 这是对外暴露的公共接口，保持向后兼容
type Indexer interface {
	// IndexWorkspace 索引整个工作区
	IndexWorkspace(ctx context.Context, workspacePath string) (*types.IndexTaskMetrics, error)

	// IndexFiles 根据工作区路径、文件路径，批量保存索引
	IndexFiles(ctx context.Context, workspacePath string, filePaths []string) error

	// RenameIndexes 重命名索引，根据路径（文件或文件夹）
	RenameIndexes(ctx context.Context, workspacePath string, sourceFilePath string, targetFilePath string) error

	// RemoveIndexes 根据工作区路径、文件路径/文件夹路径前缀，批量删除索引
	RemoveIndexes(ctx context.Context, workspacePath string, filePaths []string) error

	// RemoveAllIndexes 删除工作区的所有索引
	RemoveAllIndexes(ctx context.Context, workspacePath string) error

	// QueryReferences 查询引用
	QueryReferences(ctx context.Context, opts *types.QueryReferenceOptions) ([]*types.RelationNode, error)

	// QueryDefinitions 查询定义
	QueryDefinitions(ctx context.Context, options *types.QueryDefinitionOptions) ([]*types.Definition, error)

	// QueryCallGraph 查询代码片段内部元素或单符号的调用链及其里面的元素定义，支持代码片段检索
	QueryCallGraph(ctx context.Context, opts *types.QueryCallGraphOptions) ([]*types.RelationNode, error)

	// GetSummary 获取代码图摘要信息
	GetSummary(ctx context.Context, workspacePath string) (*types.CodeGraphSummary, error)

	// IndexIter 获取索引迭代器
	IndexIter(ctx context.Context, projectUuid string) store.Iterator

	// GetFileElementTable 获取文件元素表
	GetFileElementTable(ctx context.Context, workspacePath string, filePath string) (*codegraphpb.FileElementTable, error)
}

// IndexerConfig 索引器配置（类型别名，保持向后兼容）
type IndexerConfig = indexer.Config

// NewCodeIndexer 创建新的代码索引器
// 这是外观层的构造函数，委托给内部实现
func NewCodeIndexer(
	ignoreScanner repository.ScannerInterface,
	parserInstance *parser.SourceFileParser,
	analyzerInstance *analyzer.DependencyAnalyzer,
	workspaceReader workspace.WorkspaceReader,
	storage store.GraphStorage,
	workspaceRepository repository.WorkspaceRepository,
	config IndexerConfig,
	loggerInstance logger.Logger,
) Indexer {
	// 直接委托给内部实现
	return indexer.NewIndexer(
		ignoreScanner,
		parserInstance,
		analyzerInstance,
		workspaceReader,
		storage,
		workspaceRepository,
		config,
		loggerInstance,
	)
}
