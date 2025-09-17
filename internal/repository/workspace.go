package repository

import (
	"database/sql"
	"fmt"
	"strings"
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
	// UpdateWorkspaceByMap 根据map更新工作区
	UpdateWorkspaceByMap(path string, updates map[string]interface{}) error
	// DeleteWorkspace 删除工作区
	DeleteWorkspace(path string) error
	// ListWorkspaces 列出所有工作区
	ListWorkspaces() ([]*model.Workspace, error)
	// GetActiveWorkspaces 获取活跃的工作区
	GetActiveWorkspaces() ([]*model.Workspace, error)
	// UpdateEmbeddingInfo 更新语义构建信息
	UpdateEmbeddingInfo(path string, fileNum int, timestamp int64, message, failedFilePaths string) error
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
			embedding_file_num, embedding_ts, codegraph_file_num, codegraph_ts,
			deepwiki_file_num, deepwiki_ts, deepwiki_message, deepwiki_failed_file_paths)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		workspace.DeepwikiFileNum,
		workspace.DeepwikiTs,
		workspace.DeepwikiMessage,
		workspace.DeepwikiFailedFilePaths,
	)
	if err != nil {
		return fmt.Errorf("[DB] failed to create workspace: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("[DB] failed to get last insert ID: %w", err)
	}

	workspace.ID = id
	return nil
}

// GetWorkspaceByPath 根据路径获取工作区
func (r *workspaceRepository) GetWorkspaceByPath(path string) (*model.Workspace, error) {
	query := `
		SELECT id, workspace_name, workspace_path, active, file_num,
			embedding_file_num, embedding_ts, embedding_message, embedding_failed_file_paths,
			codegraph_file_num, codegraph_ts, codegraph_message, codegraph_failed_file_paths,
			deepwiki_file_num, deepwiki_ts, deepwiki_message, deepwiki_failed_file_paths,
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
		&workspace.EmbeddingMessage,
		&workspace.EmbeddingFailedFilePaths,
		&workspace.CodegraphFileNum,
		&workspace.CodegraphTs,
		&workspace.CodegraphMessage,
		&workspace.CodegraphFailedFilePaths,
		&workspace.DeepwikiFileNum,
		&workspace.DeepwikiTs,
		&workspace.DeepwikiMessage,
		&workspace.DeepwikiFailedFilePaths,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("[DB] workspace not found: %s", path)
		}
		return nil, fmt.Errorf("[DB] failed to get workspace by path: %w", err)
	}

	workspace.CreatedAt = createdAt
	workspace.UpdatedAt = updatedAt

	return &workspace, nil
}

// GetWorkspaceByID 根据ID获取工作区
func (r *workspaceRepository) GetWorkspaceByID(id int64) (*model.Workspace, error) {
	query := `
		SELECT id, workspace_name, workspace_path, active, file_num,
			embedding_file_num, embedding_ts, embedding_message, embedding_failed_file_paths,
			codegraph_file_num, codegraph_ts, codegraph_message, codegraph_failed_file_paths,
			deepwiki_file_num, deepwiki_ts, deepwiki_message, deepwiki_failed_file_paths,
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
		&workspace.EmbeddingMessage,
		&workspace.EmbeddingFailedFilePaths,
		&workspace.CodegraphFileNum,
		&workspace.CodegraphTs,
		&workspace.CodegraphMessage,
		&workspace.CodegraphFailedFilePaths,
		&workspace.DeepwikiFileNum,
		&workspace.DeepwikiTs,
		&workspace.DeepwikiMessage,
		&workspace.DeepwikiFailedFilePaths,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("[DB] workspace not found: %d", id)
		}
		return nil, fmt.Errorf("[DB] failed to get workspace by ID: %w", err)
	}

	workspace.CreatedAt = createdAt
	workspace.UpdatedAt = updatedAt

	return &workspace, nil
}

