package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// EmbeddingProcessService 事件处理服务接口
type EmbeddingProcessService interface {
	ProcessActiveWorkspaces() ([]*model.Workspace, error)
	ProcessAddFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error)
	ProcessModifyFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error)
	ProcessDeleteFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error)
	ProcessRenameFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error)
	ProcessDeleteFileEvents(ctx context.Context, events []*model.Event) error
	ProcessRenameFileEvents(ctx context.Context, events []*model.Event) error
	ProcessEmbeddingEvents(ctx context.Context, workspacePaths []string) error
	CleanWorkspaceFilePath(ctx context.Context, fileStatus *utils.FileStatus, event *model.Event) error
	CleanWorkspaceFilePaths(ctx context.Context, events []*model.Event) error
}

// embeddingProcessService 事件处理服务实现
type embeddingProcessService struct {
	workspaceRepo         repository.WorkspaceRepository
	eventRepo             repository.EventRepository
	codebaseEmbeddingRepo repository.EmbeddingFileRepository
	uploadService         UploadService
	logger                logger.Logger
}

// NewEmbeddingProcessService 创建事件处理服务
func NewEmbeddingProcessService(
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	codebaseEmbeddingRepo repository.EmbeddingFileRepository,
	uploadService UploadService,
	logger logger.Logger,
) EmbeddingProcessService {
	return &embeddingProcessService{
		workspaceRepo:         workspaceRepo,
		eventRepo:             eventRepo,
		codebaseEmbeddingRepo: codebaseEmbeddingRepo,
		uploadService:         uploadService,
		logger:                logger,
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
		if workspace.Active == "true" {
			activeWorkspaces = append(activeWorkspaces, workspace)
		}
	}

	return activeWorkspaces, nil
}

// ProcessAddFileEvent 处理添加文件事件
func (ep *embeddingProcessService) ProcessAddFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error) {
	ep.logger.Info("processing add file event: %s", event.SourceFilePath)

	// 更新事件状态为上传中
	updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploading}
	err := ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to uploading: %w", err)
	}

	// 调用上报逻辑进行上报
	fileStatus, err := ep.uploadService.UploadFileWithRetry(event.WorkspacePath, event.SourceFilePath, utils.FILE_STATUS_ADDED, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploadFailed}
		updateErr := ep.eventRepo.UpdateEvent(&updateEvent)
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update event status to uploadFailed: %w", updateErr)
		}
		return nil, fmt.Errorf("failed to upload add file %s: %w", event.SourceFilePath, err)
	}

	updateEvent = model.Event{
		ID:              event.ID,
		EmbeddingStatus: model.EmbeddingStatusBuilding,
		SyncId:          fileStatus.RequestId,
		FileHash:        fileStatus.Hash,
	}
	err = ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to building: %w", err)
	}

	return fileStatus, nil
}

// ProcessModifyFileEvent 处理修改文件事件
func (ep *embeddingProcessService) ProcessModifyFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error) {
	ep.logger.Info("processing modify file event: %s", event.SourceFilePath)

	// 更新事件状态为上传中
	updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploading}
	err := ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to uploading: %w", err)
	}

	// 调用上报逻辑进行上报
	fileStatus, err := ep.uploadService.UploadFileWithRetry(event.WorkspacePath, event.SourceFilePath, utils.FILE_STATUS_MODIFIED, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploadFailed}
		updateErr := ep.eventRepo.UpdateEvent(&updateEvent)
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update event status to upload failed: %w", updateErr)
		}
		return nil, fmt.Errorf("failed to upload modified file %s: %w", event.SourceFilePath, err)
	}

	updateEvent = model.Event{
		ID:              event.ID,
		EmbeddingStatus: model.EmbeddingStatusBuilding,
		SyncId:          fileStatus.RequestId,
		FileHash:        fileStatus.Hash,
	}
	err = ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to building: %w", err)
	}

	return fileStatus, nil
}

