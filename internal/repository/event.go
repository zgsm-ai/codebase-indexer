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
	// GetEventsByTypeAndStatus 根据事件类型和状态获取事件
	GetEventsByTypeAndStatus(eventType string, limit int, isDesc bool, statuses []int) ([]*model.Event, error)
	// GetEventsByTypeAndStatusAndWorkspaces 根据事件类型、状态和工作空间路径获取事件
	GetEventsByTypeAndStatusAndWorkspaces(eventType string, workspacePaths []string, limit int, isDesc bool, statuses []int) ([]*model.Event, error)
	// UpdateEvent 更新事件
	UpdateEvent(event *model.Event) error
	// DeleteEvent 删除事件
	DeleteEvent(id int64) error
	// GetRecentEvents 获取最近的事件
	GetRecentEvents(workspacePath string, limit int) ([]*model.Event, error)
	// GetEventsByWorkspaceForDeduplication 获取工作区内所有事件用于去重（无限制，用于内存中比较）
	GetEventsByWorkspaceForDeduplication(workspacePath string) ([]*model.Event, error)
	// GetEventsCountByType 获取满足事件类型条件的事件总数
	GetEventsCountByType(eventTypes []string) (int64, error)
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

// GetEventsByTypeAndStatus 根据事件类型和状态获取事件
func (r *eventRepository) GetEventsByTypeAndStatus(eventType string, limit int, isDesc bool, statuses []int) ([]*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, created_at, updated_at
		FROM events
		WHERE event_type = ?
	`

	args := []interface{}{eventType}

	// 如果提供了状态列表，添加状态过滤条件
	if len(statuses) > 0 {
		placeholders := ""
		for i := range statuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		query += fmt.Sprintf(" AND embedding_status IN (%s)", placeholders)
		for _, status := range statuses {
			args = append(args, status)
		}
	}

	query += " ORDER BY created_at %s LIMIT ?"

	if isDesc {
		query = fmt.Sprintf(query, "DESC")
	} else {
		query = fmt.Sprintf(query, "ASC")
	}
	args = append(args, limit)

	rows, err := r.db.GetDB().Query(query, args...)
	if err != nil {
		r.logger.Error("Failed to get events by type and status: %v", err)
		return nil, fmt.Errorf("failed to get events by type and status: %w", err)
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

// GetEventsByTypeAndStatusAndWorkspaces 根据事件类型、状态和工作空间路径获取事件
func (r *eventRepository) GetEventsByTypeAndStatusAndWorkspaces(eventType string, workspacePaths []string, limit int, isDesc bool, statuses []int) ([]*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, created_at, updated_at
		FROM events
		WHERE event_type = ?
	`

	args := []interface{}{eventType}

	// 添加工作空间路径过滤条件
	if len(workspacePaths) > 0 {
		placeholders := ""
		for i := range workspacePaths {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		query += fmt.Sprintf(" AND workspace_path IN (%s)", placeholders)
		for _, path := range workspacePaths {
			args = append(args, path)
		}
	}

	// 如果提供了状态列表，添加状态过滤条件
	if len(statuses) > 0 {
		placeholders := ""
		for i := range statuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		query += fmt.Sprintf(" AND embedding_status IN (%s)", placeholders)
		for _, status := range statuses {
			args = append(args, status)
		}
	}

	query += " ORDER BY created_at %s LIMIT ?"

	if isDesc {
		query = fmt.Sprintf(query, "DESC")
	} else {
		query = fmt.Sprintf(query, "ASC")
	}
	args = append(args, limit)

	rows, err := r.db.GetDB().Query(query, args...)
	if err != nil {
		r.logger.Error("Failed to get events by type, status and workspaces: %v", err)
		return nil, fmt.Errorf("failed to get events by type, status and workspaces: %w", err)
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

// GetEventsByWorkspaceForDeduplication 获取工作区内所有事件用于去重（无限制，用于内存中比较）
func (r *eventRepository) GetEventsByWorkspaceForDeduplication(workspacePath string) ([]*model.Event, error) {
	const batchSize = 1000
	var allEvents []*model.Event
	offset := 0

	for {
		query := `
			SELECT id, workspace_path, event_type, source_file_path, target_file_path, created_at
			FROM events
			WHERE workspace_path = ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`

		rows, err := r.db.GetDB().Query(query, workspacePath, batchSize, offset)
		if err != nil {
			r.logger.Error("Failed to query events batch for deduplication: %v", err)
			return nil, fmt.Errorf("failed to query events batch: %w", err)
		}

		var batchEvents []*model.Event
		for rows.Next() {
			var event model.Event
			var createdAt time.Time

			err := rows.Scan(
				&event.ID,
				&event.WorkspacePath,
				&event.EventType,
				&event.SourceFilePath,
				&event.TargetFilePath,
				&createdAt,
			)
			if err != nil {
				rows.Close()
				r.logger.Error("Failed to scan event row for deduplication: %v", err)
				return nil, fmt.Errorf("failed to scan event row: %w", err)
			}

			event.CreatedAt = createdAt
			batchEvents = append(batchEvents, &event)
		}
		rows.Close()

		if len(batchEvents) == 0 {
			break
		}

		allEvents = append(allEvents, batchEvents...)
		offset += len(batchEvents)

		// 如果返回的记录数小于批次大小，说明已经查询完毕
		if len(batchEvents) < batchSize {
			break
		}
	}

	r.logger.Info("Retrieved %d events for deduplication in workspace: %s", len(allEvents), workspacePath)
	return allEvents, nil
}

// GetEventsCountByType 获取满足事件类型条件的事件总数
func (r *eventRepository) GetEventsCountByType(eventTypes []string) (int64, error) {
	// 如果没有提供事件类型，返回0
	if len(eventTypes) == 0 {
		return 0, nil
	}

	query := `
		SELECT COUNT(*)
		FROM events
		WHERE event_type IN (`

	args := make([]interface{}, len(eventTypes))
	placeholders := ""
	for i, eventType := range eventTypes {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = eventType
	}

	query += placeholders + ")"

	var count int64
	err := r.db.GetDB().QueryRow(query, args...).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		r.logger.Error("Failed to get events count by types: %v", err)
		return 0, fmt.Errorf("failed to get events count by types: %w", err)
	}

	return count, nil
}
