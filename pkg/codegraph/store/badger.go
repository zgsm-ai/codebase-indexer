package store

import (
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"codebase-indexer/pkg/logger"

	"github.com/dgraph-io/badger/v4"
	"google.golang.org/protobuf/proto"
)

// BadgerStorage implements GraphStorage interface using BadgerDB
type BadgerStorage struct {
	baseDir   string
	logger    logger.Logger
	clients   sync.Map // projectUuid -> *badger.DB
	closeOnce sync.Once
	closed    bool
	dbMutex   sync.Map // projectUuid -> *sync.Mutex
}

// NewBadgerStorage creates new BadgerDB storage instance
func NewBadgerStorage(baseDir string, logger logger.Logger) (*BadgerStorage, error) {
	logger.Info("new_badger_storage: checking base directory", "baseDir", baseDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	logger.Info("new_badger_storage: checking directory permissions")
	if err := checkDirWritable(baseDir); err != nil {
		return nil, fmt.Errorf("directory not writable: %w", err)
	}

	storage := &BadgerStorage{
		baseDir: baseDir,
		logger:  logger,
	}

	logger.Info("new_badger_storage: initialized successfully", "baseDir", baseDir)
	return storage, nil
}

// getDB gets or creates BadgerDB instance for specified project
func (s *BadgerStorage) getDB(projectUuid string) (*badger.DB, error) {
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
		return db.(*badger.DB), nil
	}

	db, err := s.createDB(projectUuid)
	if err != nil {
		return nil, err
	}

	actual, loaded := s.clients.LoadOrStore(projectUuid, db)
	if loaded {
		db.Close()
		return actual.(*badger.DB), nil
	}

	return db, nil
}

// createDB creates new BadgerDB instance
func (s *BadgerStorage) createDB(projectUuid string) (*badger.DB, error) {
	s.logger.Debug("create_db: creating project directory", "project", projectUuid)
	projectDir := filepath.Join(s.baseDir, projectUuid)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory %s: %w", projectDir, err)
	}

	dbPath := filepath.Join(projectDir, "badger")
	s.logger.Debug("create_db: opening database", "project", projectUuid, "path", dbPath)

	opts := badger.DefaultOptions(dbPath)
	opts.SyncWrites = false // 提高性能，但可能在崩溃时丢失数据
	opts.Logger = nil       // 禁用Badger的默认日志

	db, err := badger.Open(opts)
	if err != nil {
		s.logger.Warn("create_db: database open failed, attempting to recreate", "project", projectUuid, "error", err)

		// 尝试删除损坏的数据库文件并重建
		if removeErr := os.RemoveAll(dbPath); removeErr != nil {
			s.logger.Error("create_db: failed to remove corrupted database", "project", projectUuid, "error", removeErr)
			return nil, fmt.Errorf("failed to open project database %s: %w (and failed to remove corrupted dir: %v)", dbPath, err, removeErr)
		}

		// 重新尝试创建数据库
		db, err = badger.Open(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate project database %s: %w", dbPath, err)
		}
	}

	s.logger.Debug("create_db: created new project database", "project", projectUuid, "path", dbPath)
	return db, nil
}

// BatchSave saves multiple values in batch
func (s *BadgerStorage) BatchSave(ctx context.Context, projectUuid string, values Values) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("batch_save: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	s.logger.Debug("batch_save: starting batch transaction", "project", projectUuid, "count", values.Len())

	err = db.Update(func(txn *badger.Txn) error {
		for i := 0; i < values.Len(); i++ {
			if err := utils.CheckContext(ctx); err != nil {
				return fmt.Errorf("context cancelled during batch save: %w", err)
			}

			key := values.Key(i)
			value := values.Value(i)

			var data []byte
			var err error

			// 处理自定义测试消息类型
			if customMsg, ok := value.(interface {
				Marshal() ([]byte, error)
			}); ok {
				data, err = customMsg.Marshal()
			} else {
				data, err = proto.Marshal(value)
			}

			if err != nil {
				return fmt.Errorf("failed to marshal data for key %s: %w", key, err)
			}

			if err := txn.Set([]byte(key), data); err != nil {
				return fmt.Errorf("failed to save data for key %s: %w", key, err)
			}
		}
		return nil
	})

	s.logger.Debug("batch_save: completed", "project", projectUuid, "count", values.Len())
	return err
}

