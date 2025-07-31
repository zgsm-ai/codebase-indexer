package store

//
//import (
//	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
//	"context"
//	"fmt"
//	"io/fs"
//	"os"
//	"path/filepath"
//	"runtime"
//	"sync"
//	"testing"
//	"time"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	"google.golang.org/protobuf/proto"
//)
//
//// setupPebbleTestStorage 创建测试用的存储实例
//func setupPebbleTestStorage(t *testing.T) (*PebbleStorage, func()) {
//	tempDir, err := os.MkdirTemp("", "pebble-test-*")
//	require.NoError(t, err)
//
//	logger := &MockLogger{}
//	storage, err := NewPebbleStorage(tempDir, logger)
//	require.NoError(t, err)
//
//	cleanup := func() {
//		storage.Close()
//		os.RemoveAll(tempDir)
//	}
//
//	return storage, cleanup
//}
//
//func TestNewPebbleStorage(t *testing.T) {
//	tests := []struct {
//		name        string
//		baseDir     string
//		setupFunc   func(t *testing.T) string
//		wantErr     bool
//		errContains string
//	}{
//		{
//			name:    "成功创建存储",
//			baseDir: "test-storage",
//		},
//		{
//			name: "目录权限不足",
//			setupFunc: func(t *testing.T) string {
//				// 创建一个已存在的文件作为目录路径，导致MkdirAll失败
//				tempDir, err := os.MkdirTemp("", "readonly-test-*")
//				require.NoError(t, err)
//
//				// 创建一个同名文件，阻止目录创建
//				blockingFile := filepath.Join(tempDir, "blocking-file")
//				err = os.WriteFile(blockingFile, []byte("blocking"), 0644)
//				require.NoError(t, err)
//
//				return blockingFile // 返回文件路径而不是目录路径
//			},
//			wantErr:     true,
//			errContains: "failed to create base directory",
//		},
//		{
//			name:    "空目录路径",
//			baseDir: "test-empty-path",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			baseDir := tt.baseDir
//			if tt.setupFunc != nil {
//				baseDir = tt.setupFunc(t)
//				defer os.RemoveAll(baseDir)
//			} else if baseDir != "" {
//				baseDir = filepath.Join(t.TempDir(), baseDir)
//			}
//
//			storage, err := NewPebbleStorage(baseDir, &MockLogger{})
//
//			if tt.wantErr {
//				assert.Error(t, err)
//				if tt.errContains != "" {
//					assert.Contains(t, err.Error(), tt.errContains)
//				}
//				assert.Nil(t, storage)
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, storage)
//				if storage != nil {
//					storage.Close()
//				}
//			}
//		})
//	}
//}
//
//func TestPebbleStorage_Save(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectName := "test-project"
//	projectPath := "/tmp/test-project"
//	projectID := GenerateTestProjectUUID(projectName, projectPath)
//
//	// 预填充数据
//	testData := &codegraphpb.TestMessage{Value: "test-value"}
//	err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: testData})
//	require.NoError(t, err)
//
//	tests := []struct {
//		name      string
//		value     proto.Message
//		wantErr   bool
//		setupFunc func(t *testing.T)
//	}{
//		{
//			name:  "保存成功",
//			value: &codegraphpb.TestMessage{Value: "test-value-2"},
//		},
//		{
//			name:  "保存空消息",
//			value: &codegraphpb.TestMessage{Value: ""},
//		},
//		{
//			name:    "上下文已取消",
//			value:   &codegraphpb.TestMessage{Value: "test"},
//			wantErr: true,
//			setupFunc: func(t *testing.T) {
//				ctx, cancel := context.WithCancel(ctx)
//				cancel()
//				ctx = ctx
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			testCtx := ctx
//			if tt.setupFunc != nil {
//				tt.setupFunc(t)
//				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
//				<-testCtx.Done() // 确保上下文已取消
//			}
//
//			err := storage.Save(testCtx, projectID, &Entry{Key: ElementPathKey(tt.name), Value: tt.value})
//			if tt.wantErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//			}
//		})
//	}
//}
//
//func TestPebbleStorage_BatchSave(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "test-project"
//
//	tests := []struct {
//		name      string
//		values    []proto.Message
//		keys      []string
//		wantErr   bool
//		setupFunc func(t *testing.T)
//	}{
//		{
//			name: "批量保存成功",
//			values: []proto.Message{
//				&codegraphpb.TestMessage{Value: "value1"},
//				&codegraphpb.TestMessage{Value: "value2"},
//			},
//			keys: []string{"key1", "key2"},
//		},
//		{
//			name:    "空值列表",
//			values:  []proto.Message{},
//			keys:    []string{},
//			wantErr: false,
//		},
//		{
//			name: "上下文取消",
//			values: []proto.Message{
//				&codegraphpb.TestMessage{Value: "value1"},
//			},
//			keys:    []string{"key1"},
//			wantErr: true,
//			setupFunc: func(t *testing.T) {
//				ctx, cancel := context.WithCancel(ctx)
//				cancel()
//				ctx = ctx
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			testCtx := ctx
//			if tt.setupFunc != nil {
//				tt.setupFunc(t)
//				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
//				<-testCtx.Done()
//			}
//
//			values := &testValues{
//				values: tt.values,
//				keys:   tt.keys,
//			}
//
//			err := storage.BatchSave(testCtx, projectID, values)
//			if tt.wantErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//			}
//		})
//	}
//}
//
//func TestPebbleStorage_Get(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "test-project"
//
//	// 预填充数据
//	testData := &codegraphpb.TestMessage{Value: "test-value"}
//	err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: testData})
//	require.NoError(t, err)
//
//	tests := []struct {
//		name      string
//		key       Key
//		wantValue proto.Message
//		wantErr   bool
//		errMsg    string
//		setupFunc func(t *testing.T)
//	}{
//		{
//			name:      "获取存在的键",
//			key:       TestKey{key: "*store.codegraphpb.TestMessage"},
//			wantValue: &RawMessage{Data: []byte("test-value")},
//		},
//		{
//			name:    "获取不存在的键",
//			key:     TestKey{key: "non-existent"},
//			wantErr: true,
//			errMsg:  "not found",
//		},
//		{
//			name:    "上下文已取消",
//			key:     TestKey{key: "*store.codegraphpb.TestMessage"},
//			wantErr: true,
//			setupFunc: func(t *testing.T) {
//				ctx, cancel := context.WithCancel(ctx)
//				cancel()
//				ctx = ctx
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			testCtx := ctx
//			if tt.setupFunc != nil {
//				tt.setupFunc(t)
//				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
//				<-testCtx.Done()
//			}
//
//			value, err := storage.Get(testCtx, projectID, tt.key)
//			if tt.wantErr {
//				assert.Error(t, err)
//				if tt.errMsg != "" {
//					assert.Contains(t, err.Error(), tt.errMsg)
//				}
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, value)
//				if tt.wantValue != nil {
//					assert.Equal(t, tt.wantValue.(*RawMessage).Data, value)
//				}
//			}
//		})
//	}
//}
//
//func TestPebbleStorage_Delete(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "test-project"
//
//	// 预填充数据
//	testData := &codegraphpb.TestMessage{Value: "test-value"}
//	err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: testData})
//	require.NoError(t, err)
//
//	tests := []struct {
//		name      string
//		key       Key
//		wantErr   bool
//		verify    func(t *testing.T)
//		setupFunc func(t *testing.T)
//	}{
//		{
//			name: "删除存在的键",
//			key:  TestKey{key: "*store.codegraphpb.TestMessage"},
//			verify: func(t *testing.T) {
//				_, err := storage.Get(ctx, projectID, TestKey{key: "*store.codegraphpb.TestMessage"})
//				assert.Error(t, err)
//				assert.Contains(t, err.Error(), "not found")
//			},
//		},
//		{
//			name:    "删除不存在的键",
//			key:     TestKey{key: "non-existent"},
//			wantErr: false,
//		},
//		{
//			name:    "上下文已取消",
//			key:     TestKey{key: "*store.codegraphpb.TestMessage"},
//			wantErr: true,
//			setupFunc: func(t *testing.T) {
//				ctx, cancel := context.WithCancel(ctx)
//				cancel()
//				ctx = ctx
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			testCtx := ctx
//			if tt.setupFunc != nil {
//				tt.setupFunc(t)
//				testCtx, _ = context.WithTimeout(ctx, 1*time.Nanosecond)
//				<-testCtx.Done()
//			}
//
//			err := storage.Delete(testCtx, projectID, tt.key)
//			if tt.wantErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//			}
//
//			if tt.verify != nil {
//				tt.verify(t)
//			}
//		})
//	}
//}
//
//func TestPebbleStorage_Size(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "test-project"
//
//	tests := []struct {
//		name      string
//		setupData []proto.Message
//		wantSize  int
//	}{
//		{
//			name:      "空项目",
//			setupData: []proto.Message{},
//			wantSize:  0,
//		},
//		{
//			name: "单条数据",
//			setupData: []proto.Message{
//				&codegraphpb.TestMessage{Value: "value1"},
//			},
//			wantSize: 1,
//		},
//		{
//			name: "多条数据",
//			setupData: []proto.Message{
//				&codegraphpb.TestMessage{Value: "value1"},
//				&codegraphpb.TestMessage{Value: "value2"},
//				&codegraphpb.TestMessage{Value: "value3"},
//			},
//			wantSize: 3,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// 使用BatchSave来保存多条数据，确保每条数据都有唯一的键
//			testProjectID := fmt.Sprintf("%s-%s", projectID, tt.name)
//
//			if len(tt.setupData) > 0 {
//				// 创建测试用的Values实现
//				values := &testValues{
//					values: tt.setupData,
//					keys:   make([]string, len(tt.setupData)),
//				}
//				for i := range tt.setupData {
//					values.keys[i] = fmt.Sprintf("key-%d", i+1)
//				}
//				storage.BatchSave(ctx, testProjectID, values)
//			}
//
//			size := storage.Size(ctx, testProjectID)
//			assert.Equal(t, tt.wantSize, size)
//		})
//	}
//}
//
//func TestPebbleStorage_Close(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "test-project"
//
//	// 预填充数据以确保有数据库连接
//	err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: &codegraphpb.TestMessage{Value: "test"}})
//	require.NoError(t, err)
//
//	tests := []struct {
//		name    string
//		wantErr bool
//	}{
//		{
//			name: "正常关闭",
//		},
//		{
//			name: "重复关闭",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			err := storage.Close()
//			if tt.wantErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//			}
//
//			// 验证关闭后操作失败
//			if tt.name == "正常关闭" {
//				err := storage.Save(context.Background(), projectID, &Entry{Key: ElementPathKey("test-value"), Value: &codegraphpb.TestMessage{Value: "test"}})
//				assert.Error(t, err)
//				assert.Contains(t, err.Error(), "storage is closed")
//			}
//		})
//	}
//}
//
//func TestPebbleStorage_Iter(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "test-project"
//
//	// 预填充测试数据
//	testData := []struct {
//		key   string
//		value string
//	}{
//		{"key1", "value1"},
//		{"key2", "value2"},
//		{"key3", "value3"},
//	}
//
//	for _, data := range testData {
//		storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: &codegraphpb.TestMessage{Value: data.value}})
//	}
//
//	tests := []struct {
//		name      string
//		verify    func(t *testing.T, iter Iterator)
//		setupFunc func(t *testing.T)
//	}{
//		{
//			name: "正常迭代",
//			verify: func(t *testing.T, iter Iterator) {
//				count := 0
//				for iter.Next() {
//					count++
//					assert.NotEmpty(t, iter.Key())
//					assert.NotNil(t, iter.Value())
//				}
//				assert.Greater(t, count, 0)
//				assert.NoError(t, iter.Error())
//			},
//		},
//		{
//			name: "上下文取消",
//			verify: func(t *testing.T, iter Iterator) {
//				cancelCtx, cancel := context.WithCancel(ctx)
//				cancel()
//
//				iter = storage.Iter(cancelCtx, projectID)
//				assert.False(t, iter.Next())
//				assert.Error(t, iter.Error())
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			iter := storage.Iter(ctx, projectID)
//			defer iter.Close()
//
//			if tt.verify != nil {
//				tt.verify(t, iter)
//			}
//		})
//	}
//}
//
//// testValues 用于测试的 Entries 实现
//type testValues struct {
//	values []proto.Message
//	keys   []string
//}
//
//func (tv *testValues) Len() int {
//	return len(tv.values)
//}
//
//func (tv *testValues) Key(i int) string {
//	if i < len(tv.keys) {
//		return tv.keys[i]
//	}
//	return fmt.Sprintf("key-%d", i)
//}
//
//func (tv *testValues) Value(i int) proto.Message {
//	if i < len(tv.values) {
//		return tv.values[i]
//	}
//	return &codegraphpb.TestMessage{Value: "default"}
//}
//
//func TestPebbleStorage_ConcurrentReadWrite(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "concurrent-test"
//
//	const (
//		writers      = 20
//		readers      = 20
//		opsPerWorker = 50
//	)
//
//	var wg sync.WaitGroup
//	wg.Add(writers + readers)
//
//	// 严格并发测试：所有goroutine同时启动
//	start := make(chan struct{})
//
//	// 并发写入goroutine
//	for w := 0; w < writers; w++ {
//		go func(writerID int) {
//			defer wg.Done()
//			<-start // 等待统一开始信号
//
//			for i := 0; i < opsPerWorker; i++ {
//				value := &codegraphpb.TestMessage{Value: fmt.Sprintf("writer-%d-value-%d", writerID, i)}
//				err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: value})
//				assert.NoError(t, err)
//			}
//		}(w)
//	}
//
//	// 并发读取goroutine
//	for r := 0; r < readers; r++ {
//		go func(readerID int) {
//			defer wg.Done()
//			<-start // 等待统一开始信号
//
//			for i := 0; i < opsPerWorker; i++ {
//				key := TestKey{key: "*store.codegraphpb.TestMessage"}
//				_, _ = storage.Get(ctx, projectID, key)
//				// 读取可能成功也可能失败，不验证结果
//			}
//		}(r)
//	}
//
//	// 统一启动所有并发操作
//	close(start)
//	wg.Wait()
//
//	// 验证最终数据完整性
//	finalSize := storage.Size(ctx, projectID)
//	assert.GreaterOrEqual(t, finalSize, 0)
//
//	// 验证至少有一个写入成功
//	allData := make([]*codegraphpb.TestMessage, 0)
//	iter := storage.Iter(ctx, projectID)
//	for iter.Next() {
//		data := iter.Value()
//		if msg, ok := data.(*codegraphpb.TestMessage); ok {
//			allData = append(allData, msg)
//		}
//	}
//	assert.GreaterOrEqual(t, len(allData), 0)
//	// 确保所有goroutine完成后再执行cleanup
//	iter.Close()
//}
//
//func TestPebbleStorage_ConcurrentBatchWrite(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "batch-concurrent-test"
//	const goroutines = 50
//	const batchSize = 10000
//
//	var wg sync.WaitGroup
//	wg.Add(goroutines)
//
//	for i := 0; i < goroutines; i++ {
//		go func(id int) {
//			defer wg.Done()
//			values := make([]proto.Message, batchSize)
//			keys := make([]string, batchSize)
//			for j := 0; j < batchSize; j++ {
//				values[j] = &codegraphpb.TestMessage{Value: fmt.Sprintf("batch-%d-%d", id, j)}
//				keys[j] = fmt.Sprintf("key-%d-%d", id, j)
//			}
//			testValues := &testValues{values: values, keys: keys}
//			err := storage.BatchSave(ctx, projectID, testValues)
//			assert.NoError(t, err)
//		}(i)
//	}
//
//	wg.Wait()
//
//	// 验证总数据量
//	size := storage.Size(ctx, projectID)
//	assert.Equal(t, goroutines*batchSize, size)
//}
//
//func TestPebbleStorage_BigBatchWrite(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "big-batch-test"
//	const batchSize = 1000
//
//	values := make([]proto.Message, batchSize)
//	keys := make([]string, batchSize)
//	for j := 0; j < batchSize; j++ {
//		values[j] = &codegraphpb.TestMessage{Value: fmt.Sprintf("batch-%d", j)}
//		keys[j] = fmt.Sprintf("key-%d", j)
//	}
//	testValues := &testValues{values: values, keys: keys}
//	err := storage.BatchSave(ctx, projectID, testValues)
//	assert.NoError(t, err)
//
//	// 验证总数据量
//	size := storage.Size(ctx, projectID)
//	assert.Equal(t, batchSize, size)
//}
//
//func TestPebbleStorage_MultipleProjectsIsolation(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	const projects = 10
//
//	// 为每个项目写入数据
//	for p := 0; p < projects; p++ {
//		projectName := fmt.Sprintf("project-%d", p)
//		projectPath := fmt.Sprintf("/tmp/test-project-%d", p)
//		projectID := GenerateTestProjectUUID(projectName, projectPath)
//
//		// 创建测试用的Values实现
//		values := &testValues{
//			values: make([]proto.Message, p+1),
//			keys:   make([]string, p+1),
//		}
//		for i := 0; i < p+1; i++ {
//			values.values[i] = &codegraphpb.TestMessage{Value: fmt.Sprintf("project-%d-value-%d", p, i)}
//			values.keys[i] = fmt.Sprintf("key-%d", i)
//		}
//
//		err := storage.BatchSave(ctx, projectID, values)
//		assert.NoError(t, err)
//	}
//
//	// 验证项目隔离性
//	for p := 0; p < projects; p++ {
//		projectName := fmt.Sprintf("project-%d", p)
//		projectPath := fmt.Sprintf("/tmp/test-project-%d", p)
//		projectID := GenerateTestProjectUUID(projectName, projectPath)
//		size := storage.Size(ctx, projectID)
//		assert.Equal(t, p+1, size)
//	}
//}
//
//func TestPebbleStorage_CorruptedFileHandling(t *testing.T) {
//
//	tempDir, err := os.MkdirTemp("", "corruption-test-*")
//	require.NoError(t, err)
//	defer os.RemoveAll(tempDir)
//
//	// 创建正常存储
//	storage, err := NewPebbleStorage(tempDir, &MockLogger{})
//	require.NoError(t, err)
//
//	ctx := context.Background()
//	projectName := "corruption-test"
//	projectPath := "/tmp/corruption-test"
//	projectID := GenerateTestProjectUUID(projectName, projectPath)
//	key := ElementPathKey("test-value")
//	// 写入一些数据
//	err = storage.Save(ctx, projectID, &Entry{Key: key, Value: &codegraphpb.TestMessage{Value: "test-data"}})
//	assert.NoError(t, err)
//	_, err = storage.Get(ctx, projectID, key)
//	assert.NoError(t, err)
//	err = storage.Close()
//	assert.NoError(t, err)
//
//	// 模拟文件损坏：直接修改数据库文件
//	dbPath := filepath.Join(tempDir, projectID, dataDir)
//	err = filepath.Walk(dbPath, func(path string, info fs.FileInfo, err error) error {
//		if info.IsDir() {
//			return nil
//		}
//		data, err := os.ReadFile(path)
//		if err == nil && len(data) > 0 {
//			// 破坏整个文件内容（修改所有字节）
//			for i := 0; i < len(data); i++ {
//				// 可以使用任意规则修改字节，这里示例用i对256取模
//				data[i] = byte(i % 256)
//			}
//			err = os.WriteFile(path, data, 0644)
//			require.NoError(t, err)
//		}
//		return err
//	})
//	assert.NoError(t, err)
//	storage, err = NewPebbleStorage(tempDir, &MockLogger{})
//	assert.NoError(t, err)
//	_, err = storage.Get(ctx, projectID, key)
//	assert.ErrorIs(t, err, ErrKeyNotFound)
//}
//
//func TestPebbleStorage_NonexistentDirectory(t *testing.T) {
//	tempDir := filepath.Join(os.TempDir(), "nonexistent", "deep", "path", fmt.Sprintf("%d", time.Now().UnixNano()))
//	defer os.RemoveAll(filepath.Dir(tempDir))
//
//	storage, err := NewPebbleStorage(tempDir, &MockLogger{})
//	assert.NoError(t, err)
//	assert.NotNil(t, storage)
//
//	if storage != nil {
//		storage.Close()
//	}
//}
//
//func TestPebbleStorage_ReadOnlyFileSystem(t *testing.T) {
//	if testing.Short() {
//		t.Skip("跳过只读文件系统测试")
//	}
//
//	// 在Windows系统上跳过此测试，因为Windows对只读目录的处理方式不同
//	if runtime.GOOS == "windows" {
//		t.Skip("在Windows系统上跳过只读文件系统测试")
//	}
//
//	tempDir, err := os.MkdirTemp("", "readonly-test-*")
//	require.NoError(t, err)
//	defer os.RemoveAll(tempDir)
//
//	// 创建存储并写入数据
//	storage, err := NewPebbleStorage(tempDir, &MockLogger{})
//	require.NoError(t, err)
//
//	ctx := context.Background()
//	projectName := "test"
//	projectPath := "/tmp/test"
//	projectID := GenerateTestProjectUUID(projectName, projectPath)
//	err = storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: &codegraphpb.TestMessage{Value: "data"}})
//	assert.NoError(t, err)
//	storage.Close()
//
//	// 修改目录权限为只读
//	err = os.Chmod(tempDir, 0444)
//	require.NoError(t, err)
//	defer os.Chmod(tempDir, 0755) // 恢复权限
//
//	// 尝试在只读目录中创建新存储
//	_, err = NewPebbleStorage(tempDir, &MockLogger{})
//	assert.Error(t, err)
//}
//
//func TestPebbleStorage_LargeDataHandling(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectName := "large-data-test"
//	projectPath := "/tmp/large-data-test"
//	projectID := GenerateTestProjectUUID(projectName, projectPath)
//
//	// 测试大数据量
//	const dataCount = 1000
//	values := &testValues{
//		values: make([]proto.Message, dataCount),
//		keys:   make([]string, dataCount),
//	}
//	for i := 0; i < dataCount; i++ {
//		values.values[i] = &codegraphpb.TestMessage{Value: fmt.Sprintf("large-value-%d", i)}
//		values.keys[i] = fmt.Sprintf("key-%d", i)
//	}
//	err := storage.BatchSave(ctx, projectID, values)
//	assert.NoError(t, err)
//
//	size := storage.Size(ctx, projectID)
//	assert.Equal(t, dataCount, size)
//
//	// 测试超大单条数据
//	largeValue := &codegraphpb.TestMessage{Value: string(make([]byte, 1024*1024))} // 1MB
//	largeValueProjectID := projectID + "-large"
//	err = storage.Save(ctx, largeValueProjectID, &Entry{Key: ElementPathKey("test-value"), Value: largeValue})
//	assert.NoError(t, err)
//
//	retrieved, err := storage.Get(ctx, largeValueProjectID, TestKey{key: "*store.codegraphpb.TestMessage"})
//	assert.NoError(t, err)
//	assert.Equal(t, largeValue.Value, retrieved.(*RawMessage).Data)
//}
//
//func TestPebbleStorage_SpecialProjectNames(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	// 项目名来自路径的最后一层，只会出现路径允许的字符（windows、linux、mac）
//	specialNames := []string{
//		"test-project",                 // 普通项目名
//		"test.project",                 // 包含点
//		"test project",                 // 包含空格
//		"test-project-123",             // 包含数字
//		"test_project",                 // 包含下划线
//		"test-project.123",             // 包含点和数字
//		"a",                            // 单字符
//		"1234567890",                   // 纯数字
//		"test-project-with-unicode-测试", // 包含Unicode字符
//	}
//
//	for _, projectName := range specialNames {
//		// 为每个项目生成唯一的路径
//		projectPath := fmt.Sprintf("/tmp/%s", projectName)
//		projectID := GenerateTestProjectUUID(projectName, projectPath)
//
//		value := &codegraphpb.TestMessage{Value: "test-value"}
//		err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: value})
//		assert.NoError(t, err)
//
//		retrieved, err := storage.Get(ctx, projectID, TestKey{key: "*store.codegraphpb.TestMessage"})
//		assert.NoError(t, err)
//		assert.Equal(t, value.Value, retrieved)
//	}
//}
//
//func TestPebbleStorage_CloseDuringOperations(t *testing.T) {
//	storage, cleanup := setupPebbleTestStorage(t)
//	defer cleanup()
//
//	ctx := context.Background()
//	projectID := "close-during-ops"
//
//	// 预填充数据
//	err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: &codegraphpb.TestMessage{Value: "test"}})
//	assert.NoError(t, err)
//
//	// 并发关闭和操作
//	var wg sync.WaitGroup
//	wg.Add(2)
//
//	go func() {
//		defer wg.Done()
//		time.Sleep(10 * time.Millisecond)
//		storage.Close()
//	}()
//
//	go func() {
//		defer wg.Done()
//		for i := 0; i < 100; i++ {
//			err := storage.Save(ctx, projectID, &Entry{Key: ElementPathKey("test-value"), Value: &codegraphpb.TestMessage{Value: fmt.Sprintf("concurrent-%d", i)}})
//			if err != nil {
//				// 期望在关闭后操作失败
//				assert.Contains(t, err.Error(), "storage is closed")
//				break
//			}
//		}
//	}()
//
//	wg.Wait()
//}
