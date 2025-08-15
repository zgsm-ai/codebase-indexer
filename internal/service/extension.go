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
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// ExtensionService 处理扩展接口相关的业务逻辑
type ExtensionService interface {
	// RegisterCodebase 注册代码库
	RegisterCodebase(ctx context.Context, clientID, workspacePath, workspaceName string) ([]*config.CodebaseConfig, error)

	// UnregisterCodebase 注销代码库
	UnregisterCodebase(ctx context.Context, clientID, workspacePath, workspaceName string) ([]*config.CodebaseConfig, error)

	// SyncCodebase 同步代码库
	SyncCodebase(ctx context.Context, clientID, workspacePath, workspaceName string, filePaths []string) ([]*config.CodebaseConfig, error)

	// UpdateSyncConfig 更新同步配置
	UpdateSyncConfig(ctx context.Context, clientID, serverEndpoint, accessToken string) error

	// CheckIgnoreFiles 检查文件是否应该被忽略
	CheckIgnoreFiles(ctx context.Context, clientID, workspacePath, workspaceName string, filePaths []string) (*CheckIgnoreResult, error)

	// SwitchIndex 切换索引状态
	SwitchIndex(ctx context.Context, workspacePath, switchStatus, clientID string) error

	// PublishEvents 发布工作区事件
	PublishEvents(ctx context.Context, workspacePath, clientID string, events []dto.WorkspaceEvent) (int, error)

	// TriggerIndex 触发索引构建
	TriggerIndex(ctx context.Context, workspacePath, indexType, clientID string) error

	// GetIndexStatus 获取索引状态
	GetIndexStatus(ctx context.Context, clientID, workspacePath string) (*dto.IndexStatusResponse, error)
}

// CheckIgnoreResult 检查结果
type CheckIgnoreResult struct {
	ShouldIgnore bool
	Reason       string
	IgnoredFiles []string
}

// NewExtensionService 创建新的扩展接口服务
func NewExtensionService(
	storage repository.StorageInterface,
	httpSync repository.SyncInterface,
	fileScanner repository.ScannerInterface,
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	codebaseEmbeddingRepo repository.EmbeddingFileRepository,
	codebaseService CodebaseService,
	logger logger.Logger,
) ExtensionService {
	return &extensionService{
		storage:               storage,
		httpSync:              httpSync,
		fileScanner:           fileScanner,
		workspaceRepo:         workspaceRepo,
		eventRepo:             eventRepo,
		codebaseEmbeddingRepo: codebaseEmbeddingRepo,
		codebaseService:       codebaseService,
		logger:                logger,
	}
}

type extensionService struct {
	storage               repository.StorageInterface
	httpSync              repository.SyncInterface
	fileScanner           repository.ScannerInterface
	workspaceRepo         repository.WorkspaceRepository
	eventRepo             repository.EventRepository
	codebaseEmbeddingRepo repository.EmbeddingFileRepository
	codebaseService       CodebaseService
	logger                logger.Logger
}

// RegisterCodebase 注册代码库
func (s *extensionService) RegisterCodebase(ctx context.Context, clientID, workspacePath, workspaceName string) ([]*config.CodebaseConfig, error) {
	s.logger.Info("registering codebase for client %s, path: %s", clientID, workspacePath)

	// 查找代码库配置
	codebaseConfigs, err := s.codebaseService.FindCodebasePaths(ctx, workspacePath, workspaceName)
	if err != nil {
		s.logger.Error("failed to find codebase paths: %v", err)
		return nil, fmt.Errorf("failed to find codebase paths: %w", err)
	}

	var registeredConfigs []*config.CodebaseConfig

	// 注册每个代码库
	for _, codebaseConfig := range codebaseConfigs {
		// 生成代码库ID
		codebaseID := utils.GenerateCodebaseID(codebaseConfig.CodebasePath)

		// 创建存储配置
		storageConfig := &config.CodebaseConfig{
			ClientID:     clientID,
			CodebaseId:   codebaseID,
			CodebaseName: codebaseConfig.CodebaseName,
			CodebasePath: codebaseConfig.CodebasePath,
			HashTree:     make(map[string]string),
			LastSync:     time.Time{},
			RegisterTime: time.Now(),
		}

		// 保存到存储
		if err := s.storage.SaveCodebaseConfig(storageConfig); err != nil {
			s.logger.Error("failed to save codebase config for %s: %v", codebaseConfig.CodebasePath, err)
			continue
		}

		registeredConfigs = append(registeredConfigs, storageConfig)
		s.logger.Info("registered codebase %s (%s) for client %s", codebaseConfig.CodebaseName, codebaseID, clientID)
	}

	return registeredConfigs, nil
}

