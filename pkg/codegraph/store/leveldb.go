package store

import (
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"os"
	"path/filepath"
	"sync"

	"codebase-indexer/pkg/logger"

	"github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/protobuf/proto"
)

// LevelDBStorage implements GraphStorage interface using LevelDB
type LevelDBStorage struct {
	baseDir   string
	logger    logger.Logger
	clients   sync.Map // projectUuid -> *leveldb.DB
	closeOnce sync.Once
	closed    bool
	dbMutex   sync.Map // projectUuid -> *sync.Mutex
}

// NewLevelDBStorage creates new LevelDB storage instance
func NewLevelDBStorage(baseDir string, logger logger.Logger) (*LevelDBStorage, error) {
	logger.Info("new_leveldb_storage: checking base directory", "baseDir", baseDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	logger.Info("new_leveldb_storage: checking directory permissions")
	if err := checkDirWritable(baseDir); err != nil {
		return nil, fmt.Errorf("directory not writable: %w", err)
	}

	storage := &LevelDBStorage{
		baseDir: baseDir,
		logger:  logger,
	}

	logger.Info("new_leveldb_storage: initialized successfully", "baseDir", baseDir)
	return storage, nil
}

// getDB gets or creates LevelDB instance for specified project
func (s *LevelDBStorage) getDB(projectUuid string) (*leveldb.DB, error) {
	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	// 获取或创建项目级别的互斥锁
	mutexInterface, _ := s.dbMutex.LoadOrStore(projectUuid, &sync.Mutex{})
	mutex := mutexInterface.(*sync.Mutex)

	// 加锁防止并发创建数据库
	mutex.Lock()
	defer mutex.Unlock()

	s.logger.Debug("get_db: checking existing client", "project", projectUuid)

	if db, exists := s.clients.Load(projectUuid); exists {
		return db.(*leveldb.DB), nil
	}

	db, err := s.createDB(projectUuid)
	if err != nil {
		return nil, err
	}

	actual, loaded := s.clients.LoadOrStore(projectUuid, db)
	if loaded {
		db.Close()
		return actual.(*leveldb.DB), nil
	}

	return db, nil
}

// createDB creates new LevelDB instance
func (s *LevelDBStorage) createDB(projectUuid string) (*leveldb.DB, error) {
	s.logger.Debug("create_db: creating project directory", "project", projectUuid)
	projectDir := filepath.Join(s.baseDir, projectUuid)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory %s: %w", projectDir, err)
	}

	dbPath := filepath.Join(projectDir, "leveldb")
	s.logger.Debug("create_db: opening database", "project", projectUuid, "path", dbPath)

	// 配置LevelDB选项
	dbOptions := &opt.Options{
		BlockCacheCapacity: 64 * 1024 * 1024, // 64MB block cache
		WriteBuffer:        16 * 1024 * 1024, // 16MB write buffer
	}

	db, err := leveldb.OpenFile(dbPath, dbOptions)
	if err != nil {
		s.logger.Warn("create_db: database open failed, attempting to recreate", "project", projectUuid, "error", err)

		// 尝试删除损坏的数据库文件并重建
		if removeErr := os.RemoveAll(dbPath); removeErr != nil {
			s.logger.Error("create_db: failed to remove corrupted database", "project", projectUuid, "error", removeErr)
			return nil, fmt.Errorf("failed to open project database %s: %w (and failed to remove corrupted dir: %v)", dbPath, err, removeErr)
		}

		// 重新尝试创建数据库
		db, err = leveldb.OpenFile(dbPath, dbOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate project database %s: %w", dbPath, err)
		}
	}

	s.logger.Debug("create_db: created new project database", "project", projectUuid, "path", dbPath)
	return db, nil
}

// BatchSave saves multiple values in batch
func (s *LevelDBStorage) BatchSave(ctx context.Context, projectUuid string, values Values) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("batch_save: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	s.logger.Debug("batch_save: starting batch transaction", "project", projectUuid, "count", values.Len())

	batch := new(leveldb.Batch)
	for i := 0; i < values.Len(); i++ {
		if err := utils.CheckContext(ctx); err != nil {
			return fmt.Errorf("context cancelled during batch save: %w", err)
		}

		key := values.Key(i)
		value := values.Value(i)

		var data []byte
		var marshalErr error

		// 处理自定义测试消息类型
		if customMsg, ok := value.(interface {
			Marshal() ([]byte, error)
		}); ok {
			data, marshalErr = customMsg.Marshal()
		} else {
			data, marshalErr = proto.Marshal(value)
		}

		if marshalErr != nil {
			return fmt.Errorf("failed to marshal data for key %s: %w", key, marshalErr)
		}

		batch.Put([]byte(key), data)
	}

	err = db.Write(batch, nil)
	s.logger.Debug("batch_save: completed", "project", projectUuid, "count", values.Len())
	return err
}

