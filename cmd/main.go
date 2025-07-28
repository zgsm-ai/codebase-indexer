// cmd/main.go - Program entry
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "codebase-indexer/api"
	"codebase-indexer/internal/daemon"
	"codebase-indexer/internal/handler"
	"codebase-indexer/internal/scanner"
	"codebase-indexer/internal/scheduler"
	"codebase-indexer/internal/server"
	"codebase-indexer/internal/service"
	"codebase-indexer/internal/storage"
	"codebase-indexer/internal/syncer"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"

	"google.golang.org/grpc"
)

var (
	// set by the linker during build
	osName   string
	archName string
	version  string
)

func main() {
	if osName != "" {
		fmt.Printf("OS: %s\n", osName)
	}
	if archName != "" {
		fmt.Printf("Arch: %s\n", archName)
	}
	if version != "" {
		fmt.Printf("Version: %s\n", version)
	}

	// Parse command line arguments
	appName := flag.String("appname", "zgsm", "app name")
	grpcServer := flag.String("grpc", "localhost:51353", "gRPC server address")
	httpServer := flag.String("http", "localhost:11380", "HTTP server address")
	logLevel := flag.String("loglevel", "info", "log level (debug, info, warn, error)")
	clientId := flag.String("clientid", "", "client id")
	serverEndpoint := flag.String("server", "", "server endpoint")
	token := flag.String("token", "", "authentication token")
	enableSwagger := flag.Bool("swagger", false, "enable swagger documentation")
	flag.Parse()

	// Initialize directories
	if err := initDir(*appName); err != nil {
		fmt.Printf("failed to initialize directory: %v\n", err)
		return
	}
	// Initialize configuration
	initConfig(*appName)

	// Initialize logging system
	logger, err := logger.NewLogger(utils.LogsDir, *logLevel)
	if err != nil {
		fmt.Printf("failed to initialize logging system: %v\n", err)
		return
	}
	logger.Info("OS: %s, Arch: %s, App: %s, Version: %s, Starting...", osName, archName, *appName, version)

	// Initialize infrastructure layer
	storageManager, err := storage.NewStorageManager(utils.CacheDir, logger)
	if err != nil {
		logger.Fatal("failed to initialize storage manager: %v", err)
		return
	}
	fileScanner := scanner.NewFileScanner(logger)
	var syncConfig *syncer.SyncConfig
	if *clientId != "" && *serverEndpoint != "" && *token != "" {
		syncConfig = &syncer.SyncConfig{ClientId: *clientId, ServerURL: *serverEndpoint, Token: *token}
	}
	httpSync := syncer.NewHTTPSync(syncConfig, logger)
	syncScheduler := scheduler.NewScheduler(httpSync, fileScanner, storageManager, logger)

	// Initialize service layer
	codebaseService := service.NewCodebaseService(logger)
	syncService := service.NewExtensionService(storageManager, httpSync, fileScanner, codebaseService, logger)

	// Initialize handler layer
	grpcHandler := handler.NewGRPCHandler(httpSync, fileScanner, storageManager, syncScheduler, logger)
	extensionHandler := handler.NewExtensionHandler(syncService, logger)
	backendHandler := handler.NewBackendHandler(codebaseService, logger)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", *grpcServer)
	if err != nil {
		logger.Fatal("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()
	api.RegisterSyncServiceServer(s, grpcHandler)

	// Initialize HTTP server
	httpServerInstance := server.NewServer(extensionHandler, backendHandler, logger)
	if *enableSwagger {
		httpServerInstance.EnableSwagger()
		logger.Info("swagger documentation enabled")
	}

	// Start daemon process
	daemon := daemon.NewDaemon(syncScheduler, s, lis, httpSync, fileScanner, storageManager, logger)
	go daemon.Start()

	// Start HTTP server
	go func() {
		if err := httpServerInstance.Start(*httpServer); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	logger.Info("application started successfully")
	logger.Info("gRPC server listening on %s", *grpcServer)
	logger.Info("HTTP server listening on %s", *httpServer)
	if *enableSwagger {
		logger.Info("swagger documentation available at http://localhost%s/docs", *httpServer)
	}

	// Handle system signals for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	logger.Info("received shutdown signal, shutting down gracefully...")
	daemon.Stop()

	// 优雅关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServerInstance.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error: %v", err)
	}

	logger.Info("client has been successfully closed")
}

// initDir initializes directories
func initDir(appName string) error {
	// Initialize root directory
	rootPath, err := utils.GetRootDir(appName)
	if err != nil {
		return fmt.Errorf("failed to get root directory: %v", err)
	}
	fmt.Printf("root directory: %s\n", rootPath)

	// Initialize log directory
	logPath, err := utils.GetLogDir(rootPath)
	if err != nil {
		return fmt.Errorf("failed to get log directory: %v", err)
	}
	fmt.Printf("log directory: %s\n", logPath)

	// Initialize cache directory
	cachePath, err := utils.GetCacheDir(rootPath)
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %v", err)
	}
	fmt.Printf("cache directory: %s\n", cachePath)

	// Initialize upload temp directory
	uploadTmpPath, err := utils.GetUploadTmpDir(rootPath)
	if err != nil {
		return fmt.Errorf("failed to get upload temporary directory: %v", err)
	}
	fmt.Printf("upload temporary directory: %s\n", uploadTmpPath)

	return nil
}

// initConfig initializes configuration
func initConfig(appName string) {
	// Set app info
	appInfo := storage.AppInfo{
		AppName:  appName,
		ArchName: archName,
		OSName:   osName,
		Version:  version,
	}
	storage.SetAppInfo(appInfo)
	// Set client default configuration
	storage.SetClientConfig(storage.DefaultClientConfig)
}
