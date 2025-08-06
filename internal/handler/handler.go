// handler/handler.go - gRPC service handler
package handler

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	api "codebase-indexer/api"
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// GRPCHandler handles gRPC services
type GRPCHandler struct {
	httpSync    repository.SyncInterface
	fileScanner repository.ScannerInterface
	storage     repository.StorageInterface
	scheduler   *service.Scheduler
	logger      logger.Logger
	api.UnimplementedSyncServiceServer
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(httpSync repository.SyncInterface, fileScanner repository.ScannerInterface, storage repository.StorageInterface, scheduler *service.Scheduler, logger logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		httpSync:    httpSync,
		fileScanner: fileScanner,
		storage:     storage,
		scheduler:   scheduler,
		logger:      logger,
	}
}

// RegisterSync registers sync service
func (h *GRPCHandler) RegisterSync(ctx context.Context, req *api.RegisterSyncRequest) (*api.RegisterSyncResponse, error) {
	h.logger.Info("workspace registration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// Check request parameters
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid workspace registration parameters")
		return &api.RegisterSyncResponse{Success: false, Message: "invalid parameters"}, nil
	}

	codebaseConfigsToRegister, err := h.findCodebasePaths(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths to register: %v", err)
		return &api.RegisterSyncResponse{Success: false, Message: "failed to find codebase paths"}, nil
	}

	if len(codebaseConfigsToRegister) == 0 {
		h.logger.Warn("no codebase found to register: %s", req.WorkspacePath)
		return &api.RegisterSyncResponse{Success: false, Message: "no codebase found"}, nil
	}

	var addCodebaseConfigs []*config.CodebaseConfig
	var registeredCount int
	var lastError error

	nowTime := time.Now()
	codebaseConfigs := h.storage.GetCodebaseConfigs()
	for _, pendingConfig := range codebaseConfigsToRegister {
		codebaseId := fmt.Sprintf("%s_%x", pendingConfig.CodebaseName, md5.Sum([]byte(pendingConfig.CodebasePath)))
		h.logger.Info("preparing to register/update codebase: Name=%s, Path=%s, Id=%s", pendingConfig.CodebaseName, pendingConfig.CodebasePath, codebaseId)

		codebaseConfig, ok := codebaseConfigs[codebaseId]
		if !ok {
			h.logger.Warn("failed to get codebase config (Id: %s), will register", codebaseId)
			codebaseConfig = &config.CodebaseConfig{
				ClientID:     req.ClientId,
				CodebaseName: pendingConfig.CodebaseName,
				CodebasePath: pendingConfig.CodebasePath,
				CodebaseId:   codebaseId,
				RegisterTime: nowTime, // Set registration time to now
			}
		} else {
			h.logger.Info("found existing codebase config (Id: %s), will update it", codebaseId)
			codebaseConfig.ClientID = req.ClientId
			codebaseConfig.CodebaseName = pendingConfig.CodebaseName
			codebaseConfig.CodebasePath = pendingConfig.CodebasePath
			codebaseConfig.CodebaseId = codebaseId
			codebaseConfig.RegisterTime = nowTime // Update registration time to now
		}

		if errSave := h.storage.SaveCodebaseConfig(codebaseConfig); errSave != nil {
			h.logger.Error("failed to save codebase config (Name: %s, Path: %s, Id: %s): %v", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId, errSave)
			lastError = errSave // Record the last error
			continue
		}
		registeredCount++
		if !ok {
			addCodebaseConfigs = append(addCodebaseConfigs, codebaseConfig)
		}
	}

	if registeredCount == 0 && lastError != nil {
		h.logger.Error("all codebase config saves failed: %v", lastError)
		return &api.RegisterSyncResponse{Success: false, Message: "all codebase registrations failed"}, nil
	}

	// Sync newly registered codebases
	if len(addCodebaseConfigs) > 0 && h.httpSync.GetSyncConfig() != nil {
		go h.syncCodebases(addCodebaseConfigs)
	}

	// If partially succeeded
	if registeredCount < len(codebaseConfigsToRegister) && lastError != nil {
		h.logger.Warn("partial codebase registration failures. Successful: %d, Failed: %d. Last error: %v", registeredCount, len(codebaseConfigsToRegister)-registeredCount, lastError)
		return &api.RegisterSyncResponse{
			Success: true,
			Message: fmt.Sprintf("partial codebase registration success (%d/%d). Last error: %v", registeredCount, len(codebaseConfigsToRegister), lastError),
		}, nil
	}

	h.logger.Info("all %d codebases registered/updated successfully", registeredCount)
	return &api.RegisterSyncResponse{Success: true, Message: fmt.Sprintf("%d codebases registered successfully", registeredCount)}, nil
}

