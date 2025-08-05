package model

import "time"

// Workspace 工作区数据模型
type Workspace struct {
	ID                       int64     `json:"id" db:"id"`
	WorkspaceName            string    `json:"workspaceName" db:"workspace_name"`
	WorkspacePath            string    `json:"workspacePath" db:"workspace_path"`
	Active                   bool      `json:"active" db:"active"`
	FileNum                  int       `json:"fileNum" db:"file_num"`
	EmbeddingFileNum         int       `json:"embeddingFileNum" db:"embedding_file_num"`
	EmbeddingTs              int64     `json:"embeddingTs" db:"embedding_ts"`
	EmbeddingMessage         string    `json:"embeddingMessage" db:"embedding_message"`
	EmbeddingFailedFilePaths string    `json:"embeddingFailedFilePaths" db:"embedding_failed_file_paths"`
	CodegraphFileNum         int       `json:"codegraphFileNum" db:"codegraph_file_num"`
	CodegraphTs              int64     `json:"codegraphTs" db:"codegraph_ts"`
	CodegraphMessage         string    `json:"codegraphMessage" db:"codegraph_message"`
	CodegraphFailedFilePaths string    `json:"codegraphFailedFilePaths" db:"codegraph_failed_file_paths"`
	CreatedAt                time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt                time.Time `json:"updatedAt" db:"updated_at"`
}

// Event 事件数据模型
type Event struct {
	ID              int64     `json:"id" db:"id"`
	WorkspacePath   string    `json:"workspacePath" db:"workspace_path"`
	EventType       string    `json:"eventType" db:"event_type"`
	SourceFilePath  string    `json:"sourceFilePath" db:"source_file_path"`
	TargetFilePath  string    `json:"targetFilePath" db:"target_file_path"`
	SyncId          string    `json:"syncId" db:"sync_id"`
	EmbeddingStatus int       `json:"embeddingStatus" db:"embedding_status"`
	CodegraphStatus int       `json:"codegraphStatus" db:"codegraph_status"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

// EmbeddingState 语义构建状态数据模型
type EmbeddingState struct {
	SyncID        string    `json:"syncId" db:"sync_id"`
	WorkspacePath string    `json:"workspacePath" db:"workspace_path"`
	FilePath      string    `json:"filePath" db:"file_path"`
	Status        int       `json:"status" db:"status"`
	Message       string    `json:"message" db:"message"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

// CodegraphState 代码构建状态数据模型
type CodegraphState struct {
	WorkspacePath string    `json:"workspacePath" db:"workspace_path"`
	FilePath      string    `json:"filePath" db:"file_path"`
	Status        int       `json:"status" db:"status"`
	Message       string    `json:"message" db:"message"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}
