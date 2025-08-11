package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// FileScanJob 文件扫描任务
type FileScanJob struct {
	scanner  service.FileScanService
	storage  repository.StorageInterface
	logger   logger.Logger
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewFileScanJob 创建文件扫描任务
func NewFileScanJob(
	scanner service.FileScanService,
	storage repository.StorageInterface,
	logger logger.Logger,
	interval time.Duration,
) *FileScanJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileScanJob{
		scanner:  scanner,
		storage:  storage,
		logger:   logger,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动文件扫描任务
func (j *FileScanJob) Start() {
	j.logger.Info("starting file scan job with interval: %v", j.interval)

	// 立即执行一次扫描
	j.scanWorkspaces()

	j.wg.Add(1)
	go func() {
		defer j.wg.Done()
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		for {
			select {
			case <-j.ctx.Done():
				return
			case <-ticker.C:
				j.scanWorkspaces()
			}
		}
	}()
}

// Stop 停止文件扫描任务
func (j *FileScanJob) Stop() {
	j.logger.Info("stopping file scan job...")
	j.cancel()
	j.wg.Wait()
	j.logger.Info("file scan job stopped")
}

// scanWorkspaces 扫描工作区
func (j *FileScanJob) scanWorkspaces() {
	// 检查上下文是否已取消
	select {
	case <-j.ctx.Done():
		j.logger.Info("context cancelled, skipping workspace scan")
		return
	default:
		// 继续执行
	}
	j.logger.Info("starting workspace scan")

	// 检查是否关闭codebase
	codebaseEnv := j.storage.GetCodebaseEnv()
	if codebaseEnv == nil {
		codebaseEnv = &config.CodebaseEnv{
			Switch: dto.SwitchOn,
		}
	}
	if codebaseEnv.Switch == dto.SwitchOff {
		j.logger.Info("codebase is disabled, skipping workspace scan")
		return
	}

	// 获取活跃工作区
	workspaces, err := j.scanner.ScanActiveWorkspaces()
	if err != nil {
		j.logger.Error("failed to scan active workspaces: %v", err)
		return
	}

	if len(workspaces) == 0 {
		j.logger.Debug("no active workspaces found")
		return
	}

	// 检查上下文是否已取消
	select {
	case <-j.ctx.Done():
		j.logger.Info("context cancelled, skipping workspace scan")
		return
	default:
		// 继续执行
	}

	// 扫描每个工作区
	for _, workspace := range workspaces {
		err := j.scanWorkspace(workspace)
		if err != nil {
			j.logger.Error("failed to scan workspace %s: %v", workspace.WorkspacePath, err)
			continue
		}
	}

	j.logger.Info("workspace scan completed")
}

// scanWorkspace 扫描单个工作区
func (j *FileScanJob) scanWorkspace(workspace *model.Workspace) error {
	j.logger.Info("scanning workspace: %s", workspace.WorkspacePath)

	// 检测文件变更
	events, err := j.scanner.DetectFileChanges(workspace)
	if err != nil {
		return fmt.Errorf("failed to detect file changes: %w", err)
	}

	j.logger.Info("detected %d file changes in workspace: %s", len(events), workspace.WorkspacePath)

	// 更新工作区统计信息
	err = j.scanner.UpdateWorkspaceStats(workspace)
	if err != nil {
		j.logger.Error("failed to update workspace stats: %v", err)
	}

	return nil
}
