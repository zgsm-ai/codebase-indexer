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
	"time"

	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"codebase-indexer/pkg/logger"

	"github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/protobuf/proto"
)

const (
	// InactiveThreshold 定义数据库实例不活跃的时间阈值，超过这个时间将被清理
	InactiveThreshold = 6 * time.Hour
	// CleanupInterval 定义清理任务的执行间隔
	CleanupInterval = time.Hour
)

// dbAccessRecord 记录数据库实例的访问信息
type dbAccessRecord struct {
	lastAccessTime time.Time
	db             *leveldb.DB
}

// LevelDBStorage implements GraphStorage interface using LevelDB
type LevelDBStorage struct {
	baseDir       string
	logger        logger.Logger
	clients       sync.Map // projectUuid -> *dbAccessRecord
	closeOnce     sync.Once
	closed        bool
	dbMutex       sync.Map // projectUuid -> *sync.Mutex
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
	cleanupWG     sync.WaitGroup
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

	// 启动后台清理任务
	storage.startCleanupTask()

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

	now := time.Now()

	if record, exists := s.clients.Load(projectUuid); exists {
		accessRecord := record.(*dbAccessRecord)
		// 更新最后访问时间
		accessRecord.lastAccessTime = now
		return accessRecord.db, nil
	}

	db, err := s.createDB(projectUuid)
	if err != nil {
		return nil, err
	}

	accessRecord := &dbAccessRecord{
		lastAccessTime: now,
		db:             db,
	}

	actual, loaded := s.clients.LoadOrStore(projectUuid, accessRecord)
	if loaded {
		db.Close()
		return actual.(*dbAccessRecord).db, nil
	}

	return db, nil
}

func (s *LevelDBStorage) generateDbPath(projectUuid string) string {
	return filepath.Join(s.baseDir, projectUuid, dataDir)
}

// createDB creates new LevelDB instance
func (s *LevelDBStorage) createDB(projectUuid string) (*leveldb.DB, error) {
	s.logger.Info("creating project directory project %s", projectUuid)
	projectDir := filepath.Join(s.baseDir, projectUuid)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory %s: %w", projectDir, err)
	}

	dbPath := s.generateDbPath(projectUuid)
	s.logger.Info("opening database project %s path %s", projectUuid, dbPath)

	db, err := openLevelDB(dbPath)
	if err != nil {
		s.logger.Warn("database open failed, attempting to recreate. project %s err:%v", projectUuid, err)

		// 尝试删除损坏的数据库文件并重建
		if removeErr := os.RemoveAll(dbPath); removeErr != nil {
			s.logger.Error("failed to remove corrupted database. project %s err:%v", projectUuid, removeErr)
			return nil, fmt.Errorf("failed to open project database %s: %w (and failed to remove corrupted dir: %v)", dbPath, err, removeErr)
		}

		// 重新尝试创建数据库
		db, err = openLevelDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate project database %s: %w", dbPath, err)
		}
	}

	s.logger.Debug("created new project database. project %s path %s", projectUuid, dbPath)
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

	// 停止后台清理任务
	s.stopCleanupTask()

	var errs []error
	s.clients.Range(func(key, value interface{}) bool {
		projectID := key.(string)
		record := value.(*dbAccessRecord)
		db := record.db

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

// cleanupInactiveDBs 清理不活跃的数据库实例
func (s *LevelDBStorage) cleanupInactiveDBs() {
	if s.closed {
		return
	}

	now := time.Now()
	var cleanedCount int

	s.clients.Range(func(key, value interface{}) bool {
		projectUuid := key.(string)
		record := value.(*dbAccessRecord)

		// 检查是否超过不活跃阈值
		if now.Sub(record.lastAccessTime) > InactiveThreshold {
			s.logger.Info("cleanup: cleaning up inactive database. project %s, last access: %v",
				projectUuid, record.lastAccessTime)

			// 获取项目级别的互斥锁
			mutexInterface, _ := s.dbMutex.LoadOrStore(projectUuid, &sync.Mutex{})
			mutex := mutexInterface.(*sync.Mutex)

			// 加锁防止并发操作
			mutex.Lock()

			// 再次检查，防止在等待锁的过程中数据库被重新访问
			if currentRecord, exists := s.clients.Load(projectUuid); exists {
				currentAccessRecord := currentRecord.(*dbAccessRecord)
				if now.Sub(currentAccessRecord.lastAccessTime) > InactiveThreshold {
					// TODO 清理数据库数据
					// // 清理数据库数据
					// if err := s.cleanupDBData(projectUuid, currentAccessRecord.db); err != nil {
					// 	s.logger.Error("cleanup: failed to cleanup database data. project %s, err: %v",
					// 		projectUuid, err)
					// 	mutex.Unlock()
					// 	return true // 继续处理其他数据库
					// }

					// 关闭数据库连接
					if err := currentAccessRecord.db.Close(); err != nil {
						s.logger.Error("cleanup: failed to close database. project %s, err: %v",
							projectUuid, err)
					}

					// 从内存中移除
					s.clients.Delete(projectUuid)
					s.dbMutex.Delete(projectUuid)
					cleanedCount++
					s.logger.Info("cleanup: successfully cleaned up inactive database. project %s", projectUuid)
				}
			}

			mutex.Unlock()
		}

		return true
	})

	if cleanedCount > 0 {
		s.logger.Info("cleanup: completed cleanup of %d inactive databases", cleanedCount)
	}
}

// cleanupDBData 使用数据库的Delete方法清理数据
func (s *LevelDBStorage) cleanupDBData(projectUuid string, db *leveldb.DB) error {
	s.logger.Info("cleanup_db: starting data cleanup for project %s", projectUuid)

	count := 0

	// 创建迭代器遍历所有键
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		// 使用Delete方法删除数据
		if err := db.Delete(key, nil); err != nil {
			s.logger.Error("cleanup_db: failed to delete key %s for project %s, err: %v",
				string(key), projectUuid, err)
			continue
		}
		count++
	}

	if err := iter.Error(); err != nil {
		s.logger.Error("cleanup_db: iterator error for project %s, err: %v", projectUuid, err)
		return fmt.Errorf("iterator error: %w", err)
	}

	s.logger.Info("cleanup_db: successfully deleted %d keys for project %s", count, projectUuid)
	return nil
}

// startCleanupTask 启动后台清理任务
func (s *LevelDBStorage) startCleanupTask() {
	s.logger.Info("cleanup_task: starting background cleanup task")

	ctx, cancel := context.WithCancel(context.Background())
	s.cleanupCtx = ctx
	s.cleanupCancel = cancel

	s.cleanupWG.Add(1)
	go func() {
		defer s.cleanupWG.Done()

		ticker := time.NewTicker(CleanupInterval)
		defer ticker.Stop()

		s.logger.Info("cleanup_task: background cleanup task started, interval: %v", CleanupInterval)

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("cleanup_task: background cleanup task stopped")
				return
			case <-ticker.C:
				s.logger.Debug("cleanup_task: triggering cleanup check")
				s.cleanupInactiveDBs()
			}
		}
	}()
}

// stopCleanupTask 停止后台清理任务
func (s *LevelDBStorage) stopCleanupTask() {
	if s.cleanupCancel != nil {
		s.logger.Info("cleanup_task: stopping background cleanup task")
		s.cleanupCancel()
		s.cleanupWG.Wait()
		s.logger.Info("cleanup_task: background cleanup task stopped")
	}
}

func (s *LevelDBStorage) ProjectIndexExists(projectUuid string) (bool, error) {
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
