package service

import (
	"fmt"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// FileScanService 工作区扫描服务接口
type FileScanService interface {
	ScanActiveWorkspaces() ([]*model.Workspace, error)
	DetectFileChanges(workspace *model.Workspace) ([]*model.Event, error)
	UpdateWorkspaceStats(workspace *model.Workspace) error
	MapFileStatusToEventType(status string) string
}

// fileScanService 工作区扫描服务实现
type fileScanService struct {
	workspaceRepo repository.WorkspaceRepository
	eventRepo     repository.EventRepository
	fileScanner   repository.ScannerInterface
	storage       repository.StorageInterface
	logger        logger.Logger
}

// NewFileScanService 创建工作区扫描服务
func NewFileScanService(
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	fileScanner repository.ScannerInterface,
	storage repository.StorageInterface,
	logger logger.Logger,
) FileScanService {
	return &fileScanService{
		workspaceRepo: workspaceRepo,
		eventRepo:     eventRepo,
		fileScanner:   fileScanner,
		storage:       storage,
		logger:        logger,
	}
}

// ScanActiveWorkspaces 扫描活跃工作区
func (ws *fileScanService) ScanActiveWorkspaces() ([]*model.Workspace, error) {
	workspaces, err := ws.workspaceRepo.GetActiveWorkspaces()
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

// DetectFileChanges 检测文件变更
func (ws *fileScanService) DetectFileChanges(workspace *model.Workspace) ([]*model.Event, error) {
	ws.logger.Info("scanning workspace: %s", workspace.WorkspacePath)

	// 获取当前文件哈希树
	currentHashTree, err := ws.fileScanner.ScanCodebase(workspace.WorkspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan codebase: %w", err)
	}

	// 获取上次保存的哈希树
	// 生成codebaseId
	codebaseId := fmt.Sprintf("%s_%x", workspace.WorkspaceName, []byte(workspace.WorkspacePath))
	codebaseConfig, err := ws.storage.GetCodebaseConfig(codebaseId)
	if err != nil {
		return nil, fmt.Errorf("failed to get codebase config: %w", err)
	}

	// 计算文件变更
	changes := ws.fileScanner.CalculateFileChanges(currentHashTree, codebaseConfig.HashTree)

	// 在生成新事件后，查询工作区内所有现有事件
	existingEvents, err := ws.eventRepo.GetEventsByWorkspaceForDeduplication(workspace.WorkspacePath)
	if err != nil {
		ws.logger.Error("failed to get existing events for deduplication: %v", err)
		// 降级处理：继续执行，但跳过去重逻辑
		return ws.handleEventsWithoutDeduplication(changes, workspace)
	}

	// 构建源文件路径到事件记录的映射，用于快速查找
	eventPathMap := make(map[string]*model.Event)
	for _, existingEvent := range existingEvents {
		// 如果同一路径有多个事件，保留最新的一个
		if currentEvent, exists := eventPathMap[existingEvent.SourceFilePath]; !exists ||
			existingEvent.CreatedAt.After(currentEvent.CreatedAt) {
			eventPathMap[existingEvent.SourceFilePath] = existingEvent
		}
	}

	// 生成事件并进行去重处理
	var events []*model.Event
	for _, change := range changes {
		event := &model.Event{
			WorkspacePath:  workspace.WorkspacePath,
			EventType:      ws.MapFileStatusToEventType(change.Status),
			SourceFilePath: change.Path,
			TargetFilePath: change.Path,
		}

		// 检查是否已存在相同路径的事件
		if existingEvent, exists := eventPathMap[change.Path]; exists {
			// 更新现有事件
			err := ws.updateExistingEvent(existingEvent, event)
			if err != nil {
				ws.logger.Error("failed to update existing event for path %s: %v", change.Path, err)
				continue
			}
			events = append(events, existingEvent)
		} else {
			// 创建新事件
			err := ws.eventRepo.CreateEvent(event)
			if err != nil {
				ws.logger.Error("failed to create event for path %s: %v", change.Path, err)
				continue
			}
			events = append(events, event)
		}
	}

	// 更新哈希树
	codebaseConfig.HashTree = currentHashTree
	err = ws.storage.SaveCodebaseConfig(codebaseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to save codebase config: %w", err)
	}

	return events, nil
}

// MapFileStatusToEventType 映射文件状态到事件类型
func (ws *fileScanService) MapFileStatusToEventType(status string) string {
	switch status {
	case utils.FILE_STATUS_ADDED:
		return model.EventTypeAddFile
	case utils.FILE_STATUS_MODIFIED:
		return model.EventTypeModifyFile
	case utils.FILE_STATUS_DELETED:
		return model.EventTypeDeleteFile
	default:
		return model.EventTypeUnknown
	}
}

// UpdateWorkspaceStats 更新工作区统计信息
func (ws *fileScanService) UpdateWorkspaceStats(workspace *model.Workspace) error {
	// 获取当前文件数量
	// 生成codebaseId
	codebaseId := fmt.Sprintf("%s_%x", workspace.WorkspaceName, []byte(workspace.WorkspacePath))
	codebaseConfig, err := ws.storage.GetCodebaseConfig(codebaseId)
	if err != nil {
		return fmt.Errorf("failed to get codebase config: %w", err)
	}
	fileNum := len(codebaseConfig.HashTree)

	// 更新工作区文件数量
	updateWorkspace := model.Workspace{WorkspacePath: workspace.WorkspacePath, FileNum: fileNum}

	// 更新工作区信息
	err = ws.workspaceRepo.UpdateWorkspace(&updateWorkspace)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}

// handleEventsWithoutDeduplication 当去重逻辑失败时的降级处理方法
func (ws *fileScanService) handleEventsWithoutDeduplication(changes []*utils.FileStatus, workspace *model.Workspace) ([]*model.Event, error) {
	ws.logger.Warn("deduplication failed, falling back to direct event creation")

	var events []*model.Event
	for _, change := range changes {
		event := &model.Event{
			WorkspacePath:  workspace.WorkspacePath,
			EventType:      ws.MapFileStatusToEventType(change.Status),
			SourceFilePath: change.Path,
			TargetFilePath: change.Path,
		}

		err := ws.eventRepo.CreateEvent(event)
		if err != nil {
			ws.logger.Error("failed to create event for path %s: %v", change.Path, err)
			continue
		}

		events = append(events, event)
	}

	return events, nil
}

// updateExistingEvent 更新现有事件的信息
func (ws *fileScanService) updateExistingEvent(existingEvent, newEvent *model.Event) error {
	if existingEvent.EmbeddingStatus == model.EmbeddingStatusBuilding || existingEvent.EmbeddingStatus == model.EmbeddingStatusUploading {
		if newEvent.EventType == existingEvent.EventType {
			return nil
		}
		ws.logger.Info("building event, create new event for path: %s, type: %s", existingEvent.SourceFilePath, newEvent.EventType)
		return ws.eventRepo.CreateEvent(newEvent)
	}

	ws.logger.Info("update existing event for path: %s, type: %s, embedding status: %s", existingEvent.SourceFilePath, newEvent.EventType, existingEvent.EmbeddingStatus)
	// 更新事件类型和其他必要信息
	updateEvent := &model.Event{ID: existingEvent.ID, EventType: newEvent.EventType}
	// 调用 repository 更新事件
	return ws.eventRepo.UpdateEvent(updateEvent)
}
