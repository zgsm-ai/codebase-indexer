package job

import (
	"context"
	"sync"
	"time"

	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// StatusCheckerJob 状态检查任务
type StatusCheckerJob struct {
	checker  service.EmbeddingStatusService
	logger   logger.Logger
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewStatusCheckerJob 创建状态检查任务
func NewStatusCheckerJob(
	checker service.EmbeddingStatusService,
	logger logger.Logger,
	interval time.Duration,
) *StatusCheckerJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &StatusCheckerJob{
		checker:  checker,
		logger:   logger,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动状态检查任务
func (j *StatusCheckerJob) Start() {
	j.logger.Info("starting status checker job with interval: %v", j.interval)

	j.wg.Add(1)
	go func() {
		defer j.wg.Done()

		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		// 立即执行一次检查
		j.checkBuildingStates()

		for {
			select {
			case <-j.ctx.Done():
				return
			case <-ticker.C:
				j.checkBuildingStates()
			}
		}
	}()
}

// Stop 停止状态检查任务
func (j *StatusCheckerJob) Stop() {
	j.logger.Info("stopping status checker job...")
	j.cancel()
	j.wg.Wait()
	j.logger.Info("status checker job stopped")
}

// checkBuildingStates 检查所有building状态
func (j *StatusCheckerJob) checkBuildingStates() {
	// 获取活跃工作区
	workspaces, err := j.checker.CheckActiveWorkspaces()
	if err != nil {
		j.logger.Error("failed to check active workspaces: %v", err)
		return
	}

	if len(workspaces) == 0 {
		j.logger.Debug("no active workspaces found")
		return
	}

	workspacePaths := make([]string, len(workspaces))
	for i, workspace := range workspaces {
		workspacePaths[i] = workspace.WorkspacePath
	}

	err = j.checker.CheckAllBuildingStates(workspacePaths)
	if err != nil {
		j.logger.Error("failed to check building states: %v", err)
		return
	}

	err = j.checker.CheckAllUploadingStatues(workspacePaths)
	if err != nil {
		j.logger.Error("failed to check uploading states: %v", err)
	}
}
