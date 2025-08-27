package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/codegraph/types"
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
	GetIndexStatus(ctx context.Context, workspacePath string) (*dto.IndexStatusResponse, error)
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
	scanService FileScanService,
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
		scanService:           scanService,
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
	scanService           FileScanService
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
	for _, codebaseConfig := range configs {
		codebaseID := utils.GenerateCodebaseID(codebaseConfig.CodebasePath)

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
		authInfo := config.GetAuthInfo()
		if authInfo.ServerURL == "" || authInfo.Token == "" {
			s.logger.Warn("auth info not properly set for codebase %s", codebaseID)
			continue
		}

		// 获取服务器哈希树
		_, err = s.httpSync.FetchServerHashTree(codebaseConfig.CodebasePath)
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
		s.logger.Info("synced codebase %s (%s) for client %s", codebaseConfig.CodebaseName, codebaseID, clientID)
	}

	return syncedConfigs, nil
}

// UpdateSyncConfig 更新同步配置
func (s *extensionService) UpdateSyncConfig(ctx context.Context, clientID, serverEndpoint, accessToken string) error {
	// 更新同步器配置
	syncConfig := &config.SyncConfig{
		ClientId:  clientID,
		Token:     accessToken,
		ServerURL: serverEndpoint,
	}
	s.httpSync.SetSyncConfig(syncConfig)

	s.logger.Info("updated sync config for client %s with server %s and access token %s", clientID, serverEndpoint, accessToken)
	return nil
}

// CheckIgnoreFiles 检查文件是否应该被忽略
func (s *extensionService) CheckIgnoreFiles(ctx context.Context, clientID, workspacePath, workspaceName string, filePaths []string) (*CheckIgnoreResult, error) {
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
			// Check ignore rules
			if ignore.MatchesPath(checkPath) {
				s.logger.Info("ignore dir found: %s in codebase %s", checkPath, workspacePath)
				return &CheckIgnoreResult{
					ShouldIgnore: false,
					Reason:       "ignore dir found: " + filePath,
					IgnoredFiles: []string{filePath},
				}, nil
			}
			continue
		}

		if fileInfo.Size() > maxFileSize {
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
				Reason:       "ignore file found: " + filePath,
				IgnoredFiles: []string{filePath},
			}, nil
		}

		if len(fileIncludeMap) > 0 {
			fileExt := filepath.Ext(filePath)
			if _, ok := fileIncludeMap[fileExt]; !ok {
				s.logger.Info("file not included: %s in codebase %s", filePath, workspacePath)
				return &CheckIgnoreResult{
					ShouldIgnore: true,
					Reason:       "file not included: " + filePath,
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

// SwitchIndex 索引开关切换
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

	if active == "true" {
		// 创建代码库配置
		fileNum, err := s.createCodebaseConfig(workspacePath, clientID)
		if err != nil {
			s.logger.Error("failed to create codebase config: %v", err)
		}
		updateWorkspace.FileNum = fileNum

		// 创建代码库嵌入配置
		if err := s.initCodebaseEmbeddingConfig(workspacePath, clientID); err != nil {
			s.logger.Error("failed to init codebase embedding config: %v", err)
		}

		if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
			return fmt.Errorf("failed to update workspace: %w", err)
		}

		// 事务处理新增open_worksapce事件，删除其余非进行中状态事件
		s.deleteNonProcessingEvents(workspacePath)

		// 创建新的open_workspace事件
		newOpenWorkspaceEvent := &model.Event{
			WorkspacePath:   workspacePath,
			EventType:       model.EventTypeOpenWorkspace,
			SourceFilePath:  "",
			TargetFilePath:  "",
			SyncId:          "",
			EmbeddingStatus: model.EmbeddingStatusInit,
			CodegraphStatus: model.CodegraphStatusInit,
		}

		if err := s.eventRepo.CreateEvent(newOpenWorkspaceEvent); err != nil {
			s.logger.Error("failed to create open workspace event: %v", err)
		} else {
			s.logger.Info("switch on, created new open workspace event for workspace: %s", workspacePath)
		}
		// switch on, 触发文件扫描
		go s.scanService.DetectFileChanges(workspace.WorkspacePath)
	} else {
		if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
			return fmt.Errorf("failed to update workspace: %w", err)
		}
		// 删除非进行中状态事件
		s.deleteNonProcessingEvents(workspacePath)
	}

	s.logger.Info("index switch for workspace %s set to %s", workspacePath, switchStatus)
	return nil
}

