-- 更新工作区表，添加 deepwiki 相关字段
ALTER TABLE workspaces ADD COLUMN deepwiki_file_num INTEGER NOT NULL DEFAULT 0;
ALTER TABLE workspaces ADD COLUMN deepwiki_ts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE workspaces ADD COLUMN deepwiki_message VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE workspaces ADD COLUMN deepwiki_failed_file_paths TEXT NOT NULL DEFAULT '';

-- 为 deepwiki_ts 字段创建索引
CREATE INDEX IF NOT EXISTS idx_workspaces_deepwiki_ts ON workspaces(deepwiki_ts);