// UpdateWorkspace 更新工作区
func (r *workspaceRepository) UpdateWorkspace(workspace *model.Workspace) error {
	// 构建SET子句，只包含非默认值的字段
	var setClauses []string
	var args []interface{}

	// 检查workspace_name是否为非默认值
	if workspace.WorkspaceName != "" {
		setClauses = append(setClauses, "workspace_name = ?")
		args = append(args, workspace.WorkspaceName)
	}

	// 检查active是否为非默认值（bool的默认值是false）
	if workspace.Active == "true" || workspace.Active == "false" {
		setClauses = append(setClauses, "active = ?")
		args = append(args, workspace.Active)
	}

	// 检查file_num是否为非默认值
	if workspace.FileNum != 0 {
		setClauses = append(setClauses, "file_num = ?")
		args = append(args, workspace.FileNum)
	}

	// 检查embedding_file_num是否为非默认值
	if workspace.EmbeddingFileNum != 0 {
		setClauses = append(setClauses, "embedding_file_num = ?")
		args = append(args, workspace.EmbeddingFileNum)
	}

	// 检查embedding_ts是否为非默认值
	if workspace.EmbeddingTs != 0 {
		setClauses = append(setClauses, "embedding_ts = ?")
		args = append(args, workspace.EmbeddingTs)
	}

	// 检查embedding_message是否为非默认值
	if workspace.EmbeddingMessage != "" {
		setClauses = append(setClauses, "embedding_message = ?")
		args = append(args, workspace.EmbeddingMessage)
	}

	// 检查embedding_failed_file_paths是否为非默认值
	if workspace.EmbeddingFailedFilePaths != "" {
		setClauses = append(setClauses, "embedding_failed_file_paths = ?")
		args = append(args, workspace.EmbeddingFailedFilePaths)
	}

	// 检查codegraph_file_num是否为非默认值
	if workspace.CodegraphFileNum != 0 {
		setClauses = append(setClauses, "codegraph_file_num = ?")
		args = append(args, workspace.CodegraphFileNum)
	}

	// 检查codegraph_ts是否为非默认值
	if workspace.CodegraphTs != 0 {
		setClauses = append(setClauses, "codegraph_ts = ?")
		args = append(args, workspace.CodegraphTs)
	}

	// 检查codegraph_message是否为非默认值
	if workspace.CodegraphMessage != "" {
		setClauses = append(setClauses, "codegraph_message = ?")
		args = append(args, workspace.CodegraphMessage)
	}

	// 检查codegraph_failed_file_paths是否为非默认值
	if workspace.CodegraphFailedFilePaths != "" {
		setClauses = append(setClauses, "codegraph_failed_file_paths = ?")
		args = append(args, workspace.CodegraphFailedFilePaths)
	}

	// 检查deepwiki_file_num是否为非默认值
	if workspace.DeepwikiFileNum != 0 {
		setClauses = append(setClauses, "deepwiki_file_num = ?")
		args = append(args, workspace.DeepwikiFileNum)
	}

	// 检查deepwiki_ts是否为非默认值
	if workspace.DeepwikiTs != 0 {
		setClauses = append(setClauses, "deepwiki_ts = ?")
		args = append(args, workspace.DeepwikiTs)
	}

	// 检查deepwiki_message是否为非默认值
	if workspace.DeepwikiMessage != "" {
		setClauses = append(setClauses, "deepwiki_message = ?")
		args = append(args, workspace.DeepwikiMessage)
	}

	// 检查deepwiki_failed_file_paths是否为非默认值
	if workspace.DeepwikiFailedFilePaths != "" {
		setClauses = append(setClauses, "deepwiki_failed_file_paths = ?")
		args = append(args, workspace.DeepwikiFailedFilePaths)
	}

	// 如果没有需要更新的字段，直接返回
	if len(setClauses) == 0 {
		return nil
	}

	// 添加updated_at字段
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())

	// 构建完整查询
	query := fmt.Sprintf("UPDATE workspaces SET %s WHERE workspace_path = ?", strings.Join(setClauses, ", "))
	args = append(args, workspace.WorkspacePath)

	result, err := r.db.GetDB().Exec(query, args...)
	if err != nil {
		return fmt.Errorf("[DB] failed to update workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("[DB] failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("[DB] workspace not found: %s", workspace.WorkspacePath)
	}

	return nil
}

