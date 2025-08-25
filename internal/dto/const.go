package dto

// 服务端Embedding构建状态常量
const (
	EmbeddingStatusPending = "pending"
	EmbeddingProcessing    = "processing"
	EmbeddingComplete      = "completed"
	EmbeddingFailed        = "failed"
	EmbeddingUnsupported   = "unsupported"
)

// 索引构建状态常量
const (
	ProcessStatusPending = "pending"
	ProcessStatusRunning = "running"
	ProcessStatusSuccess = "success"
	ProcessStatusFailed  = "failed"
)

// 索引构建类型常量
const (
	IndexTypeEmbedding = "embedding"
	IndexTypeCodegraph = "codegraph"
	IndexTypeAll       = "all"
)

// 索引开关状态常量
const (
	SwitchOn  = "on"
	SwitchOff = "off"
)

const (
	True  = "true"
	False = "false"
)
