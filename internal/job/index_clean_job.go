package job

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/daemon"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
)

const defaultCleanInterval = 60 * time.Minute
const defaultExpiryPeriod = 3 * 24 * time.Hour

type IndexCleanJob struct {
	logger                logger.Logger
	indexer               service.Indexer
	workspaceRepo         repository.WorkspaceRepository
	storageRepo           repository.StorageInterface
	embeddingRepo         repository.EmbeddingFileRepository
	syncRepo              repository.SyncInterface
	eventRepo             repository.EventRepository
	checkInterval         time.Duration
	expiryPeriod          time.Duration
	embeddingExpiryPeriod time.Duration
}

func NewIndexCleanJob(logger logger.Logger, indexer service.Indexer,
	workspaceRepository repository.WorkspaceRepository, storageRepo repository.StorageInterface,
	embeddingRepo repository.EmbeddingFileRepository, syncRepo repository.SyncInterface,
	eventRepo repository.EventRepository) daemon.Job {
	var checkInterval time.Duration
	var expiryPeriod time.Duration
	var embeddingExpiryPeriod time.Duration

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

	// 默认 embedding 过期时间为 7 天
	if env, ok := os.LookupEnv("EMBEDDING_EXPIRY_PERIOD_DAYS"); ok {
		if val, err := strconv.Atoi(env); err == nil {
			embeddingExpiryPeriod = time.Duration(val) * 24 * time.Hour
		}
	}

	if embeddingExpiryPeriod == 0 {
		embeddingExpiryPeriod = 7 * 24 * time.Hour // 默认 7 天
	}

	return &IndexCleanJob{
		logger:                logger,
		indexer:               indexer,
		workspaceRepo:         workspaceRepository,
		storageRepo:           storageRepo,
		embeddingRepo:         embeddingRepo,
		syncRepo:              syncRepo,
		eventRepo:             eventRepo,
		checkInterval:         checkInterval,
		expiryPeriod:          expiryPeriod,
		embeddingExpiryPeriod: embeddingExpiryPeriod,
	}
}

