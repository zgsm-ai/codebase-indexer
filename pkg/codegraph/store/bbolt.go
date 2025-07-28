package store

import (
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"codebase-indexer/pkg/logger"
)

// BBoltStorage implements GraphStorage interface using bbolt
type BBoltStorage struct {
	baseDir   string
	logger    logger.Logger
	clients   sync.Map // projectUuid -> *bbolt.DB
	closeOnce sync.Once
	closed    bool
	dbMutex   sync.Map // projectUuid -> *sync.Mutex
}

// NewBBoltStorage creates new bbolt storage instance
func NewBBoltStorage(baseDir string, logger logger.Logger) (*BBoltStorage, error) {
	logger.Info("new_bbolt_storage: checking base directory", "baseDir", baseDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	logger.Info("new_bbolt_storage: checking directory permissions")
	if err := checkDirWritable(baseDir); err != nil {
		return nil, fmt.Errorf("directory not writable: %w", err)
	}

	storage := &BBoltStorage{
		baseDir: baseDir,
		logger:  logger,
	}

	logger.Info("new_bbolt_storage: initialized successfully", "baseDir", baseDir)
	return storage, nil
}

// getDB gets or creates bbolt instance for specified project
func (s *BBoltStorage) getDB(projectUuid string) (*bbolt.DB, error) {
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
		return db.(*bbolt.DB), nil
	}

	db, err := s.createDB(projectUuid)
	if err != nil {
		return nil, err
	}

	actual, loaded := s.clients.LoadOrStore(projectUuid, db)
	if loaded {
		db.Close()
		return actual.(*bbolt.DB), nil
	}

	return db, nil
}

// createDB 创建新的数据库实例
func (s *BBoltStorage) createDB(projectUuid string) (*bbolt.DB, error) {
	s.logger.Debug("create_db: creating project directory", "project", projectUuid)
	// 对项目ID进行编码，避免特殊字符导致目录创建失败
	projectDir := filepath.Join(s.baseDir, projectUuid)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory %s: %w", projectDir, err)
	}

	dbPath := filepath.Join(projectDir, "data.db")
	s.logger.Debug("create_db: opening database", "project", projectUuid, "path", dbPath)

	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		s.logger.Warn("create_db: database open failed, attempting to recreate", "project", projectUuid, "error", err)

		// 尝试删除损坏的数据库文件并重建
		if removeErr := os.Remove(dbPath); removeErr != nil {
			s.logger.Error("create_db: failed to remove corrupted database", "project", projectUuid, "error", removeErr)
			return nil, fmt.Errorf("failed to open project database %s: %w (and failed to remove corrupted file: %v)", dbPath, err, removeErr)
		}

		// 重新尝试创建数据库
		db, err = bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
		if err != nil {
			return nil, fmt.Errorf("failed to recreate project database %s: %w", dbPath, err)
		}
	}

	s.logger.Debug("create_db: initializing bucket", "project", projectUuid)
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("data"))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize project bucket: %w", err)
	}

	s.logger.Debug("create_db: created new project database", "project", projectUuid, "path", dbPath)
	return db, nil
}

// BatchSave saves multiple values in batch
func (s *BBoltStorage) BatchSave(ctx context.Context, projectUuid string, values Values) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("batch_save: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	s.logger.Debug("batch_save: starting transaction", "project", projectUuid, "count", values.Len())
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("data"))
		if bucket == nil {
			return fmt.Errorf("data bucket not found in project %s", projectUuid)
		}

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

			if err := bucket.Put([]byte(key), data); err != nil {
				return fmt.Errorf("failed to save data for key %s: %w", key, err)
			}
		}

		s.logger.Debug("batch_save: completed", "project", projectUuid, "count", values.Len())
		return nil
	})
}

// Save saves single value
func (s *BBoltStorage) Save(ctx context.Context, projectUuid string, value proto.Message) error {
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

	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("data"))
		if bucket == nil {
			return fmt.Errorf("data bucket not found in project %s", projectUuid)
		}

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

		if err := bucket.Put([]byte(key), data); err != nil {
			return fmt.Errorf("failed to save data for type %s: %w", key, err)
		}

		s.logger.Debug("save: completed", "project", projectUuid, "type", key)
		return nil
	})
}