// UnregisterCodebase 注销代码库
func (s *extensionService) UnregisterCodebase(ctx context.Context, clientID, workspacePath, workspaceName string) ([]*config.CodebaseConfig, error) {
	s.logger.Info("unregistering codebase for client %s, path: %s", clientID, workspacePath)

	// 查找代码库配置
	codebaseConfigs, err := s.codebaseService.FindCodebasePaths(ctx, workspacePath, workspaceName)
	if err != nil {
		s.logger.Error("failed to find codebase paths: %v", err)
		return nil, fmt.Errorf("failed to find codebase paths: %w", err)
	}

	var unregisteredConfigs []*config.CodebaseConfig

	// 注销每个代码库
	for _, codebaseConfig := range codebaseConfigs {
		codebaseID := utils.GenerateCodebaseID(codebaseConfig.CodebasePath)

		// 获取现有配置
		existingConfig, err := s.storage.GetCodebaseConfig(codebaseID)
		if err != nil {
			s.logger.Error("failed to get codebase config %s: %v", codebaseID, err)
			continue
		}

		// 检查是否属于该客户端
		if existingConfig.ClientID != clientID {
			s.logger.Warn("codebase %s does not belong to client %s", codebaseID, clientID)
			continue
		}

		// 从存储中删除
		if err := s.storage.DeleteCodebaseConfig(codebaseID); err != nil {
			s.logger.Error("failed to delete codebase config %s: %v", codebaseID, err)
			continue
		}

		// 创建已注销的配置信息
		unregisteredConfig := &config.CodebaseConfig{
			ClientID:     clientID,
			CodebaseId:   codebaseID,
			CodebaseName: codebaseConfig.CodebaseName,
			CodebasePath: codebaseConfig.CodebasePath,
		}

		unregisteredConfigs = append(unregisteredConfigs, unregisteredConfig)
		s.logger.Info("unregistered codebase %s (%s) for client %s", codebaseConfig.CodebaseName, codebaseID, clientID)
	}

	return unregisteredConfigs, nil
}

// SyncCodebase 同步代码库
func (s *extensionService) SyncCodebase(ctx context.Context, clientID, workspacePath, workspaceName string, filePaths []string) ([]*config.CodebaseConfig, error) {
	s.logger.Info("syncing codebase for client %s, path: %s", clientID, workspacePath)

	// 查找代码库配置
	configs, err := s.codebaseService.FindCodebasePaths(ctx, workspacePath, workspaceName)
	if err != nil {
		s.logger.Error("failed to find codebase paths: %v", err)
		return nil, fmt.Errorf("failed to find codebase paths: %w", err)
	}

	var syncedConfigs []*config.CodebaseConfig

	// 同步每个代码库
	for _, config := range configs {
		codebaseID := utils.GenerateCodebaseID(config.CodebasePath)

		// 获取存储中的配置
		storageConfig, err := s.storage.GetCodebaseConfig(codebaseID)
		if err != nil {
			s.logger.Error("failed to get codebase config %s: %v", codebaseID, err)
			continue
		}

		// 检查是否属于该客户端
		if storageConfig.ClientID != clientID {
			s.logger.Warn("codebase %s does not belong to client %s", codebaseID, clientID)
			continue
		}

		// 检查同步配置是否设置
		syncConfig := s.httpSync.GetSyncConfig()
		if syncConfig == nil || syncConfig.ServerURL == "" || syncConfig.Token == "" {
			s.logger.Warn("sync config not properly set for codebase %s", codebaseID)
			continue
		}

		// 获取服务器哈希树
		_, err = s.httpSync.FetchServerHashTree(config.CodebasePath)
		if err != nil {
			s.logger.Error("failed to fetch server hash tree for %s: %v", codebaseID, err)
			continue
		}

		// 更新最后同步时间
		storageConfig.LastSync = time.Now()
		if err := s.storage.SaveCodebaseConfig(storageConfig); err != nil {
			s.logger.Error("failed to update last sync time for %s: %v", codebaseID, err)
			continue
		}

		syncedConfigs = append(syncedConfigs, storageConfig)
		s.logger.Info("synced codebase %s (%s) for client %s", config.CodebaseName, codebaseID, clientID)
	}

	return syncedConfigs, nil
}

// UpdateSyncConfig 更新同步配置
func (s *extensionService) UpdateSyncConfig(ctx context.Context, clientID, serverEndpoint, accessToken string) error {
	s.logger.Info("updating sync config for client %s", clientID)

	// 更新同步器配置
	syncConfig := &config.SyncConfig{
		ClientId:  clientID,
		Token:     accessToken,
		ServerURL: serverEndpoint,
	}
	s.httpSync.SetSyncConfig(syncConfig)

	s.logger.Info("updated sync config for client %s with server %s", clientID, serverEndpoint)
	return nil
}

