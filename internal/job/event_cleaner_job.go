// job/event_cleaner_job.go - Event expiration cleanup job
package job

import (
	"context"
	"time"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/logger"
)

// EventCleanerJob 事件过期清理任务
type EventCleanerJob struct {
	eventRepo repository.EventRepository
	logger    logger.Logger
}

// NewEventCleanerJob 创建新的事件清理任务
func NewEventCleanerJob(eventRepo repository.EventRepository, logger logger.Logger) *EventCleanerJob {
	return &EventCleanerJob{
		eventRepo: eventRepo,
		logger:    logger,
	}
}

// Start 启动事件清理任务
func (j *EventCleanerJob) Start(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			j.logger.Error("recovered from panic in event cleaner job: %v", r)
		}
	}()

	j.logger.Info("event cleaner job started")

	// 立即执行一次清理
	j.executeCleanup()

	// 创建定时器，每天22:00执行
	for {
		nextRun := j.getNextRunTime()
		if nextRun.IsZero() {
			j.logger.Error("failed to calculate next run time")
			return
		}

		j.logger.Info("next event cleanup scheduled at: %s", nextRun.Format(time.RFC3339))

		select {
		case <-ctx.Done():
			j.logger.Info("event cleaner job stopped")
			return
		case <-time.After(time.Until(nextRun)):
			j.executeCleanup()
		}
	}
}

// getNextRunTime 计算下一个22:00的运行时间
func (j *EventCleanerJob) getNextRunTime() time.Time {
	now := time.Now()

	// 获取今天的22:00
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, now.Location())

	// 如果今天的22:00已经过了，计算明天的22:00
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun
}

// executeCleanup 执行清理操作
func (j *EventCleanerJob) executeCleanup() {
	j.logger.Info("starting event cleanup process")

	// 计算2天前的时间
	cutoffTime := time.Now().Add(-48 * time.Hour)

	// 获取需要删除的事件ID
	eventIDs, err := j.eventRepo.GetExpiredEventIDs(cutoffTime)
	if err != nil {
		j.logger.Error("failed to get expired event IDs: %v", err)
		return
	}

	if len(eventIDs) > 0 {
		j.logger.Info("found %d expired events to delete", len(eventIDs))
		// 批量删除过期事件
		err = j.eventRepo.BatchDeleteEvents(eventIDs)
		if err != nil {
			j.logger.Error("failed to batch delete expired events: %v", err)
			return
		}
		j.logger.Info("successfully deleted %d expired events", len(eventIDs))
	}

	// 检查是否所有事件都为已成功状态
	allEventsSuccess, err := j.checkAllEventsSuccess()
	if err != nil {
		j.logger.Error("failed to check if all events are success: %v", err)
		return
	}

	if allEventsSuccess {
		j.logger.Info("all events are in success status, clearing events table")
		// 直接通过 eventRepo 清理 events 表
		err := j.eventRepo.ClearTable()
		if err != nil {
			j.logger.Error("failed to clear events table: %v", err)
			return
		}
		j.logger.Info("events table cleared successfully and ID reset")
	}
}

// checkAllEventsSuccess 检查是否所有事件都为已成功状态
func (j *EventCleanerJob) checkAllEventsSuccess() (bool, error) {
	// 定义非成功的状态
	nonSuccessEmbeddingStatus := []int{
		model.EmbeddingStatusInit,
		model.EmbeddingStatusUploading,
		model.EmbeddingStatusBuilding,
		model.EmbeddingStatusUploadFailed,
		model.EmbeddingStatusBuildFailed,
	}
	nonSuccessCodegraphStatus := []int{
		model.CodegraphStatusInit,
		model.CodegraphStatusBuilding,
		model.CodegraphStatusFailed,
	}

	// 检查是否存在非成功状态的嵌入事件
	// 使用空的 workspacePaths 列表来检查所有工作区
	nonSuccessEmbeddingCount, err := j.eventRepo.GetEventsCountByWorkspaceAndStatus([]string{}, nonSuccessEmbeddingStatus, []int{})
	if err != nil {
		j.logger.Error("failed to get embedding event count: %v", err)
		return false, err
	}

	// 检查是否存在非成功状态的代码图事件
	nonSuccessCodegraphCount, err := j.eventRepo.GetEventsCountByWorkspaceAndStatus([]string{}, []int{}, nonSuccessCodegraphStatus)
	if err != nil {
		j.logger.Error("failed to get codegraph event count: %v", err)
		return false, err
	}

	// 如果没有非成功状态的事件，则所有事件都为成功状态
	return nonSuccessEmbeddingCount == 0 && nonSuccessCodegraphCount == 0, nil
}
