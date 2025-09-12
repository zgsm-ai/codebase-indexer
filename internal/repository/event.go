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

// EventRepository 事件数据访问层
type EventRepository interface {
	// CreateEvent 创建事件
	CreateEvent(event *model.Event) error
	// GetEventByID 根据ID获取事件
	GetEventByID(id int64) (*model.Event, error)
	// GetEventsByWorkspace 根据工作区路径获取事件
	GetEventsByWorkspace(workspacePath string, limit int, isDesc bool) ([]*model.Event, error)
	// GetEventsByType 根据事件类型获取事件
	GetEventsByType(eventTypes []string, limit int, isDesc bool) ([]*model.Event, error)
	// GetEventsByWorkspaceAndType 根据工作区路径和事件类型获取事件
	GetEventsByWorkspaceAndType(workspacePath string, eventTypes []string, limit int, isDesc bool) ([]*model.Event, error)
	// GetEventsByWorkspaceAndEmbeddingStatus 根据工作区路径和嵌入状态获取事件
	GetEventsByWorkspaceAndEmbeddingStatus(workspacePath string, limit int, isDesc bool, statuses []int) ([]*model.Event, error)
	// GetEventsByTypeAndEmbeddingStatus 根据事件类型和状态获取事件
	GetEventsByTypeAndEmbeddingStatus(eventTypes []string, limit int, isDesc bool, statuses []int) ([]*model.Event, error)
	// GetEventsByTypeAndStatusAndWorkspaces 根据事件类型、状态和工作空间路径获取事件
	GetEventsByTypeAndStatusAndWorkspaces(eventTypes []string, workspacePaths []string, limit int, isDesc bool, embeddingStatuses []int, codegraphStatuses []int) ([]*model.Event, error)
	// UpdateEvent 更新事件
	UpdateEvent(event *model.Event) error
	// UpdateEventByMap 根据map更新事件
	UpdateEventByMap(id int64, updates map[string]interface{}) error
	// DeleteEvent 删除事件
	DeleteEvent(id int64) error
	// GetRecentEvents 获取最近的事件
	GetRecentEvents(workspacePath string, limit int) ([]*model.Event, error)
	// GetEventsByWorkspaceForDeduplication 获取工作区内所有事件用于去重（无限制，用于内存中比较）
	GetEventsByWorkspaceForDeduplication(workspacePath string) ([]*model.Event, error)
	// GetEventsCountByType 获取满足事件类型条件的事件总数
	GetEventsCountByType(eventTypes []string) (int64, error)
	// GetEventsCountByWorkspaceAndStatus 根据工作区路径、嵌入状态和代码图状态获取事件总数
	GetEventsCountByWorkspaceAndStatus(workspacePaths []string, embeddingStatuses []int, codegraphStatuses []int) (int64, error)
	// GetLatestEventByWorkspaceAndSourcePath 根据工作区路径和源文件路径获取最新记录
	GetLatestEventByWorkspaceAndSourcePath(workspacePath, sourceFilePath string) (*model.Event, error)
	// BatchCreateEvents 批量创建事件
	BatchCreateEvents(events []*model.Event) error
	// BatchDeleteEvents 批量删除事件
	BatchDeleteEvents(ids []int64) error
	// UpdateEvents 批量更新事件嵌入信息
	UpdateEventsEmbedding(events []*model.Event) error
	// UpdateEventsEmbeddingStatus 批量更新事件嵌入状态
	UpdateEventsEmbeddingStatus(eventIDs []int64, status int) error
	// GetExpiredEventIDs 获取过期事件的ID列表
	GetExpiredEventIDs(cutoffTime time.Time) ([]int64, error)
	// GetTableName 获取表名
	GetTableName() string
	// ClearTable 清理表数据并重置ID
	ClearTable() error
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
		INSERT INTO events (workspace_path, event_type, source_file_path, target_file_path, embedding_status, codegraph_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	nowTime := time.Now()
	result, err := r.db.GetDB().Exec(query,
		event.WorkspacePath,
		event.EventType,
		event.SourceFilePath,
		event.TargetFilePath,
		event.EmbeddingStatus,
		event.CodegraphStatus,
		nowTime,
		nowTime,
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
			embedding_status, codegraph_status, sync_id, file_hash, created_at, updated_at
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
		&event.SyncId,
		&event.FileHash,
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
			embedding_status, codegraph_status, sync_id, file_hash, created_at, updated_at
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
			&event.SyncId,
			&event.FileHash,
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
func (r *eventRepository) GetEventsByType(eventTypes []string, limit int, isDesc bool) ([]*model.Event, error) {
	// 基础查询语句
	baseQuery := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
		FROM events
	`

	args := []interface{}{}
	whereClause := ""

	// 构造事件类型过滤条件
	if len(eventTypes) > 0 {
		placeholders := strings.Repeat("?,", len(eventTypes))
		placeholders = placeholders[:len(placeholders)-1] // 移除最后一个逗号
		whereClause = fmt.Sprintf("WHERE event_type IN (%s)", placeholders)
		for _, eventType := range eventTypes {
			args = append(args, eventType)
		}
	}

	// 构造排序条件
	orderDirection := "ASC"
	if isDesc {
		orderDirection = "DESC"
	}

	// 组装完整查询
	query := fmt.Sprintf("%s %s ORDER BY created_at %s LIMIT ?", baseQuery, whereClause, orderDirection)
	args = append(args, limit)

	rows, err := r.db.GetDB().Query(query, args...)
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
			&event.SyncId,
			&event.FileHash,
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
func (r *eventRepository) GetEventsByWorkspaceAndType(workspacePath string, eventTypes []string, limit int, isDesc bool) ([]*model.Event, error) {
	// 基础查询语句
	baseQuery := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
		FROM events
		WHERE workspace_path = ?
	`

	args := []interface{}{workspacePath}

	// 构造事件类型过滤条件
	if len(eventTypes) > 0 {
		placeholders := strings.Repeat("?,", len(eventTypes))
		placeholders = placeholders[:len(placeholders)-1] // 移除最后一个逗号
		baseQuery += fmt.Sprintf(" AND event_type IN (%s)", placeholders)
		for _, eventType := range eventTypes {
			args = append(args, eventType)
		}
	}

	// 构造排序条件
	orderDirection := "ASC"
	if isDesc {
		orderDirection = "DESC"
	}

	// 组装完整查询
	query := fmt.Sprintf("%s ORDER BY created_at %s LIMIT ?", baseQuery, orderDirection)
	args = append(args, limit)

	rows, err := r.db.GetDB().Query(query, args...)
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
			&event.SyncId,
			&event.FileHash,
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

// GetEventsByWorkspaceAndEmbeddingStatus 根据工作区路径和嵌入状态获取事件
func (r *eventRepository) GetEventsByWorkspaceAndEmbeddingStatus(workspacePath string, limit int, isDesc bool, statuses []int) ([]*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
		FROM events
		WHERE workspace_path = ?
	`

	args := []interface{}{workspacePath}

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
		r.logger.Error("Failed to get events by workspace and embedding status: %v", err)
		return nil, fmt.Errorf("failed to get events by workspace and embedding status: %w", err)
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
			&event.SyncId,
			&event.FileHash,
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

// GetEventsByTypeAndEmbeddingStatus 根据事件类型和状态获取事件
func (r *eventRepository) GetEventsByTypeAndEmbeddingStatus(eventTypes []string, limit int, isDesc bool, statuses []int) ([]*model.Event, error) {
	// 基础查询语句
	baseQuery := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
		FROM events
	`

	args := []interface{}{}
	whereClause := ""

	// 构造事件类型过滤条件
	if len(eventTypes) > 0 {
		placeholders := strings.Repeat("?,", len(eventTypes))
		placeholders = placeholders[:len(placeholders)-1] // 移除最后一个逗号
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE event_type IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND event_type IN (%s)", placeholders)
		}
		for _, eventType := range eventTypes {
			args = append(args, eventType)
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
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE embedding_status IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND embedding_status IN (%s)", placeholders)
		}
		for _, status := range statuses {
			args = append(args, status)
		}
	}

	// 构造排序条件
	orderDirection := "ASC"
	if isDesc {
		orderDirection = "DESC"
	}

	// 组装完整查询
	query := fmt.Sprintf("%s %s ORDER BY created_at %s LIMIT ?", baseQuery, whereClause, orderDirection)
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
			&event.SyncId,
			&event.FileHash,
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
func (r *eventRepository) GetEventsByTypeAndStatusAndWorkspaces(eventTypes []string, workspacePaths []string, limit int,
	isDesc bool, embeddingStatuses []int, codegraphStatuses []int) ([]*model.Event, error) {
	// 如果limit为-1，表示查询所有记录，使用分批查询
	if limit == -1 {
		return r.getAllEventsByTypeAndStatusAndWorkspaces(eventTypes, workspacePaths, isDesc, embeddingStatuses, codegraphStatuses)
	}

	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
		FROM events
	`

	args := []interface{}{}
	whereClause := ""

	// 构造事件类型过滤条件
	if len(eventTypes) > 0 {
		placeholders := strings.Repeat("?,", len(eventTypes))
		placeholders = placeholders[:len(placeholders)-1] // 移除最后一个逗号
		whereClause = fmt.Sprintf("WHERE event_type IN (%s)", placeholders)
		for _, eventType := range eventTypes {
			args = append(args, eventType)
		}
	}

	// 添加工作空间路径过滤条件
	if len(workspacePaths) > 0 {
		placeholders := ""
		for i := range workspacePaths {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE workspace_path IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND workspace_path IN (%s)", placeholders)
		}
		for _, path := range workspacePaths {
			args = append(args, path)
		}
	}

	// 如果提供了状态列表，添加状态过滤条件
	if len(embeddingStatuses) > 0 {
		placeholders := ""
		for i := range embeddingStatuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE embedding_status IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND embedding_status IN (%s)", placeholders)
		}
		for _, status := range embeddingStatuses {
			args = append(args, status)
		}
	}

	if len(codegraphStatuses) > 0 {
		placeholders := ""
		for i := range codegraphStatuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE codegraph_status IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND codegraph_status IN (%s)", placeholders)
		}
		for _, status := range codegraphStatuses {
			args = append(args, status)
		}
	}

	// 构造排序条件
	orderDirection := "ASC"
	if isDesc {
		orderDirection = "DESC"
	}

	// 组装完整查询
	query = fmt.Sprintf("%s %s ORDER BY created_at %s LIMIT ?", query, whereClause, orderDirection)
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
			&event.SyncId,
			&event.FileHash,
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

// getAllEventsByTypeAndStatusAndWorkspaces 获取所有符合条件的事件（分批查询）
func (r *eventRepository) getAllEventsByTypeAndStatusAndWorkspaces(eventTypes []string, workspacePaths []string,
	isDesc bool, embeddingStatuses []int, codegraphStatuses []int) ([]*model.Event, error) {
	const batchSize = 1000
	var allEvents []*model.Event
	offset := 0

	// 构建基础查询条件
	baseQuery := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
		FROM events
	`

	args := []interface{}{}
	whereClause := ""

	// 构造事件类型过滤条件
	if len(eventTypes) > 0 {
		placeholders := strings.Repeat("?,", len(eventTypes))
		placeholders = placeholders[:len(placeholders)-1] // 移除最后一个逗号
		whereClause = fmt.Sprintf("WHERE event_type IN (%s)", placeholders)
		for _, eventType := range eventTypes {
			args = append(args, eventType)
		}
	}

	// 添加工作空间路径过滤条件
	if len(workspacePaths) > 0 {
		placeholders := ""
		for i := range workspacePaths {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE workspace_path IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND workspace_path IN (%s)", placeholders)
		}
		for _, path := range workspacePaths {
			args = append(args, path)
		}
	}

	// 如果提供了状态列表，添加状态过滤条件
	if len(embeddingStatuses) > 0 {
		placeholders := ""
		for i := range embeddingStatuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE embedding_status IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND embedding_status IN (%s)", placeholders)
		}
		for _, status := range embeddingStatuses {
			args = append(args, status)
		}
	}

	if len(codegraphStatuses) > 0 {
		placeholders := ""
		for i := range codegraphStatuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereClause == "" {
			whereClause = fmt.Sprintf("WHERE codegraph_status IN (%s)", placeholders)
		} else {
			whereClause += fmt.Sprintf(" AND codegraph_status IN (%s)", placeholders)
		}
		for _, status := range codegraphStatuses {
			args = append(args, status)
		}
	}

	// 构造排序条件
	orderDirection := "ASC"
	if isDesc {
		orderDirection = "DESC"
	}

	for {
		// 构建分页查询
		batchArgs := make([]interface{}, len(args))
		copy(batchArgs, args)
		query := fmt.Sprintf("%s %s ORDER BY created_at %s LIMIT ? OFFSET ?", baseQuery, whereClause, orderDirection)
		batchArgs = append(batchArgs, batchSize, offset)

		rows, err := r.db.GetDB().Query(query, batchArgs...)
		if err != nil {
			r.logger.Error("Failed to query events batch (offset %d): %v", offset, err)
			return nil, fmt.Errorf("failed to query events batch (offset %d): %w", offset, err)
		}

		var batchEvents []*model.Event
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
				&event.SyncId,
				&event.FileHash,
				&createdAt,
				&updatedAt,
			)
			if err != nil {
				rows.Close()
				r.logger.Error("Failed to scan event row (offset %d): %v", offset, err)
				return nil, fmt.Errorf("failed to scan event row (offset %d): %w", offset, err)
			}

			event.CreatedAt = createdAt
			event.UpdatedAt = updatedAt
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

	r.logger.Info("Retrieved %d events by type, status and workspaces", len(allEvents))
	return allEvents, nil
}

// UpdateEvent 更新事件
func (r *eventRepository) UpdateEvent(event *model.Event) error {
	// 构建SET子句，只包含非默认值的字段
	var setClauses []string
	var args []interface{}

	// 检查workspace_path是否为非默认值
	if event.WorkspacePath != "" {
		setClauses = append(setClauses, "workspace_path = ?")
		args = append(args, event.WorkspacePath)
	}

	// 检查event_type是否为非默认值
	if event.EventType != "" {
		setClauses = append(setClauses, "event_type = ?")
		args = append(args, event.EventType)
	}

	// 检查source_file_path是否为非默认值
	if event.SourceFilePath != "" {
		setClauses = append(setClauses, "source_file_path = ?")
		args = append(args, event.SourceFilePath)
	}

	// 检查target_file_path是否为非默认值
	if event.TargetFilePath != "" {
		setClauses = append(setClauses, "target_file_path = ?")
		args = append(args, event.TargetFilePath)
	}

	// 检查embedding_status是否为非默认值
	if event.EmbeddingStatus != 0 {
		setClauses = append(setClauses, "embedding_status = ?")
		args = append(args, event.EmbeddingStatus)
	}

	// 检查codegraph_status是否为非默认值
	if event.CodegraphStatus != 0 {
		setClauses = append(setClauses, "codegraph_status = ?")
		args = append(args, event.CodegraphStatus)
	}

	// 检查sync_id是否为非默认值
	if event.SyncId != "" {
		setClauses = append(setClauses, "sync_id = ?")
		args = append(args, event.SyncId)
	}

	// 检查file_hash是否为非默认值
	if event.FileHash != "" {
		setClauses = append(setClauses, "file_hash = ?")
		args = append(args, event.FileHash)
	}

	// 如果没有需要更新的字段，直接返回
	if len(setClauses) == 0 {
		return nil
	}

	// 添加updated_at字段
	setClauses = append(setClauses, "updated_at = ?")
	nowTime := time.Now()
	args = append(args, nowTime)

	// 构建完整查询
	query := fmt.Sprintf("UPDATE events SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	args = append(args, event.ID)

	result, err := r.db.GetDB().Exec(query, args...)
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

// UpdateEventByMap 根据map更新事件
func (r *eventRepository) UpdateEventByMap(id int64, updates map[string]interface{}) error {
	// 如果没有需要更新的字段，直接返回
	if len(updates) == 0 {
		return nil
	}

	// 构建SET子句和参数
	var setClauses []string
	var args []interface{}

	// 支持的字段映射
	fieldMapping := map[string]string{
		"workspace_path":   "workspace_path",
		"event_type":       "event_type",
		"source_file_path": "source_file_path",
		"target_file_path": "target_file_path",
		"embedding_status": "embedding_status",
		"codegraph_status": "codegraph_status",
		"sync_id":          "sync_id",
		"file_hash":        "file_hash",
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
	query := fmt.Sprintf("UPDATE events SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	args = append(args, id)

	result, err := r.db.GetDB().Exec(query, args...)
	if err != nil {
		r.logger.Error("Failed to update event by map: %v", err)
		return fmt.Errorf("failed to update event by map: %w", err)
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
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
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
			&event.SyncId,
			&event.FileHash,
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
			SELECT id, workspace_path, event_type, source_file_path, target_file_path, embedding_status, codegraph_status, sync_id, file_hash, created_at, updated_at
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
			var createdAt, updatedAt time.Time

			err := rows.Scan(
				&event.ID,
				&event.WorkspacePath,
				&event.EventType,
				&event.SourceFilePath,
				&event.TargetFilePath,
				&event.EmbeddingStatus,
				&event.CodegraphStatus,
				&event.SyncId,
				&event.FileHash,
				&createdAt,
				&updatedAt,
			)
			if err != nil {
				rows.Close()
				r.logger.Error("Failed to scan event row for deduplication: %v", err)
				return nil, fmt.Errorf("failed to scan event row: %w", err)
			}

			event.CreatedAt = createdAt
			event.UpdatedAt = updatedAt
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

// GetEventsCountByWorkspaceAndStatus 根据工作区路径、嵌入状态和代码图状态获取事件总数
func (r *eventRepository) GetEventsCountByWorkspaceAndStatus(workspacePaths []string, embeddingStatuses []int, codegraphStatuses []int) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM events
	`

	args := []interface{}{}
	whereAdded := false

	// 如果提供了工作区路径列表，添加工作区路径过滤条件
	if len(workspacePaths) > 0 {
		placeholders := ""
		for i := range workspacePaths {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		query += fmt.Sprintf(" WHERE workspace_path IN (%s)", placeholders)
		whereAdded = true
		for _, path := range workspacePaths {
			args = append(args, path)
		}
	}

	// 如果提供了嵌入状态列表，添加嵌入状态过滤条件
	if len(embeddingStatuses) > 0 {
		placeholders := ""
		for i := range embeddingStatuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereAdded {
			query += fmt.Sprintf(" AND embedding_status IN (%s)", placeholders)
		} else {
			query += fmt.Sprintf(" WHERE embedding_status IN (%s)", placeholders)
			whereAdded = true
		}
		for _, status := range embeddingStatuses {
			args = append(args, status)
		}
	}

	// 如果提供了代码图状态列表，添加代码图状态过滤条件
	if len(codegraphStatuses) > 0 {
		placeholders := ""
		for i := range codegraphStatuses {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		if whereAdded {
			query += fmt.Sprintf(" AND codegraph_status IN (%s)", placeholders)
		} else {
			query += fmt.Sprintf(" WHERE codegraph_status IN (%s)", placeholders)
		}
		for _, status := range codegraphStatuses {
			args = append(args, status)
		}
	}

	var count int64
	err := r.db.GetDB().QueryRow(query, args...).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}

// GetLatestEventByWorkspaceAndSourcePath 根据工作区路径和源文件路径获取最新记录
func (r *eventRepository) GetLatestEventByWorkspaceAndSourcePath(workspacePath, sourceFilePath string) (*model.Event, error) {
	query := `
		SELECT id, workspace_path, event_type, source_file_path, target_file_path,
			codegraph_status, embedding_status, sync_id, file_hash, created_at, updated_at
		FROM events
		WHERE workspace_path = ? AND source_file_path = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	row := r.db.GetDB().QueryRow(query, workspacePath, sourceFilePath)

	var event model.Event
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&event.ID,
		&event.WorkspacePath,
		&event.EventType,
		&event.SourceFilePath,
		&event.TargetFilePath,
		&event.CodegraphStatus,
		&event.EmbeddingStatus,
		&event.SyncId,
		&event.FileHash,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to get latest event by workspace and source path: %v", err)
		return nil, err
	}

	event.CreatedAt = createdAt
	event.UpdatedAt = updatedAt

	return &event, nil
}

// BatchCreateEvents 批量创建事件
func (r *eventRepository) BatchCreateEvents(events []*model.Event) error {
	if len(events) == 0 {
		return nil
	}

	const batchSize = 1000
	nowTime := time.Now()
	totalCreated := int64(0)

	// 分批处理
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		batch := events[i:end]

		// 构建批量插入的SQL语句
		valueStrings := make([]string, 0, len(batch))
		valueArgs := make([]interface{}, 0, len(batch)*6)

		for _, event := range batch {
			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				event.WorkspacePath,
				event.EventType,
				event.SourceFilePath,
				event.TargetFilePath,
				event.EmbeddingStatus,
				event.CodegraphStatus,
				nowTime,
				nowTime,
			)
		}

		query := fmt.Sprintf("INSERT INTO events (workspace_path, event_type, source_file_path, target_file_path, embedding_status, codegraph_status, created_at, updated_at) VALUES %s",
			strings.Join(valueStrings, ","))

		result, err := r.db.GetDB().Exec(query, valueArgs...)
		if err != nil {
			r.logger.Error("Failed to batch create events (batch %d-%d): %v", i+1, end, err)
			return fmt.Errorf("failed to batch create events (batch %d-%d): %w", i+1, end, err)
		}

		// 获取最后插入的ID，用于设置事件的ID
		lastInsertID, err := result.LastInsertId()
		if err != nil {
			r.logger.Error("Failed to get last insert ID (batch %d-%d): %v", i+1, end, err)
			return fmt.Errorf("failed to get last insert ID (batch %d-%d): %w", i+1, end, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			r.logger.Error("Failed to get rows affected (batch %d-%d): %v", i+1, end, err)
			return fmt.Errorf("failed to get rows affected (batch %d-%d): %w", i+1, end, err)
		}

		totalCreated += rowsAffected

		// 设置每个事件的ID
		for j, event := range batch {
			event.ID = lastInsertID - int64(len(batch)-1-j)
		}

		r.logger.Info("Successfully created batch %d-%d: %d events", i+1, end, rowsAffected)
	}

	r.logger.Info("Successfully created total %d events", totalCreated)
	return nil
}

// BatchDeleteEvents 批量删除事件
func (r *eventRepository) BatchDeleteEvents(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	const batchSize = 1000
	totalDeleted := int64(0)

	// 分批处理
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		// 构建批量删除的SQL语句
		placeholders := strings.Repeat("?,", len(batch))
		placeholders = placeholders[:len(placeholders)-1] // 移除最后一个逗号

		query := fmt.Sprintf("DELETE FROM events WHERE id IN (%s)", placeholders)

		// 转换batch为interface{}切片
		args := make([]interface{}, len(batch))
		for j, id := range batch {
			args[j] = id
		}

		result, err := r.db.GetDB().Exec(query, args...)
		if err != nil {
			r.logger.Error("Failed to batch delete events (batch %d-%d): %v", i+1, end, err)
			return fmt.Errorf("failed to batch delete events (batch %d-%d): %w", i+1, end, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			r.logger.Error("Failed to get rows affected (batch %d-%d): %v", i+1, end, err)
			return fmt.Errorf("failed to get rows affected (batch %d-%d): %w", i+1, end, err)
		}

		totalDeleted += rowsAffected
		r.logger.Info("Successfully deleted batch %d-%d: %d events", i+1, end, rowsAffected)
	}

	r.logger.Info("Successfully deleted total %d events", totalDeleted)
	return nil
}

// UpdateEvents 批量更新事件嵌入信息
func (r *eventRepository) UpdateEventsEmbedding(events []*model.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `
		UPDATE events
		SET embedding_status = ?, sync_id = ?, file_hash = ?, updated_at = ?
		WHERE id = ?
	`

	nowTime := time.Now()
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		_, err = stmt.Exec(
			event.EmbeddingStatus,
			event.SyncId,
			event.FileHash,
			nowTime,
			event.ID,
		)
		if err != nil {
			r.logger.Error("Failed to update event %d: %v", event.ID, err)
			return fmt.Errorf("failed to update event %d: %w", event.ID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Successfully updated %d events", len(events))
	return nil
}

// UpdateEventsEmbeddingStatus 批量更新事件嵌入状态
func (r *eventRepository) UpdateEventsEmbeddingStatus(eventIDs []int64, status int) error {
	if len(eventIDs) == 0 {
		return nil
	}

	tx, err := r.db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `
		UPDATE events
		SET embedding_status = ?, updated_at = ?
		WHERE id = ?
	`

	nowTime := time.Now()
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, id := range eventIDs {
		_, err = stmt.Exec(status, nowTime, id)
		if err != nil {
			r.logger.Error("Failed to update event status for ID %d: %v", id, err)
			return fmt.Errorf("failed to update event status for ID %d: %w", id, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Successfully updated status for %d events", len(eventIDs))
	return nil
}

// GetExpiredEventIDs 获取过期事件的ID列表
func (r *eventRepository) GetExpiredEventIDs(cutoffTime time.Time) ([]int64, error) {
	query := `
		SELECT id
		FROM events
		WHERE updated_at < ?
		ORDER BY updated_at ASC
	`

	rows, err := r.db.GetDB().Query(query, cutoffTime)
	if err != nil {
		r.logger.Error("Failed to get expired event IDs: %v", err)
		return nil, fmt.Errorf("failed to get expired event IDs: %w", err)
	}
	defer rows.Close()

	var eventIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			r.logger.Error("Failed to scan event ID: %v", err)
			return nil, fmt.Errorf("failed to scan event ID: %w", err)
		}
		eventIDs = append(eventIDs, id)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating expired event IDs: %v", err)
		return nil, fmt.Errorf("error iterating expired event IDs: %w", err)
	}

	r.logger.Info("Found %d expired events before %s", len(eventIDs), cutoffTime.Format(time.RFC3339))
	return eventIDs, nil
}

// GetTableName 获取表名
func (r *eventRepository) GetTableName() string {
	return "events"
}

// ClearTable 清理表数据并重置ID
func (r *eventRepository) ClearTable() error {
	return r.db.ClearTable(r.GetTableName())
}
