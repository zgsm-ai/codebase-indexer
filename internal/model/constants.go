package model

// EmbeddingStatus 语义构建状态常量
const (
	EmbeddingStatusUploading    = 1 // 上报中
	EmbeddingStatusBuilding     = 2 // 构建中
	EmbeddingStatusUploadFailed = 3 // 上报失败
	EmbeddingStatusBuildFailed  = 4 // 构建失败
	EmbeddingStatusSuccess      = 5 // 构建成功
)

// CodegraphStatus 代码构建状态常量
const (
	CodegraphStatusBuilding = 1 // 构建中
	CodegraphStatusFailed   = 2 // 构建失败
	CodegraphStatusSuccess  = 3 // 构建成功
)

// EventType 事件类型常量
const (
	EventTypeAddFile      = "add_file"      // 创建文件事件
	EventTypeModifyFile   = "modify_file"   // 更新文件事件
	EventTypeDeleteFile   = "delete_file"   // 删除文件事件
	EventTypeRenameFile   = "rename_file"   // 移动文件事件
	EventTypeDeleteFolder = "delete_folder" // 删除文件夹事件
	EventTypeRenameFolder = "rename_folder" // 移动文件夹事件
)
