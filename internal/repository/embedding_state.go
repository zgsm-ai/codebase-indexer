package repository

import (
	"database/sql"
	"fmt"
	"time"

	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/pkg/logger"
)

// EmbeddingStateRepository 语义构建状态数据访问层
type EmbeddingStateRepository interface {
	// CreateEmbeddingState 创建语义构建状态
	CreateEmbeddingState(state *model.EmbeddingState) error
	// GetEmbeddingStateBySyncID 根据同步ID获取语义构建状态
	GetEmbeddingStateBySyncID(syncID string) (*model.EmbeddingState, error)
	// GetEmbeddingStateByFile 根据工作区路径和文件路径获取语义构建状态
	GetEmbeddingStateByFile(workspacePath, filePath string) (*model.EmbeddingState, error)
	// GetEmbeddingStatesByWorkspace 根据工作区路径获取所有语义构建状态
	GetEmbeddingStatesByWorkspace(workspacePath string) ([]*model.EmbeddingState, error)
	// GetEmbeddingStatesByStatus 根据状态获取语义构建状态
	GetEmbeddingStatesByStatus(status int) ([]*model.EmbeddingState, error)
	// UpdateEmbeddingState 更新语义构建状态
	UpdateEmbeddingState(state *model.EmbeddingState) error
	// DeleteEmbeddingState 删除语义构建状态
	DeleteEmbeddingState(syncID string) error
	// DeleteEmbeddingStatesByWorkspace 删除工作区的所有语义构建状态
	DeleteEmbeddingStatesByWorkspace(workspacePath string) error
	// GetPendingEmbeddingStates 获取待处理的语义构建状态
	GetPendingEmbeddingStates(limit int) ([]*model.EmbeddingState, error)
	// UpdateEmbeddingStateStatus 更新语义构建状态
	UpdateEmbeddingStateStatus(syncID string, status int, message string) error
}

// embeddingStateRepository 语义构建状态Repository实现
type embeddingStateRepository struct {
	db     database.DatabaseManager
	logger logger.Logger
}

// NewEmbeddingStateRepository 创建语义构建状态Repository
func NewEmbeddingStateRepository(db database.DatabaseManager, logger logger.Logger) EmbeddingStateRepository {
	return &embeddingStateRepository{
		db:     db,
		logger: logger,
	}
}

// CreateEmbeddingState 创建语义构建状态
func (r *embeddingStateRepository) CreateEmbeddingState(state *model.EmbeddingState) error {
	query := `
		INSERT INTO embedding_states (sync_id, workspace_path, file_path, status, message)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.GetDB().Exec(query,
		state.SyncID,
		state.WorkspacePath,
		state.FilePath,
		state.Status,
		state.Message,
	)
	if err != nil {
		r.logger.Error("Failed to create embedding state: %v", err)
		return fmt.Errorf("failed to create embedding state: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("Failed to get last insert ID: %v", err)
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	state.SyncID = fmt.Sprintf("%d", id) // TODO: 后续改为文件上传请求ID
	return nil
}

// GetEmbeddingStateBySyncID 根据同步ID获取语义构建状态
func (r *embeddingStateRepository) GetEmbeddingStateBySyncID(syncID string) (*model.EmbeddingState, error) {
	query := `
		SELECT sync_id, workspace_path, file_path, status, message, created_at, updated_at
		FROM embedding_states 
		WHERE sync_id = ?
	`

	row := r.db.GetDB().QueryRow(query, syncID)

	var state model.EmbeddingState
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&state.SyncID,
		&state.WorkspacePath,
		&state.FilePath,
		&state.Status,
		&state.Message,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("embedding state not found: %s", syncID)
		}
		r.logger.Error("Failed to get embedding state by sync ID: %v", err)
		return nil, fmt.Errorf("failed to get embedding state by sync ID: %w", err)
	}

	state.CreatedAt = createdAt
	state.UpdatedAt = updatedAt

	return &state, nil
}

// GetEmbeddingStateByFile 根据工作区路径和文件路径获取语义构建状态
func (r *embeddingStateRepository) GetEmbeddingStateByFile(workspacePath, filePath string) (*model.EmbeddingState, error) {
	query := `
		SELECT sync_id, workspace_path, file_path, status, message, created_at, updated_at
		FROM embedding_states 
		WHERE workspace_path = ? AND file_path = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	row := r.db.GetDB().QueryRow(query, workspacePath, filePath)

	var state model.EmbeddingState
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&state.SyncID,
		&state.WorkspacePath,
		&state.FilePath,
		&state.Status,
		&state.Message,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("embedding state not found: %s/%s", workspacePath, filePath)
		}
		r.logger.Error("Failed to get embedding state by file: %v", err)
		return nil, fmt.Errorf("failed to get embedding state by file: %w", err)
	}

	state.CreatedAt = createdAt
	state.UpdatedAt = updatedAt

	return &state, nil
}