// deleteNonProcessingEvents 删除非进行中状态事件
func (s *extensionService) deleteNonProcessingEvents(workspacePath string) error {
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
	nonProcessingCodegraphEvents, err := s.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(
		[]string{}, // 空字符串表示所有事件类型
		[]string{workspacePath},
		-1, // 足够大的限制值
		false,
		nonProcessingEmbeddingStatuses,
		nonProcessingCodegraphStatuses,
	)
	if err != nil {
		s.logger.Warn("failed to get non-processing events for deletion: %v", err)
	} else {
		// 删除所有非进行中状态的事件
		deleteEventIds := []int64{}
		for _, event := range nonProcessingCodegraphEvents {
			deleteEventIds = append(deleteEventIds, event.ID)
		}
		if len(deleteEventIds) > 0 {
			if err := s.eventRepo.BatchDeleteEvents(deleteEventIds); err != nil {
				s.logger.Warn("failed to batch delete non-processing events: %v", err)
			} else {
				s.logger.Debug("batch deleted non-processing events for workspace: %s", workspacePath)
			}
		}
	}
	return nil
}

// PublishEvents 发布工作区事件
func (s *extensionService) PublishEvents(ctx context.Context, workspacePath, clientID string, events []dto.WorkspaceEvent) (int, error) {
	successCount := s.processEvents(workspacePath, clientID, events)

	s.logger.Info("successfully published %d/%d events for workspace: %s",
		successCount, len(events), workspacePath)

	return successCount, nil
}

