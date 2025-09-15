package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/errs"
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
	ProcessEmbeddingEvents(ctx context.Context, workspacePaths []string) error
	CleanWorkspaceFilePath(ctx context.Context, fileStatus *utils.FileStatus, event *model.Event) error
	CleanWorkspaceFilePaths(ctx context.Context, workspacePath string, events []*model.Event) error
}

// embeddingProcessService 事件处理服务实现
type embeddingProcessService struct {
	workspaceRepo repository.WorkspaceRepository
	eventRepo     repository.EventRepository
	embeddingRepo repository.EmbeddingFileRepository
	uploadService UploadService
	syncer        repository.SyncInterface
	logger        logger.Logger
}

// NewEmbeddingProcessService 创建事件处理服务
func NewEmbeddingProcessService(
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	embeddingRepo repository.EmbeddingFileRepository,
	uploadService UploadService,
	syncer repository.SyncInterface,
	logger logger.Logger,
) EmbeddingProcessService {
	return &embeddingProcessService{
		workspaceRepo: workspaceRepo,
		eventRepo:     eventRepo,
		embeddingRepo: embeddingRepo,
		uploadService: uploadService,
		syncer:        syncer,
		logger:        logger,
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
		ep.uploadFilePathFailed(event, err)
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
		ep.uploadFilePathFailed(event, err)
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
		ep.uploadFilePathFailed(event, err)
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
		ep.uploadFilePathFailed(event, err)
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

// ProcessEmbeddingEvents 处理事件记录
func (ep *embeddingProcessService) ProcessEmbeddingEvents(ctx context.Context, workspacePaths []string) error {
	// 遍历每个工作区，逐个处理
	for _, workspacePath := range workspacePaths {
		err := ep.processWorkspaceEvents(ctx, workspacePath)
		if err != nil {
			ep.logger.Error("failed to process events for workspace %s: %v", workspacePath, err)
			// 继续处理其他工作区，不因单个工作区失败而中断整个处理
			continue
		}
	}
	return nil
}

// processWorkspaceEvents 处理指定工作区的事件
func (ep *embeddingProcessService) processWorkspaceEvents(ctx context.Context, workspacePath string) error {
	// 定义需要处理的事件状态：初始化、上报失败、构建失败
	targetStatuses := []int{
		model.EmbeddingStatusInit,
		model.EmbeddingStatusUploadFailed,
		model.EmbeddingStatusBuildFailed,
	}

	// 获取待处理的添加和修改文件事件（合并处理）
	addModifyEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeAddFile, model.EventTypeModifyFile}, []string{workspacePath}, 100, false, targetStatuses, nil)
	if err != nil {
		return fmt.Errorf("failed to get add/modify file events: %w", err)
	}

	// 获取待处理的重命名和删除文件事件（合并处理）
	renameDeleteEvents, err := ep.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeRenameFile, model.EventTypeDeleteFile}, []string{workspacePath}, 100, false, targetStatuses, nil)
	if err != nil {
		return fmt.Errorf("failed to get rename/delete file events: %w", err)
	}

	if len(addModifyEvents) == 0 && len(renameDeleteEvents) == 0 {
		ep.logger.Debug("no events to process for workspace: %s", workspacePath)
		return nil
	}

	// 获取上传令牌
	workspaceName := filepath.Base(workspacePath)
	authInfo := config.GetAuthInfo()
	tokenReq := dto.UploadTokenReq{
		ClientId:     authInfo.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: workspaceName,
	}

	// 需要通过 uploadService 获取 syncer 来获取上传令牌
	uploadTokenResp, err := ep.syncer.FetchUploadToken(tokenReq)
	if err != nil {
		return fmt.Errorf("failed to get upload token for workspace %s: %w", workspacePath, err)
	}
	uploadToken := uploadTokenResp.Data.Token

	config := config.GetClientConfig()
	maxFileSizeKB := config.Sync.MaxFileSizeKB

	// 批量处理添加和修改事件
	if len(addModifyEvents) > 0 {
		err := ep.processBatchAddModifyEvents(ctx, workspacePath, addModifyEvents, uploadToken, maxFileSizeKB)
		if err != nil {
			ep.logger.Error("failed to process batch add/modify events for workspace %s: %v", workspacePath, err)
		}
	}

	// 批量处理重命名和删除事件
	if len(renameDeleteEvents) > 0 {
		err := ep.processBatchRenameDeleteEvents(ctx, workspacePath, renameDeleteEvents, uploadToken, maxFileSizeKB)
		if err != nil {
			ep.logger.Error("failed to process batch rename/delete events for workspace %s: %v", workspacePath, err)
		}
	}

	return nil
}