// Save saves single value
func (s *BadgerStorage) Save(ctx context.Context, projectUuid string, value proto.Message) error {
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

	return db.Update(func(txn *badger.Txn) error {
		var data []byte
		var err error

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

		if err := txn.Set([]byte(key), data); err != nil {
			return fmt.Errorf("failed to save data for type %s: %w", key, err)
		}

		s.logger.Debug("save: completed", "project", projectUuid, "type", key)
		return nil
	})
}

// Get retrieves data by key
func (s *BadgerStorage) Get(ctx context.Context, projectUuid string, key Key) (proto.Message, error) {
	if err := utils.CheckContext(ctx); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("get: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var result proto.Message
	s.logger.Debug("get: starting transaction", "project", projectUuid, "key", key.Get())

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key.Get()))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("key %s not found in project %s", key.Get(), projectUuid)
			}
			return fmt.Errorf("failed to get key %s: %w", key.Get(), err)
		}

		var data []byte
		err = item.Value(func(val []byte) error {
			data = append([]byte{}, val...) // 复制数据
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to read value for key %s: %w", key.Get(), err)
		}

		result = &RawMessage{Data: data}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Delete deletes data by key
func (s *BadgerStorage) Delete(ctx context.Context, projectUuid string, key Key) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("delete: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	s.logger.Debug("delete: starting transaction", "project", projectUuid, "key", key.Get())

	return db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete([]byte(key.Get())); err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // 键不存在时不报错
			}
			return fmt.Errorf("failed to delete key %s: %w", key.Get(), err)
		}

		s.logger.Debug("delete: completed", "project", projectUuid, "key", key.Get())
		return nil
	})
}

// Iter creates iterator
func (s *BadgerStorage) Iter(ctx context.Context, projectUuid string) Iterator {
	s.logger.Debug("iter: creating iterator", "project", projectUuid)
	return &badgerIterator{
		storage:   s,
		projectID: projectUuid,
		ctx:       ctx,
	}
}

// Size returns project data size
func (s *BadgerStorage) Size(ctx context.Context, projectUuid string) int {
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

	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // 只计算键，不加载值以提高性能
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})

	if err != nil {
		s.logger.Debug("size: failed to count records", "project", projectUuid, "error", err)
		return 0
	}

	return count
}

// Close closes all database connections
func (s *BadgerStorage) Close() error {
	s.logger.Info("close: closing all connections")

	var errs []error
	s.clients.Range(func(key, value interface{}) bool {
		projectID := key.(string)
		db := value.(*badger.DB)

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

// badgerIterator implements Iterator interface
type badgerIterator struct {
	storage   *BadgerStorage
	projectID string
	ctx       context.Context
	txn       *badger.Txn
	it        *badger.Iterator
	currentK  []byte
	currentV  []byte
	err       error
	closed    bool
}

func (it *badgerIterator) Next() bool {
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

	if it.txn == nil {
		it.storage.logger.Debug("next: getting database", "project", it.projectID)
		db, err := it.storage.getDB(it.projectID)
		if err != nil {
			it.err = fmt.Errorf("failed to get database: %w", err)
			return false
		}

		it.storage.logger.Debug("next: starting transaction", "project", it.projectID)
		it.txn = db.NewTransaction(false)

		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 100
		it.it = it.txn.NewIterator(opts)
		it.it.Rewind()
	} else {
		it.it.Next()
	}

	if it.it.Valid() {
		item := it.it.Item()
		it.currentK = item.KeyCopy(nil)

		err := item.Value(func(val []byte) error {
			it.currentV = append([]byte{}, val...)
			return nil
		})
		if err != nil {
			it.err = fmt.Errorf("failed to read value: %w", err)
			return false
		}
		return true
	}

	return false
}

func (it *badgerIterator) Key() string {
	if it.currentK == nil {
		return ""
	}
	return string(it.currentK)
}

func (it *badgerIterator) Value() proto.Message {
	if it.currentV == nil {
		return nil
	}
	return &RawMessage{Data: it.currentV}
}

func (it *badgerIterator) Error() error {
	return it.err
}

func (it *badgerIterator) Close() error {
	if it.closed {
		return nil
	}

	it.closed = true
	if it.it != nil {
		it.it.Close()
		it.it = nil
	}
	if it.txn != nil {
		it.txn.Discard()
		it.txn = nil
	}
	it.currentK = nil
	it.currentV = nil
	return nil
}
