package cache

import (
	"testing"
)

// TestNewLRUCache 测试 NewLRUCache 函数的基本功能
func TestNewLRUCache(t *testing.T) {
	testCases := []struct {
		name           string
		initCapacity   int
		maxCapacity    int
		expectPanic    bool
		expectedSize   int
		expectedMaxCap int
	}{
		{
			name:           "正常情况：initCapacity=0, maxCapacity=10",
			initCapacity:   0,
			maxCapacity:    10,
			expectPanic:    false,
			expectedSize:   0,
			expectedMaxCap: 10,
		},
		{
			name:           "正常情况：initCapacity=5, maxCapacity=10",
			initCapacity:   5,
			maxCapacity:    10,
			expectPanic:    false,
			expectedSize:   0,
			expectedMaxCap: 10,
		},
		{
			name:           "正常情况：initCapacity=10, maxCapacity=10",
			initCapacity:   10,
			maxCapacity:    10,
			expectPanic:    false,
			expectedSize:   0,
			expectedMaxCap: 10,
		},
		{
			name:           "边界情况：maxCapacity=1",
			initCapacity:   0,
			maxCapacity:    1,
			expectPanic:    false,
			expectedSize:   0,
			expectedMaxCap: 1,
		},
		{
			name:           "大容量测试：initCapacity=1000, maxCapacity=10000",
			initCapacity:   1000,
			maxCapacity:    10000,
			expectPanic:    false,
			expectedSize:   0,
			expectedMaxCap: 10000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 测试创建缓存
			cache := NewLRUCache[string](tc.initCapacity, tc.maxCapacity)

			// 验证缓存不为nil
			if cache == nil {
				t.Fatalf("NewLRUCache() 返回 nil，期望非 nil")
			}

			// 验证初始大小
			if cache.size != tc.expectedSize {
				t.Errorf("cache.size = %d, 期望 %d", cache.size, tc.expectedSize)
			}

			// 验证最大容量
			if cache.maxCapacity != tc.expectedMaxCap {
				t.Errorf("cache.maxCapacity = %d, 期望 %d", cache.maxCapacity, tc.expectedMaxCap)
			}

			// 验证缓存map已初始化
			if cache.cache == nil {
				t.Fatal("cache.cache 为 nil，期望已初始化")
			}

			// 验证map初始容量（如果指定了initCapacity）
			if tc.initCapacity > 0 {
				// 注意：Go的map容量是内部实现细节，我们只能验证map不为空
				// 实际容量可能大于或等于指定的initCapacity
				if len(cache.cache) != 0 {
					t.Errorf("新创建的缓存map长度应为0，实际为 %d", len(cache.cache))
				}
			}

			// 验证双向链表结构
			if cache.head == nil {
				t.Fatal("cache.head 为 nil，期望已初始化")
			}
			if cache.tail == nil {
				t.Fatal("cache.tail 为 nil，期望已初始化")
			}

			// 验证头尾节点连接正确
			if cache.head.next != cache.tail {
				t.Error("cache.head.next 应该指向 cache.tail")
			}
			if cache.tail.prev != cache.head {
				t.Error("cache.tail.prev 应该指向 cache.head")
			}

			// 验证头尾节点没有前驱/后继（除了互相指向）
			if cache.head.prev != nil {
				t.Error("cache.head.prev 应该为 nil")
			}
			if cache.tail.next != nil {
				t.Error("cache.tail.next 应该为 nil")
			}
		})
	}
}

// TestNewLRUCacheWithDifferentTypes 测试 NewLRUCache 函数支持不同类型
func TestNewLRUCacheWithDifferentTypes(t *testing.T) {
	// 测试int类型
	intCache := NewLRUCache[int](0, 5)
	if intCache == nil {
		t.Fatal("无法创建int类型的LRU缓存")
	}

	// 测试string类型
	stringCache := NewLRUCache[string](0, 5)
	if stringCache == nil {
		t.Fatal("无法创建string类型的LRU缓存")
	}

	// 测试struct类型
	type Person struct {
		Name string
		Age  int
	}
	structCache := NewLRUCache[Person](0, 5)
	if structCache == nil {
		t.Fatal("无法创建struct类型的LRU缓存")
	}

	// 测试指针类型
	pointerCache := NewLRUCache[*Person](0, 5)
	if pointerCache == nil {
		t.Fatal("无法创建指针类型的LRU缓存")
	}
}

// TestNewLRUCacheInitialCapacity 验证初始容量的影响
func TestNewLRUCacheInitialCapacity(t *testing.T) {
	// 创建具有不同初始容量的缓存
	cache1 := NewLRUCache[string](0, 10)   // 无预分配
	cache2 := NewLRUCache[string](100, 10) // 预分配100

	// 两者都应该正常工作
	if cache1 == nil || cache2 == nil {
		t.Fatal("缓存创建失败")
	}

	// 验证初始状态相同
	if cache1.size != cache2.size {
		t.Errorf("缓存初始大小不同：%d vs %d", cache1.size, cache2.size)
	}

	if cache1.maxCapacity != cache2.maxCapacity {
		t.Errorf("缓存最大容量不同：%d vs %d", cache1.maxCapacity, cache2.maxCapacity)
	}
}

// TestNewLRUCacheLinkedListStructure 详细测试双向链表结构
func TestNewLRUCacheLinkedListStructure(t *testing.T) {
	cache := NewLRUCache[string](0, 10)

	// 验证头节点
	head := cache.head
	if head == nil {
		t.Fatal("头节点为nil")
	}
	if head.key != "" {
		t.Errorf("头节点key应该为空字符串，实际为：%q", head.key)
	}
	// 注意：泛型的零值检查比较复杂，我们主要验证结构

	// 验证尾节点
	tail := cache.tail
	if tail == nil {
		t.Fatal("尾节点为nil")
	}
	if tail.key != "" {
		t.Errorf("尾节点key应该为空字符串，实际为：%q", tail.key)
	}

	// 验证连接关系
	if head.next != tail {
		t.Error("头节点的next应该指向尾节点")
	}
	if tail.prev != head {
		t.Error("尾节点的prev应该指向头节点")
	}

	// 验证头尾节点的外部连接为nil
	if head.prev != nil {
		t.Error("头节点的prev应该为nil")
	}
	if tail.next != nil {
		t.Error("尾节点的next应该为nil")
	}
}

// TestNewLRUCacheMutex 验证互斥锁已正确初始化
func TestNewLRUCacheMutex(t *testing.T) {
	cache := NewLRUCache[string](0, 10)

	// 验证互斥锁已初始化（通过尝试锁定和解锁）
	cache.mu.Lock()
	cache.mu.Unlock()

	// 如果没有panic，说明互斥锁工作正常
}

// TestNewLRUCacheLenAndMaxCapacity 验证Len()和MaxCapacity()方法
func TestNewLRUCacheLenAndMaxCapacity(t *testing.T) {
	initCapacity := 5
	maxCapacity := 10

	cache := NewLRUCache[string](initCapacity, maxCapacity)

	// 验证初始长度
	if got := cache.Len(); got != 0 {
		t.Errorf("Len() = %d, 期望 0", got)
	}

	// 验证最大容量
	if got := cache.MaxCapacity(); got != maxCapacity {
		t.Errorf("MaxCapacity() = %d, 期望 %d", got, maxCapacity)
	}
}
