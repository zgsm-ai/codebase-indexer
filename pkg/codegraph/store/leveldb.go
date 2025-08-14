package store

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"

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
	logger.Info("leveldb: checking base directory baseDir %s", baseDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	logger.Info("leveldb: checking directory permissions")
	if err := checkDirWritable(baseDir); err != nil {
		return nil, fmt.Errorf("directory not writable: %w", err)
	}

	storage := &LevelDBStorage{
		baseDir: baseDir,
		logger:  logger,
	}

	logger.Info("leveldb: initialized successfully baseDir %s", baseDir)
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

func (s *LevelDBStorage) generateDbPath(projectUuid string) string {
	return filepath.Join(s.baseDir, projectUuid, dataDir)
}

// createDB creates new LevelDB instance
func (s *LevelDBStorage) createDB(projectUuid string) (*leveldb.DB, error) {
	s.logger.Info("create_db: creating project directory project %s", projectUuid)
	projectDir := filepath.Join(s.baseDir, projectUuid)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory %s: %w", projectDir, err)
	}

	dbPath := s.generateDbPath(projectUuid)
	s.logger.Info("create_db: opening database project %s path %s", projectUuid, dbPath)

	db, err := openLevelDB(dbPath)
	if err != nil {
		s.logger.Warn("create_db: database open failed, attempting to recreate. project %s err:%v", projectUuid, err)

		// 尝试删除损坏的数据库文件并重建
		if removeErr := os.RemoveAll(dbPath); removeErr != nil {
			s.logger.Error("create_db: failed to remove corrupted database. project %s err:%v", projectUuid, removeErr)
			return nil, fmt.Errorf("failed to open project database %s: %w (and failed to remove corrupted dir: %v)", dbPath, err, removeErr)
		}

		// 重新尝试创建数据库
		db, err = openLevelDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate project database %s: %w", dbPath, err)
		}
	}

	s.logger.Debug("create_db: created new project database. project %s path %s", projectUuid, dbPath)
	return db, nil
}

func openLevelDB(dbPath string) (*leveldb.DB, error) {
	// 配置LevelDB选项
	dbOptions := &opt.Options{
		WriteBuffer:        4 * 1024 * 1024, // 16MB write buffer
		BlockCacheCapacity: 8 * 1024 * 1024, // 64MB block cache
		// 限制内存表大小，更早刷盘
		//WriteBuffer: 2 * 1024 * 1024, // 2MiB
		//// 减小块缓存，减少缓冲区占用
		//BlockCacheCapacity: 4 * 1024 * 1024, // 4MiB
		//// 禁用缓冲区池，避免缓冲区累积
		//DisableBufferPool: false,
		//// 更早触发合并，减少Level-0表数量
		//CompactionL0Trigger: 2,
		//// 减少打开文件缓存，降低mmap内存
		//OpenFilesCacheCapacity: 100,
	}

	db, err := leveldb.OpenFile(dbPath, dbOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbPath, err)
	}

	return db, nil
}

// BatchSave saves multiple values in batch
func (s *LevelDBStorage) BatchSave(ctx context.Context, projectUuid string, values Entries) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	for i := 0; i < values.Len(); i++ {
		if err := utils.CheckContext(ctx); err != nil {
			return fmt.Errorf("context cancelled during batch save: %w", err)
		}

		key, err := values.Key(i).Get()
		if err != nil {
			s.logger.Error("level_db batch save error:%v", err)
			continue
		}
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
			s.logger.Error("level_db batch save failed to marshal data for key %s, %v", key, marshalErr)
			continue
		}

		_ = db.Put([]byte(key), data, &opt.WriteOptions{})
	}

	return err
}

// Save saves single value
func (s *LevelDBStorage) Put(ctx context.Context, projectUuid string, entry *Entry) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	keyStr, err := entry.Key.Get()
	if err != nil {
		return err
	}

	var data []byte
	data, err = proto.Marshal(entry.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal data for type %s: %w", keyStr, err)
	}

	err = db.Put([]byte(keyStr), data, nil)

	return err
}

// Get retrieves data by key
func (s *LevelDBStorage) Get(ctx context.Context, projectUuid string, key Key) ([]byte, error) {
	if err := utils.CheckContext(ctx); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	db, err := s.getDB(projectUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}
	keyStr, err := key.Get()
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	data, err := db.Get([]byte(keyStr), nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key %s: %w", keyStr, err)
	}

	return data, nil
}
func (s *LevelDBStorage) Exists(ctx context.Context, projectUuid string, key Key) (bool, error) {
	if err := utils.CheckContext(ctx); err != nil {
		return false, fmt.Errorf("context cancelled: %w", err)
	}

	db, err := s.getDB(projectUuid)
	if err != nil {
		return false, fmt.Errorf("failed to get database: %w", err)
	}
	keyStr, err := key.Get()
	if err != nil {
		return false, err
	}
	return db.Has([]byte(keyStr), nil)
}

