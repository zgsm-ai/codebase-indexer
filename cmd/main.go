// cmd/main.go - Program entry
package main

import (
	"codebase-indexer/internal/wiki"
	"codebase-indexer/pkg/codegraph/definition"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"runtime"

	// "net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	// api "codebase-indexer/api"
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/daemon"
	"codebase-indexer/internal/database"
	"codebase-indexer/internal/handler"
	"codebase-indexer/internal/job"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/server"
	"codebase-indexer/internal/service"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/codegraph/analyzer"
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	// "google.golang.org/grpc"
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
	appName := flag.String("appname", "codebase-indexer", "app name")
	// grpcServer := flag.String("grpc", "localhost:51353", "gRPC server address")
	httpServer := flag.String("http", "localhost:11380", "HTTP server address")
	logLevel := flag.String("loglevel", "info", "log level (debug, info, warn, error)")
	enableSwagger := flag.Bool("swagger", false, "enable swagger documentation")
	enablePprof := flag.Bool("pprof", false, "enable pprof profiling")
	pprofAddr := flag.String("pprof-addr", "localhost:6060", "pprof server address")
	flag.Parse()

	// Initialize directories
	if err := initDir(*appName); err != nil {
		fmt.Printf("failed to initialize directory: %v\n", err)
		return
	}
	// Initialize configuration
	if err := initConfig(*appName); err != nil {
		fmt.Printf("failed to initialize configuration: %v\n", err)
		return
	}

	// Update pprof configuration from command line arguments
	clientConfig := config.GetClientConfig()
	clientConfig.Pprof.Enabled = *enablePprof
	clientConfig.Pprof.Address = *pprofAddr
	config.SetClientConfig(clientConfig)
	// syncConfig
	authInfo := config.GetAuthInfo()
	if authInfo.ClientId == types.EmptyString {
		panic("missing required auth.json clientid")
	}
	if authInfo.Token == types.EmptyString {
		panic("missing required auth.json server")
	}
	if authInfo.ServerURL == types.EmptyString {
		panic("missing required auth.json token")
	}
	syncServiceConfig := &config.SyncConfig{
		ClientId:  authInfo.ClientId,
		ServerURL: authInfo.ServerURL,
		Token:     authInfo.Token,
	}

	// Initialize logging system
	appLogger, err := logger.NewLogger(utils.LogsDir, *logLevel, *appName)
	if err != nil {
		fmt.Printf("failed to initialize logging system: %v\n", err)
		return
	}
	appLogger.Info("OS: %s, Arch: %s, App: %s, Version: %s, Starting...", osName, archName, *appName, version)

	// Initialize infrastructure layer
	storageManager, err := repository.NewStorageManager(utils.WorkspaceDir, appLogger)
	if err != nil {
		appLogger.Fatal("failed to initialize workspace manager: %v", err)
		return
	}
	codebaseEmbeddingRepo, err := repository.NewEmbeddingFileRepo(utils.EmbeddingDir, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to create codebase embedding repository: %v", err)
		return
	}

	// Initialize database manager
	dbConfig := config.DefaultDatabaseConfig()
	dbManager := database.NewSQLiteManager(dbConfig, appLogger)
	if err := dbManager.Initialize(); err != nil {
		appLogger.Fatal("failed to initialize database manager: %v", err)
		return
	}

	// Initialize repositories
	workspaceRepo := repository.NewWorkspaceRepository(dbManager, appLogger)
	eventRepo := repository.NewEventRepository(dbManager, appLogger)
	scanRepo := repository.NewFileScanner(appLogger)
	syncRepo := repository.NewHTTPSync(syncServiceConfig, appLogger)

	// Initialize service layer
	schedulerService := service.NewScheduler(syncRepo, scanRepo, storageManager, appLogger)
	fileScanService := service.NewFileScanService(workspaceRepo, eventRepo, scanRepo, storageManager, codebaseEmbeddingRepo, appLogger)
	uploadService := service.NewUploadService(schedulerService, syncRepo, appLogger, syncServiceConfig)
	embeddingProcessService := service.NewEmbeddingProcessService(workspaceRepo, eventRepo, codebaseEmbeddingRepo, uploadService, syncRepo, appLogger)
	embeddingStatusService := service.NewEmbeddingStatusService(codebaseEmbeddingRepo, workspaceRepo, eventRepo, syncRepo, appLogger)

	// 创建存储
	codegraphStore, err := store.NewLevelDBStorage(utils.IndexDir, appLogger)
	defer func(codegraphStore *store.LevelDBStorage) {
		err = codegraphStore.Close()
		if err != nil {
			appLogger.Error("failed to close codegraph store: %v", err)
		}
	}(codegraphStore)

	// 创建工作区读取器
	workspaceReader := workspace.NewWorkSpaceReader(appLogger)

	// 创建源文件解析器
	sourceFileParser := parser.NewSourceFileParser(appLogger)

	packageClassifier := packageclassifier.NewPackageClassifier()

	// 创建依赖分析器
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(appLogger, packageClassifier, workspaceReader, codegraphStore)

	indexer := service.NewCodeIndexer(scanRepo, sourceFileParser, dependencyAnalyzer, workspaceReader, codegraphStore,
		workspaceRepo, service.IndexerConfig{VisitPattern: workspace.DefaultVisitPattern}, appLogger)

	codegraphProcessor := service.NewCodegraphProcessor(workspaceReader, indexer, workspaceRepo, eventRepo, appLogger)

	apiKey := ""
	baseUrl := ""
	model := ""
	if apiKey == types.EmptyString {
		apiKey = os.Getenv("API_KEY")
	}
	if baseUrl == types.EmptyString {
		apiKey = os.Getenv("BASE_URL")
	}
	if model == types.EmptyString {
		model = os.Getenv("MODEL")
	}
	
	wikiManager, err := wiki.NewWikiManager(apiKey, baseUrl, model, appLogger)
	if err != nil {
		panic(fmt.Errorf("init wiki_manager err:%w", err))
	}

	wikiProcessor := service.NewWikiProcessor(workspaceReader, wikiManager, workspaceRepo, eventRepo, appLogger)
	codebaseService := service.NewCodebaseService(storageManager, appLogger, workspaceReader, workspaceRepo, definition.NewDefinitionParser(), indexer)
	extensionService := service.NewExtensionService(storageManager, syncRepo, scanRepo, workspaceRepo, eventRepo, codebaseEmbeddingRepo, codebaseService, fileScanService, appLogger)

	// Initialize job layer
	fileScanJob := job.NewFileScanJob(fileScanService, storageManager, syncRepo, appLogger, 5*time.Minute)
	eventProcessorJob := job.NewEventProcessorJob(appLogger, syncRepo, embeddingProcessService, codegraphProcessor, wikiProcessor, 10*time.Second, storageManager)
	statusCheckerJob := job.NewStatusCheckerJob(embeddingStatusService, storageManager, syncRepo, appLogger, 5*time.Second)
	eventCleanerJob := job.NewEventCleanerJob(eventRepo, appLogger)
	indexCleanJob := job.NewIndexCleanJob(appLogger, indexer, workspaceRepo)
	// Initialize handler layer
	// grpcHandler := handler.NewGRPCHandler(syncRepo, scanRepo, storageManager, schedulerService, appLogger)
	extensionHandler := handler.NewExtensionHandler(extensionService, appLogger)
	backendHandler := handler.NewBackendHandler(codebaseService, appLogger)

	// Initialize gRPC server
	// lis, err := net.Listen("tcp", *grpcServer)
	// if err != nil {
	// 	appLogger.Fatal("failed to listen: %v", err)
	// 	return
	// }
	// s := grpc.NewServer()
	// api.RegisterSyncServiceServer(s, grpcHandler)

	// Initialize HTTP server
	httpServerInstance := server.NewServer(extensionHandler, backendHandler, appLogger)
	if *enableSwagger {
		httpServerInstance.EnableSwagger()
		appLogger.Info("swagger documentation enabled")
	}

	// Start daemonProcess process
	// daemonProcess := daemonProcess.NewDaemon(syncScheduler, s, lis, httpSync, fileScanner, storageManager, appLogger)
	daemonProcess := daemon.NewDaemon(schedulerService, syncRepo, scanRepo, storageManager, appLogger,
		fileScanJob, eventProcessorJob, statusCheckerJob, indexCleanJob, eventCleanerJob)
	go daemonProcess.Start()

	// Start pprof server if enabled
	setupPprof(appLogger)

	// Start HTTP server
	httpErrChan := make(chan error, 1)
	go func() {
		if err := httpServerInstance.Start(*httpServer); err != nil && err != http.ErrServerClosed {
			httpErrChan <- err
		}
		close(httpErrChan)
	}()

	// 等待一小段时间检查HTTP服务器是否启动成功
	select {
	case err := <-httpErrChan:
		if err != nil {
			appLogger.Error("HTTP server failed to start: %v", err)
			return
		}
	case <-time.After(2 * time.Second):
		// 2秒内没有收到错误，认为服务器启动成功
		appLogger.Info("HTTP server started successfully on %s", *httpServer)
	}

	appLogger.Info("application started successfully")
	if *enableSwagger {
		appLogger.Info("swagger documentation available at http://localhost%s/docs", *httpServer)
	}

	// Handle system signals for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	appLogger.Info("received shutdown signal, shutting down gracefully...")
	daemonProcess.Stop()

	// 优雅关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServerInstance.Shutdown(ctx); err != nil {
		appLogger.Error("HTTP server shutdown error: %v", err)
	}

	appLogger.Info("client has been successfully closed")
}
func memStatsHandler(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	json.NewEncoder(w).Encode(memStats)
	w.Header().Set("Content-Type", "application/json")
}
func setupPprof(appLogger logger.Logger) {
	pprofConfig := config.GetClientConfig().Pprof
	if pprofConfig.Enabled {
		go func() {
			pprofMux := http.NewServeMux()
			pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
			pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			pprofMux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			pprofMux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			pprofMux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
			pprofMux.Handle("/debug/pprof/block", pprof.Handler("block"))
			pprofMux.Handle("/debug/pprof/memStats", http.HandlerFunc(memStatsHandler))

			appLogger.Info("pprof server starting on %s", pprofConfig.Address)
			if err := http.ListenAndServe(pprofConfig.Address, pprofMux); err != nil && err != http.ErrServerClosed {
				appLogger.Error("pprof server error: %v", err)
			}
		}()
		appLogger.Info("pprof server listening on %s", pprofConfig.Address)
		appLogger.Info("pprof profiles available at http://%s/debug/pprof/", pprofConfig.Address)
	}
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
	cachePath, err := utils.GetCacheDir(rootPath, appName)
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %v", err)
	}
	fmt.Printf("cache directory: %s\n", cachePath)

	// Initialize env file path
	cacheEnvFilePath, err := utils.GetCacheEnvFile(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get cache env file: %v", err)
	}
	fmt.Printf("cache env file: %s\n", cacheEnvFilePath)

	// Initialize upload temp directory
	uploadTmpPath, err := utils.GetCacheUploadTmpDir(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get upload temporary directory: %v", err)
	}
	fmt.Printf("upload temporary directory: %s\n", uploadTmpPath)

	// Initialize index directory
	indexPath, err := utils.GetCacheIndexDir(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get index directory: %v", err)
	}
	fmt.Printf("index directory: %s\n", indexPath)

	// Initialize cache db directory
	cacheDbPath, err := utils.GetCacheDbDir(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get cache db directory: %v", err)
	}
	fmt.Printf("cache db directory: %s\n", cacheDbPath)

	// Initialize cache workspace directory
	cacheWorkspacePath, err := utils.GetCacheWorkspaceDir(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get cache workspace directory: %v", err)
	}
	fmt.Printf("cache workspace directory: %s\n", cacheWorkspacePath)

	// Initialize cache embedding directory
	cacheEmbeddingPath, err := utils.GetCacheEmbeddingDir(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get cache embedding directory: %v", err)
	}
	fmt.Printf("cache embedding directory: %s\n", cacheEmbeddingPath)

	// Initialize cache deepwiki directory
	cacheDeepwikiPath, err := utils.GetCacheDeepwikiDir(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get cache deepwiki directory: %v", err)
	}
	fmt.Printf("cache deepwiki directory: %s\n", cacheDeepwikiPath)

	// Initialize share auth file
	authFile, err := utils.GetAuthJsonFile(rootPath)
	if err != nil {
		return fmt.Errorf("failed to get auth file: %v", err)
	}
	fmt.Printf("share auth file: %s\n", authFile)

	return nil
}

// initConfig initializes configuration
func initConfig(appName string) error {
	// Set app info
	appInfo := config.AppInfo{
		AppName:  appName,
		ArchName: archName,
		OSName:   osName,
		Version:  version,
	}
	config.SetAppInfo(appInfo)

	// Set client default configuration
	config.SetClientConfig(config.DefaultClientConfig)

	// Load auth configuration
	err := config.LoadAuthConfig()
	if err != nil {
		return err
	}

	return nil
}