// SyncCodebase syncs codebases under specified workspace
func (h *GRPCHandler) SyncCodebase(ctx context.Context, req *api.SyncCodebaseRequest) (*api.SyncCodebaseResponse, error) {
	h.logger.Info("codebase sync request: WorkspacePath=%s, WorkspaceName=%s, FilePaths=%v", req.WorkspacePath, req.WorkspaceName, req.FilePaths)
	// Check request parameters
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid codebase sync parameters")
		return &api.SyncCodebaseResponse{Success: false, Code: "0001", Message: "invalid parameters"}, nil
	}

	codebaseConfigsToSync, err := h.findCodebasePaths(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths to sync: %v", err)
		return &api.SyncCodebaseResponse{Success: false, Code: "0010", Message: "failed to find codebase paths"}, nil
	}

	if len(codebaseConfigsToSync) == 0 {
		h.logger.Warn("no codebase found to sync: %s", req.WorkspacePath)
		return &api.SyncCodebaseResponse{Success: false, Code: "0010", Message: "no codebase found"}, nil
	}

	var syncCodebaseConfigs []*config.CodebaseConfig
	var savedCount int
	var lastError error

	nowTime := time.Now()
	codebaseConfigs := h.storage.GetCodebaseConfigs()
	for _, pendingConfig := range codebaseConfigsToSync {
		codebaseId := fmt.Sprintf("%s_%x", pendingConfig.CodebaseName, md5.Sum([]byte(pendingConfig.CodebasePath)))
		h.logger.Info("preparing to sync codebase: Name=%s, Path=%s, Id=%s", pendingConfig.CodebaseName, pendingConfig.CodebasePath, codebaseId)

		codebaseConfig, ok := codebaseConfigs[codebaseId]
		if !ok {
			h.logger.Warn("codebase config not found: Id=%s, will register", codebaseId)
			codebaseConfig = &config.CodebaseConfig{
				ClientID:     req.ClientId,
				CodebaseName: pendingConfig.CodebaseName,
				CodebasePath: pendingConfig.CodebasePath,
				CodebaseId:   codebaseId,
				RegisterTime: nowTime, // Set register time to now
			}
		} else {
			h.logger.Info("found existing codebase config (Id: %s), will update it", codebaseId)
			codebaseConfig.ClientID = req.ClientId
			codebaseConfig.CodebaseName = pendingConfig.CodebaseName
			codebaseConfig.CodebasePath = pendingConfig.CodebasePath
			codebaseConfig.CodebaseId = codebaseId
			codebaseConfig.RegisterTime = nowTime // Update registration time to now
		}

		if errSave := h.storage.SaveCodebaseConfig(codebaseConfig); errSave != nil {
			h.logger.Error("failed to save codebase config (Name: %s, Path: %s, Id: %s): %v", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId, errSave)
			lastError = errSave // Record last error
			continue
		}
		savedCount++
		syncCodebaseConfigs = append(syncCodebaseConfigs, codebaseConfig)
	}

	if savedCount == 0 && lastError != nil {
		h.logger.Error("all codebase config saves failed: %v", lastError)
		return &api.SyncCodebaseResponse{Success: false, Code: "0010", Message: "all codebase config saved failed"}, nil
	}

	// Sync codebases
	if len(syncCodebaseConfigs) > 0 && h.httpSync.GetSyncConfig() != nil {
		var err error
		if len(req.FilePaths) == 0 {
			err = h.syncCodebases(syncCodebaseConfigs)
		} else {
			err = h.syncCodebasesWithFilePaths(syncCodebaseConfigs, req.FilePaths)
		}
		if err != nil {
			h.logger.Error("failed to sync codebases: %v", err)
			if utils.IsUnauthorizedError(err) {
				return &api.SyncCodebaseResponse{Success: false, Code: utils.StatusCodeUnauthorized, Message: "unauthorized"}, nil
			}
			if utils.IsPageNotFoundError(err) {
				return &api.SyncCodebaseResponse{Success: false, Code: utils.StatusCodePageNotFound, Message: "page not found"}, nil
			}
			if utils.IsTooManyRequestsError(err) {
				return &api.SyncCodebaseResponse{Success: false, Code: utils.StatusCodeTooManyRequests, Message: "too many requests"}, nil
			}
			if utils.IsServiceUnavailableError(err) {
				return &api.SyncCodebaseResponse{Success: false, Code: utils.StatusCodeServiceUnavailable, Message: "service unavailable"}, nil
			}
			return &api.SyncCodebaseResponse{Success: false, Code: "1001", Message: fmt.Sprintf("sync codebase failed: %v", err)}, nil
		}
	}

	h.logger.Info("sync codebase success")
	return &api.SyncCodebaseResponse{Success: true, Code: "0", Message: "sync codebase success"}, nil
}

