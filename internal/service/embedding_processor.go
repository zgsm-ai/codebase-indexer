package service

import (
	"fmt"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/logger"
)

// EmbeddingProcessService 事件处理服务接口
type EmbeddingProcessService interface {
	ProcessActiveWorkspaces() ([]*model.Workspace, error)
	ProcessAddFileEvent(event *model.Event) error
	ProcessModifyFileEvent(event *model.Event) error
	ProcessDeleteFileEvent(event *model.Event) error
	ProcessRenameFileEvent(event *model.Event) error
	ProcessDeleteFileEvents(events []*model.Event) error
	ProcessRenameFileEvents(events []*model.Event) error
	ProcessEmbeddingEvents(workspacePaths []string) error
}

// embeddingProcessService 事件处理服务实现
type embeddingProcessService struct {
	workspaceRepo      repository.WorkspaceRepository
	eventRepo          repository.EventRepository
	embeddingStateRepo repository.EmbeddingStateRepository
	uploadService      UploadService
	logger             logger.Logger
}

// NewEmbeddingProcessService 创建事件处理服务
func NewEmbeddingProcessService(
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	embeddingStateRepo repository.EmbeddingStateRepository,
	uploadService UploadService,
	logger logger.Logger,
) EmbeddingProcessService {
	return &embeddingProcessService{
		workspaceRepo:      workspaceRepo,
		eventRepo:          eventRepo,
		embeddingStateRepo: embeddingStateRepo,
		uploadService:      uploadService,
		logger:             logger,
	}
}

// ProcessActiveWorkspaces 扫描活跃工作区
func (ep *embeddingProcessService) ProcessActiveWorkspaces() ([]*model.Workspace, error) {
	workspaces, err := ep.workspaceRepo.GetActiveWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get active workspaces: %w", err)
	}

	var activeWorkspaces []*model.Workspace
	for _, workspace := range workspaces {
		if workspace.Active {
			activeWorkspaces = append(activeWorkspaces, workspace)
		}
	}

	return activeWorkspaces, nil
}

