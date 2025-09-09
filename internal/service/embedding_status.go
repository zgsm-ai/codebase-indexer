package service

import (
	"fmt"
	"path/filepath"
	"runtime"
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

// StatusService 状态检查服务接口
type EmbeddingStatusService interface {
	CheckActiveWorkspaces() ([]*model.Workspace, error)
	CheckAllBuildingStates(workspacePaths []string) error
	CheckAllUploadingStatues(workspacePaths []string) error
}

// embeddingStatusService 状态检查服务实现
type embeddingStatusService struct {
	codebaseEmbeddingRepo repository.EmbeddingFileRepository
	workspaceRepo         repository.WorkspaceRepository
	eventRepo             repository.EventRepository
	syncer                repository.SyncInterface
	logger                logger.Logger
}

// NewEmbeddingStatusService 创建状态检查服务
func NewEmbeddingStatusService(
	codebaseEmbeddingRepo repository.EmbeddingFileRepository,
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	syncer repository.SyncInterface,
	logger logger.Logger,
) EmbeddingStatusService {
	return &embeddingStatusService{
		codebaseEmbeddingRepo: codebaseEmbeddingRepo,
		workspaceRepo:         workspaceRepo,
		eventRepo:             eventRepo,
		syncer:                syncer,
		logger:                logger,
	}
}

// CheckActiveWorkspaces 检查活跃工作区
func (sc *embeddingStatusService) CheckActiveWorkspaces() ([]*model.Workspace, error) {
	workspaces, err := sc.workspaceRepo.GetActiveWorkspaces()
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

func (sc *embeddingStatusService) CheckAllUploadingStatues(workspacePaths []string) error {
	// 遍历每个工作区
	for _, workspacePath := range workspacePaths {
		err := sc.checkWorkspaceUploadingStates(workspacePath)
		if err != nil {
			sc.logger.Error("failed to check uploading states for workspace %s: %v", workspacePath, err)
			continue
		}
	}
	return nil
}

// CheckAllBuildingStates 检查所有building状态
func (sc *embeddingStatusService) CheckAllBuildingStates(workspacePaths []string) error {
	// 遍历每个工作区
	for _, workspacePath := range workspacePaths {
		err := sc.checkWorkspaceBuildingStates(workspacePath)
		if err != nil {
			sc.logger.Error("failed to check building states for workspace %s: %v", workspacePath, err)
			continue
		}
	}
	return nil
}

// checkWorkspaceBuildingStates 检查指定工作区的building状态
func (sc *embeddingStatusService) checkWorkspaceBuildingStates(workspacePath string) error {
	// 获取指定工作区的building状态events
	events, err := sc.getBuildingEventsForWorkspace(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to get building events: %w", err)
	}

	if len(events) == 0 {
		sc.logger.Debug("no building events for workspace: %s", workspacePath)
		return nil
	}

	sc.logger.Info("found %d building events for workspace: %s", len(events), workspacePath)

	// 检查每个event的构建状态
	nowTime := time.Now()

	// 首先处理超时的事件
	var validEvents []*model.Event
	for _, event := range events {
		if nowTime.Sub(event.UpdatedAt) > time.Minute*3 {
			updateEvent := &model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusBuildFailed}
			err := sc.eventRepo.UpdateEvent(updateEvent)
			if err != nil {
				sc.logger.Error("failed to update event status: %v", err)
			}
			sc.buildFilePathFailed(event)
			continue
		}
		validEvents = append(validEvents, event)
	}

	// 按syncId分组有效事件
	syncIdEventsMap := make(map[string][]*model.Event)
	for _, event := range validEvents {
		if event.SyncId == "" {
			sc.logger.Warn("event has empty syncId, workspace: %s, file: %s", workspacePath, event.SourceFilePath)
			continue
		}
		syncIdEventsMap[event.SyncId] = append(syncIdEventsMap[event.SyncId], event)
	}

	// 批量处理每个syncId的事件
	for syncId, syncIdEvents := range syncIdEventsMap {
		sc.logger.Info("processing %d events for syncId: %s", len(syncIdEvents), syncId)

		// 获取文件状态（每个syncId只请求一次）
		fileStatusResp, err := sc.fetchFileStatus(workspacePath, syncId)
		if err != nil {
			sc.logger.Error("failed to fetch file status for syncId %s: %v", syncId, err)
			continue
		}
		sc.logger.Info("syncId %s file status resp: %v", syncId, fileStatusResp)

		// 批量处理该syncId下的所有事件
		if err := sc.processEventsWithFileStatus(workspacePath, syncIdEvents, fileStatusResp); err != nil {
			sc.logger.Error("failed to process events with file status for syncId %s: %v", syncId, err)
		}
	}

	return nil
}

// checkWorkspaceUploadingStates 检查指定工作区的uploading状态
func (sc *embeddingStatusService) checkWorkspaceUploadingStates(workspacePath string) error {
	// 获取指定工作区的uploading状态events
	events, err := sc.getUploadingEventsForWorkspace(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to get uploading events: %w", err)
	}

	if len(events) == 0 {
		sc.logger.Debug("no uploading events for workspace: %s", workspacePath)
		return nil
	}

	sc.logger.Info("found %d uploading events for workspace: %s", len(events), workspacePath)

	// 检查每个event的上传状态
	nowTime := time.Now()
	for _, event := range events {
		if nowTime.Sub(event.UpdatedAt) < time.Minute*3 {
			continue
		}
		updateEvent := &model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusUploadFailed}
		err := sc.eventRepo.UpdateEvent(updateEvent)
		if err != nil {
			sc.logger.Error("failed to update event status: %v", err)
		}
		sc.buildFilePathFailed(event)
	}

	return nil
}

