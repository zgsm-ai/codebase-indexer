package store

//
//import (
//	"codebase-indexer/pkg/codegraph/utils"
//	"context"
//	"errors"
//	"fmt"
//	"os"
//	"path/filepath"
//	"sync"
//
//	"github.com/cockroachdb/pebble"
//	"google.golang.org/protobuf/proto"
//
//	"codebase-indexer/pkg/logger"
//)
//
//// PebbleStorage implements GraphStorage interface using Pebble
//type PebbleStorage struct {
//	baseDir   string
//	logger    logger.Logger
//	clients   sync.Map // projectUuid -> *pebble.DB
//	closeOnce sync.Once
//	closed    bool
//	dbMutex   sync.Map // projectUuid -> *sync.Mutex
//}
//
//// NewPebbleStorage creates new Pebble storage instance
//func NewPebbleStorage(baseDir string, logger logger.Logger) (*PebbleStorage, error) {
//	logger.Info("new_pebble_storage: checking base directory", "baseDir", baseDir)
//	if err := os.MkdirAll(baseDir, 0755); err != nil {
//		return nil, fmt.Errorf("failed to create base directory: %w", err)
//	}
//
//	logger.Info("new_pebble_storage: checking directory permissions")
//	if err := checkDirWritable(baseDir); err != nil {
//		return nil, fmt.Errorf("directory not writable: %w", err)
//	}
//
//	storage := &PebbleStorage{
//		baseDir: baseDir,
//		logger:  logger,
//	}
//
//	logger.Info("new_pebble_storage: initialized successfully", "baseDir", baseDir)
//	return storage, nil
//}
//
//// getDB gets or creates Pebble instance for specified project
//func (s *PebbleStorage) getDB(projectUuid string) (*pebble.DB, error) {
//	if s.closed {
//		return nil, fmt.Errorf("storage is closed")
//	}
//
//	// 获取或创建项目级别的互斥锁
//	mutexInterface, _ := s.dbMutex.LoadOrStore(projectUuid, &sync.Mutex{})
//	mutex := mutexInterface.(*sync.Mutex)
//
//	// 加锁防止并发创建数据库
//	mutex.Lock()
//	defer mutex.Unlock()
//
//	s.logger.Debug("get_db: checking existing client", "project", projectUuid)
//
//	if db, exists := s.clients.Load(projectUuid); exists {
//		return db.(*pebble.DB), nil
//	}
//
//	db, err := s.createDB(projectUuid)
//	if err != nil {
//		return nil, err
//	}
//
//	actual, loaded := s.clients.LoadOrStore(projectUuid, db)
//	if loaded {
//		db.Close()
//		return actual.(*pebble.DB), nil
//	}
//
//	return db, nil
//}
//
//// createDB 创建新的数据库实例
//func (s *PebbleStorage) createDB(projectUuid string) (*pebble.DB, error) {
//	s.logger.Debug("create_db: creating project directory", "project", projectUuid)
//	// 对项目ID进行编码，避免特殊字符导致目录创建失败
//	projectDir := filepath.Join(s.baseDir, projectUuid)
//	if err := os.MkdirAll(projectDir, 0755); err != nil {
//		return nil, fmt.Errorf("failed to create project directory %s: %w", projectDir, err)
//	}
//
//	dbPath := filepath.Join(projectDir, dataDir)
//	s.logger.Debug("create_db: opening database", "project", projectUuid, "path", dbPath)
//
//	db, err := openPebbleDb(dbPath)
//	if err != nil {
//		s.logger.Warn("create_db: database open failed, attempting to recreate", "project", projectUuid, "error", err)
//
//		// 尝试删除损坏的数据库文件并重建
//		if removeErr := os.RemoveAll(dbPath); removeErr != nil {
//			s.logger.Error("create_db: failed to remove corrupted database", "project", projectUuid, "error", removeErr)
//			return nil, fmt.Errorf("failed to open project database %s: %w (and failed to remove corrupted file: %v)", dbPath, err, removeErr)
//		}
//
//		// 重新尝试创建数据库
//		db, err = openPebbleDb(dbPath)
//		if err != nil {
//			return nil, fmt.Errorf("failed to recreate project database %s: %w", dbPath, err)
//		}
//	}
//
//	s.logger.Debug("create_db: created new project database", "project", projectUuid, "path", dbPath)
//	return db, nil
//}
//
//func openPebbleDb(dbPath string) (*pebble.DB, error) {
//	cache := pebble.NewCache(256 << 20) // 256MB cache
//	opts := &pebble.Options{
//		Cache:        cache,
//		MaxOpenFiles: 1000,
//		MemTableSize: 64 << 20, // 64MB memtable
//		// 启用WAL以确保数据持久性
//	}
//	// 设置LSM树配置
//	opts.Levels = make([]pebble.LevelOptions, 7)
//	for i := range opts.Levels {
//		opts.Levels[i].Compression = pebble.ZstdCompression
//		opts.Levels[i].BlockSize = 32 << 10      // 32KB
//		opts.Levels[i].TargetFileSize = 64 << 20 // 64MB
//	}
//	return pebble.Open(dbPath, opts)
//}
//
//// BatchSave saves multiple values in batch
//func (s *PebbleStorage) BatchSave(ctx context.Context, projectUuid string, values Entries) error {
//	if err := utils.CheckContext(ctx); err != nil {
//		return fmt.Errorf("context cancelled: %w", err)
//	}
//
//	s.logger.Debug("batch_save: getting database", "project", projectUuid)
//	db, err := s.getDB(projectUuid)
//	if err != nil {
//		return fmt.Errorf("failed to get database: %w", err)
//	}
//
//	s.logger.Debug("batch_save: starting batch transaction", "project", projectUuid, "count", values.Len())
//
//	// 使用 Pebble 的 Batch API 来提高性能
//	batch := db.NewBatch()
//	defer batch.Close()
//
//	for i := 0; i < values.Len(); i++ {
//		if err := utils.CheckContext(ctx); err != nil {
//			return fmt.Errorf("context cancelled during batch save: %w", err)
//		}
//
//		key := values.Key(i)
//		value := values.Value(i)
//
//		var data []byte
//		var err error
//
//		// 处理自定义测试消息类型
//		if customMsg, ok := value.(interface {
//			Marshal() ([]byte, error)
//		}); ok {
//			data, err = customMsg.Marshal()
//		} else {
//			data, err = proto.Marshal(value)
//		}
//
//		if err != nil {
//			return fmt.Errorf("failed to marshal data for key %s: %w", key, err)
//		}
//
//		if err := batch.Set([]byte(key), data, pebble.Sync); err != nil {
//			return fmt.Errorf("failed to save data for key %s: %w", key, err)
//		}
//	}
//
//	// 提交批次
//	err = batch.Commit(pebble.Sync)
//	s.logger.Debug("batch_save: completed", "project", projectUuid, "count", values.Len())
//	return err
//}
//
//// Save saves single value
//func (s *PebbleStorage) Save(ctx context.Context, projectUuid string, entry *Entry) error {
//	if err := utils.CheckContext(ctx); err != nil {
//		return fmt.Errorf("context cancelled: %w", err)
//	}
//
//	s.logger.Debug("save: getting database", "project", projectUuid)
//	db, err := s.getDB(projectUuid)
//	if err != nil {
//		return fmt.Errorf("failed to get database: %w", err)
//	}
//
//	key := entry.Key.Get()
//	s.logger.Debug("save: starting transaction", "project", projectUuid, "type", key)
//
//	var data []byte
//	var marshalErr error
//
//	// 处理自定义测试消息类型
//	data, marshalErr = proto.Marshal(entry.Value)
//	if marshalErr != nil {
//		return fmt.Errorf("failed to marshal data for type %s: %w", key, marshalErr)
//	}
//
//	// 使用同步写入确保数据持久性
//	// 检查数据库是否已关闭
//	if s.closed {
//		return fmt.Errorf("storage is closed")
//	}
//
//	err = db.Set([]byte(key), data, pebble.Sync)
//	s.logger.Debug("save: completed", "project", projectUuid, "type", key)
//	return err
//}
//
//// Get retrieves data by key
//func (s *PebbleStorage) Get(ctx context.Context, projectUuid string, key Key) ([]byte, error) {
//	if err := utils.CheckContext(ctx); err != nil {
//		return nil, fmt.Errorf("context cancelled: %w", err)
//	}
//
//	s.logger.Debug("get: getting database", "project", projectUuid)
//	db, err := s.getDB(projectUuid)
//	if err != nil {
//		return nil, fmt.Errorf("failed to get database: %w", err)
//	}
//
//	var result proto.Message
//	s.logger.Debug("get: starting transaction", "project", projectUuid, "key", key.Get())
//
//	data, closer, err := db.Get([]byte(key.Get()))
//	if err != nil {
//		if errors.Is(err, pebble.ErrNotFound) {
//			return nil, ErrKeyNotFound
//		}
//		return nil, fmt.Errorf("failed to get key %s: %w", key.Get(), err)
//	}
//
//	if closer != nil {
//		defer closer.Close()
//	}
//
//	return data, nil
//}
//
//// Delete deletes data by key
//func (s *PebbleStorage) Delete(ctx context.Context, projectUuid string, key Key) error {
//	if err := utils.CheckContext(ctx); err != nil {
//		return fmt.Errorf("context cancelled: %w", err)
//	}
//
//	s.logger.Debug("delete: getting database", "project", projectUuid)
//	db, err := s.getDB(projectUuid)
//	if err != nil {
//		return fmt.Errorf("failed to get database: %w", err)
//	}
//
//	s.logger.Debug("delete: starting transaction", "project", projectUuid, "key", key.Get())
//
//	err = db.Delete([]byte(key.Get()), pebble.Sync)
//	if err != nil {
//		return fmt.Errorf("failed to delete key %s: %w", key.Get(), err)
//	}
//
//	s.logger.Debug("delete: completed", "project", projectUuid, "key", key.Get())
//	return nil
//}
//
//// Iter creates iterator
//func (s *PebbleStorage) Iter(ctx context.Context, projectUuid string) Iterator {
//	s.logger.Debug("iter: creating iterator", "project", projectUuid)
//	return &pebbleIterator{
//		storage:   s,
//		projectID: projectUuid,
//		ctx:       ctx,
//	}
//}
//
//// Size returns project data size
//func (s *PebbleStorage) Size(ctx context.Context, projectUuid string) int {
//	if err := utils.CheckContext(ctx); err != nil {
//		s.logger.Debug("size: context cancelled", "project", projectUuid)
//		return 0
//	}
//
//	s.logger.Debug("size: getting database", "project", projectUuid)
//	db, err := s.getDB(projectUuid)
//	if err != nil {
//		s.logger.Debug("size: failed to get database", "project", projectUuid, "error", err)
//		return 0
//	}
//
//	count := 0
//	s.logger.Debug("size: counting records", "project", projectUuid)
//
//	iter, _ := db.NewIter(nil)
//	defer iter.Close()
//
//	for iter.First(); iter.Valid(); iter.Next() {
//		count++
//	}
//
//	return count
//}
//
//// Close closes all database connections
//func (s *PebbleStorage) Close() error {
//	// 检查是否已经关闭
//	if s.closed {
//		s.logger.Info("close: storage already closed")
//		return nil
//	}
//
//	s.logger.Info("close: closing all connections")
//
//	var errs []error
//	s.clients.Range(func(key, value interface{}) bool {
//		projectID := key.(string)
//		db := value.(*pebble.DB)
//
//		s.logger.Info("close: closing database", "projectID", projectID)
//		if err := db.Close(); err != nil {
//			s.logger.Error("close: failed to close database", "projectID", projectID, "error", err)
//			errs = append(errs, fmt.Errorf("failed to close project %s database: %w", projectID, err))
//		} else {
//			s.logger.Info("close: successfully closed database", "projectID", projectID)
//		}
//		return true
//	})
//
//	// 清空客户端映射，避免重复关闭
//	s.clients = sync.Map{}
//
//	s.closeOnce.Do(func() {
//		s.closed = true
//		s.logger.Info("close: storage marked as closed")
//	})
//
//	if len(errs) > 0 {
//		return fmt.Errorf("errors occurred while closing storage: %v", errs)
//	}
//
//	s.logger.Info("close: storage closed successfully")
//	return nil
//}
//
//// pebbleIterator implements Iterator interface
//type pebbleIterator struct {
//	storage   *PebbleStorage
//	projectID string
//	ctx       context.Context
//	iter      *pebble.Iterator
//	err       error
//	closed    bool
//}
//
//func (it *pebbleIterator) Next() bool {
//	if it.closed {
//		return false
//	}
//
//	// 检查上下文取消
//	select {
//	case <-it.ctx.Done():
//		it.err = it.ctx.Err()
//		return false
//	default:
//	}
//
//	if it.iter == nil {
//		it.storage.logger.Debug("next: getting database", "project", it.projectID)
//		db, err := it.storage.getDB(it.projectID)
//		if err != nil {
//			it.err = fmt.Errorf("failed to get database: %w", err)
//			return false
//		}
//
//		it.storage.logger.Debug("next: creating iterator", "project", it.projectID)
//		var iterErr error
//		it.iter, iterErr = db.NewIter(nil)
//		if iterErr != nil {
//			it.err = fmt.Errorf("failed to create iterator: %w", iterErr)
//			return false
//		}
//
//		return it.iter.First()
//	}
//
//	return it.iter.Next()
//}
//
//func (it *pebbleIterator) Key() string {
//	if it.iter == nil || !it.iter.Valid() {
//		return ""
//	}
//	return string(it.iter.Key())
//}
//
//func (it *pebbleIterator) Value() proto.Message {
//	if it.iter == nil || !it.iter.Valid() {
//		return nil
//	}
//
//	// 复制数据以避免迭代器关闭后数据失效
//	data := make([]byte, len(it.iter.Value()))
//	copy(data, it.iter.Value())
//
//	return &RawMessage{Data: data}
//}
//
//func (it *pebbleIterator) Error() error {
//	return it.err
//}
//
//func (it *pebbleIterator) Close() error {
//	if it.closed {
//		return nil
//	}
//
//	it.closed = true
//	if it.iter != nil {
//		err := it.iter.Close()
//		it.iter = nil
//		return err
//	}
//	return nil
//}
