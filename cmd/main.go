// cmd/main.go - Program entry
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	api "codebase-syncer/api"
	"codebase-syncer/internal/daemon"
	"codebase-syncer/internal/handler"
	"codebase-syncer/internal/scanner"
	"codebase-syncer/internal/scheduler"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/internal/utils"
	"codebase-syncer/pkg/logger"

	"google.golang.org/grpc"
)

var (
	// version will be set by the linker during build
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
	logLevel := flag.String("loglevel", "info", "log level (debug, info, warn, error)")
	clientId := flag.String("clientid", "", "client id")
	serverEndpoint := flag.String("server", "", "server endpoint")
	token := flag.String("token", "", "authentication token")
	flag.Parse()

	// Initialize directories
	if err := initDir(*appName); err != nil {
		fmt.Printf("failed to initialize directory: %v\n", err)
		return
	}
	// Initialize configuration
	initConfig()

	// Initialize logging system
	logger, err := logger.NewLogger(utils.LogsDir, *logLevel)
	if err != nil {
		fmt.Printf("failed to initialize logging system: %v\n", err)
		return
	}
	logger.Info("OS: %s, Arch: %s, App: %s, Version: %s, Starting...", osName, archName, *appName, version)

	// Initialize all modules
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
	appInfo := &handler.AppInfo{AppName: *appName, ArchName: archName, OSName: osName, Version: version}
	syncScheduler := scheduler.NewScheduler(httpSync, fileScanner, storageManager, logger)
	grpcHandler := handler.NewGRPCHandler(httpSync, storageManager, syncScheduler, logger, appInfo)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", *grpcServer)
	if err != nil {
		logger.Fatal("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()
	api.RegisterSyncServiceServer(s, grpcHandler)

	// Start daemon process
	daemon := daemon.NewDaemon(syncScheduler, s, lis, httpSync, fileScanner, logger)
	go daemon.Start()

	// Handle system signals for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	logger.Info("received shutdown signal, shutting down gracefully...")
	daemon.Stop()
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
func initConfig() {
	// Set client default configuration
	storage.SetClientConfig(storage.DefaultClientConfig)
}
