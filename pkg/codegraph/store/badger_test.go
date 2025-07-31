package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"codebase-indexer/pkg/logger"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// BadgerTestMessage is a simple proto message for testing BadgerStorage
type BadgerTestMessage struct {
	Data []byte
}

func (t *BadgerTestMessage) Reset()         { *t = BadgerTestMessage{} }
func (t *BadgerTestMessage) String() string { return string(t.Data) }
func (t *BadgerTestMessage) ProtoMessage()  {}

// BadgerTestValues implements Values interface for testing
type BadgerTestValues struct {
	keys   []string
	values []proto.Message
}

func (tv *BadgerTestValues) Len() int {
	return len(tv.keys)
}

func (tv *BadgerTestValues) Value(i int) proto.Message {
	return tv.values[i]
}

func (tv *BadgerTestValues) Key(i int) string {
	return tv.keys[i]
}

func TestNewBadgerStorage(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	assert.NotNil(t, storage)
	assert.NoError(t, storage.Close())
}

func TestBadgerStorage_SaveAndGet(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectID := "test-project"

	// 测试保存和获取
	msg := &RawMessage{Data: []byte("test data")}
	err = storage.Save(ctx, projectID, msg)
	assert.NoError(t, err)

	// 测试获取
	key := ElementPathKey("test")
	retrieved, err := storage.Get(ctx, projectID, key)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestBadgerStorage_BatchSave(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectID := "test-project"

	// 准备测试数据
	values := &BadgerTestValues{
		keys: []string{"key1", "key2", "key3"},
		values: []proto.Message{
			&RawMessage{Data: []byte("value1")},
			&RawMessage{Data: []byte("value2")},
			&RawMessage{Data: []byte("value3")},
		},
	}

	err = storage.BatchSave(ctx, projectID, values)
	assert.NoError(t, err)

	// 验证数据
	for _, key := range values.keys {
		retrieved, err := storage.Get(ctx, projectID, ElementPathKey(key))
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	}
}

func TestBadgerStorage_Delete(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectID := "test-project"

	// 先保存数据
	msg := &RawMessage{Data: []byte("test data")}
	err = storage.Save(ctx, projectID, msg)
	assert.NoError(t, err)

	// 删除数据
	key := ElementPathKey("test")
	err = storage.Delete(ctx, projectID, key)
	assert.NoError(t, err)

	// 验证数据已被删除
	_, err = storage.Get(ctx, projectID, key)
	assert.Error(t, err)
}

func TestBadgerStorage_Size(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectID := "test-project"

	// 初始大小应为0
	size := storage.Size(ctx, projectID)
	assert.Equal(t, 0, size)

	// 保存一些数据
	values := &BadgerTestValues{
		keys: []string{"key1", "key2"},
		values: []proto.Message{
			&RawMessage{Data: []byte("value1")},
			&RawMessage{Data: []byte("value2")},
		},
	}

	err = storage.BatchSave(ctx, projectID, values)
	assert.NoError(t, err)

	// 验证大小
	size = storage.Size(ctx, projectID)
	assert.Equal(t, 2, size)
}

func TestBadgerStorage_Iter(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectID := "test-project"

	// 保存测试数据
	values := &BadgerTestValues{
		keys: []string{"key1", "key2", "key3"},
		values: []proto.Message{
			&RawMessage{Data: []byte("value1")},
			&RawMessage{Data: []byte("value2")},
			&RawMessage{Data: []byte("value3")},
		},
	}

	err = storage.BatchSave(ctx, projectID, values)
	assert.NoError(t, err)

	// 使用迭代器遍历
	iterator := storage.Iter(ctx, projectID)
	defer iterator.Close()

	count := 0
	for iterator.Next() {
		count++
		key := iterator.Key()
		value := iterator.Value()
		assert.NotEmpty(t, key)
		assert.NotNil(t, value)
	}

	assert.Equal(t, 3, count)
	assert.NoError(t, iterator.Error())
}

func TestBadgerStorage_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	defer storage.Close()

	// 创建取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	projectID := "test-project"

	// 测试取消的上下文
	err = storage.Save(ctx, projectID, &RawMessage{Data: []byte("test")})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")

	// 测试取消的上下文
	_, err = storage.Get(ctx, projectID, ElementPathKey("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestBadgerStorage_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := logger.NewLogger(tempDir, "debug")
	assert.NoError(t, err)

	storage, err := NewBadgerStorage(tempDir, logger)
	assert.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectID := "test-project"

	// 并发保存数据
	const goroutines = 10
	const itemsPerGoroutine = 100

	done := make(chan bool)
	for i := 0; i < goroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < itemsPerGoroutine; j++ {
				value := &RawMessage{Data: []byte(fmt.Sprintf("value-%d-%d", goroutineID, j))}

				err := storage.Save(ctx, projectID, value)
				if err != nil {
					t.Errorf("Failed to save: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < goroutines; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// 验证数据
	size := storage.Size(ctx, projectID)
	assert.Equal(t, goroutines*itemsPerGoroutine, size)
}
