package service

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"time"

	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
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
	logger logger.Logger,
) EmbeddingStatusService {
	return &embeddingStatusService{
		codebaseEmbeddingRepo: codebaseEmbeddingRepo,
		workspaceRepo:         workspaceRepo,
		eventRepo:             eventRepo,
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
	sc.logger.Info("checking uploading states for workspaces: %v", workspacePaths)

	// 遍历每个工作区
	for _, workspacePath := range workspacePaths {
		err := sc.checkWorkspaceUploadingStates(workspacePath)
		if err != nil {
			sc.logger.Error("failed to check uploading states for workspace %s: %v", workspacePath, err)
			continue
		}
	}

	sc.logger.Info("checking uploading states completed")
	return nil
}

// CheckAllBuildingStates 检查所有building状态
func (sc *embeddingStatusService) CheckAllBuildingStates(workspacePaths []string) error {
	sc.logger.Info("checking building states for workspaces: %v", workspacePaths)

	// 遍历每个工作区
	for _, workspacePath := range workspacePaths {
		err := sc.checkWorkspaceBuildingStates(workspacePath)
		if err != nil {
			sc.logger.Error("failed to check building states for workspace %s: %v", workspacePath, err)
			continue
		}
	}

	sc.logger.Info("checking building states completed")
	return nil
}

// checkWorkspaceBuildingStates 检查指定工作区的building状态
func (sc *embeddingStatusService) checkWorkspaceBuildingStates(workspacePath string) error {
	sc.logger.Info("checking building states for workspace: %s", workspacePath)

	// 获取指定工作区的building状态events
	events, err := sc.getBuildingEventsForWorkspace(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to get building events: %w", err)
	}

	if len(events) == 0 {
		sc.logger.Info("no building events found for workspace: %s", workspacePath)
		return nil
	}

	sc.logger.Info("found %d building events for workspace: %s", len(events), workspacePath)

	// 检查每个event的构建状态
	for _, event := range events {
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
	sc.logger.Info("checking uploading states for workspace: %s", workspacePath)

	// 获取指定工作区的uploading状态events
	events, err := sc.getUploadingEventsForWorkspace(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to get uploading events: %w", err)
	}

	if len(events) == 0 {
		sc.logger.Info("no uploading events found for workspace: %s", workspacePath)
		return nil
	}

	sc.logger.Info("found %d uploading events for workspace: %s", len(events), workspacePath)

	// 检查每个event的上传状态
	nowTime := time.Now()
	for _, event := range events {
		if nowTime.Sub(event.UpdatedAt) < time.Minute*10 {
			continue
		}
		event.EmbeddingStatus = model.EmbeddingStatusUploadFailed
		err := sc.eventRepo.UpdateEvent(event)
		if err != nil {
			sc.logger.Error("failed to update event status: %v", err)
		}
	}

	return nil
}

// getBuildingEventsForWorkspace 获取指定工作区的building状态events
func (sc *embeddingStatusService) getBuildingEventsForWorkspace(workspacePath string) ([]*model.Event, error) {
	buildingStatuses := []int{model.EmbeddingStatusBuilding}
	events, err := sc.eventRepo.GetEventsByWorkspaceAndEmbeddingStatus(workspacePath, 20, true, buildingStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get building events: %w", err)
	}
	return events, nil
}

// getUploadingEventsForWorkspace 获取指定工作区的uploading状态events
func (sc *embeddingStatusService) getUploadingEventsForWorkspace(workspacePath string) ([]*model.Event, error) {
	uploadingStatuses := []int{model.EmbeddingStatusUploading}
	events, err := sc.eventRepo.GetEventsByWorkspaceAndEmbeddingStatus(workspacePath, 20, true, uploadingStatuses)
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

	fileStatusData := fileStatusResp.Data
	processStatus := fileStatusData.Process

	sc.logger.Info("fetched file status for syncId %s: process=%s", event.SyncId, processStatus)

	// 当process为pending时，不处理
	if processStatus == dto.EmbeddingStatusPending {
		sc.logger.Info("build is pending for syncId: %s", event.SyncId)
		return nil
	}

	// 当process为failed时，将event的embeddingstatus改为failed
	if processStatus == dto.EmbeddingFailed {
		sc.logger.Info("build failed for syncId: %s", event.SyncId)
		event.EmbeddingStatus = model.EmbeddingStatusBuildFailed

		// 更新event记录
		err := sc.eventRepo.UpdateEvent(event)
		if err != nil {
			return fmt.Errorf("failed to update event: %w", err)
		}
		return nil
	}

	// 其他情况保持原来的处理逻辑
	sc.logger.Info("build completed for syncId: %s", event.SyncId)
	return sc.handleBuildCompletion(workspacePath, event, fileStatusData.FileList)
}

// fetchFileStatus 获取文件状态
func (sc *embeddingStatusService) fetchFileStatus(workspacePath, syncId string) (*dto.FileStatusResp, error) {
	fileStatusReq := dto.FileStatusReq{
		ClientId:     sc.syncer.GetSyncConfig().ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		SyncId:       syncId,
	}

	fileStatusResp, err := sc.syncer.FetchFileStatus(fileStatusReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file status: %w", err)
	}

	return fileStatusResp, nil
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
		if fileItem.Path == filePath {
			status = fileItem.Status
			sc.logger.Info("file %s status: %s", fileItem.Path, fileItem.Status)
			break
		}
	}

	// 更新event状态
	if status == dto.EmbeddingComplete {
		event.EmbeddingStatus = model.EmbeddingStatusSuccess
		sc.logger.Info("file %s built successfully for syncId: %s", filePath, event.SyncId)
	} else if status == dto.EmbeddingFailed {
		event.EmbeddingStatus = model.EmbeddingStatusBuildFailed
		sc.logger.Info("file %s failed to build for syncId: %s", filePath, event.SyncId)
	} else {
		return nil
	}

	// 更新event记录
	err := sc.eventRepo.UpdateEvent(event)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	workspaceName := filepath.Base(workspacePath)
	codebaseId := fmt.Sprintf("%s_%x_embedding", workspaceName, md5.Sum([]byte(workspacePath)))

	// 获取 codebase embedding 配置
	codebaseConfig, err := sc.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseId)
	if err != nil {
		sc.logger.Error("failed to get codebase embedding config for workspace %s: %v", workspacePath, err)
		return fmt.Errorf("failed to get codebase embedding config: %w", err)
	}
	if codebaseConfig.HashTree == nil {
		codebaseConfig.HashTree = make(map[string]string)
	}
	codebaseConfig.HashTree[filePath] = ""

	return nil
}
