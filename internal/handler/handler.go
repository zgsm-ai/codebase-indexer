// handler/handler.go - gRPC服务处理
package handler

import (
	"context"
	"crypto/md5"
	"fmt"

	"google.golang.org/protobuf/types/known/emptypb"

	api "codebase-syncer/api" // 已生成的protobuf包
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/pkg/logger"
)

// GRPCHandler gRPC服务处理
type GRPCHandler struct {
	httpSync *syncer.HTTPSync
	storage  *storage.ConfigManager
	logger   logger.Logger
	api.UnimplementedSyncServiceServer
}

// NewGRPCHandler 创建新的gRPC处理器
func NewGRPCHandler(httpSync *syncer.HTTPSync, storage *storage.ConfigManager, logger logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		httpSync: httpSync,
		storage:  storage,
		logger:   logger,
	}
}

// 注册项目
func (h *GRPCHandler) RegisterSync(ctx context.Context, req *api.RegisterSyncRequest) (*api.RegisterSyncResponse, error) {
	h.logger.Info("接收到codebase注册请求: %s, %s", req.ProjectPath, req.ProjectName)

	codebaseId := fmt.Sprintf("%s_%x", req.ProjectName, md5.Sum([]byte(req.ProjectPath)))
	projectConfig, err := h.storage.GetProjectConfig(codebaseId)
	if err != nil {
		h.logger.Warn("获取codebase配置失败: %v, 初始化", err)
		projectConfig = &storage.ProjectConfig{
			ClientID:     req.ClientId,
			CodebaseName: req.ProjectName,
			CodebasePath: req.ProjectPath,
			CodebaseId:   codebaseId,
		}
	} else {
		projectConfig.ClientID = req.ClientId
		projectConfig.CodebaseId = codebaseId
		projectConfig.CodebasePath = req.ProjectPath
		projectConfig.CodebaseName = req.ProjectName
	}

	// 保存项目配置
	if err := h.storage.SaveProjectConfig(projectConfig); err != nil {
		h.logger.Error("保存codebase配置失败: %v", err)
		return nil, err
	}
	h.logger.Info("codebase注册成功，codebaseId: %s", codebaseId)
	return &api.RegisterSyncResponse{Success: true, Message: "ok"}, nil
}

func (h *GRPCHandler) UnregisterSync(ctx context.Context, req *api.UnregisterSyncRequest) (*emptypb.Empty, error) {
	h.logger.Info("接收到codebase注销请求: %s, %s", req.ProjectPath, req.ProjectName)
	// 删除项目配置
	codebaseId := fmt.Sprintf("%s_%x", req.ProjectName, md5.Sum([]byte(req.ProjectPath)))
	if err := h.storage.DeleteProjectConfig(codebaseId); err != nil {
		h.logger.Error("删除codebase配置失败: %v", err)
	}
	h.logger.Info("codebase注销成功，codebaseId: %s", codebaseId)
	return &emptypb.Empty{}, nil
}

// 同步Token
func (h *GRPCHandler) ShareAccessToken(ctx context.Context, req *api.ShareAccessTokenRequest) (*api.ShareAccessTokenResponse, error) {
	h.logger.Info("接收到Token同步请求: %+v", req)
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