// processEvents 处理工作区事件
func (s *extensionService) processEvents(workspacePath, clientID string, events []dto.WorkspaceEvent) int {
	successCount := 0
	extensionEventTypeMap := model.GetExtensionEventTypeMap()

	ignoreConfig := s.fileScanner.LoadIgnoreConfig(workspacePath)
	for _, event := range events {
		if !extensionEventTypeMap[event.EventType] {
			continue
		}

		if event.EventType == model.EventTypeCloseWorkspace {
			s.handleCloseWorkspaceEvent(workspacePath)
			successCount++
			break
		}

		if event.EventType == model.EventTypeOpenWorkspace {
			s.handleOpenWorkspaceEvent(workspacePath, clientID)
			event.SourcePath = ""
			event.TargetPath = ""
			// 尝试更新现有事件
			if updated := s.tryUpdateExistingEvent(workspacePath, event); updated {
				successCount++
				break
			}

			// 创建新事件
			if s.createNewEvent(workspacePath, event) {
				successCount++
			}
			// open_workspace 事件触发文件扫描
			go s.scanService.DetectFileChanges(workspacePath)
			break
		}

		sourcePath := event.SourcePath
		targetPath := event.TargetPath
		// 校验路径是否在工作空间内，并获取相对路径
		sourceRelPath, err := filepath.Rel(workspacePath, sourcePath)
		if err != nil {
			s.logger.Warn("failed to get relative path for source path: %s, error: %v", sourcePath, err)
			continue
		}
		event.SourcePath = sourceRelPath
		if targetPath != "" {
			targetRelPath, err := filepath.Rel(workspacePath, targetPath)
			if err != nil {
				s.logger.Warn("failed to get relative path for target path: %s, error: %v", targetPath, err)
				continue
			}
			event.TargetPath = targetRelPath
		}
		// 判断是否为忽略文件
		if event.EventType != model.EventTypeDeleteFile {
			fileInfo := &types.FileInfo{
				Path: sourcePath,
			}
			if event.EventType == model.EventTypeRenameFile {
				fileInfo.Path = targetPath
			}
			// 判断fileInfo.Path是文件还是目录
			if stat, err := os.Stat(fileInfo.Path); err == nil {
				if stat.IsDir() {
					fileInfo.IsDir = true
				} else {
					fileInfo.IsDir = false
					fileInfo.Size = stat.Size()
				}
			}
			ok, err := s.fileScanner.CheckIgnoreFile(ignoreConfig, workspacePath, fileInfo)
			if err != nil {
				s.logger.Warn("failed to check ignore file: %v", err)
				continue
			}
			if ok {
				continue
			}
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
		existingEvent.EmbeddingStatus == model.EmbeddingStatusBuilding ||
		existingEvent.CodegraphStatus == model.CodegraphStatusBuilding {
		return false
	}

	// 修改eventType为请求参数中的eventType
	updateEvent := &model.Event{
		ID:              existingEvent.ID,
		EventType:       event.EventType,
		SourceFilePath:  event.SourcePath,
		TargetFilePath:  event.TargetPath,
		EmbeddingStatus: model.EmbeddingStatusInit,
		CodegraphStatus: model.CodegraphStatusInit,
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

	// 保存事件到数据库
	if err := s.eventRepo.CreateEvent(eventModel); err != nil {
		s.logger.Error("failed to create event: %v", err)
		return false
	}

	s.logger.Debug("created event: type=%s, source=%s, target=%s",
		event.EventType, event.SourcePath, event.TargetPath)
	return true
}

// handleOpenWorkspaceEvent 处理打开工作区事件
func (s *extensionService) handleOpenWorkspaceEvent(workspacePath, clientID string) {
	// 创建代码库配置
	fileNum, err := s.createCodebaseConfig(workspacePath, clientID)
	if err != nil {
		s.logger.Error("failed to create codebase config: %v", err)
	}

	// 创建代码库嵌入配置
	if err := s.createCodebaseEmbeddingConfig(workspacePath, clientID); err != nil {
		s.logger.Error("failed to create codebase embedding config: %v", err)
	}

	// 检查工作区是否已存在
	_, err = s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		// 工作区不存在，创建新的工作区
		s.createAndActivateWorkspace(workspacePath, "true", fileNum)
	} else {
		// 激活工作区
		s.activateWorkspace(workspacePath, fileNum)
	}

	// 删除所有非进行中状态的事件
	s.deleteNonProcessingEvents(workspacePath)
}

// handleCloseWorkspaceEvent 处理关闭工作区事件
func (s *extensionService) handleCloseWorkspaceEvent(workspacePath string) {
	// 检查工作区是否已存在
	_, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		// 工作区不存在，创建新的工作区
		s.createAndActivateWorkspace(workspacePath, "false", 0)
	} else {
		// 关闭工作区
		s.deactivateWorkspace(workspacePath, 0)
	}

	// 删除所有非进行中状态的事件
	s.deleteNonProcessingEvents(workspacePath)
}

// createAndActivateWorkspace 创建并激活/关闭工作区
func (s *extensionService) createAndActivateWorkspace(workspacePath, active string, fileNum int) {
	workspaceName := filepath.Base(workspacePath)
	newWorkspace := &model.Workspace{
		WorkspaceName:    workspaceName,
		WorkspacePath:    workspacePath,
		Active:           active,
		FileNum:          fileNum,
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
func (s *extensionService) activateWorkspace(workspacePath string, fileNum int) {
	updateWorkspace := &model.Workspace{
		WorkspacePath: workspacePath,
		Active:        "true",
		FileNum:       fileNum,
	}
	if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
		s.logger.Error("failed to activate workspace: %v", err)
	}
}

// deactivateWorkspace 关闭工作区
func (s *extensionService) deactivateWorkspace(workspacePath string, fileNum int) {
	updateWorkspace := &model.Workspace{
		WorkspacePath: workspacePath,
		Active:        "false",
		FileNum:       fileNum,
	}
	if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
		s.logger.Error("failed to deactivate workspace: %v", err)
	}
}

// createCodebaseConfig 创建代码库配置文件
func (s *extensionService) createCodebaseConfig(workspacePath, clientID string) (int, error) {
	var fileNum int
	workspaceName := filepath.Base(workspacePath)
	codebaseID := utils.GenerateCodebaseID(workspacePath)

	currentHashTree, err := s.fileScanner.ScanCodebase(workspacePath)
	if err != nil {
		currentHashTree = make(map[string]string)
	}
	fileNum = len(currentHashTree)

	codebaseConfig, err := s.storage.GetCodebaseConfig(codebaseID)
	if err != nil {
		// 如果配置不存在，创建新的配置
		codebaseConfig = &config.CodebaseConfig{
			ClientID:     clientID,
			CodebaseId:   codebaseID,
			CodebaseName: workspaceName,
			CodebasePath: workspacePath,
			HashTree:     currentHashTree,
			RegisterTime: time.Now(),
		}
	} else {
		if fileNum > 0 {
			codebaseConfig.HashTree = currentHashTree
		} else {
			fileNum = len(codebaseConfig.HashTree)
		}
		codebaseConfig.RegisterTime = time.Now()
	}

	// 保存到存储
	if err := s.storage.SaveCodebaseConfig(codebaseConfig); err != nil {
		s.logger.Error("failed to save codebase config for %s: %v", workspacePath, err)
		return 0, fmt.Errorf("failed to save codebase config: %w", err)
	}

	s.logger.Info("created codebase config for %s (%s)", workspaceName, codebaseID)
	return fileNum, nil
}

// createCodebaseEmbeddingConfig 创建代码库嵌入配置
func (s *extensionService) createCodebaseEmbeddingConfig(workspacePath, clientID string) error {
	workspaceName := filepath.Base(workspacePath)
	codebaseEmbeddingID := utils.GenerateCodebaseEmbeddingID(workspacePath)

	_, err := s.codebaseEmbeddingRepo.GetCodebaseEmbeddingConfig(codebaseEmbeddingID)
	if err == nil {
		s.logger.Info("codebase embedding config for %s already exists", codebaseEmbeddingID)
		return nil
	}

	codebaseEmbeddingConfig := &config.CodebaseEmbeddingConfig{
		ClientID:     clientID,
		CodebaseId:   codebaseEmbeddingID,
		CodebaseName: workspaceName,
		CodebasePath: workspacePath,
		HashTree:     make(map[string]string),
		SyncFiles:    make(map[string]string),
		SyncIds:      make(map[string]time.Time),
		FailedFiles:  make(map[string]string),
	}

	// 保存到存储
	if err := s.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig); err != nil {
		s.logger.Error("failed to save codebase embedding config for %s: %v", workspacePath, err)
		return fmt.Errorf("failed to save codebase embedding config: %w", err)
	}

	s.logger.Info("created codebase embedding config for %s (%s)", workspaceName, codebaseEmbeddingID)
	return nil
}

