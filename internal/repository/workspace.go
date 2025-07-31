package repository

import (
	"database/sql"
	"fmt"
	"time"

	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/pkg/logger"
)

// WorkspaceRepository 工作区数据访问层
type WorkspaceRepository interface {
	// CreateWorkspace 创建工作区
	CreateWorkspace(workspace *model.Workspace) error
	// GetWorkspaceByPath 根据路径获取工作区
	GetWorkspaceByPath(path string) (*model.Workspace, error)
	// GetWorkspaceByID 根据ID获取工作区
	GetWorkspaceByID(id int64) (*model.Workspace, error)
	// UpdateWorkspace 更新工作区
	UpdateWorkspace(workspace *model.Workspace) error
	// DeleteWorkspace 删除工作区
	DeleteWorkspace(path string) error
	// ListWorkspaces 列出所有工作区
	ListWorkspaces() ([]*model.Workspace, error)
	// GetActiveWorkspaces 获取活跃的工作区
	GetActiveWorkspaces() ([]*model.Workspace, error)
	// UpdateEmbeddingInfo 更新语义构建信息
	UpdateEmbeddingInfo(path string, fileNum int, timestamp int64) error
	// UpdateCodegraphInfo 更新代码构建信息
	UpdateCodegraphInfo(path string, fileNum int, timestamp int64) error
}

// workspaceRepository 工作区Repository实现
type workspaceRepository struct {
	db     database.DatabaseManager
	logger logger.Logger
}

// NewWorkspaceRepository 创建工作区Repository
func NewWorkspaceRepository(db database.DatabaseManager, logger logger.Logger) WorkspaceRepository {
	return &workspaceRepository{
		db:     db,
		logger: logger,
	}
}

// CreateWorkspace 创建工作区
func (r *workspaceRepository) CreateWorkspace(workspace *model.Workspace) error {
	query := `
		INSERT INTO workspaces (workspace_name, workspace_path, active, file_num, 
			embedding_file_num, embedding_ts, codegraph_file_num, codegraph_ts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.GetDB().Exec(query,
		workspace.WorkspaceName,
		workspace.WorkspacePath,
		workspace.Active,
		workspace.FileNum,
		workspace.EmbeddingFileNum,
		workspace.EmbeddingTs,
		workspace.CodegraphFileNum,
		workspace.CodegraphTs,
	)
	if err != nil {
		r.logger.Error("Failed to create workspace: %v", err)
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("Failed to get last insert ID: %v", err)
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	workspace.ID = id
	return nil
}

// GetWorkspaceByPath 根据路径获取工作区
func (r *workspaceRepository) GetWorkspaceByPath(path string) (*model.Workspace, error) {
	query := `
		SELECT id, workspace_name, workspace_path, active, file_num, 
			embedding_file_num, embedding_ts, codegraph_file_num, codegraph_ts, 
			created_at, updated_at
		FROM workspaces 
		WHERE workspace_path = ?
	`

	row := r.db.GetDB().QueryRow(query, path)

	var workspace model.Workspace
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&workspace.ID,
		&workspace.WorkspaceName,
		&workspace.WorkspacePath,
		&workspace.Active,
		&workspace.FileNum,
		&workspace.EmbeddingFileNum,
		&workspace.EmbeddingTs,
		&workspace.CodegraphFileNum,
		&workspace.CodegraphTs,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workspace not found: %s", path)
		}
		r.logger.Error("Failed to get workspace by path: %v", err)
		return nil, fmt.Errorf("failed to get workspace by path: %w", err)
	}

	workspace.CreatedAt = createdAt
	workspace.UpdatedAt = updatedAt

	return &workspace, nil
}

// GetWorkspaceByID 根据ID获取工作区
func (r *workspaceRepository) GetWorkspaceByID(id int64) (*model.Workspace, error) {
	query := `
		SELECT id, workspace_name, workspace_path, active, file_num, 
			embedding_file_num, embedding_ts, codegraph_file_num, codegraph_ts, 
			created_at, updated_at
		FROM workspaces 
		WHERE id = ?
	`

	row := r.db.GetDB().QueryRow(query, id)

	var workspace model.Workspace
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&workspace.ID,
		&workspace.WorkspaceName,
		&workspace.WorkspacePath,
		&workspace.Active,
		&workspace.FileNum,
		&workspace.EmbeddingFileNum,
		&workspace.EmbeddingTs,
		&workspace.CodegraphFileNum,
		&workspace.CodegraphTs,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workspace not found: %d", id)
		}
		r.logger.Error("Failed to get workspace by ID: %v", err)
		return nil, fmt.Errorf("failed to get workspace by ID: %w", err)
	}

	workspace.CreatedAt = createdAt
	workspace.UpdatedAt = updatedAt

	return &workspace, nil
}

// UpdateWorkspace 更新工作区
func (r *workspaceRepository) UpdateWorkspace(workspace *model.Workspace) error {
	query := `
		UPDATE workspaces 
		SET workspace_name = ?, active = ?, file_num = ?, 
			embedding_file_num = ?, embedding_ts = ?, 
			codegraph_file_num = ?, codegraph_ts = ?, updated_at = CURRENT_TIMESTAMP
		WHERE workspace_path = ?
	`

	result, err := r.db.GetDB().Exec(query,
		workspace.WorkspaceName,
		workspace.Active,
		workspace.FileNum,
		workspace.EmbeddingFileNum,
		workspace.EmbeddingTs,
		workspace.CodegraphFileNum,
		workspace.CodegraphTs,
		workspace.WorkspacePath,
	)
	if err != nil {
		r.logger.Error("Failed to update workspace: %v", err)
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workspace not found: %s", workspace.WorkspacePath)
	}

	return nil
}

// DeleteWorkspace 删除工作区
func (r *workspaceRepository) DeleteWorkspace(path string) error {
	query := `DELETE FROM workspaces WHERE workspace_path = ?`

	result, err := r.db.GetDB().Exec(query, path)
	if err != nil {
		r.logger.Error("Failed to delete workspace: %v", err)
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workspace not found: %s", path)
	}

	return nil
}

// ListWorkspaces 列出所有工作区
func (r *workspaceRepository) ListWorkspaces() ([]*model.Workspace, error) {
	query := `
		SELECT id, workspace_name, workspace_path, active, file_num, 
			embedding_file_num, embedding_ts, codegraph_file_num, codegraph_ts, 
			created_at, updated_at
		FROM workspaces 
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query)
	if err != nil {
		r.logger.Error("Failed to list workspaces: %v", err)
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []*model.Workspace
	for rows.Next() {
		var workspace model.Workspace
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&workspace.ID,
			&workspace.WorkspaceName,
			&workspace.WorkspacePath,
			&workspace.Active,
			&workspace.FileNum,
			&workspace.EmbeddingFileNum,
			&workspace.EmbeddingTs,
			&workspace.CodegraphFileNum,
			&workspace.CodegraphTs,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan workspace row: %v", err)
			return nil, fmt.Errorf("failed to scan workspace row: %w", err)
		}

		workspace.CreatedAt = createdAt
		workspace.UpdatedAt = updatedAt
		workspaces = append(workspaces, &workspace)
	}

	return workspaces, nil
}