// UnregisterSync unregisters sync service
func (h *GRPCHandler) UnregisterSync(ctx context.Context, req *api.UnregisterSyncRequest) (*emptypb.Empty, error) {
	h.logger.Info("workspace unregistration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// Validate request parameters
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid workspace unregistration parameters")
		return &emptypb.Empty{}, nil
	}

	codebaseConfigsToUnregister, err := h.findCodebasePaths(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths to unregister: %v", err)
		// Even if lookup fails, still return Empty since unregister goal is cleanup
		return &emptypb.Empty{}, nil // return nil error, only log
	}

	if len(codebaseConfigsToUnregister) == 0 {
		h.logger.Warn("no codebase found: %s", req.WorkspacePath)
		return &emptypb.Empty{}, nil
	}

	var unregisteredCount int
	var lastError error

	for _, config := range codebaseConfigsToUnregister {
		codebaseId := fmt.Sprintf("%s_%x", config.CodebaseName, md5.Sum([]byte(config.CodebasePath)))
		h.logger.Info("preparing to unregister codebase: Name=%s, Path=%s, Id=%s", config.CodebaseName, config.CodebasePath, codebaseId)

		if errDelete := h.storage.DeleteCodebaseConfig(codebaseId); errDelete != nil {
			h.logger.Error("failed to delete codebase config (Name: %s, Path: %s, Id: %s): %v", config.CodebaseName, config.CodebasePath, codebaseId, errDelete)
			lastError = errDelete // Record the last error
			continue
		}
		unregisteredCount++
	}

	if unregisteredCount < len(codebaseConfigsToUnregister) {
		// Even if some fail, UnregisterSync usually returns success, errors logged
		h.logger.Warn("partial codebase unregistrations failed. Successful: %d, Failed: %d. Last error: %v", unregisteredCount, len(codebaseConfigsToUnregister)-unregisteredCount, lastError)
	} else if len(codebaseConfigsToUnregister) > 0 {
		h.logger.Info("all %d matching codebases unregistered successfully", unregisteredCount)
	} else {
		// This case should ideally be caught by the len check at the beginning
		h.logger.Warn("no codebase found: %s", req.WorkspacePath)
		return &emptypb.Empty{}, nil
	}

	// UnregisterSync usually returns Empty & nil error, unless serious error
	// If all failed and there were things to delete, may return error
	if lastError != nil && unregisteredCount == 0 && len(codebaseConfigsToUnregister) > 0 {
		h.logger.Error("all codebase unregistrations failed: %v", lastError)
		return &emptypb.Empty{}, nil
	}

	h.logger.Info("unregistered %d codebase(s)", unregisteredCount)
	return &emptypb.Empty{}, nil
}

