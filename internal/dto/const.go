package dto

// 服务端Embedding构建状态常量
const (
	EmbeddingStatusPending = "pending"
	EmbeddingProcessing    = "processing"
	EmbeddingComplete      = "complete"
	EmbeddingFailed        = "failed"
)

// 索引构建状态常量
const (
	ProcessStatusPending = "pending"
	ProcessStatusRunning = "running"
	ProcessStatusSuccess = "success"
	ProcessStatusFailed  = "failed"
)
