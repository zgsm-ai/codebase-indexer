package database

import (
	"fmt"
	"os"
	"testing"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/test/mocks"

	_ "github.com/mattn/go-sqlite3" // SQLite3驱动
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSQLiteManager(t *testing.T) {
	// 创建临时目录用于测试数据库
	tempDir, err := os.MkdirTemp("", "test-db")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建数据库配置
	dbConfig := &config.DatabaseConfig{
		DataDir:         tempDir,
		DatabaseName:    "test.db",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	// 创建数据库管理器
	dbManager := NewSQLiteManager(dbConfig, logger)

	t.Run("Initialize", func(t *testing.T) {
		// 设置 mock logger 预期
		logger.On("Info", "Database initialized successfully", []interface{}(nil)).Return()

		// 测试数据库初始化
		err := dbManager.Initialize()
		require.NoError(t, err)
		assert.NotNil(t, dbManager.GetDB())

		// 验证表是否创建成功
		db := dbManager.GetDB()

		// 检查workspaces表
		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='workspaces'").Scan(&tableName)
		require.NoError(t, err)
		assert.Equal(t, "workspaces", tableName)

		// 检查events表
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='events'").Scan(&tableName)
		require.NoError(t, err)
		assert.Equal(t, "events", tableName)

		// 检查embedding_states表
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='embedding_states'").Scan(&tableName)
		require.NoError(t, err)
		assert.Equal(t, "embedding_states", tableName)

		// 检查codegraph_states表
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='codegraph_states'").Scan(&tableName)
		require.NoError(t, err)
		assert.Equal(t, "codegraph_states", tableName)
	})

	t.Run("GetDB", func(t *testing.T) {
		db := dbManager.GetDB()
		assert.NotNil(t, db)

		// 验证数据库连接是否正常
		err := db.Ping()
		require.NoError(t, err)
	})

	t.Run("BeginTransaction", func(t *testing.T) {
		tx, err := dbManager.BeginTransaction()
		require.NoError(t, err)
		assert.NotNil(t, tx)

		// 回滚事务
		err = tx.Rollback()
		require.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		// 创建新的数据库管理器用于测试关闭功能
		tempDir2, err := os.MkdirTemp("", "test-db-close")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir2)

		dbConfig2 := &config.DatabaseConfig{
			DataDir:         tempDir2,
			DatabaseName:    "test-close.db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
		}

		// 设置 mock logger 预期
		logger.On("Info", "Database initialized successfully", []interface{}(nil)).Return()

		dbManager2 := NewSQLiteManager(dbConfig2, logger).(*SQLiteManager)
		err = dbManager2.Initialize()
		require.NoError(t, err)

		// 关闭数据库连接
		err = dbManager2.Close()
		require.NoError(t, err)

		// 验证数据库连接已关闭
		db := dbManager2.GetDB()
		err = db.Ping()
		assert.Error(t, err)
	})

	t.Run("InitializeWithInvalidPath", func(t *testing.T) {
		// 测试使用无效路径初始化数据库
		invalidConfig := &config.DatabaseConfig{
			DataDir:         "/invalid/path/that/does/not/exist",
			DatabaseName:    "test.db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
		}

		// 设置 mock logger 预期（对于无效路径，可能不会调用 Info）
		logger.On("Info", "Database initialized successfully", []interface{}(nil)).Return().Maybe()
		logger.On("Error", "Failed to create table: %v", mock.Anything).Return().Maybe()

		invalidManager := NewSQLiteManager(invalidConfig, logger).(*SQLiteManager)
		err = invalidManager.Initialize()
		assert.Error(t, err)
	})
}

func TestSQLiteManagerTableCreation(t *testing.T) {
	// 创建临时目录用于测试数据库
	tempDir, err := os.MkdirTemp("", "test-db-tables")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建数据库配置
	dbConfig := &config.DatabaseConfig{
		DataDir:         tempDir,
		DatabaseName:    "test-tables.db",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	// 创建数据库管理器
	// 设置 mock logger 预期
	logger.On("Info", "Database initialized successfully", []interface{}(nil)).Return()

	dbManager := NewSQLiteManager(dbConfig, logger).(*SQLiteManager)
	err = dbManager.Initialize()
	require.NoError(t, err)

	db := dbManager.GetDB()

	t.Run("WorkspacesTableSchema", func(t *testing.T) {
		// 验证workspaces表结构
		var cid int
		var name, dtype string
		var notNull, pk int
		var dfltValue interface{} // 使用interface{}来处理可能为NULL的默认值

		rows, err := db.Query("PRAGMA table_info(workspaces)")
		require.NoError(t, err)
		defer rows.Close()

		columns := make(map[string]bool)
		for rows.Next() {
			err = rows.Scan(&cid, &name, &dtype, &notNull, &dfltValue, &pk)
			require.NoError(t, err)
			columns[name] = true
		}

		expectedColumns := []string{
			"id", "workspace_name", "workspace_path", "active", "file_num",
			"embedding_file_num", "embedding_ts", "codegraph_file_num",
			"codegraph_ts", "created_at", "updated_at",
		}

		for _, col := range expectedColumns {
			assert.True(t, columns[col], "Missing column: %s", col)
		}
	})

	t.Run("EventsTableSchema", func(t *testing.T) {
		// 验证events表结构
		rows, err := db.Query("PRAGMA table_info(events)")
		require.NoError(t, err)
		defer rows.Close()

		columns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name, dtype string
			var notNull, pk int
			var dfltValue interface{} // 使用interface{}来处理可能为NULL的默认值

			err = rows.Scan(&cid, &name, &dtype, &notNull, &dfltValue, &pk)
			require.NoError(t, err)
			columns[name] = true
		}

		expectedColumns := []string{
			"id", "workspace_path", "event_type", "source_file_path",
			"target_file_path", "created_at", "updated_at",
		}

		for _, col := range expectedColumns {
			assert.True(t, columns[col], "Missing column: %s", col)
		}
	})

	t.Run("EmbeddingStatesTableSchema", func(t *testing.T) {
		// 验证embedding_states表结构
		rows, err := db.Query("PRAGMA table_info(embedding_states)")
		require.NoError(t, err)
		defer rows.Close()

		columns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name, dtype string
			var notNull, pk int
			var dfltValue interface{} // 使用interface{}来处理可能为NULL的默认值

			err = rows.Scan(&cid, &name, &dtype, &notNull, &dfltValue, &pk)
			require.NoError(t, err)
			columns[name] = true
		}

		expectedColumns := []string{
			"sync_id", "workspace_path", "file_path", "status",
			"message", "created_at", "updated_at",
		}

		for _, col := range expectedColumns {
			assert.True(t, columns[col], "Missing column: %s", col)
		}
	})

	t.Run("CodegraphStatesTableSchema", func(t *testing.T) {
		// 验证codegraph_states表结构
		rows, err := db.Query("PRAGMA table_info(codegraph_states)")
		require.NoError(t, err)
		defer rows.Close()

		columns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name, dtype string
			var notNull, pk int
			var dfltValue interface{} // 使用interface{}来处理可能为NULL的默认值

			err = rows.Scan(&cid, &name, &dtype, &notNull, &dfltValue, &pk)
			require.NoError(t, err)
			columns[name] = true
		}

		expectedColumns := []string{
			"workspace_path", "file_path", "status",
			"message", "created_at", "updated_at",
		}

		for _, col := range expectedColumns {
			assert.True(t, columns[col], "Missing column: %s", col)
		}
	})
}

