// internal/handler/extension.go - RESTful API handler using Gin framework
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// ExtensionHandler handles RESTful API services using Gin framework
type ExtensionHandler struct {
	syncService service.ExtensionService
	logger      logger.Logger
}

// NewExtensionHandler creates a new REST handler
func NewExtensionHandler(syncService service.ExtensionService, logger logger.Logger) *ExtensionHandler {
	return &ExtensionHandler{
		syncService: syncService,
		logger:      logger,
	}
}

// RegisterSync handles workspace registration via REST API
// @Summary 注册工作空间同步
// @Description 注册工作空间用于代码库同步
// @Tags sync
// @Accept json
// @Produce json
// @Param request body RegisterSyncRequest true "注册请求"
// @Success 200 {object} RegisterSyncResponse "注册成功"
// @Failure 400 {object} RegisterSyncResponse "请求格式错误"
// @Failure 404 {object} RegisterSyncResponse "未找到代码库"
// @Failure 500 {object} RegisterSyncResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/register [post]
func (h *ExtensionHandler) RegisterSync(c *gin.Context) {
	var req dto.RegisterSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.RegisterSyncResponse{
			Success: false,
			Message: "invalid request format",
		})
		return
	}

	h.logger.Info("workspace registration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)

	// 调用service层处理业务逻辑
	configs, err := h.syncService.RegisterCodebase(c.Request.Context(), req.ClientId, req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to register codebase: %v", err)
		c.JSON(http.StatusInternalServerError, dto.RegisterSyncResponse{
			Success: false,
			Message: "failed to register codebase",
		})
		return
	}

	if len(configs) == 0 {
		h.logger.Warn("no codebase found to register: %s", req.WorkspacePath)
		c.JSON(http.StatusNotFound, dto.RegisterSyncResponse{
			Success: false,
			Message: "no codebase found",
		})
		return
	}

	h.logger.Info("registered %d codebases successfully", len(configs))
	c.JSON(http.StatusOK, dto.RegisterSyncResponse{
		Success: true,
		Message: fmt.Sprintf("%d codebases registered successfully", len(configs)),
	})
}

// SyncCodebase handles codebase synchronization via REST API
// @Summary 同步代码库
// @Description 同步代码库文件
// @Tags sync
// @Accept json
// @Produce json
// @Param request body SyncCodebaseRequest true "同步请求"
// @Success 200 {object} SyncCodebaseResponse "同步成功"
// @Failure 400 {object} SyncCodebaseResponse "请求格式错误"
// @Failure 404 {object} SyncCodebaseResponse "未找到代码库"
// @Failure 500 {object} SyncCodebaseResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/sync [post]
func (h *ExtensionHandler) SyncCodebase(c *gin.Context) {
	var req dto.SyncCodebaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.SyncCodebaseResponse{
			Success: false,
			Code:    "0001",
			Message: "invalid request format",
		})
		return
	}

	h.logger.Info("codebase sync request: WorkspacePath=%s, WorkspaceName=%s, FilePaths=%v", req.WorkspacePath, req.WorkspaceName, req.FilePaths)

	// 调用service层处理业务逻辑
	configs, err := h.syncService.SyncCodebase(c.Request.Context(), req.ClientId, req.WorkspacePath, req.WorkspaceName, req.FilePaths)
	if err != nil {
		h.logger.Error("failed to sync codebase: %v", err)
		c.JSON(http.StatusInternalServerError, dto.SyncCodebaseResponse{
			Success: false,
			Code:    "1001",
			Message: fmt.Sprintf("sync codebase failed: %v", err),
		})
		return
	}

	if len(configs) == 0 {
		h.logger.Warn("no codebase found to sync: %s", req.WorkspacePath)
		c.JSON(http.StatusNotFound, dto.SyncCodebaseResponse{
			Success: false,
			Code:    "0010",
			Message: "no codebase found",
		})
		return
	}

	h.logger.Info("synced %d codebases successfully", len(configs))
	c.JSON(http.StatusOK, dto.SyncCodebaseResponse{
		Success: true,
		Code:    "0",
		Message: "sync codebase success",
	})
}

// UnregisterSync handles workspace unregistration via REST API
// @Summary 取消注册工作空间同步
// @Description 从代码库同步中取消注册工作空间
// @Tags sync
// @Accept json
// @Produce json
// @Param request body UnregisterSyncRequest true "取消注册请求"
// @Success 200 {object} UnregisterSyncResponse "取消注册成功"
// @Failure 400 {object} UnregisterSyncResponse "请求格式错误"
// @Failure 500 {object} UnregisterSyncResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/unregister [post]
func (h *ExtensionHandler) UnregisterSync(c *gin.Context) {
	var req dto.UnregisterSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	h.logger.Info("workspace unregistration request: WorkspacePath=%s, WorkspaceName=%s", req.WorkspacePath, req.WorkspaceName)

	// 调用service层处理业务逻辑
	configs, err := h.syncService.UnregisterCodebase(c.Request.Context(), req.ClientId, req.WorkspacePath, req.WorkspaceName)
	if err != nil {
		h.logger.Error("failed to unregister codebase: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unregister codebase"})
		return
	}

	h.logger.Info("unregistered %d codebase(s)", len(configs))
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("unregistered %d codebase(s)", len(configs))})
}