// initCodebaseEmbeddingConfig 初始化代码库嵌入配置
func (s *extensionService) initCodebaseEmbeddingConfig(workspacePath, clientID string) error {
	workspaceName := filepath.Base(workspacePath)
	codebaseEmbeddingID := utils.GenerateCodebaseEmbeddingID(workspacePath)
	codebaseEmbeddingConfig := &config.CodebaseEmbeddingConfig{
		ClientID:     clientID,
		CodebaseId:   codebaseEmbeddingID,
		CodebaseName: workspaceName,
		CodebasePath: workspacePath,
		HashTree:     make(map[string]string),
		SyncFiles:    make(map[string]string),
		SyncIds:      make(map[string]time.Time),
		FailedFiles:  make(map[string]string),
	}

	// 保存到存储
	if err := s.codebaseEmbeddingRepo.SaveCodebaseEmbeddingConfig(codebaseEmbeddingConfig); err != nil {
		return fmt.Errorf("failed to save codebase embedding config: %w", err)
	}

	s.logger.Info("init codebase embedding config for %s (%s)", workspaceName, codebaseEmbeddingID)
	return nil
}

// TriggerIndex 触发索引构建
func (s *extensionService) TriggerIndex(ctx context.Context, workspacePath, indexType, clientID string) error {
	// 创建代码库配置
	fileNum, err := s.createCodebaseConfig(workspacePath, clientID)
	if err != nil {
		return fmt.Errorf("failed to create codebase config: %w", err)
	}

	// 创建代码库嵌入配置
	if indexType == dto.IndexTypeCodegraph {
		if err := s.createCodebaseEmbeddingConfig(workspacePath, clientID); err != nil {
			return fmt.Errorf("failed to create codebase embedding config: %w", err)
		}
	} else {
		if err := s.initCodebaseEmbeddingConfig(workspacePath, clientID); err != nil {
			return fmt.Errorf("failed to init codebase embedding config: %w", err)
		}
		if err := s.deleteRemoteEmbedding(clientID, workspacePath); err != nil {
			s.logger.Warn("delete remote embedding failed: %v", err)
		}
	}

	// 检查工作区是否已存在
	_, err = s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		// 工作区不存在，创建新的工作区
		s.logger.Info("workspace not found, creating new workspace: %s", workspacePath)
		s.createAndActivateWorkspace(workspacePath, "true", fileNum)
	} else {
		updateWorkspace := getUpdateWorkspaceByTriggerType(indexType)
		updateWorkspace["file_num"] = fileNum
		if err := s.workspaceRepo.UpdateWorkspaceByMap(workspacePath, updateWorkspace); err != nil {
			return fmt.Errorf("failed to update workspace: %w", err)
		}
		s.logger.Info("updated workspace active status to true: %s", workspacePath)
	}

	// 判断rebuild_workspace事件是否存在非进行中状态，若不存在则创建
	var rebuildEventId int64
	shouldCreateEvent := true
	embeddingStatus, codegraphStatus := getIndexStatusByTriggerType(indexType)
	existingRebuildEvents, err := s.eventRepo.GetEventsByWorkspaceAndType(workspacePath, []string{model.EventTypeRebuildWorkspace}, 1, true)
	if err == nil && len(existingRebuildEvents) > 0 {
		// 检查是否存在非进行中状态的rebuild_workspace事件
		for _, event := range existingRebuildEvents {
			if event.EmbeddingStatus != model.EmbeddingStatusUploading &&
				event.EmbeddingStatus != model.EmbeddingStatusBuilding &&
				event.CodegraphStatus != model.CodegraphStatusBuilding {
				// 存在非进行中状态的事件，不需要创建新事件，只更新事件状态
				updateEvent := &model.Event{
					ID:              event.ID,
					EmbeddingStatus: embeddingStatus,
					CodegraphStatus: codegraphStatus,
				}
				err := s.eventRepo.UpdateEvent(updateEvent)
				if err != nil {
					return fmt.Errorf("failed to update event: %w", err)
				}
				shouldCreateEvent = false
				rebuildEventId = event.ID
				break
			}
		}
	}

	if shouldCreateEvent {
		// 创建打开工作区事件
		eventModel := &model.Event{
			WorkspacePath:   workspacePath,
			EventType:       model.EventTypeRebuildWorkspace,
			SourceFilePath:  "",
			TargetFilePath:  "",
			SyncId:          "", // 暂时为空，后续可以生成
			EmbeddingStatus: embeddingStatus,
			CodegraphStatus: codegraphStatus,
		}
		// 保存事件到数据库
		if err := s.eventRepo.CreateEvent(eventModel); err != nil {
			return fmt.Errorf("failed to create open workspace event: %w", err)
		}
		rebuildEventId = eventModel.ID
	}

	// 获取所有非进行中状态的事件（排除新创建的事件）
	nonProcessingEmbeddingStatuses, nonProcessingCodegraphStatuses := getNonProcessingStatusesByTriggerType(indexType)
	eventsToDelete, err := s.eventRepo.GetEventsByTypeAndStatusAndWorkspaces(
		[]string{},
		[]string{workspacePath},
		-1, // 足够大的限制值
		false,
		nonProcessingEmbeddingStatuses,
		nonProcessingCodegraphStatuses,
	)
	if err != nil {
		return fmt.Errorf("failed to get non-processing events for deletion: %w", err)
	}

	// 删除这些事件（跳过新创建的事件）
	deleteEventIds := []int64{}
	for _, event := range eventsToDelete {
		if rebuildEventId == event.ID {
			continue // 跳过新创建的事件
		}
		deleteEventIds = append(deleteEventIds, event.ID)
	}

	if len(deleteEventIds) > 0 {
		if err := s.eventRepo.BatchDeleteEvents(deleteEventIds); err != nil {
			return fmt.Errorf("failed to batch delete non-processing events: %w", err)
		}
	}

	if indexType != dto.IndexTypeCodegraph {
		// rebuild_workspace 事件触发文件扫描
		go s.scanService.DetectFileChanges(workspacePath)
	}

	s.logger.Info("successfully triggered index for workspace: %s", workspacePath)
	return nil
}

