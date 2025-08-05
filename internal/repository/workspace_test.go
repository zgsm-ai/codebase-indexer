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

func setupTestWorkspaceDB(t *testing.T) (database.DatabaseManager, func()) {
	// 创建临时目录用于测试数据库
	tempDir, err := os.MkdirTemp("", "test-workspace-db")
	require.NoError(t, err)

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建数据库配置
	dbConfig := &config.DatabaseConfig{
		DataDir:         tempDir,
		DatabaseName:    "test-workspace.db",
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

func TestWorkspaceRepository(t *testing.T) {
	dbManager, cleanup := setupTestWorkspaceDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建工作区Repository
	workspaceRepo := NewWorkspaceRepository(dbManager, logger)

	t.Run("CreateWorkspace", func(t *testing.T) {
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace",
			WorkspacePath:    "/path/to/workspace",
			Active:           "true",
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
		assert.NotZero(t, workspace.ID)
	})

	t.Run("GetWorkspaceByPath", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-get",
			WorkspacePath:    "/path/to/workspace-get",
			Active:           "true",
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

		// 通过路径获取工作区
		retrieved, err := workspaceRepo.GetWorkspaceByPath(workspace.WorkspacePath)
		require.NoError(t, err)
		assert.Equal(t, workspace.ID, retrieved.ID)
		assert.Equal(t, workspace.WorkspaceName, retrieved.WorkspaceName)
		assert.Equal(t, workspace.WorkspacePath, retrieved.WorkspacePath)
		assert.Equal(t, workspace.Active, retrieved.Active)
		assert.Equal(t, workspace.FileNum, retrieved.FileNum)
		assert.Equal(t, workspace.EmbeddingFileNum, retrieved.EmbeddingFileNum)
		assert.Equal(t, workspace.EmbeddingTs, retrieved.EmbeddingTs)
		assert.Equal(t, workspace.CodegraphFileNum, retrieved.CodegraphFileNum)
		assert.Equal(t, workspace.CodegraphTs, retrieved.CodegraphTs)
	})

	t.Run("GetWorkspaceByID", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-get-id",
			WorkspacePath:    "/path/to/workspace-get-id",
			Active:           "true",
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

		// 通过ID获取工作区
		retrieved, err := workspaceRepo.GetWorkspaceByID(workspace.ID)
		require.NoError(t, err)
		assert.Equal(t, workspace.ID, retrieved.ID)
		assert.Equal(t, workspace.WorkspaceName, retrieved.WorkspaceName)
		assert.Equal(t, workspace.WorkspacePath, retrieved.WorkspacePath)
		assert.Equal(t, workspace.Active, retrieved.Active)
		assert.Equal(t, workspace.FileNum, retrieved.FileNum)
		assert.Equal(t, workspace.EmbeddingFileNum, retrieved.EmbeddingFileNum)
		assert.Equal(t, workspace.EmbeddingTs, retrieved.EmbeddingTs)
		assert.Equal(t, workspace.CodegraphFileNum, retrieved.CodegraphFileNum)
		assert.Equal(t, workspace.CodegraphTs, retrieved.CodegraphTs)
	})

	t.Run("UpdateWorkspace", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-update",
			WorkspacePath:    "/path/to/workspace-update",
			Active:           "true",
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

		// 更新工作区
		workspace.WorkspaceName = "updated-workspace"
		workspace.Active = "false"
		workspace.FileNum = 20
		workspace.EmbeddingFileNum = 10
		workspace.EmbeddingTs = time.Now().Unix()
		workspace.CodegraphFileNum = 5
		workspace.CodegraphTs = time.Now().Unix()

		err = workspaceRepo.UpdateWorkspace(workspace)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := workspaceRepo.GetWorkspaceByPath(workspace.WorkspacePath)
		require.NoError(t, err)
		assert.Equal(t, "updated-workspace", retrieved.WorkspaceName)
		assert.Equal(t, "false", retrieved.Active)
		assert.Equal(t, 20, retrieved.FileNum)
		assert.Equal(t, 10, retrieved.EmbeddingFileNum)
		assert.Equal(t, 5, retrieved.CodegraphFileNum)
	})

	t.Run("DeleteWorkspace", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-delete",
			WorkspacePath:    "/path/to/workspace-delete",
			Active:           "true",
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

		// 删除工作区
		err = workspaceRepo.DeleteWorkspace(workspace.WorkspacePath)
		require.NoError(t, err)

		// 验证删除
		retrieved, err := workspaceRepo.GetWorkspaceByPath(workspace.WorkspacePath)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("ListWorkspaces", func(t *testing.T) {
		// 创建多个工作区
		for i := 0; i < 3; i++ {
			workspace := &model.Workspace{
				WorkspaceName:    "test-workspace-list-" + string(rune('0'+i)),
				WorkspacePath:    "/path/to/workspace-list-" + string(rune('0'+i)),
				Active:           "true",
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
		}

		// 列出所有工作区
		workspaces, err := workspaceRepo.ListWorkspaces()
		require.NoError(t, err)
		assert.True(t, len(workspaces) >= 3)
	})

	t.Run("GetActiveWorkspaces", func(t *testing.T) {
		// 创建活跃和非活跃工作区
		activeWorkspace := &model.Workspace{
			WorkspaceName:    "test-workspace-active",
			WorkspacePath:    "/path/to/workspace-active",
			Active:           "true",
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		inactiveWorkspace := &model.Workspace{
			WorkspaceName:    "test-workspace-inactive",
			WorkspacePath:    "/path/to/workspace-inactive",
			Active:           "true",
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		err := workspaceRepo.CreateWorkspace(activeWorkspace)
		require.NoError(t, err)

		err = workspaceRepo.CreateWorkspace(inactiveWorkspace)
		require.NoError(t, err)

		// 获取活跃工作区
		activeWorkspaces, err := workspaceRepo.GetActiveWorkspaces()
		require.NoError(t, err)
		assert.True(t, len(activeWorkspaces) >= 1)

		// 验证所有工作区都是活跃的
		for _, workspace := range activeWorkspaces {
			assert.Equal(t, true, workspace.Active)
		}
	})

	t.Run("UpdateEmbeddingInfo", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-embedding",
			WorkspacePath:    "/path/to/workspace-embedding",
			Active:           "true",
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

		// 更新语义构建信息
		newFileNum := 15
		newTimestamp := time.Now().Unix()
		err = workspaceRepo.UpdateEmbeddingInfo(workspace.WorkspacePath, newFileNum, newTimestamp)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := workspaceRepo.GetWorkspaceByPath(workspace.WorkspacePath)
		require.NoError(t, err)
		assert.Equal(t, newFileNum, retrieved.EmbeddingFileNum)
		assert.Equal(t, newTimestamp, retrieved.EmbeddingTs)
	})

	t.Run("UpdateCodegraphInfo", func(t *testing.T) {
		// 先创建一个工作区
		workspace := &model.Workspace{
			WorkspaceName:    "test-workspace-codegraph",
			WorkspacePath:    "/path/to/workspace-codegraph",
			Active:           "true",
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

		// 更新代码构建信息
		newFileNum := 8
		newTimestamp := time.Now().Unix()
		err = workspaceRepo.UpdateCodegraphInfo(workspace.WorkspacePath, newFileNum, newTimestamp)
		require.NoError(t, err)

		// 验证更新
		retrieved, err := workspaceRepo.GetWorkspaceByPath(workspace.WorkspacePath)
		require.NoError(t, err)
		assert.Equal(t, newFileNum, retrieved.CodegraphFileNum)
		assert.Equal(t, newTimestamp, retrieved.CodegraphTs)
	})
}

func TestWorkspaceRepositoryErrorCases(t *testing.T) {
	dbManager, cleanup := setupTestWorkspaceDB(t)
	defer cleanup()

	// 创建测试日志记录器
	logger := &mocks.MockLogger{}

	// 创建工作区Repository
	workspaceRepo := NewWorkspaceRepository(dbManager, logger)

	t.Run("GetWorkspaceByPathNotFound", func(t *testing.T) {
		// 获取不存在的工作区
		retrieved, err := workspaceRepo.GetWorkspaceByPath("/nonexistent/path")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("GetWorkspaceByIDNotFound", func(t *testing.T) {
		// 获取不存在的工作区
		retrieved, err := workspaceRepo.GetWorkspaceByID(999)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("UpdateWorkspaceNotFound", func(t *testing.T) {
		// 更新不存在的工作区
		workspace := &model.Workspace{
			WorkspaceName:    "nonexistent-workspace",
			WorkspacePath:    "/nonexistent/path",
			Active:           "true",
			FileNum:          10,
			EmbeddingFileNum: 5,
			EmbeddingTs:      time.Now().Unix(),
			CodegraphFileNum: 3,
			CodegraphTs:      time.Now().Unix(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		err := workspaceRepo.UpdateWorkspace(workspace)
		assert.Error(t, err)
	})

	t.Run("DeleteWorkspaceNotFound", func(t *testing.T) {
		// 删除不存在的工作区
		err := workspaceRepo.DeleteWorkspace("/nonexistent/path")
		assert.Error(t, err)
	})

	t.Run("UpdateEmbeddingInfoNotFound", func(t *testing.T) {
		// 更新不存在的工作区的语义构建信息
		err := workspaceRepo.UpdateEmbeddingInfo("/nonexistent/path", 10, time.Now().Unix())
		assert.Error(t, err)
	})

	t.Run("UpdateCodegraphInfoNotFound", func(t *testing.T) {
		// 更新不存在的工作区的代码构建信息
		err := workspaceRepo.UpdateCodegraphInfo("/nonexistent/path", 10, time.Now().Unix())
		assert.Error(t, err)
	})
}
