package pool

import (
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// ErrPoolClosed 定义包级错误变量，用于错误比较
var ErrPoolClosed = errors.New("task pool is closed")

// Task 任务类型，接收上下文参数和任务ID
type Task func(ctx context.Context, taskID uint64)

// TaskPool 任务池结构体
type TaskPool struct {
	logger         logger.Logger
	maxConcurrency int            // 最大并发数
	tasks          chan Task      // 任务通道
	wg             sync.WaitGroup // 等待组
	mu             sync.Mutex     // 互斥锁
	closed         bool           // 关闭状态
	taskID         uint64         // 任务ID计数器，使用原子操作确保并发安全
}

// NewTaskPool 创建任务池
func NewTaskPool(maxConcurrency int, logger logger.Logger) *TaskPool {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}

	pool := &TaskPool{
		maxConcurrency: maxConcurrency,
		tasks:          make(chan Task, maxConcurrency*2),
		logger:         logger,
		taskID:         0, // 初始任务ID为0
	}

	pool.startWorkers()
	return pool
}

// 启动工作者
func (p *TaskPool) startWorkers() {
	for i := 0; i < p.maxConcurrency; i++ {
		go func(workerID int) {
			// 为每个工作者添加标识，方便日志追踪
			p.logger.Debug("worker %d started", workerID)
			for task := range p.tasks {
				// 生成唯一任务ID（通过原子操作递增）
				taskID := atomic.AddUint64(&p.taskID, 1)
				p.logger.Debug("worker %d starting task %d", workerID, taskID)

				// 执行任务并传入任务ID
				task(context.Background(), taskID)

				p.logger.Debug("worker %d finished task %d", workerID, taskID)
				p.wg.Done()
			}
			p.logger.Debug("worker %d exited", workerID)
		}(i)
	}
}

// Submit 提交任务
func (p *TaskPool) Submit(ctx context.Context, task Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	// 包装任务：处理提交阶段和执行阶段的ctx
	wrappedTask := func(poolCtx context.Context, taskID uint64) {
		// 第一阶段：检查提交后到执行前是否已取消
		select {
		case <-ctx.Done():
			p.logger.Info("task %d cancelled before execution: %v", taskID, ctx.Err())
			return
		default:
			// 第二阶段：执行任务时传入ctx和任务ID
			task(ctx, taskID)
		}
	}

	p.wg.Add(1)
	p.tasks <- wrappedTask
	return nil
}

// Wait 等待所有任务完成
func (p *TaskPool) Wait() {
	p.wg.Wait()
}

// Close 关闭任务池
func (p *TaskPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		close(p.tasks)
		p.closed = true
		p.logger.Info("task pool closed, total tasks processed: %d", p.taskID)
	}
}