// ShareAccessToken handles token sharing via REST API
// @Summary 共享访问令牌
// @Description 为同步服务共享认证令牌
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ShareAccessTokenRequest true "令牌共享请求"
// @Success 200 {object} ShareAccessTokenResponse "共享成功"
// @Failure 400 {object} ShareAccessTokenResponse "请求格式错误"
// @Failure 500 {object} ShareAccessTokenResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/token [post]
func (h *ExtensionHandler) ShareAccessToken(c *gin.Context) {
	var req dto.ShareAccessTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.ShareAccessTokenResponse{
			Code:    http.StatusBadRequest,
			Success: false,
			Message: "invalid request format",
		})
		return
	}

	h.logger.Info("token synchronization request: ClientId=%s, ServerEndpoint=%s", req.ClientId, req.ServerEndpoint)

	// 调用service层处理业务逻辑
	err := h.syncService.UpdateSyncConfig(c.Request.Context(), req.ClientId, req.ServerEndpoint, req.AccessToken)
	if err != nil {
		h.logger.Error("failed to update sync config: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ShareAccessTokenResponse{
			Code:    http.StatusInternalServerError,
			Success: false,
			Message: "failed to update sync config",
		})
		return
	}

	h.logger.Info("sync config updated successfully")
	c.JSON(http.StatusOK, dto.ShareAccessTokenResponse{
		Code:    http.StatusOK,
		Success: true,
		Message: "ok",
	})
}

// GetVersion handles version information via REST API
// @Summary 获取版本信息
// @Description 获取应用程序版本信息
// @Tags system
// @Accept json
// @Produce json
// @Param request body VersionRequest true "版本请求"
// @Success 200 {object} VersionResponse "获取成功"
// @Failure 400 {object} VersionResponse "请求格式错误"
// @Router /codebase-indexer/api/v1/version [post]
func (h *ExtensionHandler) GetVersion(c *gin.Context) {
	var req dto.VersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.VersionResponse{
			Code:    http.StatusBadRequest,
			Success: false,
			Message: "invalid request format",
			Data:    dto.VersionResponseData{},
		})
		return
	}

	h.logger.Info("version request from client: %s", req.ClientId)

	// 返回版本信息
	c.JSON(http.StatusOK, dto.VersionResponse{
		Code:    http.StatusOK,
		Success: true,
		Message: "ok",
		Data: dto.VersionResponseData{
			AppName:  "Codebase Syncer",
			Version:  "1.0.0",
			OsName:   "cross-platform",
			ArchName: "universal",
		},
	})
}

// CheckIgnoreFile handles ignore file checking via REST API
// @Summary 检查忽略文件
// @Description 检查文件是否应该被忽略
// @Tags sync
// @Accept json
// @Produce json
// @Param request body CheckIgnoreFileRequest true "检查忽略文件请求"
// @Success 200 {object} CheckIgnoreFileResponse "检查成功"
// @Failure 400 {object} CheckIgnoreFileResponse "请求格式错误"
// @Failure 404 {object} CheckIgnoreFileResponse "未找到代码库"
// @Failure 422 {object} CheckIgnoreFileResponse "文件被忽略"
// @Router /codebase-indexer/api/v1/check-ignore [post]
func (h *ExtensionHandler) CheckIgnoreFile(c *gin.Context) {
	var req dto.CheckIgnoreFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.CheckIgnoreFileResponse{
			Code:    http.StatusBadRequest,
			Success: false,
			Ignore:  false,
			Message: "invalid request format",
		})
		return
	}

	h.logger.Info("check ignore file request: WorkspacePath=%s, WorkspaceName=%s, FilePaths=%v",
		req.WorkspacePath, req.WorkspaceName, req.FilePaths)

	// 参数验证
	if req.ClientId == "" || req.WorkspacePath == "" || req.WorkspaceName == "" || len(req.FilePaths) == 0 {
		h.logger.Error("invalid check ignore file parameters")
		c.JSON(http.StatusBadRequest, dto.CheckIgnoreFileResponse{
			Code:    http.StatusBadRequest,
			Success: false,
			Ignore:  false,
			Message: "invalid parameters",
		})
		return
	}

	// 调用service层处理业务逻辑
	result, err := h.syncService.CheckIgnoreFiles(c.Request.Context(), req.ClientId, req.WorkspacePath, req.WorkspaceName, req.FilePaths)
	if err != nil {
		h.logger.Error("failed to check ignore files: %v", err)
		c.JSON(http.StatusInternalServerError, dto.CheckIgnoreFileResponse{
			Code:    http.StatusInternalServerError,
			Success: false,
			Ignore:  false,
			Message: "internal server error",
		})
		return
	}

	// 根据结果返回响应
	if result.ShouldIgnore {
		c.JSON(http.StatusUnprocessableEntity, dto.CheckIgnoreFileResponse{
			Code:    http.StatusUnprocessableEntity,
			Success: false,
			Ignore:  true,
			Message: result.Reason,
		})
		return
	}

	c.JSON(http.StatusOK, dto.CheckIgnoreFileResponse{
		Code:    http.StatusOK,
		Success: true,
		Ignore:  false,
		Message: result.Reason,
	})
}

// SetupRoutes sets up the REST API routes
// @Description 设置REST API路由
func (h *ExtensionHandler) SetupRoutes(router *gin.Engine) {
	api := router.Group("/codebase-indexer/api/v1")
	{
		api.POST("/register", h.RegisterSync)
		api.POST("/sync", h.SyncCodebase)
		api.POST("/unregister", h.UnregisterSync)
		api.POST("/token", h.ShareAccessToken)
		api.POST("/version", h.GetVersion)
		api.POST("/files/ignore", h.CheckIgnoreFile)
	}
}
