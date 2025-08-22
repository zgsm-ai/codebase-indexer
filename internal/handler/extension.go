// internal/handler/extension.go - RESTful API handler using Gin framework
package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// ExtensionHandler handles RESTful API services using Gin framework
type ExtensionHandler struct {
	extensionService service.ExtensionService
	logger           logger.Logger
}

// NewExtensionHandler creates a new REST handler
func NewExtensionHandler(extensionService service.ExtensionService, logger logger.Logger) *ExtensionHandler {
	return &ExtensionHandler{
		extensionService: extensionService,
		logger:           logger,
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
	configs, err := h.extensionService.RegisterCodebase(c.Request.Context(), req.ClientId, req.WorkspacePath, req.WorkspaceName)
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
	configs, err := h.extensionService.SyncCodebase(c.Request.Context(), req.ClientId, req.WorkspacePath, req.WorkspaceName, req.FilePaths)
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
	configs, err := h.extensionService.UnregisterCodebase(c.Request.Context(), req.ClientId, req.WorkspacePath, req.WorkspaceName)
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
			Code:    "400",
			Success: false,
			Message: "invalid request format",
		})
		return
	}

	h.logger.Info("token synchronization request: ClientId=%s, ServerEndpoint=%s, AccessToken=%s", req.ClientId, req.ServerEndpoint, req.AccessToken)

	// 调用service层处理业务逻辑
	err := h.extensionService.UpdateSyncConfig(c.Request.Context(), req.ClientId, req.ServerEndpoint, req.AccessToken)
	if err != nil {
		h.logger.Error("failed to update sync config: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ShareAccessTokenResponse{
			Code:    "500",
			Success: false,
			Message: "failed to update sync config",
		})
		return
	}

	h.logger.Info("sync config updated successfully")
	c.JSON(http.StatusOK, dto.ShareAccessTokenResponse{
		Code:    "0",
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

	appInfo := config.GetAppInfo()
	// 返回版本信息
	c.JSON(http.StatusOK, dto.VersionResponse{
		Code:    http.StatusOK,
		Success: true,
		Message: "ok",
		Data: dto.VersionResponseData{
			AppName:  appInfo.AppName,
			Version:  appInfo.Version,
			OsName:   appInfo.OSName,
			ArchName: appInfo.ArchName,
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
	// 获取请求参数
	var req dto.CheckIgnoreFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.CheckIgnoreFileResponse{
			Code:    "400",
			Success: false,
			Ignore:  false,
			Message: "invalid request format",
		})
		return
	}

	h.logger.Info("check ignore file request: WorkspacePath=%s, WorkspaceName=%s, FilePaths=%v",
		req.WorkspacePath, req.WorkspaceName, req.FilePaths)

	// 参数验证
	if req.WorkspacePath == "" || req.WorkspaceName == "" || len(req.FilePaths) == 0 {
		h.logger.Error("invalid check ignore file parameters")
		c.JSON(http.StatusBadRequest, dto.CheckIgnoreFileResponse{
			Code:    "400",
			Success: false,
			Ignore:  false,
			Message: "invalid parameters",
		})
		return
	}

	clientId := c.GetHeader("Client-ID")
	h.logger.Info("check ignore file request: ClientID=%s, WorkspacePath=%s, WorkspaceName=%s, FilePaths=%v",
		clientId, req.WorkspacePath, req.WorkspaceName, req.FilePaths)

	// 调用service层处理业务逻辑
	result, err := h.extensionService.CheckIgnoreFiles(c.Request.Context(), clientId, req.WorkspacePath, req.WorkspaceName, req.FilePaths)
	if err != nil {
		h.logger.Error("failed to check ignore files: %v", err)
		c.JSON(http.StatusInternalServerError, dto.CheckIgnoreFileResponse{
			Code:    "500",
			Success: false,
			Ignore:  false,
			Message: "internal server error",
		})
		return
	}

	// 根据结果返回响应
	if result.ShouldIgnore {
		c.JSON(http.StatusOK, dto.CheckIgnoreFileResponse{
			Code:    "0",
			Success: false,
			Ignore:  true,
			Message: result.Reason,
		})
		return
	}

	c.JSON(http.StatusOK, dto.CheckIgnoreFileResponse{
		Code:    "0",
		Success: true,
		Ignore:  false,
		Message: result.Reason,
	})
}

// PublishEvents handles workspace events publishing via REST API
// @Summary 发布工作区事件
// @Description 发布工作区事件通知
// @Tags events
// @Accept json
// @Produce json
// @Param request body PublishEventsRequest true "事件发布请求"
// @Success 200 {object} PublishEventsResponse "发布成功"
// @Failure 400 {object} PublishEventsResponse "请求格式错误"
// @Failure 500 {object} PublishEventsResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/events [post]
func (h *ExtensionHandler) PublishEvents(c *gin.Context) {
	// 获取请求参数
	var req dto.PublishEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.PublishEventsResponse{
			Code:    "400",
			Success: false,
			Message: "invalid request format",
			Data:    0,
		})
		return
	}

	clientId := c.GetHeader("Client-ID")
	h.logger.Info("publish events request: Workspace=%s, EventsNum=%d, ClientID=%s", req.Workspace, len(req.Data), clientId)

	// 调用service层处理业务逻辑
	count, err := h.extensionService.PublishEvents(c.Request.Context(), req.Workspace, clientId, req.Data)
	if err != nil {
		h.logger.Error("failed to publish events: %v", err)
		c.JSON(http.StatusInternalServerError, dto.PublishEventsResponse{
			Code:    "500",
			Success: false,
			Message: "failed to publish events",
			Data:    0,
		})
		return
	}

	c.JSON(http.StatusOK, dto.PublishEventsResponse{
		Code:    "0",
		Success: true,
		Message: "ok",
		Data:    count,
	})
}

