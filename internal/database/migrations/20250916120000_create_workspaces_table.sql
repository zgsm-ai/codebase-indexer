-- 创建工作区表
CREATE TABLE IF NOT EXISTS workspaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_name VARCHAR(255) NOT NULL,
    workspace_path VARCHAR(500) UNIQUE NOT NULL,
    active VARCHAR(10) NOT NULL DEFAULT 'true',
    file_num INTEGER NOT NULL DEFAULT 0,
    embedding_file_num INTEGER NOT NULL DEFAULT 0,
    embedding_ts INTEGER NOT NULL DEFAULT 0,
    embedding_message VARCHAR(255) NOT NULL DEFAULT '',
    embedding_failed_file_paths TEXT NOT NULL DEFAULT '',
    codegraph_file_num INTEGER NOT NULL DEFAULT 0,
    codegraph_ts INTEGER NOT NULL DEFAULT 0,
    codegraph_message VARCHAR(255) NOT NULL DEFAULT '',
    codegraph_failed_file_paths TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_workspaces_path ON workspaces(workspace_path);
CREATE INDEX IF NOT EXISTS idx_workspaces_embedding_ts ON workspaces(embedding_ts);
CREATE INDEX IF NOT EXISTS idx_workspaces_codegraph_ts ON workspaces(codegraph_ts);
CREATE INDEX IF NOT EXISTS idx_workspaces_created_at ON workspaces(created_at);
CREATE INDEX IF NOT EXISTS idx_workspaces_updated_at ON workspaces(updated_at);