func (j *IndexCleanJob) Start(ctx context.Context) {
	j.logger.Info("starting index clean job with checkInterval %.0f minutes, expiry period %.0f hours",
		j.checkInterval.Minutes(), j.expiryPeriod.Hours())

	// 原有的清理过期工作区索引的协程
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

	// 新增的清理 embedding 索引的协程，每天23点执行
	go func() {
		j.logger.Info("starting embedding clean job")
		// 立即执行一次清理
		j.cleanupInactiveWorkspaceEmbeddings(ctx)
		// 获取随机分钟数
		randMinute := utils.RandomInt(0, 30)

		for {
			nextRun := j.getNextRunTime(randMinute)
			if nextRun.IsZero() {
				j.logger.Error("failed to calculate next run time for embedding clean job")
				return
			}

			j.logger.Info("next embedding cleanup scheduled at: %s", nextRun.Format(time.RFC3339))

			select {
			case <-ctx.Done():
				j.logger.Info("embedding clean job stopped")
				return
			case <-time.After(time.Until(nextRun)):
				j.cleanupInactiveWorkspaceEmbeddings(ctx)
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
		if workspace.Active == dto.True || time.Since(workspace.UpdatedAt) < j.expiryPeriod ||
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

// cleanupInactiveWorkspaceEmbeddings 清理非活跃工作区的 embedding 索引
func (j *IndexCleanJob) cleanupInactiveWorkspaceEmbeddings(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			j.logger.Error("recovered from panic in embedding clean job: %v", r)
		}
	}()

	workspaces, err := j.workspaceRepo.ListWorkspaces()
	if err != nil {
		j.logger.Error("list workspaces failed with %v", err)
		return
	}

	if len(workspaces) == 0 {
		j.logger.Debug("no workspaces found")
		return
	}

	cleanedCount := 0
	authInfo := config.GetAuthInfo()
	clientId := authInfo.ClientId
	for _, workspace := range workspaces {
		// 只处理 active 为 false, embedding_file_num 不为 0, 且更新时间超过过期间隔的工作区
		if workspace.Active == dto.True || time.Since(workspace.UpdatedAt) < j.embeddingExpiryPeriod ||
			workspace.EmbeddingFileNum == 0 {
			continue
		}

		j.logger.Info("workspace %s is inactive and exceeds expiry period, start to cleanup embeddings.",
			workspace.WorkspacePath)

		// 更新 workspace
		updateWorkspace := map[string]interface{}{
			"file_num":                    0,
			"embedding_file_num":          0,
			"embedding_ts":                0,
			"embedding_message":           "",
			"embedding_failed_file_paths": "",
		}
		if err := j.workspaceRepo.UpdateWorkspaceByMap(workspace.WorkspacePath, updateWorkspace); err != nil {
			j.logger.Error("update workspace failed with %v", err)
			continue
		}

		// 1. 删除这个 workspace 的所有 event 表记录
		if err := j.cleanupWorkspaceEvents(workspace.WorkspacePath); err != nil {
			j.logger.Error("failed to cleanup events for workspace %s: %v", workspace.WorkspacePath, err)
			continue
		}

		// 2. 删除 codebaseConfig
		codebaseId := utils.GenerateCodebaseID(workspace.WorkspacePath)
		if err := j.storageRepo.DeleteCodebaseConfig(codebaseId); err != nil {
			j.logger.Error("failed to delete codebase config for workspace %s: %v", workspace.WorkspacePath, err)
			continue
		}

		// 3. 删除 embeddingConfig
		embeddingId := utils.GenerateEmbeddingID(workspace.WorkspacePath)
		if err := j.embeddingRepo.DeleteEmbeddingConfig(embeddingId); err != nil {
			j.logger.Error("failed to delete embedding config for workspace %s: %v", workspace.WorkspacePath, err)
			continue
		}

		// 4. 删除远程配置
		deleteReq := dto.DeleteEmbeddingReq{
			ClientId:     clientId,
			CodebasePath: workspace.WorkspacePath,
			FilePaths:    []string{}, // 空数组表示删除整个工作区的 embedding
		}

		if _, err := j.syncRepo.DeleteEmbedding(deleteReq); err != nil {
			j.logger.Error("failed to delete remote embedding for workspace %s: %v", workspace.WorkspacePath, err)
			continue
		}

		j.logger.Info("workspace %s embeddings cleaned up successfully.", workspace.WorkspacePath)
		cleanedCount++
	}

	j.logger.Info("clean up inactive workspace embeddings end, cleaned %d workspaces.", cleanedCount)
}

// cleanupWorkspaceEvents 删除指定工作区的所有事件记录
func (j *IndexCleanJob) cleanupWorkspaceEvents(workspacePath string) error {
	// 获取工作区的所有事件
	events, err := j.eventRepo.GetEventsByWorkspaceForDeduplication(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to get events for workspace %s: %w", workspacePath, err)
	}

	if len(events) == 0 {
		j.logger.Debug("no events found for workspace %s", workspacePath)
		return nil
	}

	// 批量删除事件
	var eventIDs []int64
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
	}

	if err := j.eventRepo.BatchDeleteEvents(eventIDs); err != nil {
		return fmt.Errorf("failed to batch delete events for workspace %s: %w", workspacePath, err)
	}

	j.logger.Info("deleted %d events for workspace %s", len(eventIDs), workspacePath)
	return nil
}

// getNextRunTime 计算下一个23:00后的随机分钟数的运行时间
func (j *IndexCleanJob) getNextRunTime(randMinute int) time.Time {
	now := time.Now()

	// 获取今天的23:00 加上随机分钟数
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 23, randMinute, 0, 0, now.Location())

	// 如果今天的23:00已经过了，计算明天的23:00
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun
}