// deleteRemoteEmbedding 删除远程索引
func (s *extensionService) deleteRemoteEmbedding(clientID, workspacePath string) error {
	deleteEmbeddingReq := dto.DeleteEmbeddingReq{ClientId: clientID, CodebasePath: workspacePath}
	resp, err := s.httpSync.DeleteEmbedding(deleteEmbeddingReq)
	if err != nil {
		return fmt.Errorf("http delete remote embedding failed: %w", err)
	}
	if resp.Code != -1 {
		return fmt.Errorf("delete remote embedding resp code: %d, msg: %s", resp.Code, resp.Message)
	}

	return nil
}

// getUpdateWorkspaceByTriggerType 根据触发类型获取更新工作区参数
func getUpdateWorkspaceByTriggerType(indexType string) map[string]interface{} {
	updateWorkspace := map[string]interface{}{
		"active":                      "true",
		"embedding_file_num":          0,
		"embedding_ts":                0,
		"embedding_message":           "",
		"embedding_failed_file_paths": "",
		"codegraph_file_num":          0,
		"codegraph_ts":                0,
		"codegraph_message":           "",
		"codegraph_failed_file_paths": "",
	}
	switch indexType {
	case dto.IndexTypeEmbedding:
		updateWorkspace = map[string]interface{}{
			"active":                      "true",
			"embedding_file_num":          0,
			"embedding_ts":                0,
			"embedding_message":           "",
			"embedding_failed_file_paths": "",
		}
	case dto.IndexTypeCodegraph:
		updateWorkspace = map[string]interface{}{
			"active":                      "true",
			"codegraph_file_num":          0,
			"codegraph_ts":                0,
			"codegraph_message":           "",
			"codegraph_failed_file_paths": "",
		}
	}
	return updateWorkspace
}

