package cache

import (
	"sync"
)

// 双向链表节点
type node[T any] struct {
	key   string
	value T
	prev  *node[T]
	next  *node[T]
}

// LRUCache 带并发锁的LRU缓存（仅限制最大容量，依赖map自动扩容）
type LRUCache[T any] struct {
	cache       map[string]*node[T] // 快速查找映射，依赖Go自动扩容
	head        *node[T]            // 头节点（最近使用）
	tail        *node[T]            // 尾节点（最少使用）
	maxCapacity int                 // 最大元素数量（超过则淘汰）
	size        int                 // 当前元素数量
	mu          sync.Mutex          // 互斥锁，保证并发安全
}

// NewLRUCache 创建新的LRU缓存
// initCapacity: map初始容量（用于预分配，可填0，Go会自动扩容）
// maxCapacity: 最大元素数量（必须>0）
func NewLRUCache[T any](initCapacity, maxCapacity int) *LRUCache[T] {
	head := &node[T]{}
	tail := &node[T]{}
	head.next = tail
	tail.prev = head

	return &LRUCache[T]{
		cache:       make(map[string]*node[T], initCapacity), // 用initCapacity预分配map
		head:        head,
		tail:        tail,
		maxCapacity: maxCapacity,
		size:        0,
	}
}

// 移动节点到头部（标记为最近使用）
func (c *LRUCache[T]) moveToHead(n *node[T]) {
	c.removeNode(n)
	c.addToHead(n)
}

// 添加节点到头部
func (c *LRUCache[T]) addToHead(n *node[T]) {
	n.prev = c.head
	n.next = c.head.next
	c.head.next.prev = n
	c.head.next = n
}

// 移除指定节点
func (c *LRUCache[T]) removeNode(n *node[T]) {
	n.prev.next = n.next
	n.next.prev = n.prev
}

// 移除尾节点（淘汰最少使用）
func (c *LRUCache[T]) removeTail() *node[T] {
	n := c.tail.prev
	c.removeNode(n)
	return n
}

// Get 并发安全的获取操作
func (c *LRUCache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.cache[key]; ok {
		c.moveToHead(node)
		return node.value, true
	}

	var zero T
	return zero, false
}

// Put 并发安全的添加/更新操作
func (c *LRUCache[T]) Put(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 若已存在，更新值并移到头部
	if node, ok := c.cache[key]; ok {
		node.value = value
		c.moveToHead(node)
		return
	}

	// 新增节点
	newNode := &node[T]{
		key:   key,
		value: value,
	}
	c.cache[key] = newNode
	c.addToHead(newNode)
	c.size++

	// 超过最大容量则淘汰最少使用节点（依赖map自动扩容，无需手动管理map容量）
	if c.size > c.maxCapacity {
		removedNode := c.removeTail()
		delete(c.cache, removedNode.key)
		c.size--
	}
}

// Purge 清理所有缓存
func (c *LRUCache[T]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 用初始容量重新创建map（或保留原容量，根据需求选择）
	c.cache = make(map[string]*node[T], len(c.cache))
	c.head.next = c.tail
	c.tail.prev = c.head
	c.size = 0
}

// Len 返回当前缓存大小
func (c *LRUCache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}

// MaxCapacity 返回最大容量限制
func (c *LRUCache[T]) MaxCapacity() int {
	return c.maxCapacity // 只读字段，无需加锁
}