// getBuildingEventsForWorkspace 获取指定工作区的building状态events
func (sc *embeddingStatusService) getBuildingEventsForWorkspace(workspacePath string) ([]*model.Event, error) {
	buildingStatuses := []int{model.EmbeddingStatusBuilding}
	events, err := sc.eventRepo.GetEventsByWorkspaceAndEmbeddingStatus(workspacePath, 50, false, buildingStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get building events: %w", err)
	}
	return events, nil
}

// getUploadingEventsForWorkspace 获取指定工作区的uploading状态events
func (sc *embeddingStatusService) getUploadingEventsForWorkspace(workspacePath string) ([]*model.Event, error) {
	uploadingStatuses := []int{model.EmbeddingStatusUploading}
	events, err := sc.eventRepo.GetEventsByWorkspaceAndEmbeddingStatus(workspacePath, 50, false, uploadingStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get uploading events: %w", err)
	}
	return events, nil
}

// checkEventBuildStatus 检查单个event的构建状态
func (sc *embeddingStatusService) checkEventBuildStatus(workspacePath string, event *model.Event) error {
	if event.SyncId == "" {
		sc.logger.Warn("event has empty syncId, workspace: %s, file: %s", workspacePath, event.SourceFilePath)
		return nil
	}

	sc.logger.Info("checking build status for syncId: %s, file: %s", event.SyncId, event.SourceFilePath)

	// 获取文件状态
	fileStatusResp, err := sc.fetchFileStatus(workspacePath, event.SyncId)
	if err != nil {
		return fmt.Errorf("failed to fetch file status: %w", err)
	}
	sc.logger.Info("file status resp: %v", fileStatusResp)

	fileStatusData := fileStatusResp.Data
	processStatus := fileStatusData.Process

	sc.logger.Debug("fetched file status for syncId %s: process=%s", event.SyncId, processStatus)

	// 当process为pending时，不处理
	if processStatus == dto.EmbeddingStatusPending {
		sc.logger.Info("build is pending for syncId: %s", event.SyncId)
		return nil
	}

	// 当process为failed时，将event的embeddingstatus改为failed
	if processStatus == dto.EmbeddingFailed {
		sc.logger.Info("build failed for syncId: %s", event.SyncId)

		updateEvent := &model.Event{ID: event.ID, EmbeddingStatus: model.EmbeddingStatusBuildFailed}
		// 更新event记录
		err := sc.eventRepo.UpdateEvent(updateEvent)
		if err != nil {
			return fmt.Errorf("failed to update event: %w", err)
		}
		sc.buildFilePathFailed(event)
		return nil
	}

	// 其他情况保持原来的处理逻辑
	sc.logger.Debug("build completed for syncId: %s", event.SyncId)
	return sc.handleBuildCompletion(workspacePath, event, fileStatusData.FileList)
}