// processBatchEvents 批量处理添加和修改事件
func (ep *embeddingProcessService) processBatchAddModifyEvents(ctx context.Context, workspacePath string, events []*model.Event, uploadToken string, maxFileSizeKB int) error {
	ep.logger.Info("processing %d add/modify events for workspace: %s", len(events), workspacePath)

	// 分批处理，每批10个事件
	batchSize := 10
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}

		batch := events[i:end]
		err := ep.processBatchAddModify(ctx, workspacePath, batch, uploadToken, maxFileSizeKB)
		if err != nil {
			ep.logger.Error("failed to process batch add/modify events [%d:%d]: %v", i, end, err)
			// 继续处理下一批，不因单批失败而中断整个处理
			continue
		}
	}

	return nil
}

// processBatchAddModify 批量处理添加和修改事件
func (ep *embeddingProcessService) processBatchAddModify(ctx context.Context, workspacePath string, events []*model.Event, uploadToken string, maxFileSizeKB int) error {
	if len(events) == 0 {
		return nil
	}

	// 1. 批量更新事件状态为上传中
	eventIDs := make([]int64, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}
	err := ep.eventRepo.UpdateEventsEmbeddingStatus(eventIDs, model.EmbeddingStatusUploading)
	if err != nil {
		return fmt.Errorf("failed to update events status to uploading: %w", err)
	}

	// 2. 计算changes
	changes := make([]*utils.FileStatus, len(events))
	for i, event := range events {
		status := utils.FILE_STATUS_ADDED
		if event.EventType == model.EventTypeModifyFile {
			status = utils.FILE_STATUS_MODIFIED
		}

		// 生成文件哈希
		fileHash, err := buildFileHash(workspacePath, event.SourceFilePath, maxFileSizeKB)
		if err != nil {
			ep.logger.Error("failed to build file hash: %v", err)
			continue
		}

		changes[i] = &utils.FileStatus{
			Path:       event.SourceFilePath,
			TargetPath: event.TargetFilePath,
			Hash:       fileHash,
			Status:     status,
		}
	}

	// 3. 使用UploadChangesWithRetry批量上报，传入uploadToken
	fileStatuses, err := ep.uploadService.UploadChangesWithRetryWithToken(workspacePath, changes, 1, uploadToken)
	if err != nil {
		// 上报失败，批量更新事件状态为上报失败
		updateErr := ep.eventRepo.UpdateEventsEmbeddingStatus(eventIDs, model.EmbeddingStatusUploadFailed)
		if updateErr != nil {
			ep.logger.Error("failed to update events status to uploadFailed: %v", updateErr)
		}

		// 批量处理失败的文件路径
		ep.uploadFilePathsFailed(workspacePath, events, err)

		return fmt.Errorf("failed to upload batch add/modify files: %w", err)
	}

	// 4. 批量更新事件状态为构建中
	updateEvents := make([]*model.Event, len(events))
	for i, event := range events {
		updateEvents[i] = &model.Event{
			ID:              event.ID,
			EmbeddingStatus: model.EmbeddingStatusBuilding,
			SyncId:          fileStatuses[i].RequestId,
			FileHash:        fileStatuses[i].Hash,
		}
		events[i].FileHash = fileStatuses[i].Hash
	}
	err = ep.eventRepo.UpdateEventsEmbedding(updateEvents)
	if err != nil {
		return fmt.Errorf("failed to update events status to building: %w", err)
	}

	// 5. 批量清理工作区文件路径
	err = ep.CleanWorkspaceFilePaths(ctx, workspacePath, events)
	if err != nil {
		ep.logger.Error("failed to clean workspace filepaths: %v", err)
		// 继续处理，不因清理失败而中断整个批量处理
	}

	ep.logger.Info("successfully processed batch of %d add/modify events", len(events))
	return nil
}