// UpdateWorkspaceByMap 根据map更新工作区
func (r *workspaceRepository) UpdateWorkspaceByMap(path string, updates map[string]interface{}) error {
	// 如果没有需要更新的字段，直接返回
	if len(updates) == 0 {
		return nil
	}

	// 构建SET子句和参数
	var setClauses []string
	var args []interface{}

	// 支持的字段映射
	fieldMapping := map[string]string{
		"workspace_name":              "workspace_name",
		"active":                      "active",
		"file_num":                    "file_num",
		"embedding_file_num":          "embedding_file_num",
		"embedding_ts":                "embedding_ts",
		"embedding_message":           "embedding_message",
		"embedding_failed_file_paths": "embedding_failed_file_paths",
		"codegraph_file_num":          "codegraph_file_num",
		"codegraph_ts":                "codegraph_ts",
		"codegraph_message":           "codegraph_message",
		"codegraph_failed_file_paths": "codegraph_failed_file_paths",
		"deepwiki_file_num":           "deepwiki_file_num",
		"deepwiki_ts":                 "deepwiki_ts",
		"deepwiki_message":            "deepwiki_message",
		"deepwiki_failed_file_paths":  "deepwiki_failed_file_paths",
	}

	// 遍历updates map，构建SET子句
	for key, value := range updates {
		if fieldName, exists := fieldMapping[key]; exists {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", fieldName))
			args = append(args, value)
		}
	}

	// 如果没有有效的字段需要更新，直接返回
	if len(setClauses) == 0 {
		return nil
	}

	// 添加updated_at字段
	setClauses = append(setClauses, "updated_at = ?")
	nowTime := time.Now()
	args = append(args, nowTime)

	// 构建完整查询
	query := fmt.Sprintf("UPDATE workspaces SET %s WHERE workspace_path = ?", strings.Join(setClauses, ", "))
	args = append(args, path)

	result, err := r.db.GetDB().Exec(query, args...)
	if err != nil {
		return fmt.Errorf("[DB] failed to update workspace by map: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("[DB] failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("[DB] workspace not found: %s", path)
	}

	return nil
}

// DeleteWorkspace 删除工作区
func (r *workspaceRepository) DeleteWorkspace(path string) error {
	query := `DELETE FROM workspaces WHERE workspace_path = ?`

	result, err := r.db.GetDB().Exec(query, path)
	if err != nil {
		return fmt.Errorf("[DB] failed to delete workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("[DB] failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("[DB] workspace not found, path: %s", path)
		return nil
	}

	return nil
}

// ListWorkspaces 列出所有工作区
func (r *workspaceRepository) ListWorkspaces() ([]*model.Workspace, error) {
	query := `
		SELECT id, workspace_name, workspace_path, active, file_num, 
			embedding_file_num, embedding_ts, embedding_message, embedding_failed_file_paths,
			codegraph_file_num, codegraph_ts, codegraph_message, codegraph_failed_file_paths,
			deepwiki_file_num, deepwiki_ts, deepwiki_message, deepwiki_failed_file_paths,
			created_at, updated_at
		FROM workspaces 
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("[DB] failed to list workspaces: %w", err)
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
			&workspace.EmbeddingMessage,
			&workspace.EmbeddingFailedFilePaths,
			&workspace.CodegraphFileNum,
			&workspace.CodegraphTs,
			&workspace.CodegraphMessage,
			&workspace.CodegraphFailedFilePaths,
			&workspace.DeepwikiFileNum,
			&workspace.DeepwikiTs,
			&workspace.DeepwikiMessage,
			&workspace.DeepwikiFailedFilePaths,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("[DB] failed to scan workspaces table row: %w", err)
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
			embedding_file_num, embedding_ts, embedding_message, embedding_failed_file_paths,
			codegraph_file_num, codegraph_ts, codegraph_message, codegraph_failed_file_paths,
			deepwiki_file_num, deepwiki_ts, deepwiki_message, deepwiki_failed_file_paths,
			created_at, updated_at
		FROM workspaces
		WHERE active = "true"
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("[DB] failed to get active workspaces: %w", err)
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
			&workspace.EmbeddingMessage,
			&workspace.EmbeddingFailedFilePaths,
			&workspace.CodegraphFileNum,
			&workspace.CodegraphTs,
			&workspace.CodegraphMessage,
			&workspace.CodegraphFailedFilePaths,
			&workspace.DeepwikiFileNum,
			&workspace.DeepwikiTs,
			&workspace.DeepwikiMessage,
			&workspace.DeepwikiFailedFilePaths,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("[DB] failed to scan workspaces table row: %w", err)
		}

		workspace.CreatedAt = createdAt
		workspace.UpdatedAt = updatedAt
		workspaces = append(workspaces, &workspace)
	}

	return workspaces, nil
}

// UpdateEmbeddingInfo 更新语义构建信息
func (r *workspaceRepository) UpdateEmbeddingInfo(path string, fileNum int, timestamp int64, message, failedFilePaths string) error {
	query := `
		UPDATE workspaces 
		SET embedding_file_num = ?, embedding_ts = ?, embedding_message = ?, embedding_failed_file_paths = ?, updated_at = ?
		WHERE workspace_path = ?
	`

	result, err := r.db.GetDB().Exec(query, fileNum, timestamp, message, failedFilePaths, time.Now(), path)
	if err != nil {
		return fmt.Errorf("[DB] failed to update embedding info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("[DB] failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("[DB] workspace not found: %s", path)
	}

	return nil
}

// UpdateCodegraphInfo 更新代码构建信息
func (r *workspaceRepository) UpdateCodegraphInfo(path string, fileNum int, timestamp int64) error {
	query := `
		UPDATE workspaces 
		SET codegraph_file_num = ?, codegraph_ts = ?, updated_at = ?
		WHERE workspace_path = ?
	`

	result, err := r.db.GetDB().Exec(query, fileNum, timestamp, time.Now(), path)
	if err != nil {
		return fmt.Errorf("[DB] failed to update codegraph info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("[DB] failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("[DB] workspace not found: %s", path)
	}

	return nil
}
