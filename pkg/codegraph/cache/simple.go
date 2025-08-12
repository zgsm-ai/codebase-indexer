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

// LRUCache 带并发锁的LRU缓存
type LRUCache[T any] struct {
	cache    map[string]*node[T] // 快速查找映射
	head     *node[T]            // 头节点（最近使用）
	tail     *node[T]            // 尾节点（最少使用）
	capacity int                 // 最大容量
	size     int                 // 当前大小
	mu       sync.Mutex          // 互斥锁，保证并发安全
}

// NewLRUCache 创建新的LRU缓存
func NewLRUCache[T any](capacity int) *LRUCache[T] {
	head := &node[T]{}
	tail := &node[T]{}
	head.next = tail
	tail.prev = head

	return &LRUCache[T]{
		cache:    make(map[string]*node[T], capacity),
		head:     head,
		tail:     tail,
		capacity: capacity,
		size:     0,
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

	if node, ok := c.cache[key]; ok {
		node.value = value
		c.moveToHead(node)
		return
	}

	newNode := &node[T]{
		key:   key,
		value: value,
	}
	c.cache[key] = newNode
	c.addToHead(newNode)
	c.size++

	if c.size > c.capacity {
		removedNode := c.removeTail()
		delete(c.cache, removedNode.key)
		c.size--
	}
}

// Purge 清理所有
func (c *LRUCache[T]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 1. 清空map（所有节点不再被引用，等待GC回收）
	c.cache = make(map[string]*node[T], c.capacity)
	// 2. 重置双向链表（仅保留头和尾哨兵节点，中间节点全部隔离）
	c.head.next = c.tail
	c.tail.prev = c.head
	// 3. 重置当前大小为0
	c.size = 0
}

// Len 返回当前缓存大小（用于测试）
func (c *LRUCache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}