// processBatchRenameDeleteEvents 批量处理重命名和删除事件
func (ep *embeddingProcessService) processBatchRenameDeleteEvents(ctx context.Context, workspacePath string, events []*model.Event, uploadToken string, maxFileSizeKB int) error {
	ep.logger.Info("processing %d rename/delete events for workspace: %s", len(events), workspacePath)

	// rename和delete事件不需要实际上传文件，直接处理所有事件
	err := ep.processBatchRenameDelete(ctx, workspacePath, events, uploadToken, maxFileSizeKB)
	if err != nil {
		return err
	}

	return nil
}

// processBatchRenameDelete 批量处理重命名和删除事件
func (ep *embeddingProcessService) processBatchRenameDelete(ctx context.Context, workspacePath string, events []*model.Event, uploadToken string, maxFileSizeKB int) error {
	if len(events) == 0 {
		return nil
	}

	// 1. 批量更新事件状态为上传中
	eventIDs := make([]int64, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}
	err := ep.eventRepo.UpdateEventsEmbeddingStatus(eventIDs, model.EmbeddingStatusUploading)
	if err != nil {
		return fmt.Errorf("failed to update events status to uploading: %w", err)
	}

	// 2. 计算changes
	changes := make([]*utils.FileStatus, len(events))
	for i, event := range events {
		if event.EventType == model.EventTypeRenameFile {
			fileHash, err := buildFileHash(workspacePath, event.TargetFilePath, maxFileSizeKB)
			if err != nil {
				ep.logger.Error("failed to build file hash: %v", err)
				continue
			}
			changes[i] = &utils.FileStatus{
				Path:       event.SourceFilePath,
				TargetPath: event.TargetFilePath,
				Hash:       fileHash,
				Status:     utils.FILE_STATUS_RENAME,
			}
		} else if event.EventType == model.EventTypeDeleteFile {
			changes[i] = &utils.FileStatus{
				Path:       event.SourceFilePath,
				TargetPath: event.TargetFilePath,
				Status:     utils.FILE_STATUS_DELETED,
			}
		}
	}

	// 3. 使用UploadChangesWithRetry批量上报，传入uploadToken
	fileStatuses, err := ep.uploadService.UploadChangesWithRetryWithToken(workspacePath, changes, 1, uploadToken)
	if err != nil {
		// 上报失败，批量更新事件状态为上报失败
		updateErr := ep.eventRepo.UpdateEventsEmbeddingStatus(eventIDs, model.EmbeddingStatusUploadFailed)
		if updateErr != nil {
			ep.logger.Error("failed to update events status to uploadFailed: %v", updateErr)
		}

		// 批量处理失败的文件路径
		ep.uploadFilePathsFailed(workspacePath, events, err)

		return fmt.Errorf("failed to upload batch rename/delete files: %w", err)
	}

	// 4. 批量更新事件状态为构建中
	updateEvents := make([]*model.Event, len(events))
	for i, event := range events {
		updateEvents[i] = &model.Event{
			ID:              event.ID,
			EmbeddingStatus: model.EmbeddingStatusBuilding,
			SyncId:          fileStatuses[i].RequestId,
			FileHash:        fileStatuses[i].Hash,
		}
		events[i].FileHash = fileStatuses[i].Hash
	}
	err = ep.eventRepo.UpdateEventsEmbedding(updateEvents)
	if err != nil {
		return fmt.Errorf("failed to update events status to building: %w", err)
	}

	// 5. 批量清理工作区文件路径
	err = ep.CleanWorkspaceFilePaths(ctx, workspacePath, events)
	if err != nil {
		ep.logger.Error("failed to clean workspace filepaths: %v", err)
		// 继续处理，不因清理失败而中断整个批量处理
	}

	ep.logger.Info("successfully processed batch of %d rename/delete events", len(events))
	return nil
}

