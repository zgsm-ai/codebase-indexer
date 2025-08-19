package server

import (
	"github.com/gin-gonic/gin"

	"codebase-indexer/internal/handler"
	"codebase-indexer/pkg/logger"
)

// SetupExtensionRoutes sets up the routes for the extension handlers.
// @Description 设置扩展路由
func SetupExtensionRoutes(router *gin.Engine, extensionHandler *handler.ExtensionHandler, logger logger.Logger) {
	api := router.Group("/codebase-indexer/api/v1")
	{
		api.POST("/token", HeaderConfigMiddleware(logger), ExtensionRateLimitMiddleware(logger), extensionHandler.ShareAccessToken)
		api.POST("/files/ignore", HeaderConfigMiddleware(logger), ExtensionRateLimitMiddleware(logger), extensionHandler.CheckIgnoreFile)
		api.POST("/events", HeaderConfigMiddleware(logger), ExtensionRateLimitMiddleware(logger), extensionHandler.PublishEvents)
		api.POST("/index", HeaderConfigMiddleware(logger), ExtensionRateLimitMiddleware(logger), extensionHandler.TriggerIndex)
		api.GET("/index/status", HeaderConfigMiddleware(logger), ExtensionRateLimitMiddleware(logger), extensionHandler.GetIndexStatus)
		api.GET("/switch", HeaderConfigMiddleware(logger), ExtensionRateLimitMiddleware(logger), extensionHandler.SwitchIndex)
	}
}
