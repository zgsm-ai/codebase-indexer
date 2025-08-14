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

// StatusCheckerJob 状态检查任务
type StatusCheckerJob struct {
	checker  service.EmbeddingStatusService
	storage  repository.StorageInterface
	httpSync repository.SyncInterface
	logger   logger.Logger
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewStatusCheckerJob 创建状态检查任务
func NewStatusCheckerJob(
	checker service.EmbeddingStatusService,
	storage repository.StorageInterface,
	httpSync repository.SyncInterface,
	logger logger.Logger,
	interval time.Duration,
) *StatusCheckerJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &StatusCheckerJob{
		checker:  checker,
		storage:  storage,
		httpSync: httpSync,
		logger:   logger,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动状态检查任务
func (j *StatusCheckerJob) Start(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			j.logger.Error("recovered from panic in status checker job: %v", r)
		}
	}()
	j.logger.Info("starting status checker job with interval: %v", j.interval)

	// 立即执行一次检查
	authInfo := config.GetAuthInfo()
	if j.interval > 0 && authInfo.ClientId != "" && authInfo.Token != "" && authInfo.ServerURL != "" {
		j.checkBuildingStates(ctx)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				j.logger.Error("recovered from panic in check building: %v", r)
			}
		}()
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				j.logger.Info("status checker task stopped")
				return
			case <-ticker.C:
				authInfo := config.GetAuthInfo()
				if authInfo.ClientId == "" || authInfo.Token == "" || authInfo.ServerURL == "" {
					j.logger.Warn("auth info is nil, skip status checker task")
					continue
				}
				j.checkBuildingStates(ctx)
			}
		}
	}()
}

// Stop 停止状态检查任务
func (j *StatusCheckerJob) Stop() {
	j.logger.Info("stopping status checker job...")
	j.cancel()
	j.logger.Info("status checker job stopped")
}

// checkBuildingStates 检查所有building状态
func (j *StatusCheckerJob) checkBuildingStates(ctx context.Context) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		j.logger.Info("context cancelled, skipping status check")
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
		j.logger.Info("codebase is disabled, skipping status check")
		return
	}

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

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		j.logger.Info("context cancelled, skipping embedding process")
		return
	default:
		// 继续执行
	}

	err = j.checker.CheckAllBuildingStates(workspacePaths)
	if err != nil {
		j.logger.Error("failed to check building states: %v", err)
		return
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		j.logger.Info("context cancelled, skipping uploading process")
		return
	default:
		// 继续执行
	}

	err = j.checker.CheckAllUploadingStatues(workspacePaths)
	if err != nil {
		j.logger.Error("failed to check uploading states: %v", err)
	}
}