func buildFileHash(workspacePath string, filePath string, maxFileSizeKB int) (string, error) {
	// 1. 验证文件路径
	fullPath := filepath.Join(workspacePath, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", fullPath)
	}

	// 2. 检查文件大小
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileSizeKB := float64(fileInfo.Size()) / 1024
	if fileSizeKB > float64(maxFileSizeKB) {
		return "", fmt.Errorf("file size %.2fKB exceeds limit %dKB", fileSizeKB, maxFileSizeKB)
	}

	fileTimestamp := fileInfo.ModTime().UnixMilli()

	return fmt.Sprintf("%d", fileTimestamp), nil
}

func (ep *embeddingProcessService) uploadFilePathFailed(event *model.Event, uploadErr error) error {
	filePath := event.SourceFilePath
	if event.EventType == model.EventTypeRenameFile {
		filePath = event.TargetFilePath
	}
	embeddingId := utils.GenerateEmbeddingID(event.WorkspacePath)
	embeddingConfig, err := ep.embeddingRepo.GetEmbeddingConfig(embeddingId)
	if err != nil {
		return fmt.Errorf("failed to get embedding config for workspace %s: %w", event.WorkspacePath, err)
	}
	if embeddingConfig.HashTree == nil {
		embeddingConfig.HashTree = make(map[string]string)
	}
	if embeddingConfig.FailedFiles == nil {
		embeddingConfig.FailedFiles = make(map[string]string)
	}
	if embeddingConfig.SyncFiles == nil {
		embeddingConfig.SyncFiles = make(map[string]string)
	}
	delete(embeddingConfig.HashTree, filePath)
	delete(embeddingConfig.SyncFiles, filePath)
	if event.EventType == model.EventTypeRenameFile {
		delete(embeddingConfig.HashTree, event.SourceFilePath)
		delete(embeddingConfig.SyncFiles, event.SourceFilePath)
	}
	if utils.IsUnauthorizedError(uploadErr) {
		embeddingConfig.FailedFiles[filePath] = errs.ErrAuthenticationFailed
	} else if utils.IsTooManyRequestsError(uploadErr) {
		embeddingConfig.FailedFiles[filePath] = errs.ErrInternalServerError
	} else if utils.IsServiceUnavailableError(uploadErr) {
		embeddingConfig.FailedFiles[filePath] = errs.ErrInternalServerError
	} else {
		embeddingConfig.FailedFiles[filePath] = errs.ErrFileEmbeddingFailed
	}
	// 保存 embedding 配置
	err = ep.embeddingRepo.SaveEmbeddingConfig(embeddingConfig)
	if err != nil {
		ep.logger.Error("failed to save embedding config for workspace %s: %v", event.WorkspacePath, err)
		return fmt.Errorf("failed to save embedding config: %w", err)
	}

	embeddingFileNum := len(embeddingConfig.HashTree)
	var embeddingFailedFilePaths string
	var embeddingMessage string
	embeddingFaildFiles := embeddingConfig.FailedFiles
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

	err = ep.workspaceRepo.UpdateEmbeddingInfo(event.WorkspacePath, embeddingFileNum, time.Now().Unix(), embeddingMessage, embeddingFailedFilePaths)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}
	return nil
}