// CheckIgnoreFiles 检查文件是否应该被忽略
func (s *extensionService) CheckIgnoreFiles(ctx context.Context, clientID, workspacePath, workspaceName string, filePaths []string) (*CheckIgnoreResult, error) {
	s.logger.Info("checking ignore files for client %s, workspace: %s, files: %d", clientID, workspacePath, len(filePaths))
	// 检查每个文件
	maxFileSizeKB := s.fileScanner.GetScannerConfig().MaxFileSizeKB
	maxFileSize := int64(maxFileSizeKB * 1024)
	ignore := s.fileScanner.LoadIgnoreRules(workspacePath)
	if ignore == nil {
		s.logger.Warn("no ignore file found for codebase: %s", workspacePath)
		return &CheckIgnoreResult{
			ShouldIgnore: false,
			Reason:       "no ignore file found",
			IgnoredFiles: []string{},
		}, nil
	}
	fileInclude := s.fileScanner.LoadIncludeFiles()
	fileIncludeMap := utils.StringSlice2Map(fileInclude)

	for _, filePath := range filePaths {
		// Check if the file is in this codebase
		relPath, err := filepath.Rel(workspacePath, filePath)
		if err != nil {
			s.logger.Debug("file path %s is not in codebase %s: %v", filePath, workspacePath, err)
			continue
		}

		// Check file size and ignore rules
		checkPath := relPath
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			s.logger.Warn("failed to get file info: %s, %v", filePath, err)
			continue
		}

		// If directory, append "/" and skip size check
		if fileInfo.IsDir() {
			checkPath = relPath + "/"
		} else if fileInfo.Size() > maxFileSize {
			// For regular files, check size limit
			fileSizeKB := float64(fileInfo.Size()) / 1024
			s.logger.Info("file size exceeded limit: %s (%.2fKB)", filePath, fileSizeKB)
			return &CheckIgnoreResult{
				ShouldIgnore: false,
				Reason:       fmt.Sprintf("file size exceeded limit: %s (%.2fKB)", filePath, fileSizeKB),
				IgnoredFiles: []string{filePath},
			}, nil
		}

		// Check ignore rules
		if ignore.MatchesPath(checkPath) {
			s.logger.Info("ignore file found: %s in codebase %s", checkPath, workspacePath)
			return &CheckIgnoreResult{
				ShouldIgnore: false,
				Reason:       "ignore file found:" + filePath,
				IgnoredFiles: []string{filePath},
			}, nil
		}

		if len(fileIncludeMap) > 0 {
			if _, ok := fileIncludeMap[filePath]; !ok {
				s.logger.Info("file not included: %s in codebase %s", filePath, workspacePath)
				return &CheckIgnoreResult{
					ShouldIgnore: true,
					Reason:       "file not included:" + filePath,
					IgnoredFiles: []string{filePath},
				}, nil
			}
		}
	}

	s.logger.Info("no ignored files found, numFiles: %d", len(filePaths))
	return &CheckIgnoreResult{
		ShouldIgnore: false,
		Reason:       "no ignored files found",
		IgnoredFiles: []string{},
	}, nil
}

