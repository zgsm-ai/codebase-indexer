package database

import (
	"database/sql"
	"path/filepath"
	"sync"

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

// createTables 创建数据库表结构
func (m *SQLiteManager) createTables() error {
	tables := []string{
		m.createWorkspacesTable(),
		m.createEventsTable(),
		m.createEmbeddingStatesTable(),
		m.createCodegraphStatesTable(),
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
        active BOOLEAN NOT NULL DEFAULT 1,
        file_num INTEGER NOT NULL DEFAULT 0,
        embedding_file_num INTEGER NOT NULL DEFAULT 0,
        embedding_ts INTEGER NOT NULL DEFAULT 0,
        codegraph_file_num INTEGER NOT NULL DEFAULT 0,
        codegraph_ts INTEGER NOT NULL DEFAULT 0,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_workspaces_path ON workspaces(workspace_path);
    CREATE INDEX IF NOT EXISTS idx_workspaces_active ON workspaces(active);
    CREATE INDEX IF NOT EXISTS idx_workspaces_embedding_ts ON workspaces(embedding_ts);
    CREATE INDEX IF NOT EXISTS idx_workspaces_codegraph_ts ON workspaces(codegraph_ts);
    `
}

// createEventsTable 创建事件表
func (m *SQLiteManager) createEventsTable() string {
	return `
    CREATE TABLE IF NOT EXISTS events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        workspace_path VARCHAR(500) NOT NULL,
        event_type VARCHAR(100) NOT NULL,
        source_file_path VARCHAR(500),
        target_file_path VARCHAR(500),
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_events_workspace_path ON events(workspace_path);
    CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
    CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
    CREATE INDEX IF NOT EXISTS idx_events_workspace_type ON events(workspace_path, event_type);
    `
}

// createEmbeddingStatesTable 创建语义构建状态表
func (m *SQLiteManager) createEmbeddingStatesTable() string {
	return `
    CREATE TABLE IF NOT EXISTS embedding_states (
        sync_id VARCHAR(100) PRIMARY KEY,
        workspace_path VARCHAR(500) NOT NULL,
        file_path VARCHAR(500) NOT NULL,
        status TINYINT NOT NULL DEFAULT 0,
        message TEXT,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (sync_id)
    );
    
    CREATE INDEX IF NOT EXISTS idx_embedding_workspace_path ON embedding_states(workspace_path);
    CREATE INDEX IF NOT EXISTS idx_embedding_file_path ON embedding_states(file_path);
    CREATE INDEX IF NOT EXISTS idx_embedding_status ON embedding_states(status);
    CREATE INDEX IF NOT EXISTS idx_embedding_workspace_file ON embedding_states(workspace_path, file_path);
    `
}

// createCodegraphStatesTable 创建代码构建状态表
func (m *SQLiteManager) createCodegraphStatesTable() string {
	return `
    CREATE TABLE IF NOT EXISTS codegraph_states (
        workspace_path VARCHAR(500) NOT NULL,
        file_path VARCHAR(500) NOT NULL,
        status TINYINT NOT NULL DEFAULT 0,
        message TEXT,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        PRIMARY KEY (workspace_path, file_path)
    );
    
    CREATE INDEX IF NOT EXISTS idx_codegraph_status ON codegraph_states(status);
    CREATE INDEX IF NOT EXISTS idx_codegraph_created_at ON codegraph_states(created_at);
	CREATE INDEX IF NOT EXISTS idx_codegraph_updated_at ON codegraph_states(updated_at);
    `
}
