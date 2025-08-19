// internal/server/middleware.go - 中间件定义
package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// RecoveryMiddleware panic恢复中间件
func RecoveryMiddleware(logger logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			logger.Error("panic recovered: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "internal server error",
			})
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// LoggingMiddleware 请求日志中间件
func LoggingMiddleware(logger logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("[GIN] %s %s %d %s %s %s",
			method,
			path,
			statusCode,
			latency,
			clientIP,
			errorMessage,
		)
	}
}

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityMiddleware 安全中间件
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// AuthMiddleware 认证中间件
// 验证请求Header中的Authorization字段是否与配置中的token值一致
func AuthMiddleware(logger logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Error("missing Authorization header")
			utils.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		// 去除Bearer前缀（如果有）
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		// 获取配置中的token
		authInfo := config.GetAuthInfo()
		configToken := authInfo.Token

		// 验证token是否匹配
		if token != configToken {
			logger.Error("expired token: %s", token)
			utils.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// token验证通过，继续处理请求
		c.Next()
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(logger logger.Logger) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Every(time.Second), 100)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			logger.Error("rate limit exceeded")
			utils.TooManyRequests(c, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

// HeaderConfigMiddleware 头信息配置中间件
// 检查请求头中的Client-ID、Authorization、Server-Endpoint信息并更新配置
func HeaderConfigMiddleware(logger logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求头信息
		clientID := c.GetHeader("Client-ID")
		authorization := c.GetHeader("Authorization")
		serverEndpoint := c.GetHeader("Server-Endpoint")

		// 检查三个头信息是否都存在
		if clientID == "" {
			logger.Error("missing Client-ID header")
			utils.BadRequest(c, "Client-ID header is required")
			c.Abort()
			return
		}

		if authorization == "" {
			logger.Error("missing Authorization header")
			utils.BadRequest(c, "Authorization header is required")
			c.Abort()
			return
		}

		if serverEndpoint == "" {
			logger.Error("missing Server-Endpoint header")
			utils.BadRequest(c, "Server-Endpoint header is required")
			c.Abort()
			return
		}

		// 三个头信息都存在，更新配置
		// 获取当前的authInfo
		authInfo := config.GetAuthInfo()

		// 创建新的authInfo副本
		newAuthInfo := authInfo

		// 去除Authorization的Bearer前缀（如果有）
		token := strings.TrimPrefix(authorization, "Bearer ")
		token = strings.TrimSpace(token)

		// 更新所有字段
		if newAuthInfo.ClientId != clientID || newAuthInfo.ServerURL != serverEndpoint || newAuthInfo.Token != token {
			newAuthInfo.ClientId = clientID
			newAuthInfo.Token = token
			newAuthInfo.ServerURL = serverEndpoint
			// 更新全局authInfo配置
			config.SetAuthInfo(newAuthInfo)
			logger.Info("authInfo configuration updated from request headers - Client-ID: %s, Server-Endpoint: %s", clientID, serverEndpoint)
		}

		// 继续处理请求
		c.Next()
	}
}
