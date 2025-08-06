package pool

import (
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"sync"
)

// ErrPoolClosed 定义包级错误变量，用于错误比较
var ErrPoolClosed = errors.New("task pool is closed")

// Task 任务类型，接收上下文参数
type Task func(ctx context.Context)

// TaskPool 任务池结构体
type TaskPool struct {
	logger         logger.Logger
	maxConcurrency int            // 最大并发数
	tasks          chan Task      // 任务通道
	wg             sync.WaitGroup // 等待组
	mu             sync.Mutex     // 互斥锁
	closed         bool           // 关闭状态
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
	}

	pool.startWorkers()
	return pool
}

// 启动工作者
func (p *TaskPool) startWorkers() {
	for i := 0; i < p.maxConcurrency; i++ {
		go func() {
			for task := range p.tasks {
				task(context.Background()) // 执行任务
				p.wg.Done()
			}
		}()
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
	wrappedTask := func(poolCtx context.Context) {
		// 第一阶段：检查提交后到执行前是否已取消
		select {
		case <-ctx.Done():
			p.logger.Info("task_pool task cancelled before execution: %v", ctx.Err())
			return
		default:
			// 第二阶段：执行任务时传入ctx
			task(ctx)
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
	}
}
