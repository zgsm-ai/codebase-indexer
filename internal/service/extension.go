package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
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
	SwitchIndex(ctx context.Context, workspacePath, switchStatus string) error

	// PublishEvents 发布工作区事件
	PublishEvents(ctx context.Context, workspacePath string, events []dto.WorkspaceEvent) (int, error)

	// TriggerIndex 触发索引构建
	TriggerIndex(ctx context.Context, workspacePath string) error
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
	codebaseService CodebaseService,
	logger logger.Logger,
) ExtensionService {
	return &extensionService{
		storage:         storage,
		httpSync:        httpSync,
		fileScanner:     fileScanner,
		workspaceRepo:   workspaceRepo,
		eventRepo:       eventRepo,
		codebaseService: codebaseService,
		logger:          logger,
	}
}

type extensionService struct {
	storage         repository.StorageInterface
	httpSync        repository.SyncInterface
	fileScanner     repository.ScannerInterface
	workspaceRepo   repository.WorkspaceRepository
	eventRepo       repository.EventRepository
	codebaseService CodebaseService
	logger          logger.Logger
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
		codebaseID := s.codebaseService.GenerateCodebaseID(codebaseConfig.CodebaseName, codebaseConfig.CodebasePath)

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
		codebaseID := s.codebaseService.GenerateCodebaseID(codebaseConfig.CodebaseName, codebaseConfig.CodebasePath)

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
		codebaseID := s.codebaseService.GenerateCodebaseID(config.CodebaseName, config.CodebasePath)

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

	// 查找代码库配置
	configs, err := s.codebaseService.FindCodebasePaths(ctx, workspacePath, workspaceName)
	if err != nil {
		s.logger.Error("failed to find codebase paths: %v", err)
		return nil, fmt.Errorf("failed to find codebase paths: %w", err)
	}

	if len(configs) == 0 {
		s.logger.Warn("no codebase found in workspace: %s", workspacePath)
		return &CheckIgnoreResult{
			ShouldIgnore: false,
			Reason:       "no codebase found",
			IgnoredFiles: []string{},
		}, nil
	}

	// 检查每个文件
	maxFileSizeKB := s.fileScanner.GetScannerConfig().MaxFileSizeKB
	maxFileSize := int64(maxFileSizeKB * 1024)
	for _, config := range configs {
		ignore := s.fileScanner.LoadIgnoreRules(config.CodebasePath)
		if ignore == nil {
			s.logger.Warn("no ignore file found for codebase: %s", config.CodebasePath)
			continue
		}

		for _, filePath := range filePaths {
			// Check if the file is in this codebase
			relPath, err := filepath.Rel(config.CodebasePath, filePath)
			if err != nil {
				s.logger.Debug("file path %s is not in codebase %s: %v", filePath, config.CodebasePath, err)
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
				s.logger.Info("ignore file found: %s in codebase %s", checkPath, config.CodebasePath)
				return &CheckIgnoreResult{
					ShouldIgnore: false,
					Reason:       "ignore file found:" + filePath,
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

func (s *extensionService) SwitchIndex(ctx context.Context, workspacePath, switchStatus string) error {
	// 检查开关状态
	if switchStatus != "on" && switchStatus != "off" {
		return fmt.Errorf("invalid switch status: %s", switchStatus)
	}
	// 检查工作空间是否存在
	workspace, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// 更新工作空间的索引开关状态
	active := "false"
	if switchStatus == "on" {
		active = "true"
	}

	if workspace.Active == active {
		return fmt.Errorf("workspace index is already %s", switchStatus)
	}

	// 更新工作空间状态
	updateWorkspace := &model.Workspace{
		ID:            workspace.ID,
		WorkspacePath: workspace.WorkspacePath,
		Active:        active,
	}
	if err := s.workspaceRepo.UpdateWorkspace(updateWorkspace); err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	s.logger.Info("index switch for workspace %s set to %s", workspacePath, switchStatus)
	return nil
}

// PublishEvents 发布工作区事件
func (s *extensionService) PublishEvents(ctx context.Context, workspacePath string, events []dto.WorkspaceEvent) (int, error) {
	s.logger.Info("publishing %d events for workspace: %s", len(events), workspacePath)

	successCount := 0

	// 处理每个事件
	for _, event := range events {
		// 解析事件时间
		eventTime, err := time.Parse("2006-01-02 15:04:05", event.EventTime)
		if err != nil {
			s.logger.Warn("failed to parse event time %s: %v", event.EventTime, err)
			eventTime = time.Now() // 使用当前时间作为默认值
		}

		// 创建事件模型
		eventModel := &model.Event{
			WorkspacePath:   workspacePath,
			EventType:       event.EventType,
			SourceFilePath:  event.SourcePath,
			TargetFilePath:  event.TargetPath,
			SyncId:          "",                        // 暂时为空，后续可以生成
			EmbeddingStatus: model.EmbeddingStatusInit, // 初始状态
			CodegraphStatus: model.CodegraphStatusInit, // 初始状态
			CreatedAt:       eventTime,
			UpdatedAt:       eventTime,
		}

		// 保存事件到数据库
		if err := s.eventRepo.CreateEvent(eventModel); err != nil {
			s.logger.Error("failed to create event: %v", err)
			continue
		}

		successCount++
		s.logger.Debug("created event: type=%s, source=%s, target=%s",
			event.EventType, event.SourcePath, event.TargetPath)
	}

	s.logger.Info("successfully published %d/%d events for workspace: %s",
		successCount, len(events), workspacePath)

	// 如果是打开工作区事件，确保工作区存在且激活
	for _, event := range events {
		if event.EventType == model.EventTypeOpenWorkspace {
			// 检查工作区是否已存在
			_, err := s.workspaceRepo.GetWorkspaceByPath(workspacePath)
			if err != nil {
				// 工作区不存在，创建新的工作区
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
				} else {
					s.logger.Info("created new workspace: %s", workspacePath)
				}
			}
			break // 只需要处理一个打开工作区事件
		}
	}

	return successCount, nil
}

// TriggerIndex 触发索引构建
func (s *extensionService) TriggerIndex(ctx context.Context, workspacePath string) error {
	s.logger.Info("triggering index build for workspace: %s", workspacePath)

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

	// 创建打开工作区事件
	eventTime := time.Now()
	eventModel := &model.Event{
		WorkspacePath:   workspacePath,
		EventType:       model.EventTypeOpenWorkspace,
		SourceFilePath:  "",
		TargetFilePath:  "",
		SyncId:          "",                        // 暂时为空，后续可以生成
		EmbeddingStatus: model.EmbeddingStatusInit, // 初始状态
		CodegraphStatus: model.CodegraphStatusInit, // 初始状态
		CreatedAt:       eventTime,
		UpdatedAt:       eventTime,
	}

	// 保存事件到数据库
	if err := s.eventRepo.CreateEvent(eventModel); err != nil {
		s.logger.Error("failed to create open workspace event: %v", err)
		return fmt.Errorf("failed to create event: %w", err)
	}

	s.logger.Info("successfully triggered index build for workspace: %s", workspacePath)
	return nil
}