func (s *extensionService) SwitchIndex(ctx context.Context, workspacePath, switchStatus, clientID string) error {
	codebaseEnv := s.storage.GetCodebaseEnv()
	if codebaseEnv == nil {
		codebaseEnv = &config.CodebaseEnv{
			Switch: dto.SwitchOn,
		}
	}
	codebaseEnv.Switch = switchStatus
	err := s.storage.SaveCodebaseEnv(codebaseEnv)
	if err != nil {
		return fmt.Errorf("failed to save codebase env: %v", err)
	}

	// 检查工作空间是否存在
	workspace, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// 更新工作空间的索引开关状态
	active := "false"
	if switchStatus == dto.SwitchOn {
		active = "true"
	}

	if workspace.Active == active {
		s.logger.Info("workspace %s is already %s", workspacePath, switchStatus)
		return nil
	}

	// 更新工作空间状态
	updateWorkspace := &model.Workspace{
		WorkspacePath: workspace.WorkspacePath,
		Active:        active,
	}
	if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	if active == "true" {
		// 创建代码库配置
		if err := s.createCodebaseConfig(workspacePath, clientID); err != nil {
			s.logger.Error("failed to create codebase config: %v", err)
		}

		// 创建代码库嵌入配置
		if err := s.createCodebaseEmbeddingConfig(workspacePath, clientID); err != nil {
			s.logger.Error("failed to create codebase embedding config: %v", err)
		}

		// 事务处理新增open_worksapce事件，删除其余非进行中状态事件
		// 定义非进行中状态
		nonProcessingEmbeddingStatuses := []int{
			model.EmbeddingStatusInit,
			model.EmbeddingStatusUploadFailed,
			model.EmbeddingStatusBuildFailed,
			model.EmbeddingStatusSuccess,
		}

		nonProcessingCodegraphStatuses := []int{
			model.CodegraphStatusInit,
			model.CodegraphStatusFailed,
			model.CodegraphStatusSuccess,
		}

		// 获取所有非进行中状态的事件
		nonProcessingEvents, err := s.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(
			[]string{},
			[]string{workspacePath},
			-1, // 足够大的限制值
			false,
			nonProcessingEmbeddingStatuses,
			nonProcessingCodegraphStatuses,
		)
		if err != nil {
			s.logger.Error("failed to get non-processing open workspace events: %v", err)
		} else {
			// 删除所有非进行中状态的事件
			deleteEventIds := []int64{}
			for _, event := range nonProcessingEvents {
				deleteEventIds = append(deleteEventIds, event.ID)
			}
			if len(deleteEventIds) > 0 {
				if err := s.eventRepo.BatchDeleteEvents(deleteEventIds); err != nil {
					s.logger.Error("failed to batch delete non-processing open workspace events: %v", err)
				} else {
					s.logger.Debug("batch deleted %d non-processing open workspace events for workspace: %s", len(deleteEventIds), workspacePath)
				}
			}
		}

		// 创建新的open_workspace事件
		newOpenWorkspaceEvent := &model.Event{
			WorkspacePath:   workspacePath,
			EventType:       model.EventTypeOpenWorkspace,
			SourceFilePath:  "",
			TargetFilePath:  "",
			SyncId:          "",
			EmbeddingStatus: model.EmbeddingStatusSuccess,
			CodegraphStatus: model.CodegraphStatusInit,
		}

		if err := s.eventRepo.CreateEvent(newOpenWorkspaceEvent); err != nil {
			s.logger.Error("failed to create open workspace event: %v", err)
		} else {
			s.logger.Info("created new open workspace event for workspace: %s", workspacePath)
		}
	} else {
		// 删除非进行中状态事件
		// 定义非进行中状态
		nonProcessingEmbeddingStatuses := []int{
			model.EmbeddingStatusInit,
			model.EmbeddingStatusUploadFailed,
			model.EmbeddingStatusBuildFailed,
			model.EmbeddingStatusSuccess,
		}

		nonProcessingCodegraphStatuses := []int{
			model.CodegraphStatusInit,
			model.CodegraphStatusFailed,
			model.CodegraphStatusSuccess,
		}

		// 获取所有非进行中状态的事件
		nonProcessingEvents, err := s.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(
			[]string{}, // 空字符串表示所有事件类型
			[]string{workspacePath},
			-1, // 足够大的限制值
			false,
			nonProcessingEmbeddingStatuses,
			nonProcessingCodegraphStatuses,
		)
		if err != nil {
			s.logger.Error("failed to get non-processing events for deletion: %v", err)
		} else {
			// 删除所有非进行中状态的事件
			deleteEventIds := []int64{}
			for _, event := range nonProcessingEvents {
				deleteEventIds = append(deleteEventIds, event.ID)
			}
			if len(deleteEventIds) > 0 {
				if err := s.eventRepo.BatchDeleteEvents(deleteEventIds); err != nil {
					s.logger.Error("failed to batch delete non-processing events: %v", err)
				} else {
					s.logger.Debug("batch deleted non-processing events for workspace: %s", workspacePath)
				}
			}
		}
	}

	s.logger.Info("index switch for workspace %s set to %s", workspacePath, switchStatus)
	return nil
}

// PublishEvents 发布工作区事件
func (s *extensionService) PublishEvents(ctx context.Context, workspacePath, clientID string, events []dto.WorkspaceEvent) (int, error) {
	s.logger.Info("publishing %d events for workspace: %s", len(events), workspacePath)

	s.handleWorkspaceEvents(workspacePath, clientID, events)
	successCount := s.processEvents(workspacePath, events)

	s.logger.Info("successfully published %d/%d events for workspace: %s",
		successCount, len(events), workspacePath)

	return successCount, nil
}

// processEvents 处理工作区事件
func (s *extensionService) processEvents(workspacePath string, events []dto.WorkspaceEvent) int {
	successCount := 0

	for _, event := range events {
		sourcePath := event.SourcePath
		targetPath := event.TargetPath
		sourceRelPath, err := filepath.Rel(workspacePath, sourcePath)
		if err != nil {
			s.logger.Error("failed to get relative path for source path: %s, error: %v", sourcePath, err)
			continue
		}
		targetRelPath, err := filepath.Rel(workspacePath, targetPath)
		if err != nil {
			s.logger.Error("failed to get relative path for target path: %s, error: %v", targetPath, err)
			continue
		}
		event.SourcePath = sourceRelPath
		event.TargetPath = targetRelPath

		if s.shouldSkipEvent(event) {
			continue
		}

		// 尝试更新现有事件
		if updated := s.tryUpdateExistingEvent(workspacePath, event); updated {
			successCount++
			continue
		}

		// 创建新事件
		if s.createNewEvent(workspacePath, event) {
			successCount++
		}
	}

	return successCount
}

// shouldSkipEvent 判断是否应该跳过事件
func (s *extensionService) shouldSkipEvent(event dto.WorkspaceEvent) bool {
	return event.EventType == "" || event.EventType == model.EventTypeCloseWorkspace
}