// GetEmbeddingStatesByWorkspace 根据工作区路径获取所有语义构建状态
func (r *embeddingStateRepository) GetEmbeddingStatesByWorkspace(workspacePath string) ([]*model.EmbeddingState, error) {
	query := `
		SELECT sync_id, workspace_path, file_path, status, message, created_at, updated_at
		FROM embedding_states 
		WHERE workspace_path = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query, workspacePath)
	if err != nil {
		r.logger.Error("Failed to get embedding states by workspace: %v", err)
		return nil, fmt.Errorf("failed to get embedding states by workspace: %w", err)
	}
	defer rows.Close()

	var states []*model.EmbeddingState
	for rows.Next() {
		var state model.EmbeddingState
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&state.SyncID,
			&state.WorkspacePath,
			&state.FilePath,
			&state.Status,
			&state.Message,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan embedding state row: %v", err)
			return nil, fmt.Errorf("failed to scan embedding state row: %w", err)
		}

		state.CreatedAt = createdAt
		state.UpdatedAt = updatedAt
		states = append(states, &state)
	}

	return states, nil
}

// GetEmbeddingStatesByStatus 根据状态获取语义构建状态
func (r *embeddingStateRepository) GetEmbeddingStatesByStatus(status int) ([]*model.EmbeddingState, error) {
	query := `
		SELECT sync_id, workspace_path, file_path, status, message, created_at, updated_at
		FROM embedding_states 
		WHERE status = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().Query(query, status)
	if err != nil {
		r.logger.Error("Failed to get embedding states by status: %v", err)
		return nil, fmt.Errorf("failed to get embedding states by status: %w", err)
	}
	defer rows.Close()

	var states []*model.EmbeddingState
	for rows.Next() {
		var state model.EmbeddingState
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&state.SyncID,
			&state.WorkspacePath,
			&state.FilePath,
			&state.Status,
			&state.Message,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan embedding state row: %v", err)
			return nil, fmt.Errorf("failed to scan embedding state row: %w", err)
		}

		state.CreatedAt = createdAt
		state.UpdatedAt = updatedAt
		states = append(states, &state)
	}

	return states, nil
}

// UpdateEmbeddingState 更新语义构建状态
func (r *embeddingStateRepository) UpdateEmbeddingState(state *model.EmbeddingState) error {
	query := `
		UPDATE embedding_states 
		SET workspace_path = ?, file_path = ?, status = ?, message = ?, updated_at = CURRENT_TIMESTAMP
		WHERE sync_id = ?
	`

	result, err := r.db.GetDB().Exec(query,
		state.WorkspacePath,
		state.FilePath,
		state.Status,
		state.Message,
		state.SyncID,
	)
	if err != nil {
		r.logger.Error("Failed to update embedding state: %v", err)
		return fmt.Errorf("failed to update embedding state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("embedding state not found: %s", state.SyncID)
	}

	return nil
}

// DeleteEmbeddingState 删除语义构建状态
func (r *embeddingStateRepository) DeleteEmbeddingState(syncID string) error {
	query := `DELETE FROM embedding_states WHERE sync_id = ?`

	result, err := r.db.GetDB().Exec(query, syncID)
	if err != nil {
		r.logger.Error("Failed to delete embedding state: %v", err)
		return fmt.Errorf("failed to delete embedding state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("embedding state not found: %s", syncID)
	}

	return nil
}

// DeleteEmbeddingStatesByWorkspace 删除工作区的所有语义构建状态
func (r *embeddingStateRepository) DeleteEmbeddingStatesByWorkspace(workspacePath string) error {
	query := `DELETE FROM embedding_states WHERE workspace_path = ?`

	result, err := r.db.GetDB().Exec(query, workspacePath)
	if err != nil {
		r.logger.Error("Failed to delete embedding states by workspace: %v", err)
		return fmt.Errorf("failed to delete embedding states by workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Deleted %d embedding states for workspace: %s", rowsAffected, workspacePath)
	return nil
}

// GetPendingEmbeddingStates 获取待处理的语义构建状态
func (r *embeddingStateRepository) GetPendingEmbeddingStates(limit int) ([]*model.EmbeddingState, error) {
	query := `
		SELECT sync_id, workspace_path, file_path, status, message, created_at, updated_at
		FROM embedding_states 
		WHERE status IN (?, ?, ?, ?)
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := r.db.GetDB().Query(query,
		model.EmbeddingStatusUploading,
		model.EmbeddingStatusBuilding,
		model.EmbeddingStatusUploadFailed,
		model.EmbeddingStatusBuildFailed,
		limit,
	)
	if err != nil {
		r.logger.Error("Failed to get pending embedding states: %v", err)
		return nil, fmt.Errorf("failed to get pending embedding states: %w", err)
	}
	defer rows.Close()

	var states []*model.EmbeddingState
	for rows.Next() {
		var state model.EmbeddingState
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&state.SyncID,
			&state.WorkspacePath,
			&state.FilePath,
			&state.Status,
			&state.Message,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan embedding state row: %v", err)
			return nil, fmt.Errorf("failed to scan embedding state row: %w", err)
		}

		state.CreatedAt = createdAt
		state.UpdatedAt = updatedAt
		states = append(states, &state)
	}

	return states, nil
}

// UpdateEmbeddingStateStatus 更新语义构建状态
func (r *embeddingStateRepository) UpdateEmbeddingStateStatus(syncID string, status int, message string) error {
	query := `
		UPDATE embedding_states 
		SET status = ?, message = ?, updated_at = CURRENT_TIMESTAMP
		WHERE sync_id = ?
	`

	result, err := r.db.GetDB().Exec(query, status, message, syncID)
	if err != nil {
		r.logger.Error("Failed to update embedding state status: %v", err)
		return fmt.Errorf("failed to update embedding state status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("embedding state not found: %s", syncID)
	}

	return nil
}