// ProcessDeleteFileEvent 处理删除文件事件
func (ep *embeddingProcessService) ProcessDeleteFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error) {
	ep.logger.Info("processing delete file event: %s", event.SourceFilePath)

	// 更新事件状态为构建中
	updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusBuilding}
	err := ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to building: %w", err)
	}

	// 调用上报删除逻辑进行上报
	fileStatus, err := ep.uploadService.DeleteFileWithRetry(event.WorkspacePath, event.SourceFilePath, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploadFailed}
		updateErr := ep.eventRepo.UpdateEvent(&updateEvent)
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update event status to upload failed: %w", updateErr)
		}
		return nil, fmt.Errorf("failed to upload delete file %s: %w", event.SourceFilePath, err)
	}

	updateEvent = model.Event{
		ID:              event.ID,
		EmbeddingStatus: model.EmbeddingStatusBuilding,
		SyncId:          fileStatus.RequestId,
		FileHash:        fileStatus.Hash,
	}
	err = ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to building: %w", err)
	}

	return fileStatus, nil
}

// ProcessRenameFileEvent 处理重命名文件事件
func (ep *embeddingProcessService) ProcessRenameFileEvent(ctx context.Context, event *model.Event) (*utils.FileStatus, error) {
	ep.logger.Info("processing rename file event: %s -> %s", event.SourceFilePath, event.TargetFilePath)

	// 更新事件状态为上传中
	updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploading}
	err := ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to uploading: %w", err)
	}

	// 调用上报逻辑进行上报
	fileStatus, err := ep.uploadService.RenameFileWithRetry(event.WorkspacePath, event.SourceFilePath, event.TargetFilePath, 3)
	if err != nil {
		// 上报失败，更新事件状态为上报失败
		updateEvent := model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploadFailed}
		updateErr := ep.eventRepo.UpdateEvent(&updateEvent)
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update event status to upload failed: %w", updateErr)
		}
		return nil, fmt.Errorf("failed to upload renamed file %s->%s: %w", event.SourceFilePath, event.TargetFilePath, err)
	}

	updateEvent = model.Event{
		ID:              event.ID,
		EmbeddingStatus: model.EmbeddingStatusBuilding,
		SyncId:          fileStatus.RequestId,
		FileHash:        fileStatus.Hash,
	}
	err = ep.eventRepo.UpdateEvent(&updateEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status to building: %w", err)
	}

	return fileStatus, nil
}

// ProcessDeleteFileEvents 批量处理删除文件事件
func (ep *embeddingProcessService) ProcessDeleteFileEvents(ctx context.Context, events []*model.Event) error {
	ep.logger.Info("processing %d delete file events", len(events))

	for _, event := range events {
		_, err := ep.ProcessDeleteFileEvent(ctx, event)
		if err != nil {
			ep.logger.Error("failed to process delete file event for file %s: %v", event.SourceFilePath, err)
			// 继续处理其他事件，不因单个事件失败而中断整个批量处理
			continue
		}
	}

	return nil
}

// ProcessRenameFileEvents 批量处理重命名文件事件
func (ep *embeddingProcessService) ProcessRenameFileEvents(ctx context.Context, events []*model.Event) error {
	ep.logger.Info("processing %d rename file events", len(events))

	for _, event := range events {
		_, err := ep.ProcessRenameFileEvent(ctx, event)
		if err != nil {
			ep.logger.Error("failed to process rename file event for file %s -> %s: %v", event.SourceFilePath, event.TargetFilePath, err)
			// 继续处理其他事件，不因单个事件失败而中断整个批量处理
			continue
		}
	}

	return nil
}

