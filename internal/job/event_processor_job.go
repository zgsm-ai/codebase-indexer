package job

import (
	"context"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// EventProcessorJob 事件处理任务
type EventProcessorJob struct {
	httpSync  repository.SyncInterface
	embedding service.EmbeddingProcessService
	codegraph service.CodegraphProcessService
	storage   repository.StorageInterface
	logger    logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewEventProcessorJob 创建事件处理任务
func NewEventProcessorJob(
	logger logger.Logger,
	httpSync repository.SyncInterface,
	embedding service.EmbeddingProcessService,
	codegraph service.CodegraphProcessService,
	storage repository.StorageInterface,
) *EventProcessorJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventProcessorJob{
		httpSync:  httpSync,
		embedding: embedding,
		codegraph: codegraph,
		storage:   storage,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动事件处理任务
func (j *EventProcessorJob) Start(ctx context.Context) {
	j.logger.Info("starting event processor job")

	// 立即执行一次事件处理
	if j.httpSync.GetSyncConfig() != nil {
		j.embeddingProcessWorkspaces(ctx)
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				j.logger.Info("event processor task stopped")
				return
			case <-ticker.C:
				if j.httpSync.GetSyncConfig() == nil {
					j.logger.Warn("sync config is nil, skipping embedding process")
					continue
				}
				// 处理事件
				j.embeddingProcessWorkspaces(ctx)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				j.logger.Info("event processor task stopped")
				return
			default:
				err := j.codegraphProcessWorkSpaces(ctx)
				if err != nil {
					j.logger.Error("failed to process codegraph events: %v", err)
				}
				// 短暂休眠避免CPU占用过高
				time.Sleep(1 * time.Second)
			}
		}
	}()
}

// Stop 停止事件处理任务
func (j *EventProcessorJob) Stop() {
	j.logger.Info("stopping event processor job...")
	j.cancel()
	j.logger.Info("event processor job stopped")
}

func (j *EventProcessorJob) embeddingProcessWorkspaces(ctx context.Context) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		j.logger.Info("context cancelled, skipping embedding process")
		return
	default:
		// 继续执行
	}

	// 检查是否关闭codebase
	codebaseEnv := j.storage.GetCodebaseEnv()
	if codebaseEnv == nil {
		codebaseEnv = &config.CodebaseEnv{
			Switch: dto.SwitchOn,
		}
	}
	if codebaseEnv.Switch == dto.SwitchOff {
		j.logger.Info("codebase is disabled, skipping embedding process")
		return
	}

	// 获取活跃工作区
	workspaces, err := j.embedding.ProcessActiveWorkspaces()
	if err != nil {
		j.logger.Error("failed to scan active workspaces: %v", err)
		return
	}

	if len(workspaces) == 0 {
		j.logger.Debug("no active workspaces found")
		return
	}

	workspackePaths := make([]string, len(workspaces))
	for i, workspace := range workspaces {
		workspackePaths[i] = workspace.WorkspacePath
	}

	// 在处理事件前再次检查上下文是否已取消
	select {
	case <-ctx.Done():
		j.logger.Info("context cancelled, skipping embedding process")
		return
	default:
		// 继续执行
	}

	err = j.embedding.ProcessEmbeddingEvents(j.ctx, workspackePaths)
	if err != nil {
		j.logger.Error("failed to process embedding events: %v", err)
	}
}

func (j *EventProcessorJob) codegraphProcessWorkSpaces(ctx context.Context) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		j.logger.Info("context cancelled, skipping codegraph events processing")
		return j.ctx.Err()
	default:
		// 继续执行
	}
	// 获取活跃工作区
	workspaces, err := j.codegraph.ProcessActiveWorkspaces(j.ctx)
	if err != nil {
		return err
	}

	if len(workspaces) == 0 {
		j.logger.Debug("no active workspaces found")
		return nil
	}
	workspacesPaths := make([]string, len(workspaces))
	for i, workspace := range workspaces {
		workspacesPaths[i] = workspace.WorkspacePath
	}

	return j.codegraph.ProcessEvents(j.ctx, workspacesPaths)
}
