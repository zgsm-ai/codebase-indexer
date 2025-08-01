package repository

import (
	"database/sql"
	"fmt"
	"time"

	"codebase-indexer/internal/database"
	"codebase-indexer/internal/model"
	"codebase-indexer/pkg/logger"
)

// EventRepository 事件数据访问层
type EventRepository interface {
	// CreateEvent 创建事件
	CreateEvent(event *model.Event) error
	// GetEventByID 根据ID获取事件
	GetEventByID(id int64) (*model.Event, error)
	// GetEventsByWorkspace 根据工作区路径获取事件
	GetEventsByWorkspace(workspacePath string, limit int, isDesc bool) ([]*model.Event, error)
	// GetEventsByType 根据事件类型获取事件
	GetEventsByType(eventType string, limit int, isDesc bool) ([]*model.Event, error)
	// GetEventsByWorkspaceAndType 根据工作区路径和事件类型获取事件
	GetEventsByWorkspaceAndType(workspacePath, eventType string, limit int, isDesc bool) ([]*model.Event, error)
	// UpdateEvent 更新事件
	UpdateEvent(event *model.Event) error
	// DeleteEvent 删除事件
	DeleteEvent(id int64) error
	// GetRecentEvents 获取最近的事件
	GetRecentEvents(workspacePath string, limit int) ([]*model.Event, error)
}

// eventRepository 事件Repository实现
type eventRepository struct {
	db     database.DatabaseManager
	logger logger.Logger
}

// NewEventRepository 创建事件Repository
func NewEventRepository(db database.DatabaseManager, logger logger.Logger) EventRepository {
	return &eventRepository{
		db:     db,
		logger: logger,
	}
}

// CreateEvent 创建事件
func (r *eventRepository) CreateEvent(event *model.Event) error {
	query := `
		INSERT INTO events (workspace_path, event_type, source_file_path, target_file_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.GetDB().Exec(query,
		event.WorkspacePath,
		event.EventType,
		event.SourceFilePath,
		event.TargetFilePath,
		event.CreatedAt,
		event.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to create event: %v", err)
		return fmt.Errorf("failed to create event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("Failed to get last insert ID: %v", err)
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	event.ID = id
	return nil
}

// GetEventByID 根据ID获取事件
func (r *eventRepository) GetEventByID(id int64) (*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path, 
			embedding_status, codegraph_status, created_at, updated_at
		FROM events 
		WHERE id = ?
	`

	row := r.db.GetDB().QueryRow(query, id)

	var event model.Event
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&event.ID,
		&event.WorkspacePath,
		&event.EventType,
		&event.SourceFilePath,
		&event.TargetFilePath,
		&event.EmbeddingStatus,
		&event.CodegraphStatus,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found: %d", id)
		}
		r.logger.Error("Failed to get event by ID: %v", err)
		return nil, fmt.Errorf("failed to get event by ID: %w", err)
	}

	event.CreatedAt = createdAt
	event.UpdatedAt = updatedAt

	return &event, nil
}

// GetEventsByWorkspace 根据工作区路径获取事件
func (r *eventRepository) GetEventsByWorkspace(workspacePath string, limit int, isDesc bool) ([]*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path, 
			embedding_status, codegraph_status, created_at, updated_at
		FROM events 
		WHERE workspace_path = ?
		ORDER BY created_at %s
		LIMIT ?
	`
	if isDesc {
		query = fmt.Sprintf(query, "DESC")
	} else {
		query = fmt.Sprintf(query, "ASC")
	}
	rows, err := r.db.GetDB().Query(query, workspacePath, limit)
	if err != nil {
		r.logger.Error("Failed to get events by workspace: %v", err)
		return nil, fmt.Errorf("failed to get events by workspace: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var event model.Event
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&event.ID,
			&event.WorkspacePath,
			&event.EventType,
			&event.SourceFilePath,
			&event.TargetFilePath,
			&event.EmbeddingStatus,
			&event.CodegraphStatus,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan event row: %v", err)
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		event.CreatedAt = createdAt
		event.UpdatedAt = updatedAt
		events = append(events, &event)
	}

	return events, nil
}

// GetEventsByType 根据事件类型获取事件
func (r *eventRepository) GetEventsByType(eventType string, limit int, isDesc bool) ([]*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path, 
			codegraph_status, embedding_status, created_at, updated_at
		FROM events 
		WHERE event_type = ?
		ORDER BY created_at %s
		LIMIT ?
	`

	if isDesc {
		query = fmt.Sprintf(query, "DESC")
	} else {
		query = fmt.Sprintf(query, "ASC")
	}
	rows, err := r.db.GetDB().Query(query, eventType, limit)
	if err != nil {
		r.logger.Error("Failed to get events by type: %v", err)
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var event model.Event
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&event.ID,
			&event.WorkspacePath,
			&event.EventType,
			&event.SourceFilePath,
			&event.TargetFilePath,
			&event.CodegraphStatus,
			&event.EmbeddingStatus,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan event row: %v", err)
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		event.CreatedAt = createdAt
		event.UpdatedAt = updatedAt
		events = append(events, &event)
	}

	return events, nil
}

