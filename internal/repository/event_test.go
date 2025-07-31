package repository

import (
	"os"
	"testing"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/test/mocks"

	_ "github.com/mattn/go-sqlite3" // SQLite3驱动
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEventDB(t *testing.T) (database.DatabaseManager, func()) {
	// 创建临时目录用于测试数据库
	tempDir, err := os.MkdirTemp("", "test-event-db")
	require.NoError(t, err)

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建数据库配置
	dbConfig := &config.DatabaseConfig{
		DataDir:         tempDir,
		DatabaseName:    "test-event.db",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	// 创建数据库管理器
	dbManager := database.NewSQLiteManager(dbConfig, logger)
	err = dbManager.Initialize()
	require.NoError(t, err)

	cleanup := func() {
		dbManager.Close()
		os.RemoveAll(tempDir)
	}

	return dbManager, cleanup
}

func TestEventRepository(t *testing.T) {
	dbManager, cleanup := setupTestEventDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建事件Repository
	eventRepo := NewEventRepository(dbManager, logger)

	t.Run("CreateEvent", func(t *testing.T) {
		event := &model.Event{
			WorkspacePath:  "/path/to/workspace",
			EventType:      "file_created",
			SourceFilePath: "/path/to/source/file",
			TargetFilePath: "/path/to/target/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := eventRepo.CreateEvent(event)
		require.NoError(t, err)
		assert.NotZero(t, event.ID)
	})

	t.Run("GetEventByID", func(t *testing.T) {
		// 先创建一个事件
		event := &model.Event{
			WorkspacePath:  "/path/to/workspace",
			EventType:      "file_updated",
			SourceFilePath: "/path/to/source/file",
			TargetFilePath: "/path/to/target/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := eventRepo.CreateEvent(event)
		require.NoError(t, err)

		// 通过ID获取事件
		retrieved, err := eventRepo.GetEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, event.ID, retrieved.ID)
		assert.Equal(t, event.WorkspacePath, retrieved.WorkspacePath)
		assert.Equal(t, event.EventType, retrieved.EventType)
		assert.Equal(t, event.SourceFilePath, retrieved.SourceFilePath)
		assert.Equal(t, event.TargetFilePath, retrieved.TargetFilePath)
	})

	t.Run("GetEventsByWorkspace", func(t *testing.T) {
		// 创建多个事件
		workspacePath := "/path/to/workspace-events"
		for i := 0; i < 3; i++ {
			event := &model.Event{
				WorkspacePath:  workspacePath,
				EventType:      "file_created",
				SourceFilePath: "/path/to/source/file" + string(rune('0'+i)),
				TargetFilePath: "/path/to/target/file" + string(rune('0'+i)),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			err := eventRepo.CreateEvent(event)
			require.NoError(t, err)
		}

		// 获取工作区的所有事件
		events, err := eventRepo.GetEventsByWorkspace(workspacePath)
		require.NoError(t, err)
		assert.True(t, len(events) >= 3)

		// 验证所有事件都属于指定工作区
		for _, event := range events {
			assert.Equal(t, workspacePath, event.WorkspacePath)
		}
	})

	t.Run("GetEventsByType", func(t *testing.T) {
		// 创建多个事件
		eventType := "file_deleted"
		for i := 0; i < 3; i++ {
			event := &model.Event{
				WorkspacePath:  "/path/to/workspace-type",
				EventType:      eventType,
				SourceFilePath: "/path/to/source/file" + string(rune('0'+i)),
				TargetFilePath: "/path/to/target/file" + string(rune('0'+i)),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			err := eventRepo.CreateEvent(event)
			require.NoError(t, err)
		}

		// 创建不同类型的事件
		event := &model.Event{
			WorkspacePath:  "/path/to/workspace-type",
			EventType:      "file_created",
			SourceFilePath: "/path/to/different/file",
			TargetFilePath: "/path/to/different/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := eventRepo.CreateEvent(event)
		require.NoError(t, err)

		// 获取指定类型的事件
		events, err := eventRepo.GetEventsByType(eventType)
		require.NoError(t, err)
		assert.True(t, len(events) >= 3)

		// 验证所有事件都是指定类型
		for _, event := range events {
			assert.Equal(t, eventType, event.EventType)
		}
	})

	t.Run("GetEventsByWorkspaceAndType", func(t *testing.T) {
		// 创建多个事件
		workspacePath := "/path/to/workspace-both"
		eventType := "file_modified"
		for i := 0; i < 3; i++ {
			event := &model.Event{
				WorkspacePath:  workspacePath,
				EventType:      eventType,
				SourceFilePath: "/path/to/source/file" + string(rune('0'+i)),
				TargetFilePath: "/path/to/target/file" + string(rune('0'+i)),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			err := eventRepo.CreateEvent(event)
			require.NoError(t, err)
		}

		// 创建不同工作区和类型的事件
		event := &model.Event{
			WorkspacePath:  "/path/to/different-workspace",
			EventType:      eventType,
			SourceFilePath: "/path/to/different/file",
			TargetFilePath: "/path/to/different/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := eventRepo.CreateEvent(event)
		require.NoError(t, err)

		event = &model.Event{
			WorkspacePath:  workspacePath,
			EventType:      "file_created",
			SourceFilePath: "/path/to/different/file",
			TargetFilePath: "/path/to/different/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err = eventRepo.CreateEvent(event)
		require.NoError(t, err)

		// 获取指定工作区和类型的事件
		events, err := eventRepo.GetEventsByWorkspaceAndType(workspacePath, eventType)
		require.NoError(t, err)
		assert.True(t, len(events) >= 3)

		// 验证所有事件都符合指定工作区和类型
		for _, event := range events {
			assert.Equal(t, workspacePath, event.WorkspacePath)
			assert.Equal(t, eventType, event.EventType)
		}
	})

	t.Run("UpdateEvent", func(t *testing.T) {
		// 先创建一个事件
		event := &model.Event{
			WorkspacePath:  "/path/to/workspace",
			EventType:      "file_created",
			SourceFilePath: "/path/to/source/file",
			TargetFilePath: "/path/to/target/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := eventRepo.CreateEvent(event)
		require.NoError(t, err)

		// 更新事件
		event.EventType = "file_updated"
		event.SourceFilePath = "/path/to/new/source/file"
		event.TargetFilePath = "/path/to/new/target/file"

		err = eventRepo.UpdateEvent(event)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := eventRepo.GetEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, "file_updated", retrieved.EventType)
		assert.Equal(t, "/path/to/new/source/file", retrieved.SourceFilePath)
		assert.Equal(t, "/path/to/new/target/file", retrieved.TargetFilePath)
	})

	t.Run("DeleteEvent", func(t *testing.T) {
		// 先创建一个事件
		event := &model.Event{
			WorkspacePath:  "/path/to/workspace",
			EventType:      "file_deleted",
			SourceFilePath: "/path/to/source/file",
			TargetFilePath: "/path/to/target/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := eventRepo.CreateEvent(event)
		require.NoError(t, err)

		// 删除事件
		err = eventRepo.DeleteEvent(event.ID)
		require.NoError(t, err)

		// 验证删除
		retrieved, err := eventRepo.GetEventByID(event.ID)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("GetRecentEvents", func(t *testing.T) {
		// 创建多个事件
		workspacePath := "/path/to/workspace-recent"
		for i := 0; i < 5; i++ {
			event := &model.Event{
				WorkspacePath:  workspacePath,
				EventType:      "file_created",
				SourceFilePath: "/path/to/source/file" + string(rune('0'+i)),
				TargetFilePath: "/path/to/target/file" + string(rune('0'+i)),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			err := eventRepo.CreateEvent(event)
			require.NoError(t, err)
		}

		// 获取最近的事件
		events, err := eventRepo.GetRecentEvents(workspacePath, 3)
		require.NoError(t, err)
		assert.Equal(t, 3, len(events))

		// 验证所有事件都属于指定工作区
		for _, event := range events {
			assert.Equal(t, workspacePath, event.WorkspacePath)
		}

		// 验证事件按时间降序排列
		for i := 1; i < len(events); i++ {
			assert.True(t, events[i-1].CreatedAt.After(events[i].CreatedAt) ||
				events[i-1].CreatedAt.Equal(events[i].CreatedAt))
		}
	})
}

func TestEventRepositoryErrorCases(t *testing.T) {
	dbManager, cleanup := setupTestEventDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建事件Repository
	eventRepo := NewEventRepository(dbManager, logger)

	t.Run("GetEventByIDNotFound", func(t *testing.T) {
		// 获取不存在的事件
		retrieved, err := eventRepo.GetEventByID(999)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("GetEventsByWorkspaceNotFound", func(t *testing.T) {
		// 获取不存在工作区的事件
		events, err := eventRepo.GetEventsByWorkspace("/nonexistent/path")
		require.NoError(t, err)
		assert.Equal(t, 0, len(events))
	})

	t.Run("GetEventsByTypeNotFound", func(t *testing.T) {
		// 获取不存在类型的事件
		events, err := eventRepo.GetEventsByType("nonexistent_type")
		require.NoError(t, err)
		assert.Equal(t, 0, len(events))
	})

	t.Run("GetEventsByWorkspaceAndTypeNotFound", func(t *testing.T) {
		// 获取不存在工作区和类型的事件
		events, err := eventRepo.GetEventsByWorkspaceAndType("/nonexistent/path", "nonexistent_type")
		require.NoError(t, err)
		assert.Equal(t, 0, len(events))
	})

	t.Run("UpdateEventNotFound", func(t *testing.T) {
		// 更新不存在的事件
		event := &model.Event{
			ID:             999,
			WorkspacePath:  "/nonexistent/workspace",
			EventType:      "file_created",
			SourceFilePath: "/path/to/source/file",
			TargetFilePath: "/path/to/target/file",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := eventRepo.UpdateEvent(event)
		assert.Error(t, err)
	})

	t.Run("DeleteEventNotFound", func(t *testing.T) {
		// 删除不存在的事件
		err := eventRepo.DeleteEvent(999)
		assert.Error(t, err)
	})

	t.Run("GetRecentEventsNotFound", func(t *testing.T) {
		// 获取不存在工作区的最近事件
		events, err := eventRepo.GetRecentEvents("/nonexistent/path", 10)
		require.NoError(t, err)
		assert.Equal(t, 0, len(events))
	})
}