// tryUpdateExistingEvent 尝试更新现有事件
func (s *extensionService) tryUpdateExistingEvent(workspacePath string, event dto.WorkspaceEvent) bool {
	existingEvent, err := s.eventRepo.GetLatestEventByWorkspaceAndSourcePath(workspacePath, event.SourcePath)
	if err != nil {
		s.logger.Error("failed to get existing events: %v", err)
		return false
	}

	// 检查是否存在相同workspace和sourcePath的记录，且embeddingStatus和codegraphStatus都不为执行中状态
	if existingEvent == nil ||
		existingEvent.EmbeddingStatus == model.EmbeddingStatusUploading ||
		existingEvent.EmbeddingStatus == model.CodegraphStatusBuilding ||
		existingEvent.CodegraphStatus == model.CodegraphStatusBuilding {
		return false
	}

	// 修改eventType为请求参数中的eventType
	updateEvent := &model.Event{
		ID:              existingEvent.ID,
		EventType:       event.EventType,
		TargetFilePath:  event.TargetPath,
		EmbeddingStatus: model.EmbeddingStatusInit,
		CodegraphStatus: model.CodegraphStatusInit,
	}
	if event.EventType == model.EventTypeOpenWorkspace {
		updateEvent.EmbeddingStatus = model.EmbeddingStatusSuccess
	}

	// 更新事件记录
	if err := s.eventRepo.UpdateEvent(updateEvent); err != nil {
		s.logger.Error("failed to update event: %v", err)
		return false
	}

	s.logger.Debug("updated event: type=%s, source=%s, target=%s",
		event.EventType, event.SourcePath, event.TargetPath)
	return true
}

// createNewEvent 创建新事件
func (s *extensionService) createNewEvent(workspacePath string, event dto.WorkspaceEvent) bool {
	eventModel := &model.Event{
		WorkspacePath:   workspacePath,
		EventType:       event.EventType,
		SourceFilePath:  event.SourcePath,
		TargetFilePath:  event.TargetPath,
		SyncId:          "",                        // 暂时为空，后续可以生成
		EmbeddingStatus: model.EmbeddingStatusInit, // 初始状态
		CodegraphStatus: model.CodegraphStatusInit, // 初始状态
	}
	if event.EventType == model.EventTypeOpenWorkspace {
		eventModel.EmbeddingStatus = model.EmbeddingStatusSuccess
	}

	// 保存事件到数据库
	if err := s.eventRepo.CreateEvent(eventModel); err != nil {
		s.logger.Error("failed to create event: %v", err)
		return false
	}

	s.logger.Debug("created event: type=%s, source=%s, target=%s",
		event.EventType, event.SourcePath, event.TargetPath)
	return true
}

// handleWorkspaceEvents 处理工作区相关事件（打开/关闭工作区）
func (s *extensionService) handleWorkspaceEvents(workspacePath, clientID string, events []dto.WorkspaceEvent) {
	for _, event := range events {
		switch event.EventType {
		case model.EventTypeOpenWorkspace:
			s.handleOpenWorkspaceEvent(workspacePath, clientID)
		case model.EventTypeCloseWorkspace:
			s.handleCloseWorkspaceEvent(workspacePath)
		}
	}
}

// handleOpenWorkspaceEvent 处理打开工作区事件
func (s *extensionService) handleOpenWorkspaceEvent(workspacePath, clientID string) {
	// 检查工作区是否已存在
	_, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		// 工作区不存在，创建新的工作区
		s.createAndActivateWorkspace(workspacePath, "true")
	} else {
		// 激活工作区
		s.activateWorkspace(workspacePath)
	}

	// 创建代码库配置
	if err := s.createCodebaseConfig(workspacePath, clientID); err != nil {
		s.logger.Error("failed to create codebase config: %v", err)
	}

	// 创建代码库嵌入配置
	if err := s.createCodebaseEmbeddingConfig(workspacePath, clientID); err != nil {
		s.logger.Error("failed to create codebase embedding config: %v", err)
	}
}

// handleCloseWorkspaceEvent 处理关闭工作区事件
func (s *extensionService) handleCloseWorkspaceEvent(workspacePath string) {
	// 检查工作区是否已存在
	_, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		// 工作区不存在，创建新的工作区
		s.createAndActivateWorkspace(workspacePath, "false")
	} else {
		// 关闭工作区
		s.deactivateWorkspace(workspacePath)
	}
}

// createAndActivateWorkspace 创建并激活/关闭工作区
func (s *extensionService) createAndActivateWorkspace(workspacePath, active string) {
	workspaceName := filepath.Base(workspacePath)
	newWorkspace := &model.Workspace{
		WorkspaceName:    workspaceName,
		WorkspacePath:    workspacePath,
		Active:           active,
		FileNum:          0,
		EmbeddingFileNum: 0,
		EmbeddingTs:      0,
		CodegraphFileNum: 0,
		CodegraphTs:      0,
	}

	if err := s.workspaceRepo.CreateWorkspace(newWorkspace); err != nil {
		s.logger.Error("failed to create workspace: %v", err)
	} else {
		s.logger.Info("created new workspace: %s with active=%s", workspacePath, active)
	}
}

