package cache

//
//import (
//	"fmt"
//	"sync"
//	"testing"
//)
//
//// 基础功能测试用例
//func TestLRUCache_Basic(t *testing.T) {
//	// 测试用例表格：每个用例包含容量、操作序列、预期结果
//	testCases := []struct {
//		name     string
//		capacity int
//		ops      []op[int]   // 操作序列（put或get）
//		wants    []want[int] // 预期结果（仅get操作有结果）
//	}{
//		{
//			name:     "capacity 2, basic put and get",
//			capacity: 2,
//			ops: []op[int]{
//				{typ: "put", key: "a", val: 1},
//				{typ: "put", key: "b", val: 2},
//				{typ: "get", key: "a"},         // 预期 (1, true)
//				{typ: "put", key: "c", val: 3}, // 淘汰b
//				{typ: "get", key: "b"},         // 预期 (0, false)
//				{typ: "get", key: "c"},         // 预期 (3, true)
//			},
//			wants: []want[int]{
//				{val: 1, ok: true},
//				{val: 0, ok: false},
//				{val: 3, ok: true},
//			},
//		},
//		{
//			name:     "update existing key",
//			capacity: 1,
//			ops: []op[int]{
//				{typ: "put", key: "x", val: 10},
//				{typ: "put", key: "x", val: 20}, // 更新值
//				{typ: "get", key: "x"},          // 预期 (20, true)
//			},
//			wants: []want[int]{
//				{val: 20, ok: true},
//			},
//		},
//		{
//			name:     "get non-existent key",
//			capacity: 3,
//			ops: []op[int]{
//				{typ: "get", key: "none"}, // 预期 (0, false)
//			},
//			wants: []want[int]{
//				{val: 0, ok: false},
//			},
//		},
//	}
//
//	// 执行测试用例
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			cache := NewLRUCache[int](tc.capacity)
//			wantIdx := 0 // 跟踪预期结果的索引
//
//			for _, op := range tc.ops {
//				switch op.typ {
//				case "put":
//					cache.Put(op.key, op.val)
//				case "get":
//					val, ok := cache.Get(op.key)
//					want := tc.wants[wantIdx]
//					if val != want.val || ok != want.ok {
//						t.Errorf("Get(%q) = (%v, %v), want (%v, %v)",
//							op.key, val, ok, want.val, want.ok)
//					}
//					wantIdx++
//				}
//			}
//		})
//	}
//}
//
//func TestLRUCache_Concurrent(t *testing.T) {
//	const (
//		capacity        = 1000 // 增大容量，避免频繁淘汰
//		numGoroutines   = 10   // 并发 goroutine 数量
//		opsPerGoroutine = 100  // 每个 goroutine 操作次数
//	)
//
//	cache := NewLRUCache[int](capacity)
//	var wg sync.WaitGroup
//	var errCh = make(chan error, numGoroutines*opsPerGoroutine) // 收集错误
//
//	wg.Add(numGoroutines)
//	for i := 0; i < numGoroutines; i++ {
//		go func(goroutineID int) {
//			defer wg.Done()
//			for j := 0; j < opsPerGoroutine; j++ {
//				// 使用唯一键（每个 goroutine 独立键空间），避免互相干扰
//				key := fmt.Sprintf("key-goroutine%d-%d", goroutineID, j)
//				val := goroutineID*1000 + j
//
//				// Put 后立即 Get，确保操作原子性（在锁内完成）
//				cache.Put(key, val) // 直接调用无锁的内部逻辑（或保持原 Put 但减少锁开销）
//				gotVal, ok := cache.Get(key)
//
//				if !ok || gotVal != val {
//					errCh <- fmt.Errorf(
//						"goroutine %d: Put(%q, %d) then Get() = (%d, %v), want (%d, true)",
//						goroutineID, key, val, gotVal, ok, val,
//					)
//				}
//			}
//		}(i)
//	}
//
//	// 等待所有 goroutine 完成
//	go func() {
//		wg.Wait()
//		close(errCh)
//	}()
//
//	// 检查错误
//	for err := range errCh {
//		t.Error(err)
//	}
//
//	// 验证最终缓存大小不超过容量
//	if cache.Len() > capacity {
//		t.Errorf("final size = %d, exceeds capacity %d", cache.Len(), capacity)
//	}
//}
//
//// 辅助类型：测试用的操作和预期结果
//type op[T any] struct {
//	typ string // "put" 或 "get"
//	key string
//	val T // 仅put操作需要
//}
//
//type want[T any] struct {
//	val T
//	ok  bool
//}
