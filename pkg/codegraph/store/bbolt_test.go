package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestMessage 用于测试的 protobuf 消息
type TestMessage struct {
	Value string
}

func (t *TestMessage) Reset()         { *t = TestMessage{} }
func (t *TestMessage) String() string { return t.Value }
func (t *TestMessage) ProtoMessage()  {}
func (t *TestMessage) ProtoReflect() protoreflect.Message {
	return nil
}
func (t *TestMessage) Marshal() ([]byte, error) {
	return []byte(t.Value), nil
}
func (t *TestMessage) Unmarshal(data []byte) error {
	t.Value = string(data)
	return nil
}

// TestKey 用于测试的键类型
type TestKey struct {
	key string
}

func (k TestKey) Get() string {
	return k.key
}

// setupTestStorage 创建测试用的存储实例
func setupTestStorage(t *testing.T) (*BBoltStorage, func()) {
	tempDir, err := os.MkdirTemp("", "bbolt-test-*")
	require.NoError(t, err)

	logger := &mockLogger{}
	storage, err := NewBBoltStorage(tempDir, logger)
	require.NoError(t, err)

	cleanup := func() {
		storage.Close()
		os.RemoveAll(tempDir)
	}

	return storage, cleanup
}

// mockLogger 用于测试的 mock logger
type mockLogger struct{}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Fatal(msg string, keysAndValues ...interface{}) {}

func TestNewBBoltStorage(t *testing.T) {
	tests := []struct {
		name        string
		baseDir     string
		setupFunc   func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name:    "成功创建存储",
			baseDir: "test-storage",
		},
		{
			name: "目录权限不足",
			setupFunc: func(t *testing.T) string {
				// 创建一个已存在的文件作为目录路径，导致MkdirAll失败
				tempDir, err := os.MkdirTemp("", "readonly-test-*")
				require.NoError(t, err)

				// 创建一个同名文件，阻止目录创建
				blockingFile := filepath.Join(tempDir, "blocking-file")
				err = os.WriteFile(blockingFile, []byte("blocking"), 0644)
				require.NoError(t, err)

				return blockingFile // 返回文件路径而不是目录路径
			},
			wantErr:     true,
			errContains: "failed to create base directory",
		},
		{
			name:    "空目录路径",
			baseDir: "test-empty-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := tt.baseDir
			if tt.setupFunc != nil {
				baseDir = tt.setupFunc(t)
				defer os.RemoveAll(baseDir)
			} else if baseDir != "" {
				baseDir = filepath.Join(t.TempDir(), baseDir)
			}

			storage, err := NewBBoltStorage(baseDir, &mockLogger{})

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, storage)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, storage)
				if storage != nil {
					storage.Close()
				}
			}
		})
	}
}

