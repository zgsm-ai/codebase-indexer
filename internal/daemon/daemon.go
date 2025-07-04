// daemon/daemon.go - Daemon process
package daemon

import (
	"context"
	"net"
	"sync"
	"time"

	"codebase-syncer/internal/scanner"
	"codebase-syncer/internal/scheduler"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/internal/utils"
	"codebase-syncer/pkg/logger"

	"google.golang.org/grpc"
)

type Daemon struct {
	scheduler   *scheduler.Scheduler
	grpcServer  *grpc.Server
	grpcListen  net.Listener
	httpSync    syncer.SyncInterface
	fileScanner scanner.ScannerInterface
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	schedWG     sync.WaitGroup // Used to wait for scheduler restart
}

func NewDaemon(scheduler *scheduler.Scheduler, grpcServer *grpc.Server, grpcListen net.Listener,
	httpSync syncer.SyncInterface, fileScanner scanner.ScannerInterface, logger logger.Logger) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())
	return &Daemon{
		scheduler:   scheduler,
		grpcServer:  grpcServer,
		grpcListen:  grpcListen,
		httpSync:    httpSync,
		fileScanner: fileScanner,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
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
	go func() {
		d.logger.Info("starting gRPC server, listening on: %s", d.grpcListen.Addr().String())
		if err := d.grpcServer.Serve(d.grpcListen); err != nil {
			d.logger.Fatal("gRPC server failed to serve: %v", err)
			return
		}
	}()

	// Start sync task
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.scheduler.Start(d.ctx)
	}()

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
}

// updateConfig updates client configuration
func (d *Daemon) updateConfig() {
	d.logger.Info("updating client config")

	// Get latest client configuration
	newConfig, err := d.httpSync.GetClientConfig()
	if err != nil {
		d.logger.Error("failed to get client config: %v", err)
		return
	}
	d.logger.Info("latest client config retrieved: %+v", newConfig)

	// Get current configuration
	currentConfig := storage.GetClientConfig()
	if !configChanged(currentConfig, newConfig) {
		d.logger.Info("client config unchanged")
		return
	}

	// Update storage configuration
	storage.SetClientConfig(newConfig)
	// Update scheduler configuration
	d.scheduler.SetSchedulerConfig(&scheduler.SchedulerConfig{
		IntervalMinutes:       newConfig.Sync.IntervalMinutes,
		RegisterExpireMinutes: newConfig.Server.RegisterExpireMinutes,
		HashTreeExpireHours:   newConfig.Server.HashTreeExpireHours,
	})
	// Update file scanner configuration
	d.fileScanner.SetScannerConfig(&scanner.ScannerConfig{
		IgnorePatterns: newConfig.Sync.IgnorePatterns,
		// MaxFileSizeMB:  newConfig.Sync.MaxFileSizeMB,
		MaxFileSizeKB: newConfig.Sync.MaxFileSizeKB,
	})

	d.logger.Info("client config updated")
}

// checkAndLoadConfig checks and loads latest client configuration
func (d *Daemon) checkAndLoadConfig() {
	d.logger.Info("starting client config load check")

	// Get latest client configuration
	newConfig, err := d.httpSync.GetClientConfig()
	if err != nil {
		d.logger.Error("failed to get client config: %v", err)
		return
	}
	d.logger.Info("latest client config retrieved: %+v", newConfig)

	// Get current configuration
	currentConfig := storage.GetClientConfig()
	if !configChanged(currentConfig, newConfig) {
		d.logger.Info("client config unchanged")
		return
	}

	// Update stored configuration
	storage.SetClientConfig(newConfig)
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
func configChanged(current, new storage.ClientConfig) bool {
	return current.Server.RegisterExpireMinutes != new.Server.RegisterExpireMinutes ||
		current.Server.HashTreeExpireHours != new.Server.HashTreeExpireHours ||
		current.Sync.IntervalMinutes != new.Sync.IntervalMinutes ||
		// current.Sync.MaxFileSizeMB != new.Sync.MaxFileSizeMB ||
		current.Sync.MaxFileSizeKB != new.Sync.MaxFileSizeKB ||
		current.Sync.MaxRetries != new.Sync.MaxRetries ||
		current.Sync.RetryDelaySeconds != new.Sync.RetryDelaySeconds ||
		!equalIgnorePatterns(current.Sync.IgnorePatterns, new.Sync.IgnorePatterns)
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

// Stop stops the daemon process
func (d *Daemon) Stop() {
	d.logger.Info("stopping daemon process...")
	d.cancel()
	utils.CleanUploadTmpDir()
	d.logger.Info("temp directory cleaned up")
	d.wg.Wait()
	if d.grpcServer != nil {
		d.grpcServer.GracefulStop()
		d.logger.Info("gRPC service stopped")
	}
	d.logger.Info("daemon process stopped")
}