// ProcessEmbeddingEvents 处理事件记录
func (ep *embeddingProcessService) ProcessEmbeddingEvents(ctx context.Context, workspacePaths []string) error {
	// 定义需要处理的事件状态：初始化、上报失败、构建失败
	targetStatuses := []int{
		model.EmbeddingStatusInit,
		model.EmbeddingStatusUploadFailed,
		model.EmbeddingStatusBuildFailed,
	}

	// 获取待处理的添加文件事件
	addEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeAddFile}, workspacePaths, 20, false, targetStatuses, nil)
	if err != nil {
		return fmt.Errorf("failed to get add file events: %w", err)
	}

	// 处理添加文件事件
	for _, event := range addEvents {
		fileStatus, err := ep.ProcessAddFileEvent(ctx, event)
		if err != nil {
			ep.logger.Error("failed to process add file event: %v", err)
			continue
		}
		err = ep.CleanWorkspaceFilePath(ctx, fileStatus, event)
		if err != nil {
			ep.logger.Error("failed to clean workspace filepath %s: %v", event.SourceFilePath, err)
			continue
		}
	}

	// 获取修改文件事件
	modifyEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeModifyFile}, workspacePaths, 20, false, targetStatuses, nil)
	if err != nil {
		return fmt.Errorf("failed to get modify file events: %w", err)
	}

	// 处理修改文件事件
	for _, event := range modifyEvents {
		fileStatus, err := ep.ProcessModifyFileEvent(ctx, event)
		if err != nil {
			ep.logger.Error("failed to process modify file event: %v", err)
			continue
		}
		err = ep.CleanWorkspaceFilePath(ctx, fileStatus, event)
		if err != nil {
			ep.logger.Error("failed to clean workspace filepath %s: %v", event.SourceFilePath, err)
			continue
		}
	}

	// 获取重命名文件事件
	renameEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeRenameFile}, workspacePaths, 20, false, targetStatuses, nil)
	if err != nil {
		return fmt.Errorf("failed to get rename file events: %w", err)
	}

	// 处理重命名文件事件
	for _, event := range renameEvents {
		fileStatus, err := ep.ProcessRenameFileEvent(ctx, event)
		if err != nil {
			ep.logger.Error("failed to process rename file event: %v", err)
			continue
		}
		err = ep.CleanWorkspaceFilePath(ctx, fileStatus, event)
		if err != nil {
			ep.logger.Error("failed to clean workspace filepath %s: %v", event.SourceFilePath, err)
			continue
		}
	}

	// 获取删除文件事件
	deleteEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeDeleteFile}, workspacePaths, 20, false, targetStatuses, nil)
	if err != nil {
		return fmt.Errorf("failed to get delete file events: %w", err)
	}

	// 处理删除文件事件
	for _, event := range deleteEvents {
		fileStatus, err := ep.ProcessDeleteFileEvent(ctx, event)
		if err != nil {
			ep.logger.Error("failed to process delete file event: %v", err)
			continue
		}
		err = ep.CleanWorkspaceFilePath(ctx, fileStatus, event)
		if err != nil {
			ep.logger.Error("failed to clean workspace filepath %s: %v", event.SourceFilePath, err)
			continue
		}
	}

	return nil
}

