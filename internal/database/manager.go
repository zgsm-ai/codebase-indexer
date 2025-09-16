package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/pkg/logger"

	_ "github.com/mattn/go-sqlite3" // SQLite3驱动
)

// DatabaseManager 数据库管理器接口
type DatabaseManager interface {
	Initialize() error
	Close() error
	GetDB() *sql.DB
	BeginTransaction() (*sql.Tx, error)
	// ClearTable 清理指定表数据并重置ID
	ClearTable(tableName string) error
	// ExecuteSQLFile 执行外部 SQL 文件
	ExecuteSQLFile(filePath string) error
}

// SQLiteManager SQLite数据库管理器实现
type SQLiteManager struct {
	db       *sql.DB
	config   *config.DatabaseConfig
	logger   logger.Logger
	mutex    sync.RWMutex
	migrator *Migrator
}

// NewSQLiteManager 创建SQLite数据库管理器
func NewSQLiteManager(config *config.DatabaseConfig, logger logger.Logger) DatabaseManager {
	return &SQLiteManager{
		config: config,
		logger: logger,
	}
}

// Initialize 初始化数据库连接和表结构
func (m *SQLiteManager) Initialize() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 构建数据库文件路径
	dbPath := filepath.Join(m.config.DataDir, m.config.DatabaseName)

	// 创建数据目录
	if err := os.MkdirAll(m.config.DataDir, 0755); err != nil {
		return err
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return err
	}

	// 配置连接池
	db.SetMaxOpenConns(m.config.MaxOpenConns)
	db.SetMaxIdleConns(m.config.MaxIdleConns)
	db.SetConnMaxLifetime(m.config.ConnMaxLifetime)

	// 测试连接
	if err := db.Ping(); err != nil {
		return err
	}

	m.db = db

	// 初始化迁移器 - 使用相对于项目根目录的迁移文件路径
	migrateDir := filepath.Join(dbPath, "migrations")
	m.migrator = NewMigrator(m.db, m.logger, migrateDir)

	// 使用迁移器自动执行数据库迁移
	if err := m.migrator.AutoMigrate(); err != nil {
		return err
	}

	m.logger.Info("Database initialized successfully")
	return nil
}

// Close 关闭数据库连接
func (m *SQLiteManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// GetDB 获取数据库连接
func (m *SQLiteManager) GetDB() *sql.DB {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.db
}

// BeginTransaction 开始事务
func (m *SQLiteManager) BeginTransaction() (*sql.Tx, error) {
	return m.db.Begin()
}

// ClearTable 清理指定表数据并重置ID
func (m *SQLiteManager) ClearTable(tableName string) error {
	return m.ClearTableWithOptions(tableName, nil)
}

// ClearTableOptions 清理表选项
type ClearTableOptions struct {
	BatchSize         *int           // 分批删除的批次大小，如果为nil则使用配置中的默认值
	BatchDelay        *time.Duration // 分批删除之间的延迟，如果为nil则使用配置中的默认值
	EnableProgressLog bool           // 是否启用进度日志
}

// ClearTableWithOptions 带选项的清理表数据方法
func (m *SQLiteManager) ClearTableWithOptions(tableName string, options *ClearTableOptions) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 验证表名
	validTables := map[string]bool{
		"workspaces": true,
		"events":     true,
	}
	if !validTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// 设置默认选项
	batchSize := m.config.BatchDeleteSize
	if options != nil && options.BatchSize != nil {
		batchSize = *options.BatchSize
	}

	batchDelay := m.config.BatchDeleteDelay
	if options != nil && options.BatchDelay != nil {
		batchDelay = *options.BatchDelay
	}

	enableProgressLog := false
	if options != nil {
		enableProgressLog = options.EnableProgressLog
	}

	// 获取表中的总记录数
	var totalCount int
	err := m.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("failed to get table row count: %v", err)
	}

	if totalCount == 0 {
		m.logger.Info("Table %s is already empty", tableName)
		return nil
	}

	if enableProgressLog {
		m.logger.Info("Starting to clear table %s with %d records (batch size: %d)", tableName, totalCount, batchSize)
	}

	// 开始事务
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 分批删除数据
	deletedCount := 0
	for deletedCount < totalCount {
		// 计算当前批次要删除的记录数
		currentBatchSize := batchSize
		if deletedCount+batchSize > totalCount {
			currentBatchSize = totalCount - deletedCount
		}

		// 执行分批删除
		result, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE id IN (SELECT id FROM %s ORDER BY id LIMIT %d)", tableName, tableName, currentBatchSize))
		if err != nil {
			return fmt.Errorf("failed to delete batch: %v", err)
		}

		// 获取实际删除的记录数
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %v", err)
		}

		deletedCount += int(affected)

		if enableProgressLog {
			progress := float64(deletedCount) / float64(totalCount) * 100
			m.logger.Info("Progress: %d/%d records deleted (%.1f%%)", deletedCount, totalCount, progress)
		}

		// 如果还有记录需要删除，则等待一段时间
		if deletedCount < totalCount && batchDelay > 0 {
			time.Sleep(batchDelay)
		}
	}

	// 重置自增ID
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM sqlite_sequence WHERE name='%s'", tableName)); err != nil {
		return fmt.Errorf("failed to reset autoincrement: %v", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	if enableProgressLog {
		m.logger.Info("Successfully cleared table %s: %d records deleted, ID reset", tableName, deletedCount)
	} else {
		m.logger.Info("Table %s cleared successfully", tableName)
	}

	return nil
}

// ExecuteSQLFile 执行外部 SQL 文件
func (m *SQLiteManager) ExecuteSQLFile(filePath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.migrator == nil {
		return fmt.Errorf("migrator not initialized")
	}

	return m.migrator.ExecuteSQLFile(filePath)
}
