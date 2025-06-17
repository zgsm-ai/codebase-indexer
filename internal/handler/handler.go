// handler/handler.go - gRPC服务处理
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

// GRPCHandler gRPC服务处理
type GRPCHandler struct {
	appInfo   *AppInfo
	httpSync  syncer.SyncInterface
	storage   storage.SotrageInterface
	scheduler *scheduler.Scheduler
	logger    logger.Logger
	api.UnimplementedSyncServiceServer
}

// NewGRPCHandler 创建新的gRPC处理器
func NewGRPCHandler(httpSync syncer.SyncInterface, storage storage.SotrageInterface, scheduler *scheduler.Scheduler, logger logger.Logger, appInfo *AppInfo) *GRPCHandler {
	return &GRPCHandler{
		appInfo:   appInfo,
		httpSync:  httpSync,
		storage:   storage,
		scheduler: scheduler,
		logger:    logger,
	}
}

// RegisterSync 注册同步
func (h *GRPCHandler) RegisterSync(ctx context.Context, req *api.RegisterSyncRequest) (*api.RegisterSyncResponse, error) {
	h.logger.Info("received workspace registration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// 检查请求参数
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid workspace registration parameters")
		return &api.RegisterSyncResponse{Success: false, Message: "invalid parameters"}, nil
	}

	codebaseConfigsToRegister, err := h.findCodebasePathsToRegister(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths: %v", err)
		return &api.RegisterSyncResponse{Success: false, Message: fmt.Sprintf("failed to find codebase paths: %v", err)}, nil
	}

	if len(codebaseConfigsToRegister) == 0 {
		h.logger.Warn("no registerable codebase path found: %s", req.WorkspacePath)
		return &api.RegisterSyncResponse{Success: false, Message: "no registerable codebase found"}, nil
	}

	var codebaseConfigs []*storage.CodebaseConfig
	var registeredCount int
	var lastError error

	for _, pendingConfig := range codebaseConfigsToRegister {
		codebaseId := fmt.Sprintf("%s_%x", pendingConfig.CodebaseName, md5.Sum([]byte(pendingConfig.CodebasePath)))
		h.logger.Info("preparing to register/update codebase: Name=%s, Path=%s, Id=%s", pendingConfig.CodebaseName, pendingConfig.CodebasePath, codebaseId)

		codebaseConfig, errGet := h.storage.GetCodebaseConfig(codebaseId)
		if errGet != nil {
			h.logger.Warn("failed to get codebase config (Id: %s): %v, will initialize a new one", codebaseId, errGet)
			codebaseConfig = &storage.CodebaseConfig{
				ClientID:     req.ClientId,
				CodebaseName: pendingConfig.CodebaseName,
				CodebasePath: pendingConfig.CodebasePath,
				CodebaseId:   codebaseId,
				RegisterTime: time.Now(), // 设置注册时间为当前时间
			}
		} else {
			h.logger.Info("found existing codebase config (Id: %s), will update it", codebaseId)
			codebaseConfig.ClientID = req.ClientId
			codebaseConfig.CodebaseName = pendingConfig.CodebaseName
			codebaseConfig.CodebasePath = pendingConfig.CodebasePath
			codebaseConfig.CodebaseId = codebaseId
			codebaseConfig.RegisterTime = time.Now() // 更新注册时间为当前时间
		}

		if errSave := h.storage.SaveCodebaseConfig(codebaseConfig); errSave != nil {
			h.logger.Error("failed to save codebase config (Name: %s, Path: %s, Id: %s): %v", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId, errSave)
			lastError = errSave // 记录最后一个错误
			continue
		}
		h.logger.Info("codebase (Name: %s, Path: %s, Id: %s) registered/updated successfully", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId)
		registeredCount++
		if errGet != nil {
			codebaseConfigs = append(codebaseConfigs, codebaseConfig)
		}
	}

	if registeredCount == 0 && lastError != nil {
		return &api.RegisterSyncResponse{Success: false, Message: fmt.Sprintf("all codebase registrations failed: %v", lastError)}, lastError
	}

	if len(codebaseConfigs) > 0 && h.httpSync.GetSyncConfig() != nil {
		go func() {
			// 定义5分钟超时
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := h.scheduler.SyncForCodebases(timeoutCtx, codebaseConfigs); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					h.logger.Warn("synchronization timeout for %d codebases", len(codebaseConfigs))
				} else {
					h.logger.Error("synchronization failed: %v", err)
				}
			}
		}()
	}

	if registeredCount < len(codebaseConfigsToRegister) && lastError != nil {
		// 如果部分成功，部分失败
		h.logger.Warn("partial codebase registration failures. Successful: %d, Failed: %d. Last error: %v", registeredCount, len(codebaseConfigsToRegister)-registeredCount, lastError)
		return &api.RegisterSyncResponse{
			Success: true,
			Message: fmt.Sprintf("partial codebase registration success (%d/%d). Last error: %v", registeredCount, len(codebaseConfigsToRegister), lastError),
		}, nil
	}

	h.logger.Info("all %d codebases registered/updated successfully", registeredCount)
	return &api.RegisterSyncResponse{Success: true, Message: fmt.Sprintf("%d codebases registered successfully", registeredCount)}, nil
}

