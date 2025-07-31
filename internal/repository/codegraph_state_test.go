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

func setupTestCodegraphStateDB(t *testing.T) (database.DatabaseManager, func()) {
	// 创建临时目录用于测试数据库
	tempDir, err := os.MkdirTemp("", "test-codegraph-state-db")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建数据库配置
	dbConfig := &config.DatabaseConfig{
		DataDir:         tempDir,
		DatabaseName:    "test-codegraph-state.db",
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

func TestCodegraphStateRepository(t *testing.T) {
	dbManager, cleanup := setupTestCodegraphStateDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建工作区Repository
	workspaceRepo := NewWorkspaceRepository(dbManager, logger)

	// 创建代码构建状态Repository
	codegraphStateRepo := NewCodegraphStateRepository(dbManager, logger)

	t.Run("CreateCodegraphState", func(t *testing.T) {
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

		state := &model.CodegraphState{
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file",
			Status:        model.CodegraphStatusBuilding,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = codegraphStateRepo.CreateCodegraphState(state)
		require.NoError(t, err)
	})

	t.Run("GetCodegraphStateByFile", func(t *testing.T) {
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

		// 先创建一个代码构建状态
		state := &model.CodegraphState{
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file1",
			Status:        model.CodegraphStatusBuilding,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = codegraphStateRepo.CreateCodegraphState(state)
		require.NoError(t, err)

		// 通过文件路径获取代码构建状态
		retrieved, err := codegraphStateRepo.GetCodegraphStateByFile(state.WorkspacePath, state.FilePath)
		require.NoError(t, err)
		assert.Equal(t, state.WorkspacePath, retrieved.WorkspacePath)
		assert.Equal(t, state.FilePath, retrieved.FilePath)
		assert.Equal(t, state.Status, retrieved.Status)
		assert.Equal(t, state.Message, retrieved.Message)
	})

	t.Run("GetCodegraphStatesByWorkspace", func(t *testing.T) {
		// 创建多个代码构建状态
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
			state := &model.CodegraphState{
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file" + string(rune('0'+i)),
				Status:        model.CodegraphStatusBuilding,
				Message:       "test message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = codegraphStateRepo.CreateCodegraphState(state)
			require.NoError(t, err)
		}

		// 获取工作区的所有代码构建状态
		states, err := codegraphStateRepo.GetCodegraphStatesByWorkspace(workspacePath)
		require.NoError(t, err)
		assert.True(t, len(states) >= 3)

		// 验证所有状态都属于指定工作区
		for _, state := range states {
			assert.Equal(t, workspacePath, state.WorkspacePath)
		}
	})

	t.Run("GetCodegraphStatesByStatus", func(t *testing.T) {
		// 创建多个代码构建状态
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
			state := &model.CodegraphState{
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file" + string(rune('0'+i)),
				Status:        model.CodegraphStatusSuccess,
				Message:       "test message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = codegraphStateRepo.CreateCodegraphState(state)
			require.NoError(t, err)
		}

		// 创建不同状态的文件
		state := &model.CodegraphState{
			WorkspacePath: workspacePath,
			FilePath:      "/path/to/different-file",
			Status:        model.CodegraphStatusBuilding,
			Message:       "building message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = codegraphStateRepo.CreateCodegraphState(state)
		require.NoError(t, err)

		// 获取指定状态的文件
		states, err := codegraphStateRepo.GetCodegraphStatesByStatus(model.CodegraphStatusSuccess)
		require.NoError(t, err)
		assert.True(t, len(states) >= 3)

		// 验证所有状态都是指定状态
		for _, state := range states {
			assert.Equal(t, model.CodegraphStatusSuccess, state.Status)
		}
	})

	t.Run("GetPendingCodegraphStates", func(t *testing.T) {
		// 创建多个代码构建状态
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
			state := &model.CodegraphState{
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file" + string(rune('0'+i)),
				Status:        model.CodegraphStatusBuilding,
				Message:       "building message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = codegraphStateRepo.CreateCodegraphState(state)
			require.NoError(t, err)
		}

		// 创建非待处理状态的文件
		state := &model.CodegraphState{
			WorkspacePath: workspacePath,
			FilePath:      "/path/to/success-file",
			Status:        model.CodegraphStatusSuccess,
			Message:       "success message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = codegraphStateRepo.CreateCodegraphState(state)
		require.NoError(t, err)

		// 获取待处理状态的文件
		states, err := codegraphStateRepo.GetPendingCodegraphStates(10)
		require.NoError(t, err)
		assert.True(t, len(states) >= 3)

		// 验证所有状态都是待处理状态
		for _, state := range states {
			assert.Equal(t, model.CodegraphStatusBuilding, state.Status)
		}
	})

	t.Run("UpdateCodegraphState", func(t *testing.T) {
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

		// 先创建一个代码构建状态
		state := &model.CodegraphState{
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file-update",
			Status:        model.CodegraphStatusBuilding,
			Message:       "building message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = codegraphStateRepo.CreateCodegraphState(state)
		require.NoError(t, err)

		// 更新代码构建状态
		state.Status = model.CodegraphStatusSuccess
		state.Message = "success message"

		err = codegraphStateRepo.UpdateCodegraphState(state)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := codegraphStateRepo.GetCodegraphStateByFile(state.WorkspacePath, state.FilePath)
		require.NoError(t, err)
		assert.Equal(t, model.CodegraphStatusSuccess, retrieved.Status)
		assert.Equal(t, "success message", retrieved.Message)
	})

	t.Run("DeleteCodegraphState", func(t *testing.T) {
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

		// 先创建一个代码构建状态
		state := &model.CodegraphState{
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file-delete",
			Status:        model.CodegraphStatusSuccess,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = codegraphStateRepo.CreateCodegraphState(state)
		require.NoError(t, err)

		// 删除代码构建状态
		err = codegraphStateRepo.DeleteCodegraphState(state.WorkspacePath, state.FilePath)
		require.NoError(t, err)

		// 验证删除
		retrieved, err := codegraphStateRepo.GetCodegraphStateByFile(state.WorkspacePath, state.FilePath)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("DeleteCodegraphStatesByWorkspace", func(t *testing.T) {
		// 创建多个代码构建状态
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
			state := &model.CodegraphState{
				WorkspacePath: workspacePath,
				FilePath:      "/path/to/file-delete-all" + string(rune('0'+i)),
				Status:        model.CodegraphStatusSuccess,
				Message:       "test message",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err = codegraphStateRepo.CreateCodegraphState(state)
			require.NoError(t, err)
		}

		// 删除工作区的所有代码构建状态
		err = codegraphStateRepo.DeleteCodegraphStatesByWorkspace(workspacePath)
		require.NoError(t, err)

		// 验证删除
		states, err := codegraphStateRepo.GetCodegraphStatesByWorkspace(workspacePath)
		require.NoError(t, err)
		assert.Equal(t, 0, len(states))
	})

	t.Run("UpdateCodegraphStateStatus", func(t *testing.T) {
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

		// 先创建一个代码构建状态
		state := &model.CodegraphState{
			WorkspacePath: "/path/to/workspace",
			FilePath:      "/path/to/file-update-status",
			Status:        model.CodegraphStatusBuilding,
			Message:       "building message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = codegraphStateRepo.CreateCodegraphState(state)
		require.NoError(t, err)

		// 更新代码构建状态
		newStatus := model.CodegraphStatusSuccess
		newMessage := "success message"
		err = codegraphStateRepo.UpdateCodegraphStateStatus(state.WorkspacePath, state.FilePath, newStatus, newMessage)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := codegraphStateRepo.GetCodegraphStateByFile(state.WorkspacePath, state.FilePath)
		require.NoError(t, err)
		assert.Equal(t, newStatus, retrieved.Status)
		assert.Equal(t, newMessage, retrieved.Message)
	})
}

func TestCodegraphStateRepositoryErrorCases(t *testing.T) {
	dbManager, cleanup := setupTestCodegraphStateDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建代码构建状态Repository
	codegraphStateRepo := NewCodegraphStateRepository(dbManager, logger)

	t.Run("GetCodegraphStateByFileNotFound", func(t *testing.T) {
		// 获取不存在的文件路径的代码构建状态
		retrieved, err := codegraphStateRepo.GetCodegraphStateByFile("/nonexistent/workspace", "/nonexistent/file")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("GetCodegraphStatesByWorkspaceNotFound", func(t *testing.T) {
		// 获取不存在工作区的代码构建状态
		states, err := codegraphStateRepo.GetCodegraphStatesByWorkspace("/nonexistent/path")
		require.NoError(t, err)
		assert.Equal(t, 0, len(states))
	})

	t.Run("GetCodegraphStatesByStatusNotFound", func(t *testing.T) {
		// 获取不存在状态的代码构建状态
		states, err := codegraphStateRepo.GetCodegraphStatesByStatus(999)
		require.NoError(t, err)
		assert.Equal(t, 0, len(states))
	})

	t.Run("UpdateCodegraphStateNotFound", func(t *testing.T) {
		// 更新不存在的代码构建状态
		state := &model.CodegraphState{
			WorkspacePath: "/nonexistent/workspace",
			FilePath:      "/nonexistent/file",
			Status:        model.CodegraphStatusSuccess,
			Message:       "test message",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := codegraphStateRepo.UpdateCodegraphState(state)
		assert.Error(t, err)
	})

	t.Run("DeleteCodegraphStateNotFound", func(t *testing.T) {
		// 删除不存在的代码构建状态
		err := codegraphStateRepo.DeleteCodegraphState("/nonexistent/workspace", "/nonexistent/file")
		assert.Error(t, err)
	})

	t.Run("DeleteCodegraphStatesByWorkspaceNotFound", func(t *testing.T) {
		// 删除不存在工作区的所有代码构建状态
		err := codegraphStateRepo.DeleteCodegraphStatesByWorkspace("/nonexistent/path")
		require.NoError(t, err) // 这个操作即使工作区不存在也应该成功
	})

	t.Run("UpdateCodegraphStateStatusNotFound", func(t *testing.T) {
		// 更新不存在的代码构建状态
		err := codegraphStateRepo.UpdateCodegraphStateStatus("/nonexistent/workspace", "/nonexistent/file", model.CodegraphStatusSuccess, "test message")
		assert.Error(t, err)
	})
}
