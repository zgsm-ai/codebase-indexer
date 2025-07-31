package repository

import (
	"database/sql"
	"fmt"
	"time"

	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/pkg/logger"
)

// CodegraphStateRepository 代码构建状态数据访问层
type CodegraphStateRepository interface {
	// CreateCodegraphState 创建代码构建状态
	CreateCodegraphState(state *model.CodegraphState) error
	// GetCodegraphStateByFile 根据工作区路径和文件路径获取代码构建状态
	GetCodegraphStateByFile(workspacePath, filePath string) (*model.CodegraphState, error)
	// GetCodegraphStatesByWorkspace 根据工作区路径获取所有代码构建状态
	GetCodegraphStatesByWorkspace(workspacePath string) ([]*model.CodegraphState, error)
	// GetCodegraphStatesByStatus 根据状态获取代码构建状态
	GetCodegraphStatesByStatus(status int) ([]*model.CodegraphState, error)
	// UpdateCodegraphState 更新代码构建状态
	UpdateCodegraphState(state *model.CodegraphState) error
	// DeleteCodegraphState 删除代码构建状态
	DeleteCodegraphState(workspacePath, filePath string) error
	// DeleteCodegraphStatesByWorkspace 删除工作区的所有代码构建状态
	DeleteCodegraphStatesByWorkspace(workspacePath string) error
	// GetPendingCodegraphStates 获取待处理的代码构建状态
	GetPendingCodegraphStates(limit int) ([]*model.CodegraphState, error)
	// UpdateCodegraphStateStatus 更新代码构建状态
	UpdateCodegraphStateStatus(workspacePath, filePath string, status int, message string) error
}

// codegraphStateRepository 代码构建状态Repository实现
type codegraphStateRepository struct {
	db     database.DatabaseManager
	logger logger.Logger
}

// NewCodegraphStateRepository 创建代码构建状态Repository
func NewCodegraphStateRepository(db database.DatabaseManager, logger logger.Logger) CodegraphStateRepository {
	return &codegraphStateRepository{
		db:     db,
		logger: logger,
	}
}

// CreateCodegraphState 创建代码构建状态
func (r *codegraphStateRepository) CreateCodegraphState(state *model.CodegraphState) error {
	query := `
		INSERT INTO codegraph_states (workspace_path, file_path, status, message)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.GetDB().Exec(query,
		state.WorkspacePath,
		state.FilePath,
		state.Status,
		state.Message,
	)
	if err != nil {
		r.logger.Error("Failed to create codegraph state: %v", err)
		return fmt.Errorf("failed to create codegraph state: %w", err)
	}

	return nil
}

// GetCodegraphStateByFile 根据工作区路径和文件路径获取代码构建状态
func (r *codegraphStateRepository) GetCodegraphStateByFile(workspacePath, filePath string) (*model.CodegraphState, error) {
	query := `
		SELECT workspace_path, file_path, status, message, created_at, updated_at
		FROM codegraph_states 
		WHERE workspace_path = ? AND file_path = ?
	`

	row := r.db.GetDB().QueryRow(query, workspacePath, filePath)

	var state model.CodegraphState
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&state.WorkspacePath,
		&state.FilePath,
		&state.Status,
		&state.Message,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("codegraph state not found: %s/%s", workspacePath, filePath)
		}
		r.logger.Error("Failed to get codegraph state by file: %v", err)
		return nil, fmt.Errorf("failed to get codegraph state by file: %w", err)
	}

	state.CreatedAt = createdAt
	state.UpdatedAt = updatedAt

	return &state, nil
}

// GetCodegraphStatesByWorkspace 根据工作区路径获取所有代码构建状态
func (r *codegraphStateRepository) GetCodegraphStatesByWorkspace(workspacePath string) ([]*model.CodegraphState, error) {
	query := `
		SELECT workspace_path, file_path, status, message, created_at, updated_at
		FROM codegraph_states 
		WHERE workspace_path = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query, workspacePath)
	if err != nil {
		r.logger.Error("Failed to get codegraph states by workspace: %v", err)
		return nil, fmt.Errorf("failed to get codegraph states by workspace: %w", err)
	}
	defer rows.Close()

	var states []*model.CodegraphState
	for rows.Next() {
		var state model.CodegraphState
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&state.WorkspacePath,
			&state.FilePath,
			&state.Status,
			&state.Message,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan codegraph state row: %v", err)
			return nil, fmt.Errorf("failed to scan codegraph state row: %w", err)
		}

		state.CreatedAt = createdAt
		state.UpdatedAt = updatedAt
		states = append(states, &state)
	}

	return states, nil
}