// Get retrieves data by key
func (s *BBoltStorage) Get(ctx context.Context, projectUuid string, key Key) (proto.Message, error) {
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

	err = db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("data"))
		if bucket == nil {
			return fmt.Errorf("data bucket not found in project %s", projectUuid)
		}

		data := bucket.Get([]byte(key.Get()))
		if data == nil {
			return fmt.Errorf("key %s not found in project %s", key.Get(), projectUuid)
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
func (s *BBoltStorage) Delete(ctx context.Context, projectUuid string, key Key) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	s.logger.Debug("delete: getting database", "project", projectUuid)
	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	s.logger.Debug("delete: starting transaction", "project", projectUuid, "key", key.Get())

	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("data"))
		if bucket == nil {
			return fmt.Errorf("data bucket not found in project %s", projectUuid)
		}

		if err := bucket.Delete([]byte(key.Get())); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key.Get(), err)
		}

		s.logger.Debug("delete: completed", "project", projectUuid, "key", key.Get())
		return nil
	})
}

// Iter creates iterator
func (s *BBoltStorage) Iter(ctx context.Context, projectUuid string) Iterator {
	s.logger.Debug("iter: creating iterator", "project", projectUuid)
	return &bboltIterator{
		storage:   s,
		projectID: projectUuid,
		ctx:       ctx,
	}
}

// Size returns project data size
func (s *BBoltStorage) Size(ctx context.Context, projectUuid string) int {
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
	_ = db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("data"))
		if bucket != nil {
			stats := bucket.Stats()
			count = stats.KeyN
		}
		return nil
	})

	return count
}

// Close closes all database connections
func (s *BBoltStorage) Close() error {
	s.logger.Info("close: closing all connections")

	var errs []error
	s.clients.Range(func(key, value interface{}) bool {
		projectID := key.(string)
		db := value.(*bbolt.DB)

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

// bboltIterator implements Iterator interface
type bboltIterator struct {
	storage   *BBoltStorage
	projectID string
	ctx       context.Context
	tx        *bbolt.Tx
	cursor    *bbolt.Cursor
	currentK  []byte
	currentV  []byte
	err       error
	closed    bool
}

func (it *bboltIterator) Next() bool {
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

	if it.tx == nil {
		it.storage.logger.Debug("next: getting database", "project", it.projectID)
		db, err := it.storage.getDB(it.projectID)
		if err != nil {
			it.err = fmt.Errorf("failed to get database: %w", err)
			return false
		}

		it.storage.logger.Debug("next: starting transaction", "project", it.projectID)
		it.tx, err = db.Begin(false)
		if err != nil {
			it.err = fmt.Errorf("failed to begin transaction: %w", err)
			return false
		}

		bucket := it.tx.Bucket([]byte("data"))
		if bucket == nil {
			it.err = fmt.Errorf("data bucket not found in project %s", it.projectID)
			return false
		}

		it.cursor = bucket.Cursor()
		it.currentK, it.currentV = it.cursor.First()
	} else {
		it.currentK, it.currentV = it.cursor.Next()
	}

	return it.currentK != nil
}

func (it *bboltIterator) Key() string {
	if it.currentK == nil {
		return ""
	}
	return string(it.currentK)
}

func (it *bboltIterator) Value() proto.Message {
	if it.currentV == nil {
		return nil
	}
	return &RawMessage{Data: it.currentV}
}

func (it *bboltIterator) Error() error {
	return it.err
}

func (it *bboltIterator) Close() error {
	if it.closed {
		return nil
	}

	it.closed = true
	if it.tx != nil {
		err := it.tx.Rollback()
		it.tx = nil
		it.cursor = nil
		return err
	}
	return nil
}

// RawMessage wraps raw protobuf bytes
type RawMessage struct {
	Data []byte
}

// Marshal implements proto.Message interface
func (r *RawMessage) Marshal() ([]byte, error) {
	return r.Data, nil
}

// Unmarshal implements proto.Message interface
func (r *RawMessage) Unmarshal(data []byte) error {
	r.Data = data
	return nil
}

// Reset implements proto.Message interface
func (r *RawMessage) Reset() {
	r.Data = nil
}

// String implements proto.Message interface
func (r *RawMessage) String() string {
	return string(r.Data)
}

// ProtoMessage implements proto.Message interface
func (r *RawMessage) ProtoMessage() {}

// ProtoReflect implements proto.Message interface
func (r *RawMessage) ProtoReflect() protoreflect.Message {
	return nil
}

// checkDirWritable checks if directory is writable
func checkDirWritable(dir string) error {
	testFile := filepath.Join(dir, ".test-write")
	file, err := os.Create(testFile)
	if err != nil {
		return err
	}
	file.Close()
	return os.Remove(testFile)
}
