// daemon/daemon.go - 守护进程
package daemon

import (
	"context"
	"net"
	"sync"
	"time"

	"codebase-syncer/internal/scheduler"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/internal/utils"
	"codebase-syncer/pkg/logger"

	"google.golang.org/grpc"
)

type Daemon struct {
	scheduler  *scheduler.Scheduler
	grpcServer *grpc.Server
	grpcListen net.Listener
	httpSync   syncer.SyncInterface
	logger     logger.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	schedWG    sync.WaitGroup // 用于等待scheduler重启
}

func NewDaemon(scheduler *scheduler.Scheduler, grpcServer *grpc.Server, grpcListen net.Listener,
	httpSync syncer.SyncInterface, logger logger.Logger) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())
	return &Daemon{
		scheduler:  scheduler,
		grpcServer: grpcServer,
		grpcListen: grpcListen,
		httpSync:   httpSync,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (d *Daemon) Start() {
	d.logger.Info("守护进程已启动")

	// 启动gRPC服务端
	go func() {
		d.logger.Info("启动gRPC服务端，监听地址: %s", d.grpcListen.Addr().String())
		if err := d.grpcServer.Serve(d.grpcListen); err != nil {
			d.logger.Fatal("failed to serve: %v", err)
			return
		}
	}()

	// 启动同步任务
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.scheduler.Start(d.ctx)
	}()

	// 启动配置检查任务
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				d.logger.Info("配置检查任务已停止")
				return
			case <-ticker.C:
				d.checkAndUpdateConfig()
			}
		}
	}()
}

// checkAndUpdateConfig 检查并更新客户端配置
func (d *Daemon) checkAndUpdateConfig() {
	d.logger.Info("开始检查客户端配置更新")

	// 获取最新客户端配置
	newConfig, err := d.httpSync.GetClientConfig()
	if err != nil {
		d.logger.Error("获取客户端配置失败: %v", err)
		return
	}
	d.logger.Info("已获取最新客户端配置: %+v", newConfig)

	// 获取当前配置
	currentConfig := storage.GetClientConfig()
	if !configChanged(currentConfig, newConfig) {
		d.logger.Info("客户端配置未发生更改")
		return
	}

	// 更新存储中的配置
	storage.SetClientConfig(newConfig)
	// 检查是否需要重启scheduler
	if currentConfig.Sync.IntervalMinutes != newConfig.Sync.IntervalMinutes {
		d.schedWG.Add(1)
		go func() {
			defer d.schedWG.Done()
			d.scheduler.Restart(d.ctx)
		}()
	}

	// 更新其他配置
	d.scheduler.Update(d.ctx)

	d.schedWG.Wait()
	d.logger.Info("客户端配置更新完成")
}

// configChanged 检查配置是否有变化
func configChanged(current, new storage.ClientConfig) bool {
	return current.Server.RegisterExpireMinutes != new.Server.RegisterExpireMinutes ||
		current.Sync.IntervalMinutes != new.Sync.IntervalMinutes ||
		current.Sync.MaxFileSizeMB != new.Sync.MaxFileSizeMB ||
		current.Sync.MaxRetries != new.Sync.MaxRetries ||
		current.Sync.RetryDelaySeconds != new.Sync.RetryDelaySeconds ||
		!equalIgnorePatterns(current.Sync.IgnorePatterns, new.Sync.IgnorePatterns)
}

// equalIgnorePatterns 比较忽略模式是否相同
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

func (d *Daemon) Stop() {
	d.logger.Info("正在停止守护进程...")
	d.cancel()
	utils.CleanUploadTmpDir()
	d.logger.Info("已清理临时目录")
	d.wg.Wait()
	if d.grpcServer != nil {
		d.grpcServer.GracefulStop()
		d.logger.Info("gRPC服务已停止")
	}
	d.logger.Info("守护进程已停止")
}
