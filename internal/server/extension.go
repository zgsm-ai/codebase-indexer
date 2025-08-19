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
		api.POST("/token", HeaderConfigMiddleware(logger), RateLimitMiddleware(logger), extensionHandler.ShareAccessToken)
		api.POST("/files/ignore", HeaderConfigMiddleware(logger), RateLimitMiddleware(logger), extensionHandler.CheckIgnoreFile)
		api.POST("/events", HeaderConfigMiddleware(logger), RateLimitMiddleware(logger), extensionHandler.PublishEvents)
		api.POST("/index", HeaderConfigMiddleware(logger), RateLimitMiddleware(logger), extensionHandler.TriggerIndex)
		api.GET("/index/status", HeaderConfigMiddleware(logger), RateLimitMiddleware(logger), extensionHandler.GetIndexStatus)
		api.GET("/switch", HeaderConfigMiddleware(logger), RateLimitMiddleware(logger), extensionHandler.SwitchIndex)
	}
}