// activateWorkspace 激活工作区
func (s *extensionService) activateWorkspace(workspacePath string) {
	updateWorkspace := &model.Workspace{
		WorkspacePath: workspacePath,
		Active:        "true",
	}
	if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
		s.logger.Error("failed to activate workspace: %v", err)
	}
}

// deactivateWorkspace 关闭工作区
func (s *extensionService) deactivateWorkspace(workspacePath string) {
	updateWorkspace := &model.Workspace{
		WorkspacePath: workspacePath,
		Active:        "false",
	}
	if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
		s.logger.Error("failed to deactivate workspace: %v", err)
	}
}

func (s *extensionService) createCodebaseConfig(workspacePath, clientID string) error {
	workspaceName := filepath.Base(workspacePath)
	codebaseID := utils.GenerateCodebaseID(workspacePath)

	codebaseConfig, err := s.storage.GetCodebaseConfig(codebaseID)
	if err != nil {
		// 如果配置不存在，创建新的配置
		codebaseConfig = &config.CodebaseConfig{
			ClientID:     clientID,
			CodebaseId:   codebaseID,
			CodebaseName: workspaceName,
			CodebasePath: workspacePath,
			HashTree:     make(map[string]string),
			RegisterTime: time.Now(),
		}
	} else {
		codebaseConfig.RegisterTime = time.Now()
	}

	// 保存到存储
	if err := s.storage.SaveCodebaseConfig(codebaseConfig); err != nil {
		s.logger.Error("failed to save codebase config for %s: %v", workspacePath, err)
		return fmt.Errorf("failed to save codebase config: %w", err)
	}

	s.logger.Info("created codebase config for %s (%s)", workspaceName, codebaseID)
	return nil
}

func (s *extensionService) createCodebaseEmbeddingConfig(workspacePath, clientID string) error {
	workspaceName := filepath.Base(workspacePath)
	codebaseID := utils.GenerateCodebaseEmbeddingID(workspacePath)

	_, err := s.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseID)
	if err == nil {
		s.logger.Info("codebase embedding config for %s already exists", codebaseID)
		return nil
	}

	codebaseEmbeddingConfig := &config.CodebaseEmbeddingConfig{
		ClientID:     clientID,
		CodebaseId:   codebaseID,
		CodebaseName: workspaceName,
		CodebasePath: workspacePath,
		HashTree:     make(map[string]string),
	}

	// 保存到存储
	if err := s.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig); err != nil {
		s.logger.Error("failed to save codebase embedding config for %s: %v", workspacePath, err)
		return fmt.Errorf("failed to save codebase embedding config: %w", err)
	}

	s.logger.Info("created codebase embedding config for %s (%s)", workspaceName, codebaseID)
	return nil
}