// uploadFilePathsFailed 批量处理文件路径失败的情况
func (ep *embeddingProcessService) uploadFilePathsFailed(workspacePath string, events []*model.Event, uploadErr error) {
	embeddingId := utils.GenerateEmbeddingID(workspacePath)
	embeddingConfig, err := ep.embeddingRepo.GetEmbeddingConfig(embeddingId)
	if err != nil {
		ep.logger.Error("failed to get embedding config for workspace %s: %v", workspacePath, err)
		return
	}

	if embeddingConfig.HashTree == nil {
		embeddingConfig.HashTree = make(map[string]string)
	}
	if embeddingConfig.FailedFiles == nil {
		embeddingConfig.FailedFiles = make(map[string]string)
	}
	if embeddingConfig.SyncFiles == nil {
		embeddingConfig.SyncFiles = make(map[string]string)
	}

	// 批量处理失败文件
	for _, event := range events {
		filePath := event.SourceFilePath
		if event.EventType == model.EventTypeRenameFile {
			filePath = event.TargetFilePath
		}

		delete(embeddingConfig.HashTree, filePath)
		delete(embeddingConfig.SyncFiles, filePath)
		if event.EventType == model.EventTypeRenameFile {
			delete(embeddingConfig.HashTree, event.SourceFilePath)
			delete(embeddingConfig.SyncFiles, event.SourceFilePath)
		}

		if utils.IsUnauthorizedError(uploadErr) {
			embeddingConfig.FailedFiles[filePath] = errs.ErrAuthenticationFailed
		} else if utils.IsTooManyRequestsError(uploadErr) {
			embeddingConfig.FailedFiles[filePath] = errs.ErrInternalServerError
		} else if utils.IsServiceUnavailableError(uploadErr) {
			embeddingConfig.FailedFiles[filePath] = errs.ErrInternalServerError
		} else {
			embeddingConfig.FailedFiles[filePath] = errs.ErrFileEmbeddingFailed
		}
	}

	// 保存 embedding 配置
	err = ep.embeddingRepo.SaveEmbeddingConfig(embeddingConfig)
	if err != nil {
		ep.logger.Error("failed to save embedding config for workspace %s: %v", workspacePath, err)
		return
	}

	// 更新工作区信息
	embeddingFileNum := len(embeddingConfig.HashTree)
	var embeddingFailedFilePaths string
	var embeddingMessage string
	embeddingFaildFiles := embeddingConfig.FailedFiles
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

	err = ep.workspaceRepo.UpdateEmbeddingInfo(workspacePath, embeddingFileNum, time.Now().Unix(), embeddingMessage, embeddingFailedFilePaths)
	if err != nil {
		ep.logger.Error("failed to update workspace %s: %v", workspacePath, err)
	}
}

