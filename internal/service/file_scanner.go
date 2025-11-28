package service

import (
	"fmt"
	"time"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// FileScanService 工作区扫描服务接口
type FileScanService interface {
	ScanActiveWorkspaces() ([]*model.Workspace, error)
	DetectFileChanges(workspacePath string) ([]*model.Event, error)
	UpdateWorkspaceStats(workspace *model.Workspace) error
	MapFileStatusToEventType(status string) string
}

// fileScanService 工作区扫描服务实现
type fileScanService struct {
	workspaceRepo repository.WorkspaceRepository
	eventRepo     repository.EventRepository
	fileScanner   repository.ScannerInterface
	storage       repository.StorageInterface
	embeddingRepo repository.EmbeddingFileRepository
	logger        logger.Logger
}

// NewFileScanService 创建工作区扫描服务
func NewFileScanService(
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	fileScanner repository.ScannerInterface,
	storage repository.StorageInterface,
	embeddingRepo repository.EmbeddingFileRepository,
	logger logger.Logger,
) FileScanService {
	return &fileScanService{
		workspaceRepo: workspaceRepo,
		eventRepo:     eventRepo,
		fileScanner:   fileScanner,
		storage:       storage,
		embeddingRepo: embeddingRepo,
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
func (ws *fileScanService) DetectFileChanges(workspacePath string) ([]*model.Event, error) {
	ws.logger.Info("scanning workspace: %s", workspacePath)

	// 获取当前文件哈希树
	ignoreConfig := ws.fileScanner.LoadIgnoreConfig(workspacePath)
	currentHashTree, err := ws.fileScanner.ScanCodebase(ignoreConfig, workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan codebase: %w", err)
	}

	// 获取上次保存的哈希树
	// 生成codebaseId
	codebaseId := utils.GenerateCodebaseID(workspacePath)
	codebaseConfig, err := ws.storage.GetCodebaseConfig(codebaseId)
	if err != nil {
		return nil, fmt.Errorf("failed to get codebase config: %w", err)
	}

	// 更新哈希树
	codebaseConfig.HashTree = currentHashTree
	codebaseConfig.RegisterTime = time.Now()
	err = ws.storage.SaveCodebaseConfig(codebaseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to save codebase config: %w", err)
	}

	embeddingId := utils.GenerateEmbeddingID(workspacePath)
	embeddingConfig, err := ws.embeddingRepo.GetEmbeddingConfig(embeddingId)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding config: %w", err)
	}

	// 计算文件变更
	changes := utils.CalculateFileChanges(currentHashTree, embeddingConfig.HashTree)
	if len(changes) == 0 {
		// 查询 open_workspace 事件并更新状态为完成
		ws.updateNonSuccessOpenOrRebuildEventStatus(workspacePath)
		return nil, nil
	}

	// 在生成新事件后，查询工作区内所有现有事件
	existingEvents, err := ws.eventRepo.GetEventsByWorkspaceForDeduplication(workspacePath)
	if err != nil {
		ws.logger.Error("failed to get existing events for deduplication: %v", err)
		// 降级处理：继续执行，但跳过去重逻辑
		return ws.handleEventsWithoutDeduplication(changes, workspacePath)
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
	var eventsToCreate []*model.Event // 需要批量创建的事件（新文件 + building时需创建的）
	var eventsToUpdate []*model.Event // 需要批量更新的事件

	for _, change := range changes {
		filePth := change.Path
		event := &model.Event{
			WorkspacePath:   workspacePath,
			EventType:       ws.MapFileStatusToEventType(change.Status),
			SourceFilePath:  filePth,
			TargetFilePath:  filePth,
			FileHash:        change.Hash,
			EmbeddingStatus: model.EmbeddingStatusInit,
			CodegraphStatus: model.CodegraphStatusSuccess,
		}

		// 检查是否已存在相同路径的事件
		if existingEvent, exists := eventPathMap[filePth]; exists {
			// 分类处理现有事件
			action := ws.classifyExistingEventAction(existingEvent, event)
			switch action {
			case "skip":
				// 正在 building 且类型相同，跳过
				events = append(events, existingEvent)
			case "create":
				// 正在 building 但类型不同，需要创建新事件
				eventsToCreate = append(eventsToCreate, event)
			case "update":
				// 更新现有事件的状态
				existingEvent.EventType = event.EventType
				existingEvent.TargetFilePath = event.TargetFilePath
				existingEvent.EmbeddingStatus = model.EmbeddingStatusInit
				existingEvent.CodegraphStatus = model.CodegraphStatusSuccess
				eventsToUpdate = append(eventsToUpdate, existingEvent)
				events = append(events, existingEvent)
			}
		} else {
			// 新文件，收集后批量创建
			eventsToCreate = append(eventsToCreate, event)
		}
	}

	// 批量创建事件（减少 fsync 次数，提升性能）
	if len(eventsToCreate) > 0 {
		err := ws.eventRepo.BatchCreateEvents(eventsToCreate)
		if err != nil {
			ws.logger.Error("failed to batch create events: %v", err)
			// 降级处理：逐条创建
			for _, event := range eventsToCreate {
				if createErr := ws.eventRepo.CreateEvent(event); createErr != nil {
					ws.logger.Error("failed to create event for path %s: %v", event.SourceFilePath, createErr)
					continue
				}
				events = append(events, event)
			}
		} else {
			events = append(events, eventsToCreate...)
			ws.logger.Info("batch created %d events for workspace: %s", len(eventsToCreate), workspacePath)
		}
	}

	// 批量更新现有事件
	if len(eventsToUpdate) > 0 {
		err := ws.eventRepo.BatchUpdateEvents(eventsToUpdate)
		if err != nil {
			ws.logger.Error("failed to batch update events: %v", err)
			// 降级处理：逐条更新
			for _, event := range eventsToUpdate {
				if updateErr := ws.eventRepo.UpdateEvent(event); updateErr != nil {
					ws.logger.Error("failed to update event for path %s: %v", event.SourceFilePath, updateErr)
				}
			}
		} else {
			ws.logger.Info("batch updated %d existing events for workspace: %s", len(eventsToUpdate), workspacePath)
		}
	}

	// 查询 open_workspace 事件并更新状态为完成
	ws.updateNonSuccessOpenOrRebuildEventStatus(workspacePath)

	return events, nil
}

func (ws *fileScanService) updateNonSuccessOpenOrRebuildEventStatus(workspacePath string) {
	openWorkspaceEvents, err := ws.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(
		[]string{model.EventTypeOpenWorkspace, model.EventTypeRebuildWorkspace},
		[]string{workspacePath},
		1, // 限制查询数量
		false,
		[]int{
			model.EmbeddingStatusInit,
			model.EmbeddingStatusUploading,
			model.EmbeddingStatusBuilding,
			model.EmbeddingStatusUploadFailed,
			model.EmbeddingStatusBuildFailed,
		},
		[]int{},
	)
	if err != nil {
		ws.logger.Error("failed to get open_workspace events: %v", err)
	}
	for _, event := range openWorkspaceEvents {
		ws.logger.Info("non-success open_workspace or rebuild_workspace event: %v", event)
		if event.EmbeddingStatus != model.EmbeddingStatusSuccess {
			updateEvent := &model.Event{
				ID:              event.ID,
				EmbeddingStatus: model.EmbeddingStatusSuccess,
			}
			err := ws.eventRepo.UpdateEvent(updateEvent)
			if err != nil {
				ws.logger.Error("failed to update open_workspace event status: %v", err)
			} else {
				ws.logger.Info("updated open_workspace event status to success for workspace: %s", workspacePath)
			}
		}
	}
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
	codebaseId := utils.GenerateCodebaseID(workspace.WorkspacePath)
	codebaseConfig, err := ws.storage.GetCodebaseConfig(codebaseId)
	if err != nil {
		return fmt.Errorf("failed to get codebase config: %w", err)
	}
	fileNum := len(codebaseConfig.HashTree)

	// 更新工作区文件数量
	updateWorkspace := model.Workspace{
		WorkspacePath: workspace.WorkspacePath,
		FileNum:       fileNum,
	}
	err = ws.workspaceRepo.UpdateWorkspace(&updateWorkspace)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}

// handleEventsWithoutDeduplication 当去重逻辑失败时的降级处理方法
func (ws *fileScanService) handleEventsWithoutDeduplication(changes []*utils.FileStatus, workspacePath string) ([]*model.Event, error) {
	ws.logger.Warn("deduplication failed, falling back to direct event creation")

	var events []*model.Event
	for _, change := range changes {
		filePth := change.Path
		event := &model.Event{
			WorkspacePath:   workspacePath,
			EventType:       ws.MapFileStatusToEventType(change.Status),
			SourceFilePath:  filePth,
			TargetFilePath:  filePth,
			FileHash:        change.Hash,
			EmbeddingStatus: model.EmbeddingStatusInit,
			CodegraphStatus: model.CodegraphStatusSuccess,
		}
		events = append(events, event)
	}

	// 批量创建事件（减少 fsync 次数，提升性能）
	if len(events) > 0 {
		err := ws.eventRepo.BatchCreateEvents(events)
		if err != nil {
			ws.logger.Error("failed to batch create events: %v", err)
			// 降级处理：逐条创建
			var createdEvents []*model.Event
			for _, event := range events {
				if createErr := ws.eventRepo.CreateEvent(event); createErr != nil {
					ws.logger.Error("failed to create event for path %s: %v", event.SourceFilePath, createErr)
					continue
				}
				createdEvents = append(createdEvents, event)
			}
			events = createdEvents
		} else {
			ws.logger.Info("batch created %d events for workspace: %s", len(events), workspacePath)
		}
	}

	// 查询 open_workspace 事件并更新状态为完成
	ws.updateNonSuccessOpenOrRebuildEventStatus(workspacePath)

	return events, nil
}

// classifyExistingEventAction 判断现有事件应该执行的操作
// 返回值: "skip" - 跳过, "create" - 创建新事件, "update" - 更新现有事件
func (ws *fileScanService) classifyExistingEventAction(existingEvent, newEvent *model.Event) string {
	if existingEvent.EmbeddingStatus == model.EmbeddingStatusBuilding ||
		existingEvent.EmbeddingStatus == model.EmbeddingStatusUploading ||
		existingEvent.CodegraphStatus == model.CodegraphStatusBuilding {
		if newEvent.EventType == existingEvent.EventType {
			return "skip"
		}
		ws.logger.Debug("building event, will create new event for path: %s, type: %s", existingEvent.SourceFilePath, newEvent.EventType)
		return "create"
	}

	ws.logger.Debug("will update existing event for path: %s, type: %s", existingEvent.SourceFilePath, newEvent.EventType)
	return "update"
}

// updateExistingEvent 更新现有事件的信息（保留用于降级处理）
func (ws *fileScanService) updateExistingEvent(existingEvent, newEvent *model.Event) error {
	if existingEvent.EmbeddingStatus == model.EmbeddingStatusBuilding ||
		existingEvent.EmbeddingStatus == model.EmbeddingStatusUploading ||
		existingEvent.CodegraphStatus == model.CodegraphStatusBuilding {
		if newEvent.EventType == existingEvent.EventType {
			return nil
		}
		ws.logger.Debug("building event, create new event for path: %s, type: %s", existingEvent.SourceFilePath, newEvent.EventType)
		return ws.eventRepo.CreateEvent(newEvent)
	}

	ws.logger.Debug("update existing event for path: %s, type: %s, embedding status: %s", existingEvent.SourceFilePath, newEvent.EventType, model.EmbeddingStatusInitStr)
	// 更新事件类型和其他必要信息
	updateEvent := &model.Event{
		ID:              existingEvent.ID,
		EventType:       newEvent.EventType,
		TargetFilePath:  newEvent.TargetFilePath,
		EmbeddingStatus: model.EmbeddingStatusInit,
		CodegraphStatus: model.CodegraphStatusSuccess,
	}

	// 调用 repository 更新事件
	return ws.eventRepo.UpdateEvent(updateEvent)
}