// TriggerIndex handles manual index building via REST API
// @Summary 手动触发索引构建
// @Description 手动触发代码索引构建
// @Tags index
// @Accept json
// @Produce json
// @Param request body TriggerIndexRequest true "索引构建请求"
// @Success 200 {object} TriggerIndexResponse "触发成功"
// @Failure 400 {object} TriggerIndexResponse "请求格式错误"
// @Failure 500 {object} TriggerIndexResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/index [post]
func (h *ExtensionHandler) TriggerIndex(c *gin.Context) {
	// 获取请求参数
	var req dto.TriggerIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, dto.TriggerIndexResponse{
			Code:    "400",
			Success: false,
			Message: "invalid request format",
			Data:    0,
		})
		return
	}

	if req.Type != dto.IndexTypeAll && req.Type != dto.IndexTypeEmbedding && req.Type != dto.IndexTypeCodegraph {
		h.logger.Error("invalid index type: %s", req.Type)
		c.JSON(http.StatusBadRequest, dto.TriggerIndexResponse{
			Code:    "400",
			Success: false,
			Message: "invalid index type",
			Data:    0,
		})
		return
	}

	// 检查workspace路径文件是否存在
	if _, err := os.Stat(req.Workspace); os.IsNotExist(err) {
		h.logger.Error("workspace path does not exist: %s", req.Workspace)
		c.JSON(http.StatusBadRequest, dto.TriggerIndexResponse{
			Code:    "400",
			Success: false,
			Message: "workspace path does not exist",
			Data:    0,
		})
		return
	}

	clientId := c.GetHeader("Client-ID")
	h.logger.Info("trigger index request: Workspace=%s, Type=%s, ClientID=%s", req.Workspace, req.Type, clientId)

	// 调用service层处理业务逻辑
	err := h.extensionService.TriggerIndex(c.Request.Context(), req.Workspace, req.Type, clientId)
	if err != nil {
		h.logger.Error("failed to trigger index: %v", err)
		c.JSON(http.StatusInternalServerError, dto.TriggerIndexResponse{
			Code:    "500",
			Success: false,
			Message: fmt.Sprintf("failed to trigger index: %v", err),
			Data:    0,
		})
		return
	}

	c.JSON(http.StatusOK, dto.TriggerIndexResponse{
		Code:    "0",
		Success: true,
		Message: "ok",
		Data:    1,
	})
}