// Delete deletes data by key
func (s *LevelDBStorage) Delete(ctx context.Context, projectUuid string, key Key) error {
	if err := utils.CheckContext(ctx); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	db, err := s.getDB(projectUuid)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	keyStr, err := key.Get()
	if err != nil {
		return err
	}

	err = db.Delete([]byte(keyStr), nil)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return fmt.Errorf("failed to delete key %s: %w", keyStr, err)
	}

	return nil
}

func (s *LevelDBStorage) DeleteAll(ctx context.Context, projectUuid string) error {
	db, err := s.getDB(projectUuid)
	if err != nil {
		s.logger.Debug("iter: failed to get database. project %s, error: %v", projectUuid, err)
		return nil
	}
	s.logger.Info("level_db: start to delete all for project %s", projectUuid)
	iter := s.Iter(ctx, projectUuid)
	for iter.Next() {
		_ = db.Delete([]byte(iter.Key()), nil)
	}
	err = iter.Close()
	s.logger.Info("level_db: delete all for project %s end, after size: %d", projectUuid,
		s.Size(ctx, projectUuid, types.EmptyString))
	return err
}

// Iter creates iterator
func (s *LevelDBStorage) Iter(ctx context.Context, projectUuid string) Iterator {
	db, err := s.getDB(projectUuid)
	if err != nil {
		s.logger.Debug("iter: failed to get database. project %s, error: %v", projectUuid, err)
		return nil
	}
	return &leveldbIterator{
		storage:     s,
		projectUuid: projectUuid,
		ctx:         ctx,
		db:          db,
		iter:        db.NewIterator(nil, nil),
	}
}

// Size returns project data size
func (s *LevelDBStorage) Size(ctx context.Context, projectUuid string, keyPrefix string) int {
	if err := utils.CheckContext(ctx); err != nil {
		s.logger.Debug("size: context cancelled. project %s", projectUuid)
		return 0
	}

	db, err := s.getDB(projectUuid)
	if err != nil {
		s.logger.Debug("size: failed to get database. project %s, error:%v", projectUuid, err)
		return 0
	}

	count := 0

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		if keyPrefix == types.EmptyString || strings.HasPrefix(string(iter.Key()), keyPrefix) {
			count++
		}
	}

	if err := iter.Error(); err != nil {
		s.logger.Debug("size: failed to count records. project %s error:%v", projectUuid, err)
		return 0
	}

	return count
}

// Close closes all database connections
func (s *LevelDBStorage) Close() error {
	if s.closed {
		return nil
	}

	s.logger.Info("leveldb_close: closing all connections")

	var errs []error
	s.clients.Range(func(key, value interface{}) bool {
		projectID := key.(string)
		db := value.(*leveldb.DB)

		s.logger.Info("leveldb_close: closing database. projectUuid %s", projectID)
		if err := db.Close(); err != nil {
			s.logger.Error("leveldb_close: failed to close database. projectUuid %s, err: %v", projectID, err)
			errs = append(errs, fmt.Errorf("failed to close project %s database: %w", projectID, err))
		} else {
			s.logger.Info("leveldb_close: successfully closed database. projectUuid %s", projectID)
		}
		return true
	})

	s.closeOnce.Do(func() {
		s.closed = true
		s.logger.Info("leveldb_close: storage marked as closed")
	})

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while closing storage: %v", errs)
	}

	s.logger.Info("leveldb_close: storage closed successfully")
	return nil
}

func (s *LevelDBStorage) ExistsProject(projectUuid string) (bool, error) {
	dbPath := s.generateDbPath(projectUuid)
	// 调用os.Stat获取路径信息
	_, err := os.Stat(dbPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// 其他错误（如权限问题等）
	return false, fmt.Errorf("check project index path err: %w", err)
}

// leveldbIterator implements Iterator interface
type leveldbIterator struct {
	storage     *LevelDBStorage
	projectUuid string
	ctx         context.Context
	db          *leveldb.DB
	iter        iterator.Iterator
	currentK    []byte
	currentV    []byte
	err         error
	closed      bool
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
		it.storage.logger.Debug("next: getting database project %s", it.projectUuid)
		db, err := it.storage.getDB(it.projectUuid)
		if err != nil {
			it.err = fmt.Errorf("failed to get database: %w", err)
			return false
		}
		it.db = db

		it.storage.logger.Debug("next: creating iterator. project %s", it.projectUuid)
		it.iter = db.NewIterator(nil, nil)
		if it.iter == nil {
			it.err = fmt.Errorf("failed to create iterator")
			return false
		}
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

func (it *leveldbIterator) Value() []byte {
	if it.currentV == nil {
		return nil
	}
	return it.currentV
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
	var err error
	if it.iter != nil {
		err = it.iter.Error()
		it.iter.Release()
		it.iter = nil
	}
	it.currentK = nil
	it.currentV = nil
	it.db = nil
	return err
}
