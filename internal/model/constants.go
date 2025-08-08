package model

// EmbeddingStatus 语义构建状态常量
const (
	EmbeddingStatusInit         = 1 // 初始化
	EmbeddingStatusUploading    = 2 // 上报中
	EmbeddingStatusBuilding     = 3 // 构建中
	EmbeddingStatusUploadFailed = 4 // 上报失败
	EmbeddingStatusBuildFailed  = 5 // 构建失败
	EmbeddingStatusSuccess      = 6 // 构建成功
)

// CodegraphStatus 代码构建状态常量
const (
	CodegraphStatusInit     = 1 // 初始化
	CodegraphStatusBuilding = 2 // 构建中
	CodegraphStatusFailed   = 3 // 构建失败
	CodegraphStatusSuccess  = 4 // 构建成功
)

// EventType 事件类型常量
const (
	EventTypeUnknown        = "unknown"
	EventTypeAddFile        = "add_file"        // 创建文件事件
	EventTypeModifyFile     = "modify_file"     // 更新文件事件
	EventTypeDeleteFile     = "delete_file"     // 删除文件事件
	EventTypeRenameFile     = "rename_file"     // 移动文件事件
	EventTypeOpenWorkspace  = "open_workspace"  // 打开工作区事件
	EventTypeCloseWorkspace = "close_workspace" // 关闭工作区事件
)

const True = "true"