// GetEventsByWorkspaceAndType 根据工作区路径和事件类型获取事件
func (r *eventRepository) GetEventsByWorkspaceAndType(workspacePath, eventType string, limit int, isDesc bool) ([]*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path, 
			codegraph_status, embedding_status, created_at, updated_at
		FROM events 
		WHERE workspace_path = ? AND event_type = ?
		ORDER BY created_at %s
		LIMIT ?
	`
	if isDesc {
		query = fmt.Sprintf(query, "DESC")
	} else {
		query = fmt.Sprintf(query, "ASC")
	}
	rows, err := r.db.GetDB().Query(query, workspacePath, eventType, limit)
	if err != nil {
		r.logger.Error("Failed to get events by workspace and type: %v", err)
		return nil, fmt.Errorf("failed to get events by workspace and type: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var event model.Event
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&event.ID,
			&event.WorkspacePath,
			&event.EventType,
			&event.SourceFilePath,
			&event.TargetFilePath,
			&event.CodegraphStatus,
			&event.EmbeddingStatus,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan event row: %v", err)
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		event.CreatedAt = createdAt
		event.UpdatedAt = updatedAt
		events = append(events, &event)
	}

	return events, nil
}

// UpdateEvent 更新事件
func (r *eventRepository) UpdateEvent(event *model.Event) error {
	query := `
		UPDATE events 
		SET workspace_path = ?, event_type = ?, source_file_path = ?, 
			target_file_path = ?, embedding_status = ?, codegraph_status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.GetDB().Exec(query,
		event.WorkspacePath,
		event.EventType,
		event.SourceFilePath,
		event.TargetFilePath,
		event.EmbeddingStatus,
		event.CodegraphStatus,
		event.ID,
	)
	if err != nil {
		r.logger.Error("Failed to update event: %v", err)
		return fmt.Errorf("failed to update event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found: %d", event.ID)
	}

	return nil
}

// DeleteEvent 删除事件
func (r *eventRepository) DeleteEvent(id int64) error {
	query := `DELETE FROM events WHERE id = ?`

	result, err := r.db.GetDB().Exec(query, id)
	if err != nil {
		r.logger.Error("Failed to delete event: %v", err)
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found: %d", id)
	}

	return nil
}

// GetRecentEvents 获取最近的事件
func (r *eventRepository) GetRecentEvents(workspacePath string, limit int) ([]*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path, 
			codegraph_status, embedding_status, created_at, updated_at
		FROM events 
		WHERE workspace_path = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.GetDB().Query(query, workspacePath, limit)
	if err != nil {
		r.logger.Error("Failed to get recent events: %v", err)
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var event model.Event
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&event.ID,
			&event.WorkspacePath,
			&event.EventType,
			&event.SourceFilePath,
			&event.TargetFilePath,
			&event.CodegraphStatus,
			&event.EmbeddingStatus,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan event row: %v", err)
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		event.CreatedAt = createdAt
		event.UpdatedAt = updatedAt
		events = append(events, &event)
	}

	return events, nil
}
