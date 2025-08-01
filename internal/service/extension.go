package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codebase-indexer/internal/config"
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
	codebaseService CodebaseService,
	logger logger.Logger,
) ExtensionService {
	return &extensionService{
		storage:         storage,
		httpSync:        httpSync,
		fileScanner:     fileScanner,
		codebaseService: codebaseService,
		logger:          logger,
	}
}

type extensionService struct {
	storage         repository.StorageInterface
	httpSync        repository.SyncInterface
	fileScanner     repository.ScannerInterface
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