func TestSQLiteManagerConcurrency(t *testing.T) {
	// 创建临时目录用于测试数据库
	tempDir, err := os.MkdirTemp("", "test-db-concurrency")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建数据库配置
	dbConfig := &config.DatabaseConfig{
		DataDir:         tempDir,
		DatabaseName:    "test-concurrency.db",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	// 创建数据库管理器
	// 设置 mock logger 预期
	logger.On("Info", "Database initialized successfully", []interface{}(nil)).Return()

	dbManager := NewSQLiteManager(dbConfig, logger).(*SQLiteManager)
	err = dbManager.Initialize()
	require.NoError(t, err)

	t.Run("ConcurrentGetDB", func(t *testing.T) {
		// 测试并发获取数据库连接
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				db := dbManager.GetDB()
				assert.NotNil(t, db)
				err := db.Ping()
				assert.NoError(t, err)
				done <- true
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("ConcurrentTransactions", func(t *testing.T) {
		// 测试并发事务
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(id int) {
				tx, err := dbManager.BeginTransaction()
				require.NoError(t, err)

				// 执行简单的插入操作
				_, err = tx.Exec("INSERT INTO workspaces (workspace_name, workspace_path) VALUES (?, ?)",
					fmt.Sprintf("test_workspace_%d", id), fmt.Sprintf("/test/path/%d", id))
				require.NoError(t, err)

				err = tx.Commit()
				require.NoError(t, err)

				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < 5; i++ {
			<-done
		}

		// 验证所有插入都成功
		db := dbManager.GetDB()
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM workspaces WHERE workspace_name LIKE 'test_workspace_%'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})
}