// ShareAccessToken shares auth token
func (h *GRPCHandler) ShareAccessToken(ctx context.Context, req *api.ShareAccessTokenRequest) (*api.ShareAccessTokenResponse, error) {
	h.logger.Info("token synchronization request: ClientId=%s, ServerEndpoint=%s", req.ClientId, req.ServerEndpoint)
	if req.ClientId == "" || req.ServerEndpoint == "" || req.AccessToken == "" {
		h.logger.Error("invalid token synchronization parameters")
		return &api.ShareAccessTokenResponse{Success: false, Message: "invalid parameters"}, nil
	}
	syncConfig := &config.SyncConfig{
		ClientId:  req.ClientId,
		ServerURL: req.ServerEndpoint,
		Token:     req.AccessToken,
	}
	h.httpSync.SetSyncConfig(syncConfig)
	h.logger.Info("global token updated: %s, %s", req.ServerEndpoint, req.AccessToken)
	return &api.ShareAccessTokenResponse{Success: true, Message: "ok"}, nil
}

// GetVersion retrieves application version info
func (h *GRPCHandler) GetVersion(ctx context.Context, req *api.VersionRequest) (*api.VersionResponse, error) {
	h.logger.Info("version information request: ClientId=%s", req.ClientId)
	if req.ClientId == "" {
		h.logger.Error("invalid version information parameters")
		return &api.VersionResponse{Success: false, Message: "invalid parameters"}, nil
	}
	appInfo := config.GetAppInfo()

	return &api.VersionResponse{
		Success: true,
		Message: "ok",
		Data: &api.VersionResponse_Data{
			AppName:  appInfo.AppName,
			Version:  appInfo.Version,
			OsName:   appInfo.OSName,
			ArchName: appInfo.ArchName,
		},
	}, nil
}

// CheckIgnoreFile checks if specified files are ignored by ignore rules or exceed size limit
func (h *GRPCHandler) CheckIgnoreFile(ctx context.Context, req *api.CheckIgnoreFileRequest) (*api.SyncCodebaseResponse, error) {
	const (
		InvalidParamsCode    = "0001"
		CodebaseFindError    = "0010"
		FileSizeExceededCode = "2001"
		IgnoredFileFoundCode = "2002"
		SuccessCode          = "0"
	)

	h.logger.Info("check ignore file request: WorkspacePath=%s, WorkspaceName=%s, FilePaths=%v", req.WorkspacePath, req.WorkspaceName, req.FilePaths)

	// Validate input params
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" || len(req.FilePaths) == 0 {
		h.logger.Error("invalid check ignore file parameters")
		return &api.SyncCodebaseResponse{
			Success: false,
			Code:    InvalidParamsCode,
			Message: "invalid parameters",
		}, nil
	}

	// Find all codebases in the workspace
	codebasesToCheck, err := h.findCodebasePaths(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase to check: %v", err)
		return &api.SyncCodebaseResponse{
			Success: false,
			Code:    CodebaseFindError,
			Message: "failed to find codebase paths",
		}, nil
	}

	if len(codebasesToCheck) == 0 {
		h.logger.Warn("no codebase found to check: %s", req.WorkspacePath)
		return &api.SyncCodebaseResponse{
			Success: false,
			Code:    CodebaseFindError,
			Message: "no codebase found in workspace",
		}, nil
	}

	// Check ignore files
	maxFileSizeKB := h.fileScanner.GetScannerConfig().MaxFileSizeKB
	maxFileSize := int64(maxFileSizeKB * 1024)
	for _, config := range codebasesToCheck {
		ignore := h.fileScanner.LoadIgnoreRules(config.CodebasePath)
		if ignore == nil {
			h.logger.Warn("no ignore file found for codebase: %s", config.CodebasePath)
			continue
		}

		for _, filePath := range req.FilePaths {
			// Check if the file is in this codebase
			relPath, err := filepath.Rel(config.CodebasePath, filePath)
			if err != nil {
				h.logger.Debug("file path %s is not in codebase %s: %v", filePath, config.CodebasePath, err)
				continue
			}

			// Check file size and ignore rules
			checkPath := relPath
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				h.logger.Warn("failed to get file info: %s, %v", filePath, err)
				continue
			}

			// If directory, append "/" and skip size check
			if fileInfo.IsDir() {
				checkPath = relPath + "/"
			} else if fileInfo.Size() > maxFileSize {
				// For regular files, check size limit
				fileSizeKB := float64(fileInfo.Size()) / 1024
				h.logger.Info("file size exceeded limit: %s (%.2fKB)", filePath, fileSizeKB)
				return &api.SyncCodebaseResponse{
					Success: false,
					Code:    FileSizeExceededCode,
					Message: fmt.Sprintf("file size exceeded limit: %s (%.2fKB)", filePath, fileSizeKB),
				}, nil
			}

			// Check ignore rules
			if ignore.MatchesPath(checkPath) {
				h.logger.Info("ignore file found: %s in codebase %s", checkPath, config.CodebasePath)
				return &api.SyncCodebaseResponse{
					Success: false,
					Code:    IgnoredFileFoundCode,
					Message: "ignore file found:" + filePath,
				}, nil
			}
		}
	}

	h.logger.Info("no ignored files found, numFiles: %d", len(req.FilePaths))
	return &api.SyncCodebaseResponse{
		Success: true,
		Code:    SuccessCode,
		Message: "no ignored files found",
	}, nil
}