// processEventsWithFileStatus 使用已获取的文件状态批量处理事件
func (sc *embeddingStatusService) processEventsWithFileStatus(workspacePath string, events []*model.Event, fileStatusResp *dto.FileStatusResp) error {
	if len(events) == 0 {
		return nil
	}

	fileStatusData := fileStatusResp.Data
	processStatus := fileStatusData.Process

	// 当process为pending时，不处理
	if processStatus == dto.EmbeddingStatusPending {
		return nil
	}

	// 当process为failed时，将所有事件的embeddingstatus改为failed
	if processStatus == dto.EmbeddingFailed {
		// 构建失败，批量更新事件状态为构建失败
		eventIDs := make([]int64, 0, len(events))
		for _, event := range events {
			eventIDs = append(eventIDs, event.ID)
		}
		updateErr := sc.eventRepo.UpdateEventsEmbeddingStatus(eventIDs, model.EmbeddingStatusBuildFailed)
		if updateErr != nil {
			sc.logger.Error("failed to update events status to buildFailed: %v", updateErr)
		}
		return sc.batchHandleBuildFailed(workspacePath, events)
	}

	// 其他情况保持原来的处理逻辑，批量处理构建完成
	return sc.batchHandleBuildCompletion(workspacePath, events, fileStatusData.FileList)
}

// fetchFileStatus 获取文件状态
func (sc *embeddingStatusService) fetchFileStatus(workspacePath, syncId string) (*dto.FileStatusResp, error) {
	authInfo := config.GetAuthInfo()
	fileStatusReq := dto.FileStatusReq{
		ClientId:     authInfo.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		SyncId:       syncId,
	}

	fileStatusResp, err := sc.syncer.FetchFileStatus(fileStatusReq)
	if err != nil {
		return nil, err
	}

	return fileStatusResp, nil
}

func (sc *embeddingStatusService) buildFilePathFailed(event *model.Event) error {
	filePath := event.SourceFilePath
	if event.EventType == model.EventTypeRenameFile {
		filePath = event.TargetFilePath
	}
	workspacePath := event.WorkspacePath
	codebaseId := utils.GenerateCodebaseEmbeddingID(workspacePath)
	codebaseEmbeddingConfig, err := sc.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseId)
	if err != nil {
		return fmt.Errorf("failed to get codebase embedding config for workspace %s: %w", event.WorkspacePath, err)
	}
	if codebaseEmbeddingConfig.HashTree == nil {
		codebaseEmbeddingConfig.HashTree = make(map[string]string)
	}
	if codebaseEmbeddingConfig.FailedFiles == nil {
		codebaseEmbeddingConfig.FailedFiles = make(map[string]string)
	}
	if codebaseEmbeddingConfig.SyncFiles == nil {
		codebaseEmbeddingConfig.SyncFiles = make(map[string]string)
	}
	delete(codebaseEmbeddingConfig.SyncFiles, filePath)
	if event.EventType == model.EventTypeRenameFile {
		delete(codebaseEmbeddingConfig.SyncFiles, event.SourceFilePath)
	}
	codebaseEmbeddingConfig.FailedFiles[filePath] = errs.ErrFileEmbeddingFailed
	// 保存 codebase embedding 配置
	err = sc.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig)
	if err != nil {
		sc.logger.Error("failed to save codebase embedding config for workspace %s: %v", event.WorkspacePath, err)
		return fmt.Errorf("failed to save codebase embedding config: %w", err)
	}

	return sc.updateWorkspaceEmbeddingInfo(workspacePath, codebaseEmbeddingConfig)
}

