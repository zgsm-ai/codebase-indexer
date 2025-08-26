package job

import (
	"codebase-indexer/internal/daemon"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
	"context"
	"os"
	"strconv"
	"time"
)

const defaultCleanInterval = 60 * time.Minute
const defaultExpiryPeriod = 3 * 24 * time.Hour

type IndexCleanJob struct {
	logger        logger.Logger
	indexer       service.Indexer
	workspaceRepo repository.WorkspaceRepository
	checkInterval time.Duration
	expiryPeriod  time.Duration
}

func NewIndexCleanJob(logger logger.Logger, indexer service.Indexer,
	workspaceRepository repository.WorkspaceRepository) daemon.Job {
	var checkInterval time.Duration
	var expiryPeriod time.Duration
	if env, ok := os.LookupEnv("INDEX_CLEAN_CHECK_INTERVAL_MINUTES"); ok {
		if val, err := strconv.Atoi(env); err == nil {
			checkInterval = time.Duration(val) * time.Minute
		}
	}

	if checkInterval == 0 {
		checkInterval = defaultCleanInterval
	}

	if env, ok := os.LookupEnv("INDEX_EXPIRY_PERIOD_HOURS"); ok {
		if val, err := strconv.Atoi(env); err == nil {
			expiryPeriod = time.Duration(val) * time.Hour
		}
	}

	if expiryPeriod == 0 {
		expiryPeriod = defaultExpiryPeriod
	}

	return &IndexCleanJob{
		logger:        logger,
		indexer:       indexer,
		workspaceRepo: workspaceRepository,
		checkInterval: checkInterval,
		expiryPeriod:  expiryPeriod,
	}
}

func (j *IndexCleanJob) Start(ctx context.Context) {
	j.logger.Info("starting index clean job with checkInterval %.0f minutes, expiry period %.0f hours",
		j.checkInterval.Minutes(), j.expiryPeriod.Hours())
	go func() {
		ticker := time.NewTicker(j.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				j.logger.Info("index clean job stopped")
				return
			case <-ticker.C:
				j.cleanupExpiredWorkspaceIndexes(ctx)
			}
		}
	}()
}

func (j *IndexCleanJob) cleanupExpiredWorkspaceIndexes(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			j.logger.Error("recovered from panic in index clean job: %v", r)
		}
	}()
	j.logger.Info("start to clean up expired workspace indexes with expiry period %.0f hours", j.expiryPeriod.Hours())
	workspaces, err := j.workspaceRepo.ListWorkspaces()
	if err != nil {
		j.logger.Error("list workspaces failed with %v", err)
		return
	}

	if len(workspaces) == 0 {
		j.logger.Debug("no workspaces found")
		return
	}

	for _, workspace := range workspaces {
		// 活跃中 更新时间小于过期间隔 索引数量为0 跳过
		if workspace.Active == dto.True || time.Now().Sub(workspace.UpdatedAt).Hours() < j.expiryPeriod.Hours() ||
			workspace.CodegraphFileNum == 0 {
			continue
		}
		j.logger.Info("workspace %s updated_at %s exceeds expiry period %.0f hours, start to cleanup.",
			workspace.WorkspacePath, workspace.UpdatedAt.Format("2006-01-02 15:04:05"), j.expiryPeriod.Hours())
		// 清理索引 （有更新数据库为0的逻辑）
		if err = j.indexer.RemoveAllIndexes(ctx, workspace.WorkspacePath); err != nil {
			j.logger.Error("remove indexes failed with %v", err)
			continue
		}

		j.logger.Info("workspace %s clean up expired indexes successfully.", workspace.WorkspacePath)
	}
	j.logger.Info("clean up expired workspace indexes end.")
}
