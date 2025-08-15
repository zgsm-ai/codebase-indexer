// internal/server/router.go - 路由配置和服务器初始化
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"codebase-indexer/internal/handler"
	"codebase-indexer/pkg/logger"
)

// Server 服务器接口
type Server interface {
	Start(addr string) error
	Shutdown(ctx context.Context) error
	EnableSwagger()
}

// NewServer 创建新的HTTP服务器
func NewServer(
	extensionHandler *handler.ExtensionHandler,
	backendHandler *handler.BackendHandler,
	logger logger.Logger,
) Server {
	return &server{
		extensionHandler: extensionHandler,
		backendHandler:   backendHandler,
		logger:           logger,
	}
}

type server struct {
	engine           *gin.Engine
	extensionHandler *handler.ExtensionHandler
	backendHandler   *handler.BackendHandler
	logger           logger.Logger
	httpServer       *http.Server
	swaggerEnabled   bool
}

// Start 启动服务器
func (s *server) Start(addr string) error {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin引擎
	s.engine = gin.New()

	// 设置中间件
	s.setupMiddleware()

	// 设置路由
	s.setupRoutes()

	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        s.engine,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	s.logger.Info("starting HTTP server on %s", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown 优雅关闭服务器
func (s *server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		s.logger.Info("shutting down HTTP server")
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// EnableSwagger 启用swagger文档
func (s *server) EnableSwagger() {
	s.swaggerEnabled = true
}

// setupMiddleware 设置中间件
func (s *server) setupMiddleware() {
	// 基础中间件
	s.engine.Use(RecoveryMiddleware(s.logger))
	s.engine.Use(LoggingMiddleware(s.logger))
	s.engine.Use(CORSMiddleware())
	s.engine.Use(SecurityMiddleware())
	s.engine.Use(RateLimitMiddleware(s.logger))

	// 健康检查
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
			"success": true,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Swagger文档路由
	if s.swaggerEnabled {
		s.setupSwaggerRoutes()
	}
}

// setupSwaggerRoutes 设置swagger文档路由
func (s *server) setupSwaggerRoutes() {
	// 静态文件服务
	s.engine.Static("/swagger-ui", "./docs/swagger-ui")

	// API文档路由
	s.engine.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger-ui/index.html")
	})

	s.logger.Info("swagger documentation routes enabled")
}

// setupRoutes 设置路由
func (s *server) setupRoutes() {
	// API路由
	SetupExtensionRoutes(s.engine, s.extensionHandler, s.logger)
	SetupBackendRoutes(s.engine, s.backendHandler, s.logger)

	// 404处理
	s.engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "endpoint not found",
		})
	})

	// 405处理
	s.engine.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"success": false,
			"message": "method not allowed",
		})
	})
}

// GetEngine 获取Gin引擎（用于测试）
func (s *server) GetEngine() *gin.Engine {
	return s.engine
}