// handleBuildCompletion 处理构建完成后的状态更新
func (sc *embeddingStatusService) handleBuildCompletion(workspacePath string, event *model.Event, fileStatusList []dto.FileStatusRespFileListItem) error {
	// 检查所有文件状态是否都成功
	filePath := event.SourceFilePath
	if event.EventType == model.EventTypeRenameFile {
		filePath = event.TargetFilePath
	}
	var status string
	for _, fileItem := range fileStatusList {
		fileItemPath := fileItem.Path
		if runtime.GOOS == "windows" {
			fileItemPath = filepath.FromSlash(fileItemPath)
		}
		if fileItemPath == filePath {
			status = fileItem.Status
			sc.logger.Info("file %s status: %s", fileItemPath, fileItem.Status)
			break
		}
	}

	// 更新event状态
	updateEvent := &model.Event{ID: event.ID}
	switch status {
	case dto.EmbeddingComplete, dto.EmbeddingUnsupported:
		updateEvent.EmbeddingStatus = model.EmbeddingStatusSuccess
		sc.logger.Info("file %s built successfully for syncId: %s", filePath, event.SyncId)
	case dto.EmbeddingFailed:
		updateEvent.EmbeddingStatus = model.EmbeddingStatusBuildFailed
		sc.logger.Info("file %s failed to build for syncId: %s", filePath, event.SyncId)
	default:
		return nil
	}

	// 更新event记录
	err := sc.eventRepo.UpdateEvent(updateEvent)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	// 获取 codebase embedding 配置
	codebaseId := utils.GenerateCodebaseEmbeddingID(workspacePath)
	codebaseEmbeddingConfig, err := sc.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseId)
	if err != nil {
		return fmt.Errorf("failed to get codebase embedding config for workspace %s: %w", workspacePath, err)
	}
	if codebaseEmbeddingConfig.HashTree == nil {
		codebaseEmbeddingConfig.HashTree = make(map[string]string)
	}
	if codebaseEmbeddingConfig.FailedFiles == nil {
		codebaseEmbeddingConfig.FailedFiles = make(map[string]string)
	}
	if codebaseEmbeddingConfig.SyncFiles == nil {
		codebaseEmbeddingConfig.SyncFiles = make(map[string]string)
	}

	if status == dto.EmbeddingFailed {
		delete(codebaseEmbeddingConfig.SyncFiles, filePath)
		codebaseEmbeddingConfig.FailedFiles[filePath] = errs.ErrFileEmbeddingFailed
	} else {
		delete(codebaseEmbeddingConfig.SyncFiles, filePath)
		delete(codebaseEmbeddingConfig.FailedFiles, filePath)
		if event.EventType != model.EventTypeDeleteFile {
			codebaseEmbeddingConfig.HashTree[filePath] = event.FileHash
		}
	}
	// 保存 codebase embedding 配置
	err = sc.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig)
	if err != nil {
		sc.logger.Error("failed to save codebase embedding config for workspace %s: %v", workspacePath, err)
		return fmt.Errorf("failed to save codebase embedding config: %w", err)
	}

	return sc.updateWorkspaceEmbeddingInfo(workspacePath, codebaseEmbeddingConfig)
}

// batchHandleBuildFailed 批量处理构建失败
func (sc *embeddingStatusService) batchHandleBuildFailed(workspacePath string, events []*model.Event) error {
	if len(events) == 0 {
		return nil
	}

	// 获取 codebase embedding 配置
	codebaseId := utils.GenerateCodebaseEmbeddingID(workspacePath)
	codebaseEmbeddingConfig, err := sc.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseId)
	if err != nil {
		return fmt.Errorf("failed to get codebase embedding config for workspace %s: %w", workspacePath, err)
	}

	if codebaseEmbeddingConfig.HashTree == nil {
		codebaseEmbeddingConfig.HashTree = make(map[string]string)
	}
	if codebaseEmbeddingConfig.FailedFiles == nil {
		codebaseEmbeddingConfig.FailedFiles = make(map[string]string)
	}
	if codebaseEmbeddingConfig.SyncFiles == nil {
		codebaseEmbeddingConfig.SyncFiles = make(map[string]string)
	}

	// 批量更新事件状态和处理文件路径
	for _, event := range events {
		// 处理文件路径
		filePath := event.SourceFilePath
		if event.EventType == model.EventTypeRenameFile {
			filePath = event.TargetFilePath
		}

		delete(codebaseEmbeddingConfig.SyncFiles, filePath)
		if event.EventType == model.EventTypeRenameFile {
			delete(codebaseEmbeddingConfig.SyncFiles, event.SourceFilePath)
		}
		codebaseEmbeddingConfig.FailedFiles[filePath] = errs.ErrFileEmbeddingFailed
	}

	// 保存 codebase embedding 配置
	err = sc.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig)
	if err != nil {
		return fmt.Errorf("failed to save codebase embedding config: %w", err)
	}

	// 更新工作区信息
	return sc.updateWorkspaceEmbeddingInfo(workspacePath, codebaseEmbeddingConfig)
}