// Save saves single value
func (s *LevelDBStorage) Save(ctx context.Context, projectUuid string, value proto.Message) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("save: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	key := fmt.Sprintf("%T", value)
	s.logger.Debug("save: starting transaction", "project", projectUuid, "type", key)

	var data []byte

	// 处理自定义测试消息类型
	if customMsg, ok := value.(interface {
		Marshal() ([]byte, error)
	}); ok {
		data, err = customMsg.Marshal()
	} else {
		data, err = proto.Marshal(value)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal data for type %s: %w", key, err)
	}

	err = db.Put([]byte(key), data, nil)
	s.logger.Debug("save: completed", "project", projectUuid, "type", key)
	return err
}

// Get retrieves data by key
func (s *LevelDBStorage) Get(ctx context.Context, projectUuid string, key Key) (proto.Message, error) {
	if err := utils.CheckContext(ctx); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("get: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	s.logger.Debug("get: starting transaction", "project", projectUuid, "key", key.Get())

	data, err := db.Get([]byte(key.Get()), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("key %s not found in project %s", key.Get(), projectUuid)
		}
		return nil, fmt.Errorf("failed to get key %s: %w", key.Get(), err)
	}

	return &RawMessage{Data: data}, nil
}

// Delete deletes data by key
func (s *LevelDBStorage) Delete(ctx context.Context, projectUuid string, key Key) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("delete: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	s.logger.Debug("delete: starting transaction", "project", projectUuid, "key", key.Get())

	err = db.Delete([]byte(key.Get()), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("failed to delete key %s: %w", key.Get(), err)
	}

	s.logger.Debug("delete: completed", "project", projectUuid, "key", key.Get())
	return nil
}

// Iter creates iterator
func (s *LevelDBStorage) Iter(ctx context.Context, projectUuid string) Iterator {
	s.logger.Debug("iter: creating iterator", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		s.logger.Debug("iter: failed to get database", "project", projectUuid, "error", err)
		return nil
	}
	return &leveldbIterator{
		storage:   s,
		projectID: projectUuid,
		ctx:       ctx,
		db:        db,
		iter:      db.NewIterator(nil, nil),
	}
}

// Size returns project data size
func (s *LevelDBStorage) Size(ctx context.Context, projectUuid string) int {
	if err := utils.CheckContext(ctx); err != nil {
		s.logger.Debug("size: context cancelled", "project", projectUuid)
		return 0
	}

	s.logger.Debug("size: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		s.logger.Debug("size: failed to get database", "project", projectUuid, "error", err)
		return 0
	}

	count := 0
	s.logger.Debug("size: counting records", "project", projectUuid)

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		s.logger.Debug("size: failed to count records", "project", projectUuid, "error", err)
		return 0
	}

	return count
}

// Close closes all database connections
func (s *LevelDBStorage) Close() error {
	s.logger.Info("close: closing all connections")

	var errs []error
	s.clients.Range(func(key, value interface{}) bool {
		projectID := key.(string)
		db := value.(*leveldb.DB)

		s.logger.Info("close: closing database", "projectID", projectID)
		if err := db.Close(); err != nil {
			s.logger.Error("close: failed to close database", "projectID", projectID, "error", err)
			errs = append(errs, fmt.Errorf("failed to close project %s database: %w", projectID, err))
		} else {
			s.logger.Info("close: successfully closed database", "projectID", projectID)
		}
		return true
	})

	s.closeOnce.Do(func() {
		s.closed = true
		s.logger.Info("close: storage marked as closed")
	})

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while closing storage: %v", errs)
	}

	s.logger.Info("close: storage closed successfully")
	return nil
}

// leveldbIterator implements Iterator interface
type leveldbIterator struct {
	storage   *LevelDBStorage
	projectID string
	ctx       context.Context
	db        *leveldb.DB
	iter      iterator.Iterator
	currentK  []byte
	currentV  []byte
	err       error
	closed    bool
}

func (it *leveldbIterator) Next() bool {
	if it.closed {
		return false
	}

	// 检查上下文取消
	select {
	case <-it.ctx.Done():
		it.err = it.ctx.Err()
		return false
	default:
	}
	if it.iter == nil {
		it.storage.logger.Debug("next: getting database", "project", it.projectID)
		db, err := it.storage.getDB(it.projectID)
		if err != nil {
			it.err = fmt.Errorf("failed to get database: %w", err)
			return false
		}
		it.db = db

		it.storage.logger.Debug("next: creating iterator", "project", it.projectID)
		it.iter = db.NewIterator(nil, nil)
		it.iter.First()
	} else {
		it.iter.Next()
	}

	if it.iter.Valid() {
		it.currentK = it.iter.Key()
		it.currentV = it.iter.Value()
		return true
	}

	return false
}

func (it *leveldbIterator) Key() string {
	if it.currentK == nil {
		return ""
	}
	return string(it.currentK)
}

func (it *leveldbIterator) Value() proto.Message {
	if it.currentV == nil {
		return nil
	}
	return &RawMessage{Data: it.currentV}
}

func (it *leveldbIterator) Error() error {
	if it.err != nil {
		return it.err
	}
	return nil
}

func (it *leveldbIterator) Close() error {
	if it.closed {
		return nil
	}

	it.closed = true
	if it.iter != nil {
		it.iter.Release()
		it.iter = nil
	}
	err := it.Close()
	it.currentK = nil
	it.currentV = nil
	it.db = nil
	return err
}
