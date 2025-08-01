package job

import (
	"context"
	"sync"
	"time"

	"codebase-indexer/internal/service"
	"codebase-indexer/pkg/logger"
)

// EventProcessorJob 事件处理任务
type EventProcessorJob struct {
	processor service.EventProcessService
	logger    logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewEventProcessorJob 创建事件处理任务
func NewEventProcessorJob(
	processor service.EventProcessService,
	logger logger.Logger,
) *EventProcessorJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventProcessorJob{
		processor: processor,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动事件处理任务
func (j *EventProcessorJob) Start() {
	j.logger.Info("starting event processor job")

	j.wg.Add(1)
	go func() {
		defer j.wg.Done()

		for {
			select {
			case <-j.ctx.Done():
				j.logger.Info("event processor job stopped")
				return
			default:
				err := j.processor.ProcessEvents()
				if err != nil {
					j.logger.Error("failed to process events: %v", err)
				}
				// 短暂休眠避免CPU占用过高
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop 停止事件处理任务
func (j *EventProcessorJob) Stop() {
	j.logger.Info("stopping event processor job...")
	j.cancel()
	j.wg.Wait()
	j.logger.Info("event processor job stopped")
}