// ProcessAddFileEvent 处理添加文件事件
func (ep *embeddingProcessService) ProcessAddFileEvent(event *model.Event) error {
	ep.logger.Info("processing add file event: %s", event.SourceFilePath)

	// 更新事件状态为上传中
	originalStatus := event.EmbeddingStatus
	event.EmbeddingStatus = model.EmbeddingStatusUploading
	err := ep.eventRepo.UpdateEvent(event)
	if err != nil {
		event.EmbeddingStatus = originalStatus
		ep.logger.Error("failed to update event status to uploading for add file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 调用上报逻辑进行上报
	syncId, err := ep.uploadService.UploadFileWithRetry(event.WorkspacePath, event.SourceFilePath, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		event.EmbeddingStatus = model.EmbeddingStatusUploadFailed
		updateErr := ep.eventRepo.UpdateEvent(event)
		if updateErr != nil {
			ep.logger.Error("failed to update event status to upload failed for add file event %d: %v", event.ID, updateErr)
		}
		ep.logger.Error("failed to upload modified file %s: %v", event.SourceFilePath, err)
		return fmt.Errorf("failed to upload modified file: %w", err)
	}

	event.EmbeddingStatus = model.EmbeddingStatusBuilding
	event.SyncId = syncId
	err = ep.eventRepo.UpdateEvent(event)
	if err != nil {
		ep.logger.Error("failed to update event status to building for add file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 上报成功，检查是否已存在语义构建状态记录
	state, err := ep.embeddingStateRepo.GetEmbeddingStateByFile(event.WorkspacePath, event.SourceFilePath)
	if err != nil {
		// 不存在记录，创建新的
		state = &model.EmbeddingState{
			SyncID:        syncId, // 将在创建时生成
			WorkspacePath: event.WorkspacePath,
			FilePath:      event.SourceFilePath,
			Status:        model.EmbeddingStatusBuilding,
			Message:       "文件修改，重新构建中",
		}

		err = ep.embeddingStateRepo.CreateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to create embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to create embedding state: %w", err)
		}
	} else {
		// 已存在记录，更新状态为构建中
		state.Status = model.EmbeddingStatusBuilding
		state.Message = "文件修改，重新构建中"

		err = ep.embeddingStateRepo.UpdateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to update embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to update embedding state: %w", err)
		}
	}

	return nil
}

// ProcessModifyFileEvent 处理修改文件事件
func (ep *embeddingProcessService) ProcessModifyFileEvent(event *model.Event) error {
	ep.logger.Info("processing modify file event: %s", event.SourceFilePath)

	// 更新事件状态为上传中
	originalStatus := event.EmbeddingStatus
	event.EmbeddingStatus = model.EmbeddingStatusUploading
	err := ep.eventRepo.UpdateEvent(event)
	if err != nil {
		event.EmbeddingStatus = originalStatus
		ep.logger.Error("failed to update event status to uploading for modify file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 调用上报逻辑进行上报
	syncId, err := ep.uploadService.UploadFileWithRetry(event.WorkspacePath, event.SourceFilePath, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		event.EmbeddingStatus = model.EmbeddingStatusUploadFailed
		updateErr := ep.eventRepo.UpdateEvent(event)
		if updateErr != nil {
			ep.logger.Error("failed to update event status to upload failed for modify file event %d: %v", event.ID, updateErr)
		}
		ep.logger.Error("failed to upload modified file %s: %v", event.SourceFilePath, err)
		return fmt.Errorf("failed to upload modified file: %w", err)
	}

	event.EmbeddingStatus = model.EmbeddingStatusBuilding
	event.SyncId = syncId
	err = ep.eventRepo.UpdateEvent(event)
	if err != nil {
		ep.logger.Error("failed to update event status to building for delete file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 上报成功，检查是否已存在语义构建状态记录
	state, err := ep.embeddingStateRepo.GetEmbeddingStateByFile(event.WorkspacePath, event.SourceFilePath)
	if err != nil {
		// 不存在记录，创建新的
		state = &model.EmbeddingState{
			SyncID:        syncId, // 同步请求ID
			WorkspacePath: event.WorkspacePath,
			FilePath:      event.SourceFilePath,
			Status:        model.EmbeddingStatusBuilding,
			Message:       "文件修改，重新构建中",
		}

		err = ep.embeddingStateRepo.CreateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to create embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to create embedding state: %w", err)
		}
	} else {
		// 已存在记录，更新状态为构建中
		state.Status = model.EmbeddingStatusBuilding
		state.Message = "文件修改，重新构建中"

		err = ep.embeddingStateRepo.UpdateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to update embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to update embedding state: %w", err)
		}
	}

	return nil
}

// ProcessDeleteFileEvent 处理删除文件事件
func (ep *embeddingProcessService) ProcessDeleteFileEvent(event *model.Event) error {
	ep.logger.Info("processing delete file event: %s", event.SourceFilePath)

	// 更新事件状态为构建中
	originalStatus := event.EmbeddingStatus
	event.EmbeddingStatus = model.EmbeddingStatusBuilding
	err := ep.eventRepo.UpdateEvent(event)
	if err != nil {
		// 恢复原始状态
		event.EmbeddingStatus = originalStatus
		ep.logger.Error("failed to update event status to building for delete file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 调用上报删除逻辑进行上报
	syncId, err := ep.uploadService.DeleteFileWithRetry(event.WorkspacePath, event.SourceFilePath, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		event.EmbeddingStatus = model.EmbeddingStatusUploadFailed
		updateErr := ep.eventRepo.UpdateEvent(event)
		if updateErr != nil {
			ep.logger.Error("failed to update event status to upload failed for delete file event %d: %v", event.ID, updateErr)
		}
		ep.logger.Error("failed to delete file %s: %v", event.SourceFilePath, err)
		return fmt.Errorf("failed to delete file: %w", err)
	}

	event.EmbeddingStatus = model.EmbeddingStatusBuilding
	event.SyncId = syncId
	err = ep.eventRepo.UpdateEvent(event)
	if err != nil {
		ep.logger.Error("failed to update event status to building for delete file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 检查是否已存在语义构建状态记录
	state, err := ep.embeddingStateRepo.GetEmbeddingStateByFile(event.WorkspacePath, event.SourceFilePath)
	if err != nil {
		// 不存在记录，创建新的
		state = &model.EmbeddingState{
			SyncID:        syncId,
			WorkspacePath: event.WorkspacePath,
			FilePath:      event.SourceFilePath,
			Status:        model.EmbeddingStatusBuilding,
			Message:       "文件删除，重新构建中",
		}

		err = ep.embeddingStateRepo.CreateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to create embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to create embedding state: %w", err)
		}
	} else {
		// 已存在记录，更新状态为构建中
		state.Status = model.EmbeddingStatusBuilding
		state.Message = "文件删除，重新构建中"

		err = ep.embeddingStateRepo.UpdateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to update embedding state for modified file %s: %v", event.SourceFilePath, err)
			return fmt.Errorf("failed to update embedding state: %w", err)
		}
	}

	return nil
}

// ProcessRenameFileEvent 处理重命名文件事件
func (ep *embeddingProcessService) ProcessRenameFileEvent(event *model.Event) error {
	ep.logger.Info("processing rename file event: %s -> %s", event.SourceFilePath, event.TargetFilePath)

	// 更新事件状态为上传中
	originalStatus := event.EmbeddingStatus
	event.EmbeddingStatus = model.EmbeddingStatusUploading
	err := ep.eventRepo.UpdateEvent(event)
	if err != nil {
		event.EmbeddingStatus = originalStatus
		ep.logger.Error("failed to update event status to uploading for rename file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 调用上报逻辑进行上报
	syncId, err := ep.uploadService.UploadFileWithRetry(event.WorkspacePath, event.TargetFilePath, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		event.EmbeddingStatus = model.EmbeddingStatusUploadFailed
		updateErr := ep.eventRepo.UpdateEvent(event)
		if updateErr != nil {
			ep.logger.Error("failed to update event status to upload failed for rename file event %d: %v", event.ID, updateErr)
		}
		ep.logger.Error("failed to upload renamed file %s: %v", event.TargetFilePath, err)
		return fmt.Errorf("failed to upload renamed file: %w", err)
	}

	event.EmbeddingStatus = model.EmbeddingStatusBuilding
	event.SyncId = syncId
	err = ep.eventRepo.UpdateEvent(event)
	if err != nil {
		ep.logger.Error("failed to update event status to building for rename file event %d: %v", event.ID, err)
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// 上报成功，检查是否已存在语义构建状态记录
	state, err := ep.embeddingStateRepo.GetEmbeddingStateByFile(event.WorkspacePath, event.SourceFilePath)
	if err != nil {
		// 不存在记录，创建新的
		state = &model.EmbeddingState{
			SyncID:        syncId, // 将在创建时生成
			WorkspacePath: event.WorkspacePath,
			FilePath:      event.TargetFilePath,
			Status:        model.EmbeddingStatusBuilding,
			Message:       "文件重命名，重新构建中",
		}

		err = ep.embeddingStateRepo.CreateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to create embedding state for renamed file %s: %v", event.TargetFilePath, err)
			return fmt.Errorf("failed to create embedding state: %w", err)
		}
	} else {
		// 已存在记录，更新状态为构建中
		state.Status = model.EmbeddingStatusBuilding
		state.Message = "文件重命名，等待构建"
		err = ep.embeddingStateRepo.UpdateEmbeddingState(state)
		if err != nil {
			ep.logger.Error("failed to update embedding state for renamed file %s: %v", event.TargetFilePath, err)
			return fmt.Errorf("failed to update embedding state: %w", err)
		}
	}

	return nil
}

// ProcessDeleteFileEvents 批量处理删除文件事件
func (ep *embeddingProcessService) ProcessDeleteFileEvents(events []*model.Event) error {
	ep.logger.Info("processing %d delete file events", len(events))

	for _, event := range events {
		err := ep.ProcessDeleteFileEvent(event)
		if err != nil {
			ep.logger.Error("failed to process delete file event for file %s: %v", event.SourceFilePath, err)
			// 继续处理其他事件，不因单个事件失败而中断整个批量处理
			continue
		}
	}

	return nil
}

// ProcessRenameFileEvents 批量处理重命名文件事件
func (ep *embeddingProcessService) ProcessRenameFileEvents(events []*model.Event) error {
	ep.logger.Info("processing %d rename file events", len(events))

	for _, event := range events {
		err := ep.ProcessRenameFileEvent(event)
		if err != nil {
			ep.logger.Error("failed to process rename file event for file %s -> %s: %v", event.SourceFilePath, event.TargetFilePath, err)
			// 继续处理其他事件，不因单个事件失败而中断整个批量处理
			continue
		}
	}

	return nil
}

// ProcessEmbeddingEvents 处理事件记录
func (ep *embeddingProcessService) ProcessEmbeddingEvents(workspacePaths []string) error {
	// 定义需要处理的事件状态：初始化、上报失败、构建失败
	targetStatuses := []int{
		model.EmbeddingStatusInit,
		model.EmbeddingStatusUploadFailed,
		model.EmbeddingStatusBuildFailed,
	}

	// 获取待处理的添加文件事件
	events, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(model.EventTypeAddFile, workspacePaths, 10, false, targetStatuses)
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
	modifyEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(model.EventTypeModifyFile, workspacePaths, 10, false, targetStatuses)
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

	// 获取重命名文件事件
	renameEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(model.EventTypeRenameFile, workspacePaths, 10, false, targetStatuses)
	if err != nil {
		ep.logger.Error("failed to get rename file events: %v", err)
		return fmt.Errorf("failed to get rename file events: %w", err)
	}

	// 处理重命名文件事件
	for _, event := range renameEvents {
		err = ep.ProcessRenameFileEvent(event)
		if err != nil {
			ep.logger.Error("failed to process rename file event: %v", err)
			continue
		}
	}

	// 获取删除文件事件
	deleteEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(model.EventTypeDeleteFile, workspacePaths, 10, false, targetStatuses)
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

	return nil
}
