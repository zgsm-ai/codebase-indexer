package pool

import (
	"codebase-indexer/pkg/logger"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 测试正常提交和执行任务
func TestTaskPool_NormalExecution(t *testing.T) {
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	if err != nil {
		panic(err)
	}
	pool := NewTaskPool(2, newLogger)
	defer pool.Close()

	var counter int32
	taskCount := 5

	for i := 0; i < taskCount; i++ {
		err := pool.Submit(context.Background(), func(ctx context.Context, taskId uint64) {
			atomic.AddInt32(&counter, 1)
		})
		if err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}

	pool.Wait()

	if atomic.LoadInt32(&counter) != int32(taskCount) {
		t.Errorf("Incorrect number of tasks executed, expected %d, got %d", taskCount, counter)
	}
}

// 测试任务在等待执行时被取消
func TestTaskPool_CancelBeforeExecution(t *testing.T) {
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	if err != nil {
		panic(err)
	}
	pool := NewTaskPool(1, newLogger) // 只启动一个工作者，确保任务会排队
	defer pool.Close()

	// 第一个任务会占用工作者
	err = pool.Submit(context.Background(), func(ctx context.Context, taskId uint64) {
		time.Sleep(200 * time.Millisecond) // 长时间运行
	})
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// 这个任务会进入队列，然后被取消
	err = pool.Submit(ctx, func(ctx context.Context, taskId uint64) {
		// 这个任务不应该执行
		t.Error("Cancelled task was executed")
	})
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	pool.Wait()
}

// 测试任务执行过程中超时
func TestTaskPool_TaskTimeout(t *testing.T) {
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	if err != nil {
		panic(err)
	}
	pool := NewTaskPool(2, newLogger)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var timedOut bool

	err = pool.Submit(ctx, func(ctx context.Context, taskId uint64) {
		// 模拟长时间运行的任务
		select {
		case <-time.After(200 * time.Millisecond):
			t.Error("Timeout task was not terminated")
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				timedOut = true
			}
		}
	})
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	pool.Wait()

	if !timedOut {
		t.Error("Task did not time out as expected")
	}
}

// 测试任务执行过程中被取消
func TestTaskPool_TaskCancelDuringExecution(t *testing.T) {
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	if err != nil {
		panic(err)
	}
	pool := NewTaskPool(2, newLogger)
	defer pool.Close()

	ctx, cancel := context.WithCancel(context.Background())

	var cancelled bool
	var wg sync.WaitGroup
	wg.Add(1)

	err = pool.Submit(ctx, func(ctx context.Context, taskId uint64) {
		wg.Done() // 通知任务已开始执行
		// 等待取消信号
		select {
		case <-time.After(1 * time.Second):
			t.Error("Cancellable task was not terminated")
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				cancelled = true
			}
		}
	})
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	wg.Wait() // 等待任务开始执行
	cancel()  // 发送取消信号
	pool.Wait()

	if !cancelled {
		t.Error("Task was not cancelled as expected")
	}
}

// 测试关闭任务池后无法提交任务
func TestTaskPool_SubmitAfterClose(t *testing.T) {
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	if err != nil {
		panic(err)
	}
	pool := NewTaskPool(2, newLogger)
	pool.Close()

	err = pool.Submit(context.Background(), func(ctx context.Context, taskId uint64) {})
	if err == nil {
		t.Error("Expected submission to fail but no error was returned")
	}

	if err != ErrPoolClosed {
		t.Errorf("Expected error mismatch, got: %v", err)
	}
}

// 测试并发提交任务
func TestTaskPool_ConcurrentSubmit(t *testing.T) {
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	if err != nil {
		panic(err)
	}
	pool := NewTaskPool(5, newLogger)
	defer pool.Close()

	var counter int32
	taskCount := 1000
	submitters := 10

	var wg sync.WaitGroup
	wg.Add(submitters)

	// 使用全局上下文，避免超时导致任务被取消
	ctx := context.Background()

	// 多goroutine并发提交任务
	for s := 0; s < submitters; s++ {
		go func() {
			defer wg.Done()
			for i := 0; i < taskCount/submitters; i++ {
				err := pool.Submit(ctx, func(ctx context.Context, taskId uint64) {
					atomic.AddInt32(&counter, 1)
					time.Sleep(1 * time.Millisecond) // 模拟轻微工作
				})

				if err != nil {
					t.Errorf("Failed to submit task concurrently: %v", err)
				}
			}
		}()
	}

	wg.Wait()   // 等待所有提交完成
	pool.Wait() // 等待所有任务完成

	if atomic.LoadInt32(&counter) != int32(taskCount) {
		t.Errorf("Incorrect number of concurrent tasks executed, expected %d, got %d", taskCount, counter)
	}
}

// 测试最大并发限制
func TestTaskPool_MaxConcurrency(t *testing.T) {
	maxConcurrency := 3
	newLogger, err := logger.NewLogger("/tmp/logs", "debug")
	if err != nil {
		panic(err)
	}
	pool := NewTaskPool(maxConcurrency, newLogger)
	defer pool.Close()

	var current int32
	var maxConcurrent int32
	taskCount := 10
	var wg sync.WaitGroup
	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		err := pool.Submit(context.Background(), func(ctx context.Context, taskId uint64) {
			defer wg.Done()

			// 增加当前并发数
			c := atomic.AddInt32(&current, 1)

			// 更新最大并发数
			for {
				m := atomic.LoadInt32(&maxConcurrent)
				if c > m {
					if atomic.CompareAndSwapInt32(&maxConcurrent, m, c) {
						break
					}
				} else {
					break
				}
			}

			// 保持任务运行以确保并发检测有效
			time.Sleep(50 * time.Millisecond)

			// 减少当前并发数
			atomic.AddInt32(&current, -1)
		})
		if err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}

	wg.Wait()

	if maxConcurrent != int32(maxConcurrency) {
		t.Errorf("Max concurrency does not match expected, expected %d, got %d", maxConcurrency, maxConcurrent)
	}
}