// batchHandleBuildCompletion 批量处理构建完成后的状态更新
func (sc *embeddingStatusService) batchHandleBuildCompletion(workspacePath string, events []*model.Event, fileStatusList []dto.FileStatusRespFileListItem) error {
	if len(events) == 0 {
		return nil
	}

	// 获取 codebase embedding 配置
	codebaseId := utils.GenerateCodebaseEmbeddingID(workspacePath)
	codebaseEmbeddingConfig, err := sc.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseId)
	if err != nil {
		return fmt.Errorf("failed to get codebase embedding config for workspace %s: %w", workspacePath, err)
	}

	if codebaseEmbeddingConfig.HashTree == nil {
		codebaseEmbeddingConfig.HashTree = make(map[string]string)
	}
	if codebaseEmbeddingConfig.FailedFiles == nil {
		codebaseEmbeddingConfig.FailedFiles = make(map[string]string)
	}
	if codebaseEmbeddingConfig.SyncFiles == nil {
		codebaseEmbeddingConfig.SyncFiles = make(map[string]string)
	}

	// 预处理文件状态列表，构建路径到状态的映射
	fileStatusMap := make(map[string]string)
	for _, fileItem := range fileStatusList {
		fileItemPath := fileItem.Path
		if runtime.GOOS == "windows" {
			fileItemPath = filepath.FromSlash(fileItemPath)
		}
		fileStatusMap[fileItemPath] = fileItem.Status
	}

	// 批量更新事件状态和处理文件路径
	for _, event := range events {
		// 检查文件状态
		filePath := event.SourceFilePath
		if event.EventType == model.EventTypeRenameFile {
			filePath = event.TargetFilePath
		}

		var status string
		if fileStatus, exists := fileStatusMap[filePath]; exists {
			status = fileStatus
			sc.logger.Debug("file %s status: %s, syncId: %s", filePath, fileStatus, event.SyncId)
		}

		// 更新事件状态
		updateEvent := &model.Event{ID: event.ID}
		switch status {
		case dto.EmbeddingComplete, dto.EmbeddingUnsupported:
			updateEvent.EmbeddingStatus = model.EmbeddingStatusSuccess
			sc.logger.Debug("file %s built successfully for syncId: %s", filePath, event.SyncId)
		case dto.EmbeddingFailed:
			updateEvent.EmbeddingStatus = model.EmbeddingStatusBuildFailed
			sc.logger.Debug("file %s failed to build for syncId: %s", filePath, event.SyncId)
		default:
			continue
		}

		// 更新event记录
		if err := sc.eventRepo.UpdateEvent(updateEvent); err != nil {
			sc.logger.Error("failed to update event %d: %v", event.ID, err)
			continue
		}

		// 处理文件路径
		if status == dto.EmbeddingFailed {
			delete(codebaseEmbeddingConfig.SyncFiles, filePath)
			codebaseEmbeddingConfig.FailedFiles[filePath] = errs.ErrFileEmbeddingFailed
		} else {
			delete(codebaseEmbeddingConfig.SyncFiles, filePath)
			delete(codebaseEmbeddingConfig.FailedFiles, filePath)
			if event.EventType != model.EventTypeDeleteFile {
				codebaseEmbeddingConfig.HashTree[filePath] = event.FileHash
			}
		}
	}

	// 保存 codebase embedding 配置
	err = sc.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig)
	if err != nil {
		return fmt.Errorf("failed to save codebase embedding config: %w", err)
	}

	// 更新工作区信息
	return sc.updateWorkspaceEmbeddingInfo(workspacePath, codebaseEmbeddingConfig)
}

// updateWorkspaceEmbeddingInfo 更新工作区嵌入信息
func (sc *embeddingStatusService) updateWorkspaceEmbeddingInfo(workspacePath string, codebaseEmbeddingConfig *config.CodebaseEmbeddingConfig) error {
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

	err := sc.workspaceRepo.UpdateEmbeddingInfo(workspacePath, embeddingFileNum, time.Now().Unix(), embeddingMessage, embeddingFailedFilePaths)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}
	return nil
}
