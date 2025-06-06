// handler/handler.go - gRPC服务处理
package handler

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	api "codebase-syncer/api" // 已生成的protobuf包
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/pkg/logger"
)

// GRPCHandler gRPC服务处理
type GRPCHandler struct {
	httpSync *syncer.HTTPSync
	storage  *storage.StorageManager
	logger   logger.Logger
	appName  string
	version  string
	osName   string
	archName string
	api.UnimplementedSyncServiceServer
}

// NewGRPCHandler 创建新的gRPC处理器
func NewGRPCHandler(httpSync *syncer.HTTPSync, storage *storage.StorageManager, logger logger.Logger, appName, version, osName, archName string) *GRPCHandler {
	return &GRPCHandler{
		httpSync: httpSync,
		storage:  storage,
		logger:   logger,
		appName:  appName,
		version:  version,
		osName:   osName,
		archName: archName,
	}
}

// RegisterSync 注册同步
func (h *GRPCHandler) RegisterSync(ctx context.Context, req *api.RegisterSyncRequest) (*api.RegisterSyncResponse, error) {
	h.logger.Info("接收到工作区注册请求: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// 检查请求参数
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("工作区注册请求参数错误")
		return &api.RegisterSyncResponse{Success: false, Message: "参数错误"}, nil
	}

	codebaseConfigsToRegister, err := h.findCodebasePathsToRegister(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("查找codebase路径失败: %v", err)
		return &api.RegisterSyncResponse{Success: false, Message: fmt.Sprintf("查找codebase路径失败: %v", err)}, nil
	}

	if len(codebaseConfigsToRegister) == 0 {
		h.logger.Warn("未找到可注册的codebase路径: %s", req.WorkspacePath)
		// 根据需求，如果一个都没找到（包括原始路径本身也不作为codebase），可以返回错误或特定消息
		// 当前 findCodebasePathsToRegister 的逻辑是，如果啥都没有，会把原始路径返回，所以这里理论上不会是0
		// 但为了健壮性，可以保留一个检查
		return &api.RegisterSyncResponse{Success: false, Message: "未找到可注册的codebase"}, nil
	}

	var registeredCount int
	var lastError error

	for _, pendingConfig := range codebaseConfigsToRegister {
		codebaseId := fmt.Sprintf("%s_%x", pendingConfig.CodebaseName, md5.Sum([]byte(pendingConfig.CodebasePath)))
		h.logger.Info("准备注册或更新codebase: Name=%s, Path=%s, Id=%s", pendingConfig.CodebaseName, pendingConfig.CodebasePath, codebaseId)

		codebaseConfig, errGet := h.storage.GetCodebaseConfig(codebaseId)
		if errGet != nil {
			h.logger.Warn("获取codebase配置失败 (codebaseId: %s): %v. 将进行初始化.", codebaseId, errGet)
			codebaseConfig = &storage.CodebaseConfig{
				ClientID:     req.ClientId, // 使用请求中的 ClientId
				CodebaseName: pendingConfig.CodebaseName,
				CodebasePath: pendingConfig.CodebasePath,
				CodebaseId:   codebaseId,
				RegisterTime: time.Now(), // 设置注册时间为当前时间
			}
		} else {
			h.logger.Info("找到现有codebase配置 (codebaseId: %s). 将进行更新.", codebaseId)
			codebaseConfig.ClientID = req.ClientId                   // 更新 ClientID
			codebaseConfig.CodebaseName = pendingConfig.CodebaseName // 确保名称是最新的
			codebaseConfig.CodebasePath = pendingConfig.CodebasePath // 确保路径是最新的
			codebaseConfig.CodebaseId = codebaseId                   // 确保 ID 是最新的
			codebaseConfig.RegisterTime = time.Now()                 // 更新注册时间为当前时间
		}

		if errSave := h.storage.SaveCodebaseConfig(codebaseConfig); errSave != nil {
			h.logger.Error("保存codebase配置失败 (Name: %s, Path: %s, Id: %s): %v", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId, errSave)
			lastError = errSave // 记录最后一个错误
			// 当前选择继续尝试，但记录错误
			continue
		}
		h.logger.Info("Codebase (Name: %s, Path: %s, Id: %s) 注册/更新成功", codebaseConfig.CodebaseName, codebaseConfig.CodebasePath, codebaseConfig.CodebaseId)
		registeredCount++
	}

	if registeredCount == 0 && lastError != nil {
		// 如果一个都没成功，并且有错误
		return &api.RegisterSyncResponse{Success: false, Message: fmt.Sprintf("所有codebase注册均失败: %v", lastError)}, lastError
	}

	if registeredCount < len(codebaseConfigsToRegister) && lastError != nil {
		// 如果部分成功，部分失败
		h.logger.Warn("部分codebase注册失败. 成功: %d, 失败: %d.最后一个错误: %v", registeredCount, len(codebaseConfigsToRegister)-registeredCount, lastError)
		return &api.RegisterSyncResponse{
			Success: true, // 标记为部分成功
			Message: fmt.Sprintf("部分codebase注册成功 (%d/%d). 最后一个错误: %v", registeredCount, len(codebaseConfigsToRegister), lastError),
		}, nil // 返回nil error，因为操作部分成功
	}

	h.logger.Info("所有 %d 个codebase均已成功注册/更新.", registeredCount)
	return &api.RegisterSyncResponse{Success: true, Message: fmt.Sprintf("%d 个codebase注册成功", registeredCount)}, nil
}

