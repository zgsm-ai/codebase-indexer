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
		err := sc.checkEventBuildStatus(workspacePath, event)
		if err != nil {
			sc.logger.Error("failed to check event build status: %v", err)
			continue
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
	events, err := sc.eventRepo.GetEventsByWorkspaceAndEmbeddingStatus(workspacePath, 20, false, buildingStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get building events: %w", err)
	}
	return events, nil
}

// getUploadingEventsForWorkspace 获取指定工作区的uploading状态events
func (sc *embeddingStatusService) getUploadingEventsForWorkspace(workspacePath string) ([]*model.Event, error) {
	uploadingStatuses := []int{model.EmbeddingStatusUploading}
	events, err := sc.eventRepo.GetEventsByWorkspaceAndEmbeddingStatus(workspacePath, 20, false, uploadingStatuses)
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
	codebaseId := utils.GenerateCodebaseEmbeddingID(event.WorkspacePath)
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

	err = sc.workspaceRepo.UpdateEmbeddingInfo(event.WorkspacePath, embeddingFileNum, time.Now().Unix(), embeddingMessage, embeddingFailedFilePaths)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}
	return nil
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

	err = sc.workspaceRepo.UpdateEmbeddingInfo(event.WorkspacePath, embeddingFileNum, time.Now().Unix(), embeddingMessage, embeddingFailedFilePaths)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}