func TestBBoltStorage_Save(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"

	tests := []struct {
		name      string
		value     proto.Message
		wantErr   bool
		setupFunc func(t *testing.T)
	}{
		{
			name:  "保存成功",
			value: &TestMessage{Value: "test-value"},
		},
		{
			name:  "保存空消息",
			value: &TestMessage{Value: ""},
		},
		{
			name:    "上下文已取消",
			value:   &TestMessage{Value: "test"},
			wantErr: true,
			setupFunc: func(t *testing.T) {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := ctx
			if tt.setupFunc != nil {
				tt.setupFunc(t)
				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
				<-testCtx.Done() // 确保上下文已取消
			}

			err := storage.Save(testCtx, projectID, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBBoltStorage_BatchSave(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"

	tests := []struct {
		name      string
		values    []proto.Message
		keys      []string
		wantErr   bool
		setupFunc func(t *testing.T)
	}{
		{
			name: "批量保存成功",
			values: []proto.Message{
				&TestMessage{Value: "value1"},
				&TestMessage{Value: "value2"},
			},
			keys: []string{"key1", "key2"},
		},
		{
			name:    "空值列表",
			values:  []proto.Message{},
			keys:    []string{},
			wantErr: false,
		},
		{
			name: "上下文取消",
			values: []proto.Message{
				&TestMessage{Value: "value1"},
			},
			keys:    []string{"key1"},
			wantErr: true,
			setupFunc: func(t *testing.T) {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := ctx
			if tt.setupFunc != nil {
				tt.setupFunc(t)
				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
				<-testCtx.Done()
			}

			values := &testValues{
				values: tt.values,
				keys:   tt.keys,
			}

			err := storage.BatchSave(testCtx, projectID, values)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBBoltStorage_Get(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"

	// 预填充数据
	testData := &TestMessage{Value: "test-value"}
	err := storage.Save(ctx, projectID, testData)
	require.NoError(t, err)

	tests := []struct {
		name      string
		key       Key
		wantValue proto.Message
		wantErr   bool
		errMsg    string
		setupFunc func(t *testing.T)
	}{
		{
			name:      "获取存在的键",
			key:       TestKey{key: "*store.TestMessage"},
			wantValue: &RawMessage{Data: []byte("test-value")},
		},
		{
			name:    "获取不存在的键",
			key:     TestKey{key: "non-existent"},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name:    "上下文已取消",
			key:     TestKey{key: "*store.TestMessage"},
			wantErr: true,
			setupFunc: func(t *testing.T) {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := ctx
			if tt.setupFunc != nil {
				tt.setupFunc(t)
				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
				<-testCtx.Done()
			}

			value, err := storage.Get(testCtx, projectID, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, value)
				if tt.wantValue != nil {
					assert.Equal(t, tt.wantValue.(*RawMessage).Data, value.(*RawMessage).Data)
				}
			}
		})
	}
}

func TestBBoltStorage_Delete(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"

	// 预填充数据
	testData := &TestMessage{Value: "test-value"}
	err := storage.Save(ctx, projectID, testData)
	require.NoError(t, err)

	tests := []struct {
		name      string
		key       Key
		wantErr   bool
		verify    func(t *testing.T)
		setupFunc func(t *testing.T)
	}{
		{
			name: "删除存在的键",
			key:  TestKey{key: "*store.TestMessage"},
			verify: func(t *testing.T) {
				_, err := storage.Get(ctx, projectID, TestKey{key: "*store.TestMessage"})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name:    "删除不存在的键",
			key:     TestKey{key: "non-existent"},
			wantErr: false,
		},
		{
			name:    "上下文已取消",
			key:     TestKey{key: "*store.TestMessage"},
			wantErr: true,
			setupFunc: func(t *testing.T) {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := ctx
			if tt.setupFunc != nil {
				tt.setupFunc(t)
				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
				<-testCtx.Done()
			}

			err := storage.Delete(testCtx, projectID, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

func TestBBoltStorage_Size(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"

	tests := []struct {
		name      string
		setupData []proto.Message
		wantSize  int
	}{
		{
			name:      "空项目",
			setupData: []proto.Message{},
			wantSize:  0,
		},
		{
			name: "单条数据",
			setupData: []proto.Message{
				&TestMessage{Value: "value1"},
			},
			wantSize: 1,
		},
		{
			name: "多条数据",
			setupData: []proto.Message{
				&TestMessage{Value: "value1"},
				&TestMessage{Value: "value2"},
				&TestMessage{Value: "value3"},
			},
			wantSize: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用BatchSave来保存多条数据，确保每条数据都有唯一的键
			testProjectID := fmt.Sprintf("%s-%s", projectID, tt.name)

			if len(tt.setupData) > 0 {
				// 创建测试用的Values实现
				values := &testValues{
					values: tt.setupData,
					keys:   make([]string, len(tt.setupData)),
				}
				for i := range tt.setupData {
					values.keys[i] = fmt.Sprintf("key-%d", i+1)
				}
				storage.BatchSave(ctx, testProjectID, values)
			}

			size := storage.Size(ctx, testProjectID)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

func TestBBoltStorage_Close(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"

	// 预填充数据以确保有数据库连接
	err := storage.Save(ctx, projectID, &TestMessage{Value: "test"})
	require.NoError(t, err)

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "正常关闭",
		},
		{
			name: "重复关闭",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Close()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 验证关闭后操作失败
			if tt.name == "正常关闭" {
				err := storage.Save(context.Background(), projectID, &TestMessage{Value: "test"})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "storage is closed")
			}
		})
	}
}

func TestBBoltStorage_Iter(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"

	// 预填充测试数据
	testData := []struct {
		key   string
		value string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	for _, data := range testData {
		storage.Save(ctx, projectID, &TestMessage{Value: data.value})
	}

	tests := []struct {
		name      string
		verify    func(t *testing.T, iter Iterator)
		setupFunc func(t *testing.T)
	}{
		{
			name: "正常迭代",
			verify: func(t *testing.T, iter Iterator) {
				count := 0
				for iter.Next() {
					count++
					assert.NotEmpty(t, iter.Key())
					assert.NotNil(t, iter.Value())
				}
				assert.Greater(t, count, 0)
				assert.NoError(t, iter.Error())
			},
		},
		{
			name: "上下文取消",
			verify: func(t *testing.T, iter Iterator) {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel()

				iter = storage.Iter(cancelCtx, projectID)
				assert.False(t, iter.Next())
				assert.Error(t, iter.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := storage.Iter(ctx, projectID)
			defer iter.Close()

			if tt.verify != nil {
				tt.verify(t, iter)
			}
		})
	}
}

// testValues 用于测试的 Values 实现
type testValues struct {
	values []proto.Message
	keys   []string
}

func (tv *testValues) Len() int {
	return len(tv.values)
}

func (tv *testValues) Key(i int) string {
	if i < len(tv.keys) {
		return tv.keys[i]
	}
	return fmt.Sprintf("key-%d", i)
}

func (tv *testValues) Value(i int) proto.Message {
	if i < len(tv.values) {
		return tv.values[i]
	}
	return &TestMessage{Value: "default"}
}
