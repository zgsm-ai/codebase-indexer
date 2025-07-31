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

func setupTestEmbeddingStateDB(t *testing.T) (database.DatabaseManager, func()) {
	// 创建临时目录用于测试数据库
	tempDir, err := os.MkdirTemp("", "test-embedding-state-db")
	require.NoError(t, err)

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建数据库配置
	dbConfig := &config.DatabaseConfig{
		DataDir:         tempDir,
		DatabaseName:    "test-embedding-state.db",
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

func TestEmbeddingStateRepository(t *testing.T) {
	dbManager, cleanup := setupTestEmbeddingStateDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建工作区Repository
	workspaceRepo := NewWorkspaceRepository(dbManager, logger)

	// 创建嵌入状态Repository
	embeddingStateRepo := NewEmbeddingStateRepository(dbManager, logger)

	t.Run("CreateEmbeddingState", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace",
			WorkspacePath:    "/path/to/workspace",
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		state := &model.EmbeddingState{
			SyncID:        "test-sync-id",
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file",
			Status:        model.EmbeddingStatusSuccess,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)
		assert.NotZero(t, state.SyncID)
	})

	t.Run("GetEmbeddingStateById", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-get",
			WorkspacePath:    "/path/to/workspace",
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		// 先创建一个嵌入状态
		state := &model.EmbeddingState{
			SyncID:        "test-sync-id-get",
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file1",
			Status:        model.EmbeddingStatusBuilding,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)

		// 通过ID获取嵌入状态
		retrieved, err := embeddingStateRepo.GetEmbeddingStateBySyncID(state.SyncID)
		require.NoError(t, err)
		assert.Equal(t, state.SyncID, retrieved.SyncID)
		assert.Equal(t, state.WorkspacePath, retrieved.WorkspacePath)
		assert.Equal(t, state.FilePath, retrieved.FilePath)
	})

	t.Run("GetEmbeddingStateByFile", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-file",
			WorkspacePath:    "/path/to/workspace",
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		// 先创建一个嵌入状态
		state := &model.EmbeddingState{
			SyncID:        "test-sync-id-file",
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file2",
			Status:        model.EmbeddingStatusSuccess,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)

		// 通过文件路径获取嵌入状态
		retrieved, err := embeddingStateRepo.GetEmbeddingStateByFile(state.WorkspacePath, state.FilePath)
		require.NoError(t, err)
		assert.Equal(t, state.SyncID, retrieved.SyncID)
		assert.Equal(t, state.WorkspacePath, retrieved.WorkspacePath)
		assert.Equal(t, state.FilePath, retrieved.FilePath)
		assert.Equal(t, state.Status, retrieved.Status)
		assert.Equal(t, state.Message, retrieved.Message)
	})

	t.Run("GetEmbeddingStatesByWorkspace", func(t *testing.T) {
		// 创建多个嵌入状态
		workspacePath := "/path/to/workspace-states"

		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-states",
			WorkspacePath:    workspacePath,
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			state := &model.EmbeddingState{
				SyncID:        "test-sync-id-" + string(rune(i)),
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file" + string(rune(i)),
				Status:        model.EmbeddingStatusSuccess,
				Message:       "test message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = embeddingStateRepo.CreateEmbeddingState(state)
			require.NoError(t, err)
		}

		// 获取工作区的所有嵌入状态
		states, err := embeddingStateRepo.GetEmbeddingStatesByWorkspace(workspacePath)
		require.NoError(t, err)
		assert.True(t, len(states) >= 3)

		// 验证所有状态都属于指定工作区
		for _, state := range states {
			assert.Equal(t, workspacePath, state.WorkspacePath)
		}
	})

	t.Run("GetEmbeddingStatesByStatus", func(t *testing.T) {
		// 创建多个嵌入状态
		workspacePath := "/path/to/workspace-status"

		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-status",
			WorkspacePath:    workspacePath,
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			state := &model.EmbeddingState{
				SyncID:        "test-sync-id-" + string(rune(i)),
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file" + string(rune(i)),
				Status:        model.EmbeddingStatusSuccess,
				Message:       "test message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = embeddingStateRepo.CreateEmbeddingState(state)
			require.NoError(t, err)
		}

		// 创建不同状态的文件
		state := &model.EmbeddingState{
			SyncID:        "test-sync-id-different",
			WorkspacePath: workspacePath,
			FilePath:      "/path/to/different-file",
			Status:        model.EmbeddingStatusBuilding,
			Message:       "building message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)

		// 获取指定状态的文件
		states, err := embeddingStateRepo.GetEmbeddingStatesByStatus(model.EmbeddingStatusSuccess)
		require.NoError(t, err)
		assert.True(t, len(states) >= 3)

		// 验证所有状态都是指定状态
		for _, state := range states {
			assert.Equal(t, model.EmbeddingStatusSuccess, state.Status)
		}
	})

	t.Run("GetPendingEmbeddingStates", func(t *testing.T) {
		// 创建多个嵌入状态
		workspacePath := "/path/to/workspace-pending"

		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-pending",
			WorkspacePath:    workspacePath,
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			state := &model.EmbeddingState{
				SyncID:        "test-sync-id-pending-" + string(rune(i)),
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file" + string(rune(i)),
				Status:        model.EmbeddingStatusBuilding,
				Message:       "building message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = embeddingStateRepo.CreateEmbeddingState(state)
			require.NoError(t, err)
		}

		// 创建非待处理状态的文件
		state := &model.EmbeddingState{
			SyncID:        "test-sync-id-success",
			WorkspacePath: workspacePath,
			FilePath:      "/path/to/success-file",
			Status:        model.EmbeddingStatusSuccess,
			Message:       "success message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)

		// 获取待处理状态的文件
		states, err := embeddingStateRepo.GetPendingEmbeddingStates(10)
		require.NoError(t, err)
		assert.True(t, len(states) >= 3)

		// 验证所有状态都是待处理状态
		for _, state := range states {
			assert.True(t, state.Status == model.EmbeddingStatusUploading ||
				state.Status == model.EmbeddingStatusBuilding ||
				state.Status == model.EmbeddingStatusUploadFailed ||
				state.Status == model.EmbeddingStatusBuildFailed)
		}
	})

	t.Run("UpdateEmbeddingState", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-update",
			WorkspacePath:    "/path/to/workspace",
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		// 先创建一个嵌入状态
		state := &model.EmbeddingState{
			SyncID:        "test-sync-id",
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file-update",
			Status:        model.EmbeddingStatusBuilding,
			Message:       "building message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)

		// 更新嵌入状态
		state.Status = model.EmbeddingStatusSuccess
		state.Message = "success message"

		err = embeddingStateRepo.UpdateEmbeddingState(state)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := embeddingStateRepo.GetEmbeddingStateBySyncID(state.SyncID)
		require.NoError(t, err)
		assert.Equal(t, model.EmbeddingStatusSuccess, retrieved.Status)
		assert.Equal(t, "success message", retrieved.Message)
	})

	t.Run("DeleteEmbeddingState", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-delete",
			WorkspacePath:    "/path/to/workspace",
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		// 先创建一个嵌入状态
		state := &model.EmbeddingState{
			SyncID:        "test-sync-id",
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file-delete",
			Status:        model.EmbeddingStatusSuccess,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)

		// 删除嵌入状态
		err = embeddingStateRepo.DeleteEmbeddingState(state.SyncID)
		require.NoError(t, err)

		// 验证删除
		retrieved, err := embeddingStateRepo.GetEmbeddingStateBySyncID(state.SyncID)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("DeleteEmbeddingStatesByWorkspace", func(t *testing.T) {
		// 创建多个嵌入状态
		workspacePath := "/path/to/workspace-delete-all"

		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-delete-all",
			WorkspacePath:    workspacePath,
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			state := &model.EmbeddingState{
				SyncID:        "test-sync-id-delete-" + string(rune(i)),
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file-delete-all" + string(rune(i)),
				Status:        model.EmbeddingStatusSuccess,
				Message:       "test message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = embeddingStateRepo.CreateEmbeddingState(state)
			require.NoError(t, err)
		}

		// 删除工作区的所有嵌入状态
		err = embeddingStateRepo.DeleteEmbeddingStatesByWorkspace(workspacePath)
		require.NoError(t, err)

		// 验证删除
		states, err := embeddingStateRepo.GetEmbeddingStatesByWorkspace(workspacePath)
		require.NoError(t, err)
		assert.Equal(t, 0, len(states))
	})

	t.Run("UpdateEmbeddingStateStatus", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-update-status",
			WorkspacePath:    "/path/to/workspace",
			Active:           true,
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := workspaceRepo.CreateWorkspace(workspace)
		require.NoError(t, err)

		// 先创建一个嵌入状态
		state := &model.EmbeddingState{
			SyncID:        "test-sync-id",
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file-update-status",
			Status:        model.EmbeddingStatusBuilding,
			Message:       "building message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = embeddingStateRepo.CreateEmbeddingState(state)
		require.NoError(t, err)

		// 更新嵌入状态
		newStatus := model.EmbeddingStatusSuccess
		newMessage := "success message"
		err = embeddingStateRepo.UpdateEmbeddingStateStatus(state.SyncID, newStatus, newMessage)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := embeddingStateRepo.GetEmbeddingStateBySyncID(state.SyncID)
		require.NoError(t, err)
		assert.Equal(t, newStatus, retrieved.Status)
		assert.Equal(t, newMessage, retrieved.Message)
	})
}

func TestEmbeddingStateRepositoryErrorCases(t *testing.T) {
	dbManager, cleanup := setupTestEmbeddingStateDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建嵌入状态Repository
	embeddingStateRepo := NewEmbeddingStateRepository(dbManager, logger)

	t.Run("GetEmbeddingStateBySyncIDNotFound", func(t *testing.T) {
		// 获取不存在的嵌入状态
		retrieved, err := embeddingStateRepo.GetEmbeddingStateBySyncID("nonexistent-sync-id")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("GetEmbeddingStateByFileNotFound", func(t *testing.T) {
		// 获取不存在的文件路径的嵌入状态
		retrieved, err := embeddingStateRepo.GetEmbeddingStateByFile("/nonexistent/workspace", "/nonexistent/file")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("GetEmbeddingStatesByWorkspaceNotFound", func(t *testing.T) {
		// 获取不存在工作区的嵌入状态
		states, err := embeddingStateRepo.GetEmbeddingStatesByWorkspace("/nonexistent/path")
		require.NoError(t, err)
		assert.Equal(t, 0, len(states))
	})

	t.Run("GetEmbeddingStatesByStatusNotFound", func(t *testing.T) {
		// 获取不存在状态的嵌入状态
		states, err := embeddingStateRepo.GetEmbeddingStatesByStatus(999)
		require.NoError(t, err)
		assert.Equal(t, 0, len(states))
	})

	t.Run("UpdateEmbeddingStateNotFound", func(t *testing.T) {
		// 更新不存在的嵌入状态
		state := &model.EmbeddingState{
			SyncID:        "nonexistent-sync-id",
			WorkspacePath: "/nonexistent/workspace",
			FilePath:      "/nonexistent/file",
			Status:        model.EmbeddingStatusSuccess,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := embeddingStateRepo.UpdateEmbeddingState(state)
		assert.Error(t, err)
	})

	t.Run("DeleteEmbeddingStateNotFound", func(t *testing.T) {
		// 删除不存在的嵌入状态
		err := embeddingStateRepo.DeleteEmbeddingState("nonexistent-sync-id")
		assert.Error(t, err)
	})

	t.Run("DeleteEmbeddingStatesByWorkspaceNotFound", func(t *testing.T) {
		// 删除不存在工作区的所有嵌入状态
		err := embeddingStateRepo.DeleteEmbeddingStatesByWorkspace("/nonexistent/path")
		require.NoError(t, err) // 这个操作即使工作区不存在也应该成功
	})

	t.Run("UpdateEmbeddingStateStatusNotFound", func(t *testing.T) {
		// 更新不存在的嵌入状态
		err := embeddingStateRepo.UpdateEmbeddingStateStatus("nonexistent-sync-id", model.EmbeddingStatusSuccess, "test message")
		assert.Error(t, err)
	})
}