// CleanWorkspaceFilePath 删除 workspace 中指定文件的 filepath 记录
func (ep *embeddingProcessService) CleanWorkspaceFilePath(ctx context.Context, fileStatus *utils.FileStatus, event *model.Event) error {
	ep.logger.Info("cleaning workspace filepath for event: %s, workspace: %s", event.SourceFilePath, event.WorkspacePath)

	// 获取 embedding 配置
	embeddingId := utils.GenerateEmbeddingID(event.WorkspacePath)
	embeddingConfig, err := ep.embeddingRepo.GetEmbeddingConfig(embeddingId)
	if err != nil {
		return fmt.Errorf("failed to get embedding config for workspace %s: %w", event.WorkspacePath, err)
	}

	// 根据事件类型处理不同的文件路径
	filePath := event.SourceFilePath
	var filePaths []string
	switch event.EventType {
	case model.EventTypeAddFile, model.EventTypeModifyFile, model.EventTypeDeleteFile:
		filePaths = []string{event.SourceFilePath}
	case model.EventTypeRenameFile:
		filePaths = []string{event.SourceFilePath, event.TargetFilePath}
		filePath = event.TargetFilePath
	default:
		ep.logger.Warn("unsupported event type for cleaning filepath: %d", event.EventType)
		return nil
	}

	// 从 HashTree 中删除对应的文件路径记录
	// TODO: 判断是否为目录，是则删除目录下所有文件的记录
	updated := false
	if embeddingConfig.HashTree != nil {
		for _, filePath := range filePaths {
			if _, exists := embeddingConfig.HashTree[filePath]; exists {
				delete(embeddingConfig.HashTree, filePath)
				updated = true
				ep.logger.Debug("removed filepath from hash tree: %s", filePath)
			}
		}
	} else {
		embeddingConfig.HashTree = make(map[string]string)
	}

	// 删除FailedFiles中对应的文件路径
	if embeddingConfig.FailedFiles != nil {
		for _, filePath := range filePaths {
			if _, exists := embeddingConfig.FailedFiles[filePath]; exists {
				delete(embeddingConfig.FailedFiles, filePath)
				updated = true
				ep.logger.Debug("removed filepath from failed files: %s", filePath)
			}
		}
	} else {
		embeddingConfig.FailedFiles = make(map[string]string)
	}

	// 从SyncFiles中添加对应的文件路径
	if embeddingConfig.SyncFiles != nil {
		if oldHash, exists := embeddingConfig.SyncFiles[filePath]; !exists || oldHash != fileStatus.Hash {
			embeddingConfig.SyncFiles[filePath] = fileStatus.Hash
			updated = true
		}
	} else {
		embeddingConfig.SyncFiles = make(map[string]string)
		embeddingConfig.SyncFiles[filePath] = fileStatus.Hash
		updated = true
	}

	// // 添加syncId到SyncIds中
	// if embeddingConfig.SyncIds != nil {
	// 	embeddingConfig.SyncIds[fileStatus.RequestId] = time.Now()
	// 	updated = true
	// } else {
	// 	embeddingConfig.SyncIds = make(map[string]time.Time)
	// 	embeddingConfig.SyncIds[fileStatus.RequestId] = time.Now()
	// 	updated = true
	// }

	// 如果有更新，保存配置
	if updated {
		if err := ep.embeddingRepo.SaveEmbeddingConfig(embeddingConfig); err != nil {
			ep.logger.Error("failed to save embedding config after cleaning filepath: %v", err)
			return fmt.Errorf("failed to save embedding config: %w", err)
		}

		embeddingFileNum := len(embeddingConfig.HashTree)
		var embeddingFailedFilePaths string
		var embeddingMessage string
		embeddingFaildFiles := embeddingConfig.FailedFiles
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

		err = ep.workspaceRepo.UpdateEmbeddingInfo(event.WorkspacePath, embeddingFileNum, time.Now().Unix(), embeddingMessage, embeddingFailedFilePaths)
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
func (ep *embeddingProcessService) CleanWorkspaceFilePaths(ctx context.Context, workspacePath string, events []*model.Event) error {
	// 获取 embedding 配置
	embeddingId := utils.GenerateEmbeddingID(workspacePath)
	embeddingConfig, err := ep.embeddingRepo.GetEmbeddingConfig(embeddingId)
	if err != nil {
		return fmt.Errorf("failed to get embedding config for workspace %s: %w", workspacePath, err)
	}

	if embeddingConfig.HashTree == nil {
		embeddingConfig.HashTree = make(map[string]string)
	}
	if embeddingConfig.FailedFiles == nil {
		embeddingConfig.FailedFiles = make(map[string]string)
	}
	if embeddingConfig.SyncFiles == nil {
		embeddingConfig.SyncFiles = make(map[string]string)
	}

	// 批量处理文件路径
	for _, event := range events {
		filePath := event.SourceFilePath
		if event.EventType == model.EventTypeRenameFile {
			filePath = event.TargetFilePath
		}

		delete(embeddingConfig.HashTree, filePath)
		delete(embeddingConfig.FailedFiles, filePath)
		if event.EventType == model.EventTypeRenameFile {
			delete(embeddingConfig.HashTree, event.SourceFilePath)
			delete(embeddingConfig.FailedFiles, event.SourceFilePath)
		}
		if oldHash, exists := embeddingConfig.SyncFiles[filePath]; !exists || oldHash != event.FileHash {
			embeddingConfig.SyncFiles[filePath] = event.FileHash
		}
	}

	// 保存 embedding 配置
	if err := ep.embeddingRepo.SaveEmbeddingConfig(embeddingConfig); err != nil {
		return fmt.Errorf("failed to save embedding config: %w", err)
	}

	// 更新工作区信息
	embeddingFileNum := len(embeddingConfig.HashTree)
	var embeddingFailedFilePaths string
	var embeddingMessage string
	embeddingFaildFiles := embeddingConfig.FailedFiles
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

	err = ep.workspaceRepo.UpdateEmbeddingInfo(workspacePath, embeddingFileNum, time.Now().Unix(), embeddingMessage, embeddingFailedFilePaths)
	if err != nil {
		return fmt.Errorf("failed to update workspace file num: %w", err)
	}

	return nil
}