// getIndexStatusByTriggerType 根据触发类型获取索引状态
func getIndexStatusByTriggerType(indexType string) (int, int) {
	embeddingStatus := model.EmbeddingStatusInit
	codegraphStatus := model.CodegraphStatusInit
	switch indexType {
	case dto.IndexTypeEmbedding:
		codegraphStatus = model.CodegraphStatusSuccess
	case dto.IndexTypeCodegraph:
		embeddingStatus = model.EmbeddingStatusSuccess
	}
	return embeddingStatus, codegraphStatus
}

// getNonProcessingStatusesByTriggerType 根据触发类型获取非进行中状态
func getNonProcessingStatusesByTriggerType(indexType string) ([]int, []int) {
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

	switch indexType {
	case dto.IndexTypeEmbedding:
		nonProcessingCodegraphStatuses = []int{
			model.CodegraphStatusFailed,
			model.CodegraphStatusSuccess,
		}
	case dto.IndexTypeCodegraph:
		nonProcessingEmbeddingStatuses = []int{
			model.EmbeddingStatusUploadFailed,
			model.EmbeddingStatusBuildFailed,
			model.EmbeddingStatusSuccess,
		}
	}
	return nonProcessingEmbeddingStatuses, nonProcessingCodegraphStatuses
}

// GetIndexStatus 获取索引状态
func (s *extensionService) GetIndexStatus(ctx context.Context, workspacePath string) (*dto.IndexStatusResponse, error) {
	// 获取工作区信息
	workspace, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
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
		// 如果工作区已激活，根据事件记录计算状态
		data.Embedding = s.calculateEmbeddingStatus(workspace)
		data.Codegraph = s.calculateCodegraphStatus(workspace)
	}

	// 构建响应
	response := &dto.IndexStatusResponse{
		Code:    "0",
		Message: "ok",
		Data:    data,
	}

	s.logger.Info("successfully retrieved index status for workspace: %s, data: %v", workspacePath, data)
	return response, nil
}

