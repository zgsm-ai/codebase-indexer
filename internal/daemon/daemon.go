// daemon/daemon.go - Daemon process
package daemon

import (
	"context"
	"time"

	// "net"
	"sync"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/job"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
	// "google.golang.org/grpc"
)

type Daemon struct {
	scheduler *service.Scheduler
	// grpcServer  *grpc.Server
	// grpcListen  net.Listener
	httpSync    repository.SyncInterface
	fileScanner repository.ScannerInterface
	storage     repository.StorageInterface
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	schedWG     sync.WaitGroup // Used to wait for scheduler restart

	// 新增字段
	scannerJob        *job.FileScanJob
	eventProcessorJob *job.EventProcessorJob
	statusCheckerJob  *job.StatusCheckerJob
}

// func NewDaemon(scheduler *scheduler.Scheduler, grpcServer *grpc.Server, grpcListen net.Listener,
//
//	httpSync syncer.SyncInterface, fileScanner scanner.ScannerInterface, storage storage.SotrageInterface, logger logger.Logger) *Daemon {
func NewDaemon(scheduler *service.Scheduler, httpSync repository.SyncInterface,
	fileScanner repository.ScannerInterface, storage repository.StorageInterface, logger logger.Logger,
	scannerJob *job.FileScanJob, eventProcessorJob *job.EventProcessorJob, statusCheckerJob *job.StatusCheckerJob) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())
	return &Daemon{
		scheduler: scheduler,
		// grpcServer:  grpcServer,
		// grpcListen:  grpcListen,
		httpSync:          httpSync,
		fileScanner:       fileScanner,
		storage:           storage,
		logger:            logger,
		ctx:               ctx,
		cancel:            cancel,
		scannerJob:        scannerJob,
		eventProcessorJob: eventProcessorJob,
		statusCheckerJob:  statusCheckerJob,
	}
}

// Start starts the daemon process
func (d *Daemon) Start() {
	d.logger.Info("daemon process started")

	// Update configuration on startup
	if d.httpSync.GetSyncConfig() != nil {
		d.updateConfig()
	}

	// Start gRPC server
	// go func() {
	// 	d.logger.Info("starting gRPC server, listening on: %s", d.grpcListen.Addr().String())
	// 	if err := d.grpcServer.Serve(d.grpcListen); err != nil {
	// 		d.logger.Fatal("gRPC server failed to serve: %v", err)
	// 		return
	// 	}
	// }()

	// Start sync task
	// d.wg.Add(1)
	// go func() {
	// 	defer d.wg.Done()
	// 	d.scheduler.Start(d.ctx)
	// }()

	// Start config check task
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				d.logger.Info("config check task stopped")
				return
			case <-ticker.C:
				d.checkAndLoadConfig()
			}
		}
	}()

	// Start fetch server hash tree task
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				d.logger.Info("fetch server hash task stopped")
				return
			case <-ticker.C:
				d.fetchServerHashTree()
			}
		}
	}()

	// 启动文件扫描任务（5分钟间隔）
	d.startFileScannerTask()

	// 启动事件处理协程任务
	d.startEventProcessorTask()

	// 启动状态检查任务（3秒间隔）
	d.startStatusCheckerTask()
}

// updateConfig updates client configuration
func (d *Daemon) updateConfig() {
	d.logger.Info("updating client config")

	// Value latest client configuration
	newConfig, err := d.httpSync.GetClientConfig()
	if err != nil {
		d.logger.Error("failed to get client config: %v", err)
		return
	}
	d.logger.Info("latest client config retrieved: %+v", newConfig)

	// Value current configuration
	currentConfig := config.GetClientConfig()
	if !configChanged(currentConfig, newConfig) {
		d.logger.Info("client config unchanged")
		return
	}

	// Update storage configuration
	config.SetClientConfig(newConfig)
	// Update scheduler configuration
	d.scheduler.SetSchedulerConfig(&service.SchedulerConfig{
		IntervalMinutes:       newConfig.Sync.IntervalMinutes,
		RegisterExpireMinutes: newConfig.Server.RegisterExpireMinutes,
		HashTreeExpireHours:   newConfig.Server.HashTreeExpireHours,
	})
	// Update file scanner configuration
	d.fileScanner.SetScannerConfig(&config.ScannerConfig{
		FileIgnorePatterns:   newConfig.Sync.FileIgnorePatterns,
		FolderIgnorePatterns: newConfig.Sync.FolderIgnorePatterns,
		FileIncludePatterns:  newConfig.Sync.FileIncludePatterns,
		MaxFileSizeKB:        newConfig.Sync.MaxFileSizeKB,
	})

	d.logger.Info("client config updated")
}