// GetActiveWorkspaces 获取活跃的工作区
func (r *workspaceRepository) GetActiveWorkspaces() ([]*model.Workspace, error) {
	query := `
		SELECT id, workspace_name, workspace_path, active, file_num, 
			embedding_file_num, embedding_ts, codegraph_file_num, codegraph_ts, 
			created_at, updated_at
		FROM workspaces 
		WHERE active = 1
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query)
	if err != nil {
		r.logger.Error("Failed to get active workspaces: %v", err)
		return nil, fmt.Errorf("failed to get active workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []*model.Workspace
	for rows.Next() {
		var workspace model.Workspace
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&workspace.ID,
			&workspace.WorkspaceName,
			&workspace.WorkspacePath,
			&workspace.Active,
			&workspace.FileNum,
			&workspace.EmbeddingFileNum,
			&workspace.EmbeddingTs,
			&workspace.CodegraphFileNum,
			&workspace.CodegraphTs,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan workspace row: %v", err)
			return nil, fmt.Errorf("failed to scan workspace row: %w", err)
		}

		workspace.CreatedAt = createdAt
		workspace.UpdatedAt = updatedAt
		workspaces = append(workspaces, &workspace)
	}

	return workspaces, nil
}

// UpdateEmbeddingInfo 更新语义构建信息
func (r *workspaceRepository) UpdateEmbeddingInfo(path string, fileNum int, timestamp int64) error {
	query := `
		UPDATE workspaces 
		SET embedding_file_num = ?, embedding_ts = ?, updated_at = CURRENT_TIMESTAMP
		WHERE workspace_path = ?
	`

	result, err := r.db.GetDB().Exec(query, fileNum, timestamp, path)
	if err != nil {
		r.logger.Error("Failed to update embedding info: %v", err)
		return fmt.Errorf("failed to update embedding info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workspace not found: %s", path)
	}

	return nil
}

// UpdateCodegraphInfo 更新代码构建信息
func (r *workspaceRepository) UpdateCodegraphInfo(path string, fileNum int, timestamp int64) error {
	query := `
		UPDATE workspaces 
		SET codegraph_file_num = ?, codegraph_ts = ?, updated_at = CURRENT_TIMESTAMP
		WHERE workspace_path = ?
	`

	result, err := r.db.GetDB().Exec(query, fileNum, timestamp, path)
	if err != nil {
		r.logger.Error("Failed to update codegraph info: %v", err)
		return fmt.Errorf("failed to update codegraph info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workspace not found: %s", path)
	}

	return nil
}
