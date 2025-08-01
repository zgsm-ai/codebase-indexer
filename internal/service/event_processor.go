package service

import (
	"fmt"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/logger"
)

// EventProcessService 事件处理服务接口
type EventProcessService interface {
	ProcessAddFileEvent(event *model.Event) error
	ProcessModifyFileEvent(event *model.Event) error
	ProcessDeleteFileEvent(event *model.Event) error
	ProcessEvents() error
}

// eventProcessService 事件处理服务实现
type eventProcessService struct {
	eventRepo          repository.EventRepository
	embeddingStateRepo repository.EmbeddingStateRepository
	logger             logger.Logger
}

// NewEventProcessService 创建事件处理服务
func NewEventProcessService(
	eventRepo repository.EventRepository,
	embeddingStateRepo repository.EmbeddingStateRepository,
	logger logger.Logger,
) EventProcessService {
	return &eventProcessService{
		eventRepo:          eventRepo,
		embeddingStateRepo: embeddingStateRepo,
		logger:             logger,
	}
}

// ProcessAddFileEvent 处理添加文件事件
func (ep *eventProcessService) ProcessAddFileEvent(event *model.Event) error {
	ep.logger.Info("processing add file event: %s", event.SourceFilePath)

	// 创建语义构建状态记录
	state := &model.EmbeddingState{
		SyncID:        "", // 将在创建时生成
		WorkspacePath: event.WorkspacePath,
		FilePath:      event.SourceFilePath,
		Status:        model.EmbeddingStatusUploading,
		Message:       "文件等待上报",
	}

	err := ep.embeddingStateRepo.CreateEmbeddingState(state)
	if err != nil {
		ep.logger.Error("failed to create embedding state for file %s: %v", event.SourceFilePath, err)
		return fmt.Errorf("failed to create embedding state: %w", err)
	}

	// 删除已处理的事件
	err = ep.eventRepo.DeleteEvent(event.ID)
	if err != nil {
		ep.logger.Error("failed to delete processed event %d: %v", event.ID, err)
		return fmt.Errorf("failed to delete processed event: %w", err)
	}

	return nil
}

// ProcessModifyFileEvent 处理修改文件事件
func (ep *eventProcessService) ProcessModifyFileEvent(event *model.Event) error {
	ep.logger.Info("processing modify file event: %s", event.SourceFilePath)

	// 检查是否已存在语义构建状态记录
	state, err := ep.embeddingStateRepo.GetEmbeddingStateByFile(event.WorkspacePath, event.SourceFilePath)
	if err != nil {
		// 不存在记录，创建新的
		state = &model.EmbeddingState{
			SyncID:        "", // 将在创建时生成
			WorkspacePath: event.WorkspacePath,
			FilePath:      event.SourceFilePath,
			Status:        model.EmbeddingStatusUploading,
			Message:       "文件修改，等待重新上报",
		}

		err = ep.embeddingStateRepo.CreateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to create embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to create embedding state: %w", err)
		}
	} else {
		// 已存在记录，更新状态为需要重新上报
		state.Status = model.EmbeddingStatusUploading
		state.Message = "文件修改，等待重新上报"

		err = ep.embeddingStateRepo.UpdateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to update embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to update embedding state: %w", err)
		}
	}

	// 删除已处理的事件
	err = ep.eventRepo.DeleteEvent(event.ID)
	if err != nil {
		ep.logger.Error("failed to delete processed event %d: %v", event.ID, err)
		return fmt.Errorf("failed to delete processed event: %w", err)
	}

	return nil
}

// ProcessDeleteFileEvent 处理删除文件事件
func (ep *eventProcessService) ProcessDeleteFileEvent(event *model.Event) error {
	ep.logger.Info("processing delete file event: %s", event.SourceFilePath)

	// 检查是否已存在语义构建状态记录
	state, err := ep.embeddingStateRepo.GetEmbeddingStateByFile(event.WorkspacePath, event.SourceFilePath)
	if err == nil {
		// 存在记录，删除它
		err = ep.embeddingStateRepo.DeleteEmbeddingState(state.SyncID)
		if err != nil {
			ep.logger.Error("failed to delete embedding state for deleted file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to delete embedding state: %w", err)
		}
	}

	// 删除已处理的事件
	err = ep.eventRepo.DeleteEvent(event.ID)
	if err != nil {
		ep.logger.Error("failed to delete processed event %d: %v", event.ID, err)
		return fmt.Errorf("failed to delete processed event: %w", err)
	}

	return nil
}

// ProcessEvents 处理事件记录
func (ep *eventProcessService) ProcessEvents() error {
	// 获取待处理的事件
	events, err := ep.eventRepo.GetEventsByType(model.EventTypeAddFile, 10, false)
	if err != nil {
		ep.logger.Error("failed to get add file events: %v", err)
		return fmt.Errorf("failed to get add file events: %w", err)
	}

	// 处理添加文件事件
	for _, event := range events {
		err = ep.ProcessAddFileEvent(event)
		if err != nil {
			ep.logger.Error("failed to process add file event: %v", err)
			continue
		}
	}

	// 获取修改文件事件
	modifyEvents, err := ep.eventRepo.GetEventsByType(model.EventTypeModifyFile, 10, false)
	if err != nil {
		ep.logger.Error("failed to get modify file events: %v", err)
		return fmt.Errorf("failed to get modify file events: %w", err)
	}

	// 处理修改文件事件
	for _, event := range modifyEvents {
		err = ep.ProcessModifyFileEvent(event)
		if err != nil {
			ep.logger.Error("failed to process modify file event: %v", err)
			continue
		}
	}

	// 获取删除文件事件
	deleteEvents, err := ep.eventRepo.GetEventsByType(model.EventTypeDeleteFile, 10, false)
	if err != nil {
		ep.logger.Error("failed to get delete file events: %v", err)
		return fmt.Errorf("failed to get delete file events: %w", err)
	}

	// 处理删除文件事件
	for _, event := range deleteEvents {
		err = ep.ProcessDeleteFileEvent(event)
		if err != nil {
			ep.logger.Error("failed to process delete file event: %v", err)
			continue
		}
	}

	// TODO: 添加控制逻辑，避免

	return nil
}