// checkAndLoadConfig checks and loads latest client configuration
func (d *Daemon) checkAndLoadConfig() {
	d.logger.Info("starting client config load check")

	// Value latest client configuration
	newConfig, err := d.httpSync.GetClientConfig()
	if err != nil {
		d.logger.Error("failed to get client config: %v", err)
		return
	}
	d.logger.Info("latest client config retrieved: %+v", newConfig)

	// Value current configuration
	currentConfig := config.GetClientConfig()
	if !configChanged(currentConfig, newConfig) {
		d.logger.Info("client config unchanged")
		return
	}

	// Update stored configuration
	config.SetClientConfig(newConfig)
	// Check if scheduler needs restart
	if currentConfig.Sync.IntervalMinutes != newConfig.Sync.IntervalMinutes {
		d.schedWG.Add(1)
		go func() {
			defer d.schedWG.Done()
			d.scheduler.Restart(d.ctx)
		}()
	}

	// Load latest configuration
	d.scheduler.LoadConfig(d.ctx)

	d.schedWG.Wait()
	d.logger.Info("client config load completed")
}

// configChanged checks if configuration has changed
func configChanged(current, new config.ClientConfig) bool {
	return current.Server.RegisterExpireMinutes != new.Server.RegisterExpireMinutes ||
		current.Server.HashTreeExpireHours != new.Server.HashTreeExpireHours ||
		current.Sync.IntervalMinutes != new.Sync.IntervalMinutes ||
		current.Sync.MaxFileSizeKB != new.Sync.MaxFileSizeKB ||
		current.Sync.MaxRetries != new.Sync.MaxRetries ||
		current.Sync.RetryDelaySeconds != new.Sync.RetryDelaySeconds ||
		!equalIgnorePatterns(current.Sync.FileIgnorePatterns, new.Sync.FileIgnorePatterns) ||
		!equalIgnorePatterns(current.Sync.FolderIgnorePatterns, new.Sync.FolderIgnorePatterns) ||
		!equalIgnorePatterns(current.Sync.FileIncludePatterns, new.Sync.FileIncludePatterns)
}

// equalIgnorePatterns compares whether ignore patterns are same
func equalIgnorePatterns(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// fetchServerHashTree fetches the latest server hash tree
func (d *Daemon) fetchServerHashTree() {
	d.logger.Info("starting server hash tree fetch")

	codebaseConfigs := d.storage.GetCodebaseConfigs()
	if len(codebaseConfigs) == 0 {
		d.logger.Warn("no codebase config, skip hash tree fetch")
		return
	}

	for _, codebaseConfig := range codebaseConfigs {
		hashTree, err := d.httpSync.FetchServerHashTree(codebaseConfig.CodebasePath)
		if err != nil {
			d.logger.Warn("failed to fetch server hash tree: %v", err)
			continue
		}
		codebaseConfig.HashTree = hashTree
		err = d.storage.SaveCodebaseConfig(codebaseConfig)
		if err != nil {
			d.logger.Warn("failed to save server hash tree: %v", err)
		}
	}

	d.logger.Info("server hash tree fetch completed")
}

// Stop stops the daemon process
func (d *Daemon) Stop() {
	d.logger.Info("stopping daemon process...")
	d.cancel()

	// 停止文件扫描任务
	if d.scannerJob != nil {
		d.scannerJob.Stop()
		d.logger.Info("file scanner task stopped")
	}

	// 停止事件处理协程任务
	if d.eventProcessorJob != nil {
		d.eventProcessorJob.Stop()
		d.logger.Info("event processor task stopped")
	}

	// 停止状态检查任务
	if d.statusCheckerJob != nil {
		d.statusCheckerJob.Stop()
		d.logger.Info("status checker task stopped")
	}

	utils.CleanUploadTmpDir()
	d.logger.Info("temp directory cleaned up")
	d.wg.Wait()
	// if d.grpcServer != nil {
	// 	d.grpcServer.GracefulStop()
	// 	d.logger.Info("gRPC service stopped")
	// }
	d.logger.Info("daemon process stopped")
}

// startFileScannerTask 启动文件扫描任务
func (d *Daemon) startFileScannerTask() {
	d.logger.Info("starting file scanner task...")
	d.scannerJob.Start()
}

// startEventProcessorTask 启动事件处理协程任务
func (d *Daemon) startEventProcessorTask() {
	d.logger.Info("starting event processor task...")
	d.eventProcessorJob.Start()
}

// startStatusCheckerTask 启动状态检查任务
func (d *Daemon) startStatusCheckerTask() {
	d.logger.Info("starting status checker task...")
	d.statusCheckerJob.Start()
}
