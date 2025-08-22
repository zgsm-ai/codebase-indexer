// internal/handler/backend.go - 代码库索引器HTTP API后端处理器
package handler

import (
	"codebase-indexer/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"

	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// BackendHandler 实现BackendHandler接口的HTTP处理器
type BackendHandler struct {
	codebaseService service.CodebaseService
	logger          logger.Logger
}

// NewBackendHandler 创建新的后端处理器
func NewBackendHandler(codebaseService service.CodebaseService, logger logger.Logger) *BackendHandler {
	return &BackendHandler{
		codebaseService: codebaseService,
		logger:          logger,
	}
}

// ==================== 接口实现 ====================

// SearchReference 关系检索接口
// @Summary 关系检索
// @Description 根据代码位置检索符号的关系信息
// @Tags search
// @Accept json
// @Produce json
// @Param clientId query string true "用户机器ID"
// @Param codebasePath query string true "代码库绝对路径"
// @Param filePath query string true "文件相对路径"
// @Param startLine query int true "开始行号"
// @Param startColumn query int true "开始列号"
// @Param endLine query int true "结束行号"
// @Param endColumn query int true "结束列号"
// @Param symbolName query string false "符号名"
// @Param includeContent query bool false "是否需要返回代码内容"
// @Param maxLayer query int false "最大图层数"
// @Success 200 {object} SearchRelationResponse "成功"
// @Failure 400 {object} SearchRelationResponse "请求参数错误"
// @Failure 500 {object} SearchRelationResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/search/reference [get]
func (h *BackendHandler) SearchReference(c *gin.Context) {
	var req dto.SearchReferenceRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	h.logger.Info("relation search request: ClientId=%s, Workspace=%s, FilePath=%s", req.ClientId, req.CodebasePath, req.FilePath)

	relations, err := h.codebaseService.QueryReference(c, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.OkJson(c, relations)
}

// SearchDefinition 获取代码文件范围的内容定义
// @Summary 获取定义
// @Description 获取一个代码文件范围的内容定义
// @Tags search
// @Accept json
// @Produce json
// @Param clientId query string true "用户机器ID"
// @Param codebasePath query string true "代码库绝对路径"
// @Param filePath query string true "文件相对路径"
// @Param startLine query int false "开始行号"
// @Param endLine query int false "结束行号"
// @Param codeSnippet query string false "代码片段"
// @Success 200 {object} SearchDefinitionResponse "成功"
// @Failure 400 {object} SearchDefinitionResponse "请求参数错误"
// @Failure 500 {object} SearchDefinitionResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/search/definition [get]
func (h *BackendHandler) SearchDefinition(c *gin.Context) {
	var req dto.SearchDefinitionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	h.logger.Info("definition search request: ClientId=%s, Workspace=%s, FilePath=%s", req.ClientId, req.CodebasePath, req.FilePath)

	definitions, err := h.codebaseService.QueryDefinition(c, &req)
	if err != nil {
		h.logger.Error("search definition err:%v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.OkJson(c, definitions)
}

// GetFileContent 获取源文件内容接口
// @Summary 获取文件内容
// @Description 获取源文件内容，以二进制流形式返回
// @Tags files
// @Accept json
// @Produce application/octet-stream
// @Param clientId query string true "用户机器ID"
// @Param codebasePath query string true "代码库绝对地址"
// @Param filePath query string true "文件相对路径"
// @Param startLine query int false "开始行号"
// @Param endLine query int false "结束行号"
// @Success 200 "文件内容二进制流"
// @Failure 400 "请求参数错误"
// @Failure 404 "文件不存在"
// @Failure 500 "服务器内部错误"
// @Router /codebase-indexer/api/v1/files/content [get]
func (h *BackendHandler) GetFileContent(c *gin.Context) {
	var req dto.GetFileContentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	h.logger.Info("get file content request: ClientId=%s, Workspace=%s, FilePath=%s", req.ClientId, req.CodebasePath, req.FilePath)
	content, err := h.codebaseService.GetFileContent(c, &req)
	if err != nil {
		h.logger.Error("get file content err:%v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Bytes(c, content)
}

// GetCodebaseDirectory 获取代码库目录树
// @Summary 获取目录树
// @Description 获取代码库的目录树结构
// @Tags codebases
// @Accept json
// @Produce json
// @Param clientId query string true "用户机器ID"
// @Param codebasePath query string true "项目绝对路径"
// @Param depth query int false "递归深度"
// @Param includeFiles query bool false "是否包含文件"
// @Param subDir query string false "子目录"
// @Success 200 {object} GetCodebaseDirectoryResponse "成功"
// @Failure 400 {object} GetCodebaseDirectoryResponse "请求参数错误"
// @Failure 500 {object} GetCodebaseDirectoryResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/codebases/directory [get]
func (h *BackendHandler) GetCodebaseDirectory(c *gin.Context) {
	var req dto.GetCodebaseDirectoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	// 设置默认值
	if req.Depth == 0 {
		req.Depth = 1
	}

	h.logger.Info("get codebase directory request: ClientId=%s, Workspace=%s, Depth=%d", req.ClientId, req.CodebasePath, req.Depth)

	tree, err := h.codebaseService.GetCodebaseDirectoryTree(c, &req)
	if err != nil {
		h.logger.Error("get codebase directory err:%v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.OkJson(c, tree)
}

// GetFileStructure 获取单个代码文件结构
// @Summary 获取文件结构
// @Description 获取单个代码文件的结构信息
// @Tags files
// @Accept json
// @Produce json
// @Param clientId query string true "用户机器ID"
// @Param codebasePath query string true "项目绝对路径"
// @Param filePath query string true "文件相对路径"
// @Param types query []string false "类型列表"
// @Success 200 {object} GetFileStructureResponse "成功"
// @Failure 400 {object} GetFileStructureResponse "请求参数错误"
// @Failure 500 {object} GetFileStructureResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/files/structure [get]
func (h *BackendHandler) GetFileStructure(c *gin.Context) {
	var req dto.GetFileStructureRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	h.logger.Info("get file structure request: ClientId=%s, Workspace=%s, FilePath=%s", req.ClientId, req.CodebasePath, req.FilePath)

	definitions, err := h.codebaseService.ParseFileDefinitions(c, &req)
	if err != nil {
		h.logger.Info("get file structure err:%v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.OkJson(c, definitions)
}

// GetIndexSummary 获取代码库的索引情况
// @Summary 获取索引情况
// @Description 获取一个代码库的索引情况摘要
// @Tags index
// @Accept json
// @Produce json
// @Param clientId query string true "用户机器ID"
// @Param codebasePath query string true "项目绝对路径"
// @Success 200 {object} GetIndexSummaryResponse "成功"
// @Failure 400 {object} GetIndexSummaryResponse "请求参数错误"
// @Failure 500 {object} GetIndexSummaryResponse "服务器内部错误"
// @Router /codebase-indexer/api/v1/index/summary [get]
func (h *BackendHandler) GetIndexSummary(c *gin.Context) {
	var req dto.GetIndexSummaryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	summarize, err := h.codebaseService.Summarize(c, &req)
	if err != nil {
		h.logger.Error("get index summary: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.OkJson(c, summarize)
}

func (h *BackendHandler) ExportIndex(c *gin.Context) {
	var req dto.ExportIndexRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	err := h.codebaseService.ExportIndex(c, &req)
	if err != nil {
		h.logger.Error("export index err: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
}

func (h *BackendHandler) DeleteIndex(c *gin.Context) {
	var req dto.DeleteIndexRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	err := h.codebaseService.DeleteIndex(c, &req)
	if err != nil {
		h.logger.Error("delete index err: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.Ok(c)
}

func (h *BackendHandler) ReadCodeSnippets(c *gin.Context) {
	var req dto.ReadCodeSnippetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request format: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	list, err := h.codebaseService.ReadCodeSnippets(c, &req)
	if err != nil {
		h.logger.Error("read code snippets err: %v", err)
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	response.OkJson(c, list)
}
