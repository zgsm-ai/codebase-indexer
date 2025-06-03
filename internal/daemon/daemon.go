// daemon/daemon.go - 守护进程
package daemon

import (
	"context"
	"sync"
	"time"

	"codebase-syncer/internal/handler"
	"codebase-syncer/internal/scheduler"
	"codebase-syncer/internal/utils"
	"codebase-syncer/pkg/logger"
)

type Daemon struct {
	scheduler   *scheduler.Scheduler
	grpcHandler *handler.GRPCHandler
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewDaemon(scheduler *scheduler.Scheduler, grpcHandler *handler.GRPCHandler,
	logger logger.Logger) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())
	return &Daemon{
		scheduler:   scheduler,
		grpcHandler: grpcHandler,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (d *Daemon) Start() {
	d.logger.Info("守护进程已启动")

	// 启动同步任务
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.scheduler.Start(d.ctx)
	}()

	// 启动心跳检查
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				d.logger.Info("心跳检查已停止")
				return
			case <-ticker.C:
				// TODO: 判断注册的项目是否存活
			}
		}
	}()
}

func (d *Daemon) Stop() {
	d.logger.Info("正在停止守护进程...")
	d.cancel()
	utils.CleanUploadTmpDir()
	d.wg.Wait()
	d.logger.Info("守护进程已停止")
}