// CleanWorkspaceFilePath 删除 workspace 中指定文件的 filepath 记录
func (ep *embeddingProcessService) CleanWorkspaceFilePath(ctx context.Context, fileStatus *utils.FileStatus, event *model.Event) error {
	ep.logger.Info("cleaning workspace filepath for event: %s, workspace: %s", event.SourceFilePath, event.WorkspacePath)

	// 获取 codebase embedding 配置
	codebaseEmbeddingId := utils.GenerateCodebaseEmbeddingID(event.WorkspacePath)
	codebaseEmbeddingConfig, err := ep.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseEmbeddingId)
	if err != nil {
		return fmt.Errorf("failed to get codebase embedding config for workspace %s: %w", event.WorkspacePath, err)
	}

	// 根据事件类型处理不同的文件路径
	var filePaths []string
	switch event.EventType {
	case model.EventTypeAddFile, model.EventTypeModifyFile, model.EventTypeDeleteFile:
		filePaths = []string{event.SourceFilePath}
	case model.EventTypeRenameFile:
		filePaths = []string{event.SourceFilePath, event.TargetFilePath}
	default:
		ep.logger.Warn("unsupported event type for cleaning filepath: %d", event.EventType)
		return nil
	}

	// 从 HashTree 中删除对应的文件路径记录
	// TODO: 判断是否为目录，是则删除目录下所有文件的记录
	updated := false
	if codebaseEmbeddingConfig.HashTree != nil {
		for _, filePath := range filePaths {
			if _, exists := codebaseEmbeddingConfig.HashTree[filePath]; exists {
				delete(codebaseEmbeddingConfig.HashTree, filePath)
				updated = true
				ep.logger.Debug("removed filepath from hash tree: %s", filePath)
			}
		}
	} else {
		codebaseEmbeddingConfig.HashTree = make(map[string]string)
	}

	// 删除FailedFiles中对应的文件路径
	if codebaseEmbeddingConfig.FailedFiles != nil {
		for _, filePath := range filePaths {
			if _, exists := codebaseEmbeddingConfig.FailedFiles[filePath]; exists {
				delete(codebaseEmbeddingConfig.FailedFiles, filePath)
				updated = true
				ep.logger.Debug("removed filepath from failed files: %s", filePath)
			}
		}
	} else {
		codebaseEmbeddingConfig.FailedFiles = make(map[string]string)
	}

	// 从SyncFiles中添加对应的文件路径
	if codebaseEmbeddingConfig.SyncFiles != nil {
		for _, filePath := range filePaths {
			if oldHash, exists := codebaseEmbeddingConfig.SyncFiles[filePath]; !exists || oldHash != fileStatus.Hash {
				codebaseEmbeddingConfig.SyncFiles[filePath] = fileStatus.Hash
				updated = true
			}
		}
	} else {
		codebaseEmbeddingConfig.SyncFiles = make(map[string]string)
		for _, filePath := range filePaths {
			codebaseEmbeddingConfig.SyncFiles[filePath] = fileStatus.Hash
		}
		updated = true
	}

	// // 添加syncId到SyncIds中
	// if codebaseEmbeddingConfig.SyncIds != nil {
	// 	codebaseEmbeddingConfig.SyncIds[fileStatus.RequestId] = time.Now()
	// 	updated = true
	// } else {
	// 	codebaseEmbeddingConfig.SyncIds = make(map[string]time.Time)
	// 	codebaseEmbeddingConfig.SyncIds[fileStatus.RequestId] = time.Now()
	// 	updated = true
	// }

	// 如果有更新，保存配置
	if updated {
		if err := ep.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig); err != nil {
			ep.logger.Error("failed to save codebase embedding config after cleaning filepath: %v", err)
			return fmt.Errorf("failed to save codebase embedding config: %w", err)
		}

		embeddingFileNum := len(codebaseEmbeddingConfig.HashTree)
		var embeddingFailedFilePaths string
		var embeddingMessage string
		embeddingFaildFiles := codebaseEmbeddingConfig.FailedFiles
		failedKeys := make([]string, 0, len(embeddingFaildFiles))
		for k, v := range embeddingFaildFiles {
			failedKeys = append(failedKeys, k)
			embeddingMessage = v
			if len(failedKeys) > 5 {
				break
			}
		}
		if len(failedKeys) == 0 {
			embeddingFailedFilePaths = ""
			embeddingMessage = ""
		} else if len(failedKeys) > 5 {
			embeddingFailedFilePaths = strings.Join(failedKeys[:5], ",")
		} else {
			embeddingFailedFilePaths = strings.Join(failedKeys, ",")
		}

		err = ep.workspaceRepo.UpdateEmbeddingInfo(event.WorkspacePath, embeddingFileNum, time.Now().Unix(), embeddingFailedFilePaths, embeddingMessage)
		if err != nil {
			ep.logger.Error("failed to update workspace file num: %v", err)
			return fmt.Errorf("failed to update workspace file num: %w", err)
		}
		ep.logger.Info("workspace filepath cleaned successfully for event: %s", event.SourceFilePath)
	} else {
		ep.logger.Debug("no filepath records found to clean for event: %s", event.SourceFilePath)
	}

	return nil
}