// TriggerIndex 触发索引构建
func (s *extensionService) TriggerIndex(ctx context.Context, workspacePath, indexType, clientID string) error {
	// 检查工作区是否已存在
	workspace, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		// 工作区不存在，创建新的工作区
		s.logger.Info("workspace not found, creating new workspace: %s", workspacePath)
		workspaceName := filepath.Base(workspacePath)
		newWorkspace := &model.Workspace{
			WorkspaceName:    workspaceName,
			WorkspacePath:    workspacePath,
			Active:           "true", // 激活工作区
			FileNum:          0,
			EmbeddingFileNum: 0,
			EmbeddingTs:      0,
			CodegraphFileNum: 0,
			CodegraphTs:      0,
		}

		if err := s.workspaceRepo.CreateWorkspace(newWorkspace); err != nil {
			s.logger.Error("failed to create workspace: %v", err)
			return fmt.Errorf("failed to create workspace: %w", err)
		}

		s.logger.Info("created new workspace: %s", workspacePath)
	} else {
		// 工作区已存在，更新 active 状态
		if workspace.Active != "true" {
			updateWorkspace := &model.Workspace{
				ID:            workspace.ID,
				WorkspacePath: workspace.WorkspacePath,
				Active:        "true",
			}
			if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
				s.logger.Error("failed to update workspace active status: %v", err)
				return fmt.Errorf("failed to update workspace: %w", err)
			}
			s.logger.Info("updated workspace active status to true: %s", workspacePath)
		}
	}

	if err := s.createCodebaseConfig(workspacePath, clientID); err != nil {
		s.logger.Error("failed to create codebase config: %v", err)
		return fmt.Errorf("failed to create codebase config: %w", err)
	}

	if err := s.createCodebaseEmbeddingConfig(workspacePath, clientID); err != nil {
		s.logger.Error("failed to create codebase embedding config: %v", err)
		return fmt.Errorf("failed to create codebase embedding config: %w", err)
	}

	// 判断open_workspace事件是否存在非进行中状态，若不存在则创建
	existingOpenEvents, err := s.eventRepo.GetEventsByWorkspaceAndType(workspacePath, []string{model.EventTypeOpenWorkspace}, 1, true)
	if err != nil {
		s.logger.Error("failed to get existing open workspace events: %v", err)
		return fmt.Errorf("failed to get existing open workspace events: %w", err)
	}

	shouldCreateEvent := true
	if len(existingOpenEvents) > 0 {
		// 检查是否存在非进行中状态的open_workspace事件
		for _, event := range existingOpenEvents {
			if event.EmbeddingStatus != model.EmbeddingStatusUploading &&
				event.EmbeddingStatus != model.EmbeddingStatusBuilding &&
				event.CodegraphStatus != model.CodegraphStatusBuilding {
				// 存在非进行中状态的事件，不需要创建新事件
				shouldCreateEvent = false
				s.logger.Info("found existing non-processing open workspace event for workspace: %s", workspacePath)
				break
			}
		}
	}

	var eventModel *model.Event
	if shouldCreateEvent {
		// 创建打开工作区事件
		eventModel = &model.Event{
			WorkspacePath:  workspacePath,
			EventType:      model.EventTypeOpenWorkspace,
			SourceFilePath: "",
			TargetFilePath: "",
			SyncId:         "", // 暂时为空，后续可以生成
		}
		switch indexType {
		case dto.IndexTypeEmbedding:
			eventModel.EmbeddingStatus = model.EmbeddingStatusInit
			eventModel.CodegraphStatus = model.CodegraphStatusSuccess
		case dto.IndexTypeCodegraph:
			eventModel.EmbeddingStatus = model.EmbeddingStatusSuccess
			eventModel.CodegraphStatus = model.CodegraphStatusInit
		default:
			eventModel.EmbeddingStatus = model.EmbeddingStatusInit
			eventModel.CodegraphStatus = model.CodegraphStatusInit
		}
		// 保存事件到数据库
		if err := s.eventRepo.CreateEvent(eventModel); err != nil {
			s.logger.Error("failed to create open workspace event: %v", err)
			return fmt.Errorf("failed to create event: %w", err)
		}
	}

	// 事务处理删除其他非进行中状态事件
	// 获取所有需要删除的事件ID
	nonProcessingEmbeddingStatuses := []int{
		model.EmbeddingStatusInit,
		model.EmbeddingStatusUploadFailed,
		model.EmbeddingStatusBuildFailed,
		model.EmbeddingStatusSuccess,
	}

	nonProcessingCodegraphStatuses := []int{
		model.CodegraphStatusInit,
		model.CodegraphStatusFailed,
		model.CodegraphStatusSuccess,
	}

	// 获取所有非进行中状态的事件（排除新创建的事件）
	eventsToDelete, err := s.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(
		[]string{},
		[]string{workspacePath},
		-1, // 足够大的限制值
		false,
		nonProcessingEmbeddingStatuses,
		nonProcessingCodegraphStatuses,
	)
	if err != nil {
		s.logger.Error("failed to get non-processing events for deletion: %v", err)
		return fmt.Errorf("failed to get non-processing events: %w", err)
	}

	// 删除这些事件（跳过新创建的事件）
	deleteEventIds := []int64{}
	for _, event := range eventsToDelete {
		if shouldCreateEvent && event.ID == eventModel.ID {
			continue // 跳过新创建的事件
		}
		deleteEventIds = append(deleteEventIds, event.ID)
	}

	if len(deleteEventIds) > 0 {
		if err := s.eventRepo.BatchDeleteEvents(deleteEventIds); err != nil {
			s.logger.Error("failed to batch delete non-processing events: %v", err)
			return fmt.Errorf("failed to batch delete non-processing events: %w", err)
		}
	}

	s.logger.Info("successfully triggered index build for workspace: %s", workspacePath)
	return nil
}

// GetIndexStatus 获取索引状态
func (s *extensionService) GetIndexStatus(ctx context.Context, clientID, workspacePath string) (*dto.IndexStatusResponse, error) {
	s.logger.Info("getting index status for client %s, workspace: %s", clientID, workspacePath)

	// 获取工作区信息
	workspace, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		s.logger.Error("failed to get workspace %s: %v", workspacePath, err)
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	data := dto.IndexStatusData{}

	// 判断工作区是否激活
	if workspace.Active != "true" {
		// 如果工作区未激活，状态为 pending
		data.Embedding = dto.IndexStatus{
			Status:       "pending",
			Process:      0,
			TotalFiles:   workspace.FileNum,
			TotalSucceed: 0,
			TotalFailed:  0,
			ProcessTs:    0,
		}
		data.Codegraph = dto.IndexStatus{
			Status:       "pending",
			Process:      0,
			TotalFiles:   workspace.FileNum,
			TotalSucceed: 0,
			TotalFailed:  0,
			ProcessTs:    0,
		}
	} else {
		// 获取工作区的所有事件
		events, err := s.eventRepo.GetEventsByWorkspaceForDeduplication(workspace.WorkspacePath)
		if err != nil {
			s.logger.Warn("failed to get events for workspace %s: %v", workspace.WorkspacePath, err)
		}
		// 如果工作区已激活，根据事件记录计算状态
		data.Embedding = s.calculateEmbeddingStatus(workspace, events)
		data.Codegraph = s.calculateCodegraphStatus(workspace, events)
	}

	// 构建响应
	response := &dto.IndexStatusResponse{
		Code:    "0",
		Message: "ok",
		Data:    data,
	}

	s.logger.Info("successfully retrieved index status for workspace: %s", workspacePath)
	return response, nil
}