// UnregisterSync 注销同步
func (h *GRPCHandler) UnregisterSync(ctx context.Context, req *api.UnregisterSyncRequest) (*emptypb.Empty, error) {
	h.logger.Info("接收到工作区注销请求: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	// 检查请求参数
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" {
		h.logger.Error("工作区注销请求参数错误")
		return &emptypb.Empty{}, fmt.Errorf("参数错误")
	}

	codebaseConfigsToUnregister, err := h.findCodebasePathsToRegister(req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("查找待注销的codebase路径失败: %v. WorkspacePath=%s, WorkspaceName=%s", err, req.WorkspacePath, req.WorkspaceName)
		// 即使查找失败，也尝试返回 Empty，因为注销操作的目标是清理，不应因查找阶段的错误而阻塞
		return &emptypb.Empty{}, fmt.Errorf("查找待注销的codebase路径失败: %v", err) // 或者返回 nil error，仅记录日志
	}

	if len(codebaseConfigsToUnregister) == 0 {
		h.logger.Warn("未找到与 WorkspacePath=%s, WorkspaceName=%s 匹配的可注销codebase", req.WorkspacePath, req.WorkspaceName)
		return &emptypb.Empty{}, nil
	}

	var unregisteredCount int
	var lastError error

	for _, config := range codebaseConfigsToUnregister {
		codebaseId := fmt.Sprintf("%s_%x", config.CodebaseName, md5.Sum([]byte(config.CodebasePath)))
		h.logger.Info("准备注销codebase: Name=%s, Path=%s, Id=%s", config.CodebaseName, config.CodebasePath, codebaseId)

		if errDelete := h.storage.DeleteCodebaseConfig(codebaseId); errDelete != nil {
			h.logger.Error("删除codebase配置失败 (Name: %s, Path: %s, Id: %s): %v", config.CodebaseName, config.CodebasePath, codebaseId, errDelete)
			lastError = errDelete // 记录最后一个错误
			continue              // 继续尝试注销其他的
		}
		h.logger.Info("Codebase (Name: %s, Path: %s, Id: %s) 注销成功", config.CodebaseName, config.CodebasePath, codebaseId)
		unregisteredCount++
	}

	if unregisteredCount < len(codebaseConfigsToUnregister) {
		h.logger.Warn("部分codebase注销失败. 成功: %d, 失败: %d. 最后一个错误: %v", unregisteredCount, len(codebaseConfigsToUnregister)-unregisteredCount, lastError)
		// 即使部分失败，UnregisterSync 通常也返回成功，错误通过日志体现
	} else if len(codebaseConfigsToUnregister) > 0 {
		h.logger.Info("所有 %d 个匹配的codebase均已成功注销.", unregisteredCount)
	} else {
		// This case should ideally be caught by the len check at the beginning
		h.logger.Info("没有codebase需要注销或被找到: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)
	}

	// UnregisterSync 通常返回 Empty 和 nil error，除非发生非常严重、阻止其尝试操作的错误。
	// 单个删除失败不应导致整个操作失败。
	if lastError != nil && unregisteredCount == 0 && len(codebaseConfigsToUnregister) > 0 {
		// 如果一个都没成功，并且确实有东西要删，可以考虑返回错误
		return &emptypb.Empty{}, fmt.Errorf("所有codebase注销均失败: %w", lastError)
	}

	return &emptypb.Empty{}, nil
}

// ShareAccessToken 同步Token
func (h *GRPCHandler) ShareAccessToken(ctx context.Context, req *api.ShareAccessTokenRequest) (*api.ShareAccessTokenResponse, error) {
	h.logger.Info("接收到Token同步请求: ClientId=%s, ServerEndpoint=%s", req.ClientId, req.ServerEndpoint)
	if req.ClientId == "" || req.ServerEndpoint == "" || req.AccessToken == "" {
		h.logger.Error("Token同步请求参数错误")
		return &api.ShareAccessTokenResponse{Success: false, Message: "参数错误"}, nil
	}
	syncConfig := &syncer.SyncConfig{
		ClientId:  req.ClientId,
		ServerURL: req.ServerEndpoint,
		Token:     req.AccessToken,
	}
	h.httpSync.SetSyncConfig(syncConfig)
	h.logger.Info("更新全局token: %s, %s", req.ServerEndpoint, req.AccessToken)
	return &api.ShareAccessTokenResponse{Success: true, Message: "ok"}, nil
}

// GetVersion 获取应用版本信息
func (h *GRPCHandler) GetVersion(ctx context.Context, req *api.VersionRequest) (*api.VersionResponse, error) {
	h.logger.Info("接收到版本信息请求: ClientId=%s", req.ClientId)
	if req.ClientId == "" {
		h.logger.Error("版本信息请求参数错误")
		return &api.VersionResponse{Success: false, Message: "参数错误"}, nil
	}
	return &api.VersionResponse{
		Success: true,
		Message: "ok",
		Data: &api.VersionResponse_Data{
			AppName:  h.appName,
			Version:  h.version,
			OsName:   h.osName,
			ArchName: h.archName,
		},
	}, nil
}

// isGitRepository 检查给定路径是否是一个 .git 仓库的根目录
func (h *GRPCHandler) isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		if !os.IsNotExist(err) {
			h.logger.Debug("检查git仓库 %s 时发生错误: %v", gitPath, err)
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
		h.logger.Info("路径 %s 本身是一个git仓库", basePath)
		configs = append(configs, storage.CodebaseConfig{CodebasePath: basePath, CodebaseName: baseName})
		return configs, nil
	}

	h.logger.Info("路径 %s 不是git仓库，检查其子目录", basePath)
	subDirs, err := os.ReadDir(basePath)
	if err != nil {
		h.logger.Error("读取目录 %s 失败: %v", basePath, err)
		// 如果读取目录失败，根据策略，可以直接返回错误，或者将 basePath 作为默认
		// 这里选择返回错误，让上层处理
		return nil, fmt.Errorf("读取目录 %s 失败: %w", basePath, err)
	}

	foundSubRepo := false
	for _, entry := range subDirs {
		if entry.IsDir() {
			subDirPath := filepath.Join(basePath, entry.Name())
			if h.isGitRepository(subDirPath) {
				h.logger.Info("发现子目录git仓库: %s (名称: %s)", subDirPath, entry.Name())
				configs = append(configs, storage.CodebaseConfig{CodebasePath: subDirPath, CodebaseName: entry.Name()})
				foundSubRepo = true
			}
		}
	}

	if !foundSubRepo {
		h.logger.Info("在 %s 的子目录中未找到git仓库，将 %s 本身作为codebase", basePath, basePath)
		configs = append(configs, storage.CodebaseConfig{CodebasePath: basePath, CodebaseName: baseName})
	}

	return configs, nil
}