// CleanWorkspaceFilePaths 批量删除 workspace 中指定文件的 filepath 记录
func (ep *embeddingProcessService) CleanWorkspaceFilePaths(ctx context.Context, events []*model.Event) error {
	ep.logger.Info("cleaning workspace filepath for %d events", len(events))

	// 统计成功和失败的数量
	successCount := 0
	failureCount := 0
	codebaseEmbeddingFilePaths := make(map[string][]string)

	for _, event := range events {
		// 根据事件类型处理不同的文件路径
		codebaseEmbeddingId := utils.GenerateCodebaseEmbeddingID(event.WorkspacePath)
		switch event.EventType {
		case model.EventTypeAddFile, model.EventTypeModifyFile, model.EventTypeDeleteFile:
			if codebaseEmbeddingFilePaths[codebaseEmbeddingId] == nil {
				codebaseEmbeddingFilePaths[codebaseEmbeddingId] = []string{event.SourceFilePath}
			} else {
				codebaseEmbeddingFilePaths[codebaseEmbeddingId] = append(codebaseEmbeddingFilePaths[codebaseEmbeddingId], event.SourceFilePath)
			}
		case model.EventTypeRenameFile:
			if codebaseEmbeddingFilePaths[codebaseEmbeddingId] == nil {
				codebaseEmbeddingFilePaths[codebaseEmbeddingId] = []string{event.SourceFilePath, event.TargetFilePath}
			} else {
				codebaseEmbeddingFilePaths[codebaseEmbeddingId] = append(codebaseEmbeddingFilePaths[codebaseEmbeddingId], event.TargetFilePath)
			}
		default:
			continue
		}
	}

	codebaseEmbeddingConfigs := ep.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfigs()

	// 从 HashTree 中删除对应的文件路径记录
	for codebaseEmbeddingId, filePaths := range codebaseEmbeddingFilePaths {
		codebaseEmbeddingConfig := codebaseEmbeddingConfigs[codebaseEmbeddingId]
		if codebaseEmbeddingConfig == nil {
			ep.logger.Error("codebase config not found for codebaseEmbeddingId: %s", codebaseEmbeddingId)
			continue
		}
		updated := false
		for _, filePath := range filePaths {
			if _, exists := codebaseEmbeddingConfig.HashTree[filePath]; exists {
				delete(codebaseEmbeddingConfig.HashTree, filePath)
				updated = true
				ep.logger.Debug("removed filepath from hash tree: %s", filePath)
			}
		}

		// 如果有更新，保存配置
		if updated {
			if err := ep.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig); err != nil {
				ep.logger.Error("failed to save codebase embedding config after cleaning filepath: %v", err)
				failureCount++
				continue
			}
			embeddingFileNum := len(codebaseEmbeddingConfig.HashTree)
			updateWorkspace := model.Workspace{
				WorkspacePath:    codebaseEmbeddingConfig.CodebasePath,
				EmbeddingFileNum: embeddingFileNum,
			}
			err := ep.workspaceRepo.UpdateWorkspace(&updateWorkspace)
			if err != nil {
				ep.logger.Error("failed to update workspace: %v", err)
				failureCount++
				continue
			}
			ep.logger.Info("workspace filepath cleaned successfully for codebaseEmbeddingId: %s", codebaseEmbeddingId)
			successCount++
		} else {
			ep.logger.Debug("no filepath records found to clean for codebaseEmbeddingId: %s", codebaseEmbeddingId)
		}
	}

	ep.logger.Info("workspace filepath cleaning completed: %d succeeded, %d failed", successCount, failureCount)

	// 如果全部失败，返回错误
	if failureCount > 0 && successCount == 0 {
		return fmt.Errorf("all %d events failed to clean workspace filepath", failureCount)
	}

	return nil
}