// GetIndexStatus handles index status querying via REST API
// @Summary 查询索引状态
// @Description 查询工作区索引构建状态
// @Tags index
// @Accept json
// @Produce json
// @Param clientId query string true "客户端ID" example(111a)
// @Param workspace query string true "工作区路径" example(g:\projects\codebase-indexer)
// @Success 200 {object} IndexStatusResponse "查询成功"
// @Failure 400 {object} IndexStatusResponse "请求参数错误"
// @Failure 500 {object} IndexStatusResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/index/status [get]
func (h *ExtensionHandler) GetIndexStatus(c *gin.Context) {
	// 获取请求参数
	var query dto.IndexStatusQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Error("invalid query parameters: %v", err)
		c.JSON(http.StatusBadRequest, dto.IndexStatusResponse{
			Code:    "400",
			Message: "invalid query parameters",
		})
		return
	}

	h.logger.Info("index status query request: Workspace=%s", query.Workspace)

	// 调用service层处理业务逻辑
	response, err := h.extensionService.GetIndexStatus(c.Request.Context(), query.Workspace)
	if err != nil {
		h.logger.Error("failed to get index status: %v", err)
		c.JSON(http.StatusInternalServerError, dto.IndexStatusResponse{
			Code:    "500",
			Message: "failed to get index status",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SwitchIndex handles index feature switching via REST API
// @Summary 索引功能开关
// @Description 控制索引功能的开启和关闭
// @Tags index
// @Accept json
// @Produce json
// @Param switch query string true "开关状态，on：开启，off：关闭" example(on) Enums(on, off) default(off)
// @Success 200 {object} IndexSwitchResponse "操作成功"
// @Failure 400 {object} IndexSwitchResponse "请求参数错误"
// @Failure 500 {object} IndexSwitchResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/switch [get]
func (h *ExtensionHandler) SwitchIndex(c *gin.Context) {
	// 获取请求参数
	var query dto.IndexSwitchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Error("invalid query parameters: %v", err)
		c.JSON(http.StatusBadRequest, dto.IndexSwitchResponse{
			Code:    "400",
			Success: false,
			Message: "invalid query parameters",
			Data:    false,
		})
		return
	}

	if query.Workspace == "" || query.Switch == "" {
		h.logger.Error("invalid query parameters: %v", query)
		c.JSON(http.StatusBadRequest, dto.IndexSwitchResponse{
			Code:    "400",
			Success: false,
			Message: "invalid query parameters",
			Data:    false,
		})
		return
	}

	if query.Switch != dto.SwitchOn && query.Switch != dto.SwitchOff {
		h.logger.Error("invalid switch status: %s", query.Switch)
		c.JSON(http.StatusBadRequest, dto.IndexSwitchResponse{
			Code:    "400",
			Success: false,
			Message: "invalid query parameters",
			Data:    false,
		})
		return
	}

	clientId := c.GetHeader("Client-ID")
	h.logger.Info("index switch request: Workspace=%s, Switch=%s, ClientID=%s", query.Workspace, query.Switch, clientId)

	err := h.extensionService.SwitchIndex(c, query.Workspace, query.Switch, clientId)
	if err != nil {
		h.logger.Error("failed to switch index: %v", err)
		c.JSON(http.StatusInternalServerError, dto.IndexSwitchResponse{
			Code:    "500",
			Success: false,
			Message: "failed to switch index",
			Data:    false,
		})
		return
	}

	c.JSON(http.StatusOK, dto.IndexSwitchResponse{
		Code:    "0",
		Success: true,
		Message: "ok",
		Data:    true,
	})
}

// UpdateSyncConfig handles sync configuration update via REST API
// @Summary 更新同步配置
// @Description 从Header中获取参数更新同步配置
// @Tags sync
// @Accept json
// @Produce json
// @Success 200 {object} ShareAccessTokenResponse "更新成功"
// @Failure 400 {object} ShareAccessTokenResponse "请求参数错误"
// @Failure 500 {object} ShareAccessTokenResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/sync-config [post]
func (h *ExtensionHandler) UpdateSyncConfig(c *gin.Context) error {
	// 从Header中获取参数
	clientID := c.GetHeader("Client-ID")
	authorization := c.GetHeader("Authorization")
	serverEndpoint := c.GetHeader("Server-Endpoint")

	// 参数验证
	if clientID == "" || authorization == "" || serverEndpoint == "" {
		h.logger.Error("missing required headers: Client-ID=%s, Authorization=%s, Server-Endpoint=%s",
			clientID, authorization, serverEndpoint)
		return fmt.Errorf("missing required headers")
	}

	h.logger.Info("sync config update request: ClientID=%s, ServerEndpoint=%s", clientID, serverEndpoint)

	// 调用service层处理业务逻辑
	token := strings.TrimPrefix(authorization, "Bearer ")
	err := h.extensionService.UpdateSyncConfig(c.Request.Context(), clientID, serverEndpoint, token)
	if err != nil {
		h.logger.Error("failed to update sync config: %v", err)
		return fmt.Errorf("failed to update sync config")
	}

	h.logger.Info("sync config updated successfully")
	return nil
}