// syncCodebases actively syncs code repositories
func (h *GRPCHandler) syncCodebases(codebaseConfigs []*config.CodebaseConfig) error {
	timeout := time.Duration(config.DefaultConfigSync.IntervalMinutes) * time.Minute
	if h.scheduler.GetSchedulerConfig() != nil && h.scheduler.GetSchedulerConfig().IntervalMinutes > 0 {
		timeout = time.Duration(h.scheduler.GetSchedulerConfig().IntervalMinutes) * time.Minute
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := h.scheduler.SyncForCodebases(timeoutCtx, codebaseConfigs); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.Warn("sync timeout for %d codebases", len(codebaseConfigs))
			return fmt.Errorf("sync timeout for %d codebases: %v", len(codebaseConfigs), err)
		} else {
			h.logger.Error("sync failed: %v", err)
			return err
		}
	}

	return nil
}

// syncCodebasesWithFilePaths syncs code repositories with specified file paths
func (h *GRPCHandler) syncCodebasesWithFilePaths(codebaseConfigs []*config.CodebaseConfig, filePaths []string) error {
	for _, codebaseConfig := range codebaseConfigs {
		if err := h.scheduler.PerformSyncForCodebaseWithFilePaths(codebaseConfig, filePaths); err != nil {
			h.logger.Error("sync failed: %v", err)
			return err
		}
	}

	return nil
}

// isGitRepository checks if path is a git repo root
func (h *GRPCHandler) isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		if !os.IsNotExist(err) {
			h.logger.Warn("error checking git repository %s: %v", gitPath, err)
		}
		return false
	}
	return info.IsDir()
}

// findCodebasePaths finds codebase paths under specified path:
// 1. If basePath is a git repo, return it
// 2. If not, check first-level subdirs and return any git repos
// 3. If no git repos found in basePath or subdirs, return basePath
// Returns slice of CodebaseConfig (only Workspace and CodebaseName filled)
func (h *GRPCHandler) findCodebasePaths(basePath string, baseName string) ([]config.CodebaseConfig, error) {
	var configs []config.CodebaseConfig

	if h.isGitRepository(basePath) {
		h.logger.Info("path %s is a git repository", basePath)
		configs = append(configs, config.CodebaseConfig{CodebasePath: basePath, CodebaseName: baseName})
		return configs, nil
	}

	subDirs, err := os.ReadDir(basePath)
	if err != nil {
		h.logger.Error("failed to read directory %s: %v", basePath, err)
		return nil, fmt.Errorf("failed to read directory %s: %v", basePath, err)
	}

	foundSubRepo := false
	for _, entry := range subDirs {
		if entry.IsDir() {
			subDirPath := filepath.Join(basePath, entry.Name())
			if h.isGitRepository(subDirPath) {
				configs = append(configs, config.CodebaseConfig{CodebasePath: subDirPath, CodebaseName: entry.Name()})
				foundSubRepo = true
			}
		}
	}

	if !foundSubRepo {
		configs = append(configs, config.CodebaseConfig{CodebasePath: basePath, CodebaseName: baseName})
	}

	h.logger.Info("found %d codebase paths under %s: %+v", len(configs), basePath, configs)

	return configs, nil
}
