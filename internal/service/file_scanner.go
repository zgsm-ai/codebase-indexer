package service

import (
	"fmt"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
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
		if workspace.Active {
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

	// 生成事件
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
			ws.logger.Error("failed to create event: %v", err)
			continue
		}

		events = append(events, event)
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
	case "add": // utils.FILE_STATUS_ADDED
		return model.EventTypeAddFile
	case "modify": // utils.FILE_STATUS_MODIFIED
		return model.EventTypeModifyFile
	case "delete": // utils.FILE_STATUS_DELETED
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
	workspace.FileNum = fileNum

	// 更新工作区信息
	err = ws.workspaceRepo.UpdateWorkspace(workspace)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}
