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
}

// SQLiteManager SQLite数据库管理器实现
type SQLiteManager struct {
	db     *sql.DB
	config *config.DatabaseConfig
	logger logger.Logger
	mutex  sync.RWMutex
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

	// 创建表结构
	if err := m.createTables(); err != nil {
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

// createTables 创建数据库表结构
func (m *SQLiteManager) createTables() error {
	tables := []string{
		m.createWorkspacesTable(),
		m.createEventsTable(),
	}

	for _, tableSQL := range tables {
		if _, err := m.db.Exec(tableSQL); err != nil {
			m.logger.Error("Failed to create table: %v", err)
			return err
		}
	}

	return nil
}

// createWorkspacesTable 创建工作区表
func (m *SQLiteManager) createWorkspacesTable() string {
	return `
    CREATE TABLE IF NOT EXISTS workspaces (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        workspace_name VARCHAR(255) NOT NULL,
        workspace_path VARCHAR(500) UNIQUE NOT NULL,
        active VARCHAR(10) NOT NULL DEFAULT 'true',
        file_num INTEGER NOT NULL DEFAULT 0,
        embedding_file_num INTEGER NOT NULL DEFAULT 0,
        embedding_ts INTEGER NOT NULL DEFAULT 0,
		embedding_message VARCHAR(255) NOT NULL DEFAULT '',
		embedding_failed_file_paths TEXT NOT NULL DEFAULT '',
        codegraph_file_num INTEGER NOT NULL DEFAULT 0,
        codegraph_ts INTEGER NOT NULL DEFAULT 0,
		codegraph_message VARCHAR(255) NOT NULL DEFAULT '',
		codegraph_failed_file_paths TEXT NOT NULL DEFAULT '',
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_workspaces_path ON workspaces(workspace_path);
    CREATE INDEX IF NOT EXISTS idx_workspaces_embedding_ts ON workspaces(embedding_ts);
    CREATE INDEX IF NOT EXISTS idx_workspaces_codegraph_ts ON workspaces(codegraph_ts);
    CREATE INDEX IF NOT EXISTS idx_workspaces_created_at ON workspaces(created_at);
	CREATE INDEX IF NOT EXISTS idx_workspaces_updated_at ON workspaces(updated_at);
    `
}

// createEventsTable 创建事件表
func (m *SQLiteManager) createEventsTable() string {
	return `
    CREATE TABLE IF NOT EXISTS events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        workspace_path VARCHAR(500) NOT NULL,
        event_type VARCHAR(100) NOT NULL,
        source_file_path VARCHAR(500) NOT NULL DEFAULT '',
        target_file_path VARCHAR(500) NOT NULL DEFAULT '',
		sync_id VARCHAR(100) NOT NULL DEFAULT '',
		file_hash VARCHAR(100) NOT NULL DEFAULT '',
		embedding_status TINYINT NOT NULL DEFAULT 1,
		codegraph_status TINYINT NOT NULL DEFAULT 1,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_events_workspace_path ON events(workspace_path);
    CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
	CREATE INDEX IF NOT EXISTS idx_events_sync_id ON events(sync_id);
    CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
	CREATE INDEX IF NOT EXISTS idx_events_updated_at ON events(updated_at);
    CREATE INDEX IF NOT EXISTS idx_events_workspace_type ON events(workspace_path, event_type);
    `
}