// UnregisterSync 注销同步
func (h *GRPCHandler) UnregisterSync(ctx context.Context, req *api.UnregisterSyncRequest) (*emptypb.Empty, error) {
	h.logger.Info("received workspace unregistration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// 检查请求参数
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("invalid workspace unregistration parameters")
		return &emptypb.Empty{}, fmt.Errorf("invalid parameters")
	}

	codebaseConfigsToUnregister, err := h.findCodebasePathsToRegister(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to find codebase paths to unregister: %v. WorkspacePath=%s, WorkspaceName=%s", err, req.WorkspacePath, req.WorkspaceName)
		// 即使查找失败，也尝试返回 Empty，因为注销操作的目标是清理，不应因查找阶段的错误而阻塞
		return &emptypb.Empty{}, fmt.Errorf("failed to find codebase paths to unregister: %v", err) // 或者返回 nil error，仅记录日志
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
			lastError = errDelete // 记录最后一个错误
			continue
		}
		h.logger.Info("codebase (Name: %s, Path: %s, Id: %s) unregistered successfully", config.CodebaseName, config.CodebasePath, codebaseId)
		unregisteredCount++
	}

	if unregisteredCount < len(codebaseConfigsToUnregister) {
		// 即使部分失败，UnregisterSync 通常也返回成功，错误通过日志体现
		h.logger.Warn("partial codebase unregistrations failed. Successful: %d, Failed: %d. Last error: %v", unregisteredCount, len(codebaseConfigsToUnregister)-unregisteredCount, lastError)
	} else if len(codebaseConfigsToUnregister) > 0 {
		h.logger.Info("all %d matching codebases unregistered successfully", unregisteredCount)
	} else {
		// This case should ideally be caught by the len check at the beginning
		h.logger.Info("no codebases to unregister or found: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	}

	// UnregisterSync 通常返回 Empty 和 nil error，除非发生非常严重、阻止其尝试操作的错误。
	// 如果一个都没成功，并且确实有东西要删，可以考虑返回错误
	if lastError != nil && unregisteredCount == 0 && len(codebaseConfigsToUnregister) > 0 {
		return &emptypb.Empty{}, fmt.Errorf("all codebase unregistrations failed: %v", lastError)
	}

	return &emptypb.Empty{}, nil
}

// ShareAccessToken 同步Token
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

// GetVersion 获取应用版本信息
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

// isGitRepository 检查给定路径是否是一个 .git 仓库的根目录
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

// findCodebasePathsToRegister 查找需要注册的codebase路径和名称
// 1. 如果 basePath 本身是 .git 仓库，则返回它
// 2. 如果 basePath 不是 .git 仓库，则检查其一级子目录，返回所有作为 .git 仓库的子目录
// 3. 如果 basePath 及其一级子目录都不是 .git 仓库，则返回 basePath 本身
// 返回的是一个包含待处理 CodebaseConfig 结构体（仅填充 CodebasePath 和 CodebaseName）的切片
func (h *GRPCHandler) findCodebasePathsToRegister(basePath string, baseName string) ([]storage.CodebaseConfig, error) {
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