// calculateEmbeddingStatus 计算 embedding 状态
func (s *extensionService) calculateEmbeddingStatus(workspace *model.Workspace, events []*model.Event) dto.IndexStatus {

	status := dto.IndexStatus{
		TotalFiles:   workspace.FileNum,
		TotalSucceed: workspace.EmbeddingFileNum,
		ProcessTs:    workspace.EmbeddingTs,
	}

	// 计算进度
	if workspace.FileNum > 0 {
		status.Process = float32(workspace.EmbeddingFileNum) / float32(workspace.FileNum) * 100
		if status.Process > 100 { // 进度不能超过100%
			status.Process = 100
		}
	} else {
		status.Process = 0
	}

	// 计算失败文件数
	failedFilePaths := strings.Split(workspace.EmbeddingFailedFilePaths, ",")
	totalFailed := 0
	if len(failedFilePaths) > 0 {
		totalFailed = len(failedFilePaths)
	}

	// 统计各状态的 embedding 事件数
	initCount := 0
	uploadingCount := 0
	buildingCount := 0
	uploadFailedCount := 0
	buildFailedCount := 0
	successCount := 0

	for _, event := range events {
		switch event.EmbeddingStatus {
		case model.EmbeddingStatusInit:
			initCount++
		case model.EmbeddingStatusUploading:
			uploadingCount++
		case model.EmbeddingStatusBuilding:
			buildingCount++
		case model.EmbeddingStatusUploadFailed:
			uploadFailedCount++
		case model.EmbeddingStatusBuildFailed:
			buildFailedCount++
		case model.EmbeddingStatusSuccess:
			successCount++
		}
	}

	// 判断状态
	// 存在初始或进行中状态事件时，状态为 running
	if initCount > 0 || uploadingCount > 0 || buildingCount > 0 {
		status.Status = dto.ProcessStatusRunning
	} else if uploadFailedCount > 0 || buildFailedCount > 0 {
		// 存在失败状态时，判断比较 process 和配置中的百分比阈值
		embeddingSuccessPercent := config.GetClientConfig().Sync.EmbeddingSuccessPercent
		if status.Process < embeddingSuccessPercent {
			status.Status = dto.ProcessStatusFailed
			status.TotalFailed = totalFailed
			status.FailedFiles = failedFilePaths
			status.FailedReason = workspace.EmbeddingMessage
		} else {
			status.Status = dto.ProcessStatusSuccess
		}
	} else {
		// 其他情况返回 success
		status.Status = dto.ProcessStatusSuccess
	}

	return status
}

// calculateCodegraphStatus 计算 codegraph 状态
func (s *extensionService) calculateCodegraphStatus(workspace *model.Workspace, events []*model.Event) dto.IndexStatus {

	status := dto.IndexStatus{
		TotalFiles:   workspace.FileNum,
		TotalSucceed: workspace.CodegraphFileNum,
		ProcessTs:    workspace.CodegraphTs,
	}

	// 计算进度
	if workspace.FileNum > 0 {
		status.Process = float32(workspace.CodegraphFileNum) / float32(workspace.FileNum) * 100
		if status.Process > 100 { // 进度不能超过100%
			status.Process = 100
		}
	} else {
		status.Process = 0
	}

	// 计算失败文件数
	failedFilePaths := strings.Split(workspace.EmbeddingFailedFilePaths, ",")
	totalFailed := 0
	if len(failedFilePaths) > 0 {
		totalFailed = len(failedFilePaths)
	}

	// 统计各状态的 codegraph 事件数
	initCount := 0
	buildingCount := 0
	failedCount := 0
	successCount := 0

	for _, event := range events {
		switch event.CodegraphStatus {
		case model.CodegraphStatusInit:
			initCount++
		case model.CodegraphStatusBuilding:
			buildingCount++
		case model.CodegraphStatusFailed:
			failedCount++
		case model.CodegraphStatusSuccess:
			successCount++
		}
	}

	// 判断状态
	// 存在初始或进行中状态事件时，状态为 running
	if initCount > 0 || buildingCount > 0 {
		status.Status = dto.ProcessStatusRunning
	} else if failedCount > 0 {
		// 存在失败状态时，判断比较 process 和配置中的百分比阈值
		codegraphSuccessPercent := config.GetClientConfig().Sync.CodegraphSuccessPercent
		if status.Process < codegraphSuccessPercent {
			status.Status = dto.ProcessStatusFailed
			status.TotalFailed = totalFailed
			status.FailedFiles = failedFilePaths
			status.FailedReason = workspace.EmbeddingMessage
		} else {
			status.Status = dto.ProcessStatusSuccess
		}
	} else {
		// 其他情况返回 success
		status.Status = dto.ProcessStatusSuccess
	}

	return status
}
