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

	api "codebase-syncer/api"
	"codebase-syncer/internal/scheduler"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/pkg/logger"
)

type AppInfo struct {
	AppName  string `json:"appName"`
	Version  string `json:"version"`
	OSName   string `json:"osName"`
	ArchName string `json:"archName"`
}

// GRPCHandler handles gRPC services
type GRPCHandler struct {
	appInfo   *AppInfo
	httpSync  syncer.SyncInterface
	storage   storage.SotrageInterface
	scheduler *scheduler.Scheduler
	logger    logger.Logger
	api.UnimplementedSyncServiceServer
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(httpSync syncer.SyncInterface, storage storage.SotrageInterface, scheduler *scheduler.Scheduler, logger logger.Logger, appInfo *AppInfo) *GRPCHandler {
	return &GRPCHandler{
		appInfo:   appInfo,
		httpSync:  httpSync,
		storage:   storage,
		scheduler: scheduler,
		logger:    logger,
	}
}

// RegisterSync registers sync service
func (h *GRPCHandler) RegisterSync(ctx context.Context, req *api.RegisterSyncRequest) (*api.RegisterSyncResponse, error) {
	h.logger.Info("received workspace registration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// Check request parameters
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid workspace registration parameters")
		return &api.RegisterSyncResponse{Success: false, Message: "invalid parameters"}, nil
	}

	codebaseConfigsToRegister, err := h.findCodebasePaths(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths: %v", err)
		return &api.RegisterSyncResponse{Success: false, Message: fmt.Sprintf("failed to find codebase paths: %v", err)}, nil
	}

	if len(codebaseConfigsToRegister) == 0 {
		h.logger.Warn("no registerable codebase path found: %s", req.WorkspacePath)
		return &api.RegisterSyncResponse{Success: false, Message: "no registerable codebase found"}, nil
	}

	var addCodebaseConfigs []*storage.CodebaseConfig
	var registeredCount int
	var lastError error

	codebaseConfigs := h.storage.GetCodebaseConfigs()
	for _, pendingConfig := range codebaseConfigsToRegister {
		codebaseId := fmt.Sprintf("%s_%x", pendingConfig.CodebaseName, md5.Sum([]byte(pendingConfig.CodebasePath)))
		h.logger.Info("preparing to register/update codebase: Name=%s, Path=%s, Id=%s", pendingConfig.CodebaseName, pendingConfig.CodebasePath, codebaseId)

		codebaseConfig, ok := codebaseConfigs[codebaseId]
		if !ok {
			h.logger.Warn("failed to get codebase config (Id: %s), will initialize a new one", codebaseId)
			codebaseConfig = &storage.CodebaseConfig{
				ClientID:     req.ClientId,
				CodebaseName: pendingConfig.CodebaseName,
				CodebasePath: pendingConfig.CodebasePath,
				CodebaseId:   codebaseId,
				RegisterTime: time.Now(), // Set registration time to now
			}
		} else {
			h.logger.Info("found existing codebase config (Id: %s), will update it", codebaseId)
			codebaseConfig.ClientID = req.ClientId
			codebaseConfig.CodebaseName = pendingConfig.CodebaseName
			codebaseConfig.CodebasePath = pendingConfig.CodebasePath
			codebaseConfig.CodebaseId = codebaseId
			codebaseConfig.RegisterTime = time.Now() // Update registration time to now
		}

		if errSave := h.storage.SaveCodebaseConfig(codebaseConfig); errSave != nil {
			h.logger.Error("failed to save codebase config (Name: %s, Path: %s, Id: %s): %v", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId, errSave)
			lastError = errSave // Record the last error
			continue
		}
		h.logger.Info("codebase (Name: %s, Path: %s, Id: %s) registered/updated successfully", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId)
		registeredCount++
		if !ok {
			addCodebaseConfigs = append(addCodebaseConfigs, codebaseConfig)
		}
	}

	if registeredCount == 0 && lastError != nil {
		return &api.RegisterSyncResponse{Success: false, Message: fmt.Sprintf("all codebase registrations failed: %v", lastError)}, lastError
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
	h.logger.Info("received codebase sync request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// Check request parameters
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid codebase sync parameters")
		return &api.SyncCodebaseResponse{Success: false, Message: "invalid parameters"}, nil
	}

	codebaseConfigsToSync, err := h.findCodebasePaths(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths: %v", err)
		return &api.SyncCodebaseResponse{Success: false, Message: fmt.Sprintf("failed to find codebase paths: %v", err)}, nil
	}

	if len(codebaseConfigsToSync) == 0 {
		h.logger.Warn("no codebase found: %s", req.WorkspacePath)
		return &api.SyncCodebaseResponse{Success: false, Message: "no codebase found"}, nil
	}

	var syncCodebaseConfigs []*storage.CodebaseConfig
	var savedCount int
	var lastError error

	codebaseConfigs := h.storage.GetCodebaseConfigs()
	for _, pendingConfig := range codebaseConfigsToSync {
		codebaseId := fmt.Sprintf("%s_%x", pendingConfig.CodebaseName, md5.Sum([]byte(pendingConfig.CodebasePath)))
		h.logger.Info("preparing to sync codebase: Name=%s, Path=%s, Id=%s", pendingConfig.CodebaseName, pendingConfig.CodebasePath, codebaseId)

		codebaseConfig, ok := codebaseConfigs[codebaseId]
		if !ok {
			h.logger.Warn("codebase config not found: Id=%s", codebaseId)
			codebaseConfig = &storage.CodebaseConfig{
				ClientID:     req.ClientId,
				CodebaseName: pendingConfig.CodebaseName,
				CodebasePath: pendingConfig.CodebasePath,
				CodebaseId:   codebaseId,
				RegisterTime: time.Now(), // Set register time to now
			}
		} else {
			h.logger.Info("found existing codebase config (Id: %s), will update it", codebaseId)
			codebaseConfig.ClientID = req.ClientId
			codebaseConfig.CodebaseName = pendingConfig.CodebaseName
			codebaseConfig.CodebasePath = pendingConfig.CodebasePath
			codebaseConfig.CodebaseId = codebaseId
			codebaseConfig.RegisterTime = time.Now() // Update registration time to now
		}

		if errSave := h.storage.SaveCodebaseConfig(codebaseConfig); errSave != nil {
			h.logger.Error("failed to save codebase config (Name: %s, Path: %s, Id: %s): %v", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId, errSave)
			lastError = errSave // Record last error
			continue
		}
		h.logger.Info("codebase (Name: %s, Path: %s, Id: %s) saved successfully", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId)
		savedCount++
		syncCodebaseConfigs = append(syncCodebaseConfigs, codebaseConfig)
	}

	if savedCount == 0 && lastError != nil {
		return &api.SyncCodebaseResponse{Success: false, Message: fmt.Sprintf("all codebase config saved failed: %v", lastError)}, lastError
	}

	// Sync codebases
	if len(syncCodebaseConfigs) > 0 && h.httpSync.GetSyncConfig() != nil {
		err := h.syncCodebases(syncCodebaseConfigs)
		if err != nil {
			return &api.SyncCodebaseResponse{Success: false, Message: fmt.Sprintf("sync codebase failed: %v", err)}, err
		}
	}

	return &api.SyncCodebaseResponse{Success: true, Message: "sync codebase success"}, nil
}

// UnregisterSync unregisters sync service
func (h *GRPCHandler) UnregisterSync(ctx context.Context, req *api.UnregisterSyncRequest) (*emptypb.Empty, error) {
	h.logger.Info("received workspace unregistration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// Validate request parameters
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid workspace unregistration parameters")
		return &emptypb.Empty{}, fmt.Errorf("invalid parameters")
	}

	codebaseConfigsToUnregister, err := h.findCodebasePaths(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths to unregister: %v. WorkspacePath=%s, WorkspaceName=%s", err, req.WorkspacePath, req.WorkspaceName)
		// Even if lookup fails, still return Empty since unregister goal is cleanup
		return &emptypb.Empty{}, fmt.Errorf("failed to find codebase paths to unregister: %v", err) // Or return nil error, only log
	}

	if len(codebaseConfigsToUnregister) == 0 {
		h.logger.Warn("no matching codebase found to unregister for WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
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
		h.logger.Info("codebase (Name: %s, Path: %s, Id: %s) unregistered successfully", config.CodebaseName, config.CodebasePath, codebaseId)
		unregisteredCount++
	}

	if unregisteredCount < len(codebaseConfigsToUnregister) {
		// Even if some fail, UnregisterSync usually returns success, errors logged
		h.logger.Warn("partial codebase unregistrations failed. Successful: %d, Failed: %d. Last error: %v", unregisteredCount, len(codebaseConfigsToUnregister)-unregisteredCount, lastError)
	} else if len(codebaseConfigsToUnregister) > 0 {
		h.logger.Info("all %d matching codebases unregistered successfully", unregisteredCount)
	} else {
		// This case should ideally be caught by the len check at the beginning
		h.logger.Info("no codebases to unregister or found: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	}

	// UnregisterSync usually returns Empty & nil error, unless serious error
	// If all failed and there were things to delete, may return error
	if lastError != nil && unregisteredCount == 0 && len(codebaseConfigsToUnregister) > 0 {
		return &emptypb.Empty{}, fmt.Errorf("all codebase unregistrations failed: %v", lastError)
	}

	return &emptypb.Empty{}, nil
}

// ShareAccessToken shares auth token
func (h *GRPCHandler) ShareAccessToken(ctx context.Context, req *api.ShareAccessTokenRequest) (*api.ShareAccessTokenResponse, error) {
	h.logger.Info("token synchronization request received: ClientId=%s, ServerEndpoint=%s", req.ClientId, req.ServerEndpoint)
	if req.ClientId == "" || req.ServerEndpoint == "" || req.AccessToken == "" {
		h.logger.Error("invalid token synchronization parameters")
		return &api.ShareAccessTokenResponse{Success: false, Message: "invalid parameters"}, nil
	}
	syncConfig := &syncer.SyncConfig{
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
	h.logger.Info("version information request received: ClientId=%s", req.ClientId)
	if req.ClientId == "" {
		h.logger.Error("invalid version information parameters")
		return &api.VersionResponse{Success: false, Message: "invalid parameters"}, nil
	}
	return &api.VersionResponse{
		Success: true,
		Message: "ok",
		Data: &api.VersionResponse_Data{
			AppName:  h.appInfo.AppName,
			Version:  h.appInfo.Version,
			OsName:   h.appInfo.OSName,
			ArchName: h.appInfo.ArchName,
		},
	}, nil
}

// syncCodebases actively syncs code repositories
func (h *GRPCHandler) syncCodebases(codebaseConfigs []*storage.CodebaseConfig) error {
	timeout := time.Duration(storage.DefaultConfigSync.IntervalMinutes) * time.Minute
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
// Returns slice of CodebaseConfig (only CodebasePath and CodebaseName filled)
func (h *GRPCHandler) findCodebasePaths(basePath string, baseName string) ([]storage.CodebaseConfig, error) {
	var configs []storage.CodebaseConfig

	if h.isGitRepository(basePath) {
		h.logger.Info("path %s is a git repository", basePath)
		configs = append(configs, storage.CodebaseConfig{CodebasePath: basePath, CodebaseName: baseName})
		return configs, nil
	}

	h.logger.Info("path %s is not a git repository, checking its subdirectories", basePath)
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
				h.logger.Info("found git repository in subdirectory: %s (name: %s)", subDirPath, entry.Name())
				configs = append(configs, storage.CodebaseConfig{CodebasePath: subDirPath, CodebaseName: entry.Name()})
				foundSubRepo = true
			}
		}
	}

	if !foundSubRepo {
		h.logger.Info("no git repositories found in subdirectories of %s, using %s itself as codebase", basePath, basePath)
		configs = append(configs, storage.CodebaseConfig{CodebasePath: basePath, CodebaseName: baseName})
	}

	return configs, nil
}
