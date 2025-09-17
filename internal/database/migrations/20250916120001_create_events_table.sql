-- 创建事件表
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_path VARCHAR(500) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    source_file_path VARCHAR(500) NOT NULL DEFAULT '',
    target_file_path VARCHAR(500) NOT NULL DEFAULT '',
    sync_id VARCHAR(100) NOT NULL DEFAULT '',
    file_hash VARCHAR(100) NOT NULL DEFAULT '',
    embedding_status TINYINT NOT NULL DEFAULT 1,
    codegraph_status TINYINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_events_workspace_path ON events(workspace_path);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_sync_id ON events(sync_id);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_events_updated_at ON events(updated_at);
CREATE INDEX IF NOT EXISTS idx_events_workspace_type ON events(workspace_path, event_type);