// calculateEmbeddingStatus 计算 embedding 状态
func (s *extensionService) calculateEmbeddingStatus(workspace *model.Workspace) dto.IndexStatus {
	status := dto.IndexStatus{
		TotalFiles:   workspace.FileNum,
		TotalSucceed: workspace.EmbeddingFileNum,
		ProcessTs:    workspace.EmbeddingTs,
	}

	// 计算进度
	if workspace.FileNum > 0 {
		status.Process = float32(math.Round(float64(workspace.EmbeddingFileNum)/float64(workspace.FileNum)*100*10) / 10)
		if status.Process > 100 { // 进度不能超过100%
			status.Process = 100
			status.Status = dto.ProcessStatusSuccess
			return status
		}
	} else {
		status.Process = 0
		status.Status = dto.ProcessStatusPending
		return status
	}

	// 计算失败文件数
	failedFilePaths := strings.Split(workspace.EmbeddingFailedFilePaths, ",")
	fullFailedfilePaths := make([]string, 0, len(failedFilePaths))
	for _, failedFilePath := range failedFilePaths {
		if failedFilePath != "" {
			fullFailedfilePaths = append(fullFailedfilePaths, filepath.Join(workspace.WorkspacePath, failedFilePath))
		}
	}
	totalFailed := len(fullFailedfilePaths)

	// 统计各状态的 embedding 事件数
	processingCount, err := s.eventRepo.GetEventsCountByWorkspaceAndStatus(
		[]string{workspace.WorkspacePath},
		[]int{model.EmbeddingStatusInit, model.EmbeddingStatusUploading, model.EmbeddingStatusBuilding},
		[]int{},
	)
	if err != nil {
		s.logger.Warn("failed to get embedding events count by workspace and status: %v", err)
	}

	failedCount, err := s.eventRepo.GetEventsCountByWorkspaceAndStatus(
		[]string{workspace.WorkspacePath},
		[]int{model.EmbeddingStatusUploadFailed, model.EmbeddingStatusBuildFailed},
		[]int{},
	)
	if err != nil {
		s.logger.Warn("failed to get embedding events count by workspace and status: %v", err)
	}

	// 判断状态
	// 存在初始或进行中状态事件时，状态为 running
	if processingCount > 0 {
		status.Status = dto.ProcessStatusRunning
		return status
	}
	// 存在失败状态时，判断比较 process 和配置中的百分比阈值
	if failedCount > 0 {
		clientConfig := config.GetClientConfig()
		embeddingSuccessPercent := clientConfig.Sync.EmbeddingSuccessPercent
		if status.Process < embeddingSuccessPercent {
			status.Status = dto.ProcessStatusFailed
			status.TotalFailed = totalFailed
			status.FailedFiles = fullFailedfilePaths
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
func (s *extensionService) calculateCodegraphStatus(workspace *model.Workspace) dto.IndexStatus {
	status := dto.IndexStatus{
		TotalFiles:   workspace.FileNum,
		TotalSucceed: workspace.CodegraphFileNum,
		ProcessTs:    workspace.CodegraphTs,
	}

	// 计算进度
	if workspace.FileNum > 0 {
		status.Process = float32(math.Round(float64(workspace.CodegraphFileNum)/float64(workspace.FileNum)*100*10) / 10)
		if status.Process > 100 { // 进度不能超过100%
			status.Process = 100
			status.Status = dto.ProcessStatusSuccess
			return status
		}
	} else {
		status.Process = 0
		status.Status = dto.ProcessStatusPending
		return status
	}

	// 计算失败文件数
	failedFilePaths := strings.Split(workspace.EmbeddingFailedFilePaths, ",")
	fullFailedfilePaths := make([]string, 0, len(failedFilePaths))
	for _, failedFilePath := range failedFilePaths {
		if failedFilePath != "" {
			fullFailedfilePaths = append(fullFailedfilePaths, filepath.Join(workspace.WorkspacePath, failedFilePath))
		}
	}
	totalFailed := len(fullFailedfilePaths)

	// 统计各状态的 embedding 事件数
	processingCount, err := s.eventRepo.GetEventsCountByWorkspaceAndStatus(
		[]string{workspace.WorkspacePath},
		[]int{},
		[]int{model.CodegraphStatusInit, model.CodegraphStatusBuilding},
	)
	if err != nil {
		s.logger.Warn("failed to get codegraph events count by workspace and status: %v", err)
	}

	failedCount, err := s.eventRepo.GetEventsCountByWorkspaceAndStatus(
		[]string{workspace.WorkspacePath},
		[]int{},
		[]int{model.CodegraphStatusFailed},
	)
	if err != nil {
		s.logger.Warn("failed to get codegraph events count by workspace and status: %v", err)
	}

	// 判断状态
	// 存在初始或进行中状态事件时，状态为 running
	if processingCount > 0 {
		status.Status = dto.ProcessStatusRunning
		return status
	}
	// 存在失败状态时，判断比较 process 和配置中的百分比阈值
	if failedCount > 0 {
		clientConfig := config.GetClientConfig()
		codegraphSuccessPercent := clientConfig.Sync.CodegraphSuccessPercent
		if status.Process < codegraphSuccessPercent {
			status.Status = dto.ProcessStatusFailed
			status.TotalFailed = totalFailed
			status.FailedFiles = fullFailedfilePaths
			status.FailedReason = workspace.CodegraphMessage
		} else {
			status.Status = dto.ProcessStatusSuccess
		}
	} else {
		// 其他情况返回 success
		status.Status = dto.ProcessStatusSuccess
	}

	return status
}
