// internal/server/backend_routes.go - 后端API路由配置
package server

import (
	"github.com/gin-gonic/gin"

	"codebase-indexer/internal/handler"
	"codebase-indexer/pkg/logger"
)

// SetupBackendRoutes 设置后端API路由，并为每个路由添加认证中间件
// @Description 设置后端路由
func SetupBackendRoutes(router *gin.Engine, backendHandler *handler.BackendHandler, logger logger.Logger) {
	api := router.Group("/codebase-indexer/api/v1")
	{
		api.GET("/search/reference", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.SearchReference)
		api.GET("/search/definition", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.SearchDefinition)
		api.GET("/files/content", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.GetFileContent)
		api.POST("/snippets/read", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.ReadCodeSnippets)
		api.GET("/codebases/directory", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.GetCodebaseDirectory)
		api.GET("/files/structure", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.GetFileStructure)
		api.GET("/index/summary", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.GetIndexSummary)
		api.GET("/index/export", AuthMiddleware(logger), RateLimitMiddleware(logger), backendHandler.ExportIndex)
	}
}