// GetCodegraphStatesByStatus 根据状态获取代码构建状态
func (r *codegraphStateRepository) GetCodegraphStatesByStatus(status int) ([]*model.CodegraphState, error) {
	query := `
		SELECT workspace_path, file_path, status, message, created_at, updated_at
		FROM codegraph_states 
		WHERE status = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query, status)
	if err != nil {
		r.logger.Error("Failed to get codegraph states by status: %v", err)
		return nil, fmt.Errorf("failed to get codegraph states by status: %w", err)
	}
	defer rows.Close()

	var states []*model.CodegraphState
	for rows.Next() {
		var state model.CodegraphState
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&state.WorkspacePath,
			&state.FilePath,
			&state.Status,
			&state.Message,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan codegraph state row: %v", err)
			return nil, fmt.Errorf("failed to scan codegraph state row: %w", err)
		}

		state.CreatedAt = createdAt
		state.UpdatedAt = updatedAt
		states = append(states, &state)
	}

	return states, nil
}

// UpdateCodegraphState 更新代码构建状态
func (r *codegraphStateRepository) UpdateCodegraphState(state *model.CodegraphState) error {
	query := `
		UPDATE codegraph_states 
		SET status = ?, message = ?, updated_at = CURRENT_TIMESTAMP
		WHERE workspace_path = ? AND file_path = ?
	`

	result, err := r.db.GetDB().Exec(query,
		state.Status,
		state.Message,
		state.WorkspacePath,
		state.FilePath,
	)
	if err != nil {
		r.logger.Error("Failed to update codegraph state: %v", err)
		return fmt.Errorf("failed to update codegraph state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("codegraph state not found: %s/%s", state.WorkspacePath, state.FilePath)
	}

	return nil
}

// DeleteCodegraphState 删除代码构建状态
func (r *codegraphStateRepository) DeleteCodegraphState(workspacePath, filePath string) error {
	query := `DELETE FROM codegraph_states WHERE workspace_path = ? AND file_path = ?`

	result, err := r.db.GetDB().Exec(query, workspacePath, filePath)
	if err != nil {
		r.logger.Error("Failed to delete codegraph state: %v", err)
		return fmt.Errorf("failed to delete codegraph state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("codegraph state not found: %s/%s", workspacePath, filePath)
	}

	return nil
}

// DeleteCodegraphStatesByWorkspace 删除工作区的所有代码构建状态
func (r *codegraphStateRepository) DeleteCodegraphStatesByWorkspace(workspacePath string) error {
	query := `DELETE FROM codegraph_states WHERE workspace_path = ?`

	result, err := r.db.GetDB().Exec(query, workspacePath)
	if err != nil {
		r.logger.Error("Failed to delete codegraph states by workspace: %v", err)
		return fmt.Errorf("failed to delete codegraph states by workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Deleted %d codegraph states for workspace: %s", rowsAffected, workspacePath)
	return nil
}

// GetPendingCodegraphStates 获取待处理的代码构建状态
func (r *codegraphStateRepository) GetPendingCodegraphStates(limit int) ([]*model.CodegraphState, error) {
	query := `
		SELECT workspace_path, file_path, status, message, created_at, updated_at
		FROM codegraph_states 
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := r.db.GetDB().Query(query,
		model.CodegraphStatusBuilding,
		limit,
	)
	if err != nil {
		r.logger.Error("Failed to get pending codegraph states: %v", err)
		return nil, fmt.Errorf("failed to get pending codegraph states: %w", err)
	}
	defer rows.Close()

	var states []*model.CodegraphState
	for rows.Next() {
		var state model.CodegraphState
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&state.WorkspacePath,
			&state.FilePath,
			&state.Status,
			&state.Message,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan codegraph state row: %v", err)
			return nil, fmt.Errorf("failed to scan codegraph state row: %w", err)
		}

		state.CreatedAt = createdAt
		state.UpdatedAt = updatedAt
		states = append(states, &state)
	}

	return states, nil
}

// UpdateCodegraphStateStatus 更新代码构建状态
func (r *codegraphStateRepository) UpdateCodegraphStateStatus(workspacePath, filePath string, status int, message string) error {
	query := `
		UPDATE codegraph_states 
		SET status = ?, message = ?, updated_at = CURRENT_TIMESTAMP
		WHERE workspace_path = ? AND file_path = ?
	`

	result, err := r.db.GetDB().Exec(query, status, message, workspacePath, filePath)
	if err != nil {
		r.logger.Error("Failed to update codegraph state status: %v", err)
		return fmt.Errorf("failed to update codegraph state status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("codegraph state not found: %s/%s", workspacePath, filePath)
	}

	return nil
}
