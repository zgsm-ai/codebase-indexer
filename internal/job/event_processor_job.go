package job

import (
	"context"
	"sync"
	"time"

	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// EventProcessorJob 事件处理任务
type EventProcessorJob struct {
	embedding service.EmbeddingProcessService
	codegraph service.CodegraphProcessService
	logger    logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewEventProcessorJob 创建事件处理任务
func NewEventProcessorJob(
	logger logger.Logger,
	embedding service.EmbeddingProcessService,
	codegraph service.CodegraphProcessService,
) *EventProcessorJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventProcessorJob{
		embedding: embedding,
		codegraph: codegraph,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动事件处理任务
func (j *EventProcessorJob) Start() {
	j.logger.Info("starting event processor job")

	j.wg.Add(1)
	go func() {
		defer j.wg.Done()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// 立即执行一次事件处理
		j.embeddingProcessWorkspaces()

		for {
			select {
			case <-j.ctx.Done():
				j.logger.Info("event embedding job stopped")
				return
			case <-ticker.C:
				// 处理事件
				j.embeddingProcessWorkspaces()
				// 短暂休眠避免CPU占用过高
				time.Sleep(1 * time.Second)
			}
		}
	}()

	j.wg.Add(1)
	go func() {
		defer j.wg.Done()

		for {
			select {
			case <-j.ctx.Done():
				return
			default:
				err := j.codegraphProcessWorkSpaces()
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
	j.wg.Wait()
	j.logger.Info("event processor job stopped")
}

func (j *EventProcessorJob) embeddingProcessWorkspaces() {
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

	err = j.embedding.ProcessEmbeddingEvents(workspackePaths)
	if err != nil {
		j.logger.Error("failed to process embedding events: %v", err)
	}
}

func (j *EventProcessorJob) codegraphProcessWorkSpaces() error {
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
