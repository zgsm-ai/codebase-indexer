// internal/dto/extension.go - Extension API DTOs
package dto

// RegisterSyncRequest represents the request for registering sync service
// @Description 注册同步服务的请求参数
type RegisterSyncRequest struct {
	// 客户端ID
	// required: true
	// example: client-123456
	ClientId string `json:"clientId" binding:"required"`

	// 工作空间路径
	// required: true
	// example: /home/user/workspace/project
	WorkspacePath string `json:"workspacePath" binding:"required"`

	// 工作空间名称
	// required: true
	// example: my-project
	WorkspaceName string `json:"workspaceName" binding:"required"`
}

// RegisterSyncResponse represents the response for registering sync service
// @Description 注册同步服务的响应数据
type RegisterSyncResponse struct {
	// 是否成功
	// example: true
	Success bool `json:"success"`

	// 响应消息
	// example: 3 codebases registered successfully
	Message string `json:"message"`
}

// SyncCodebaseRequest represents the request for syncing codebase
// @Description 同步代码库的请求参数
type SyncCodebaseRequest struct {
	// 客户端ID
	// required: true
	// example: client-123456
	ClientId string `json:"clientId" binding:"required"`

	// 工作空间路径
	// required: true
	// example: /home/user/workspace/project
	WorkspacePath string `json:"workspacePath" binding:"required"`

	// 工作空间名称
	// required: true
	// example: my-project
	WorkspaceName string `json:"workspaceName" binding:"required"`

	// 文件路径列表（可选）
	// example: ["src/main.go", "README.md"]
	FilePaths []string `json:"filePaths"`
}

// SyncCodebaseResponse represents the response for syncing codebase
// @Description 同步代码库的响应数据
type SyncCodebaseResponse struct {
	// 是否成功
	// example: true
	Success bool `json:"success"`

	// 响应代码
	// example: 0
	Code string `json:"code"`

	// 响应消息
	// example: sync codebase success
	Message string `json:"message"`
}

// UnregisterSyncRequest represents the request for unregistering sync service
// @Description 取消注册同步服务的请求参数
type UnregisterSyncRequest struct {
	// 客户端ID
	// required: true
	// example: client-123456
	ClientId string `json:"clientId" binding:"required"`

	// 工作空间路径
	// required: true
	// example: /home/user/workspace/project
	WorkspacePath string `json:"workspacePath" binding:"required"`

	// 工作空间名称
	// required: true
	// example: my-project
	WorkspaceName string `json:"workspaceName" binding:"required"`
}

// UnregisterSyncResponse represents the response for unregistering sync service
// @Description 取消注册同步服务的响应数据
type UnregisterSyncResponse struct {
	// 响应消息
	// example: unregistered 2 codebase(s)
	Message string `json:"message"`

	// 是否成功
	// example: true
	Success bool `json:"success"`
}

// ShareAccessTokenRequest represents the request for sharing access token
// @Description 共享访问令牌的请求参数
type ShareAccessTokenRequest struct {
	// 客户端ID
	// required: true
	// example: client-123456
	ClientId string `json:"clientId" binding:"required"`

	// 服务器端点
	// required: true
	// example: https://api.example.com
	ServerEndpoint string `json:"serverEndpoint" binding:"required"`

	// 访问令牌
	// required: true
	// example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
	AccessToken string `json:"accessToken" binding:"required"`
}

// ShareAccessTokenResponse represents the response for sharing access token
// @Description 共享访问令牌的响应数据
type ShareAccessTokenResponse struct {
	// 响应代码
	// example: 0
	Code int `json:"code"`
	// 是否成功
	// example: true
	Success bool `json:"success"`

	// 响应消息
	// example: ok
	Message string `json:"message"`
}

// VersionRequest represents the request for getting version info
// @Description 获取版本信息的请求参数
type VersionRequest struct {
	// 客户端ID
	// required: true
	// example: client-123456
	ClientId string `json:"clientId" binding:"required"`
}

// VersionResponseData represents the version data
// @Description 版本信息数据
type VersionResponseData struct {
	// 应用名称
	// example: Codebase Syncer
	AppName string `json:"appName"`

	// 版本号
	// example: 1.0.0
	Version string `json:"version"`

	// 操作系统名称
	// example: cross-platform
	OsName string `json:"osName"`

	// 架构名称
	// example: universal
	ArchName string `json:"archName"`
}

// VersionResponse represents the response for getting version info
// @Description 获取版本信息的响应数据
type VersionResponse struct {
	// 响应代码
	// example: 0
	Code int `json:"code"`
	// 是否成功
	// example: true
	Success bool `json:"success"`
	// 响应消息
	// example: ok
	Message string `json:"message"`
	// 版本数据
	Data VersionResponseData `json:"data"`
}

// CheckIgnoreFileRequest represents the request for checking ignore file
// @Description 检查忽略文件的请求参数
type CheckIgnoreFileRequest struct {
	// 客户端ID
	// required: true
	// example: client-123456
	ClientId string `json:"clientId" binding:"required"`

	// 工作空间路径
	// required: true
	// example: /home/user/workspace/project
	WorkspacePath string `json:"workspacePath" binding:"required"`

	// 工作空间名称
	// required: true
	// example: project
	WorkspaceName string `json:"workspaceName" binding:"required"`

	// 文件路径列表
	// required: true
	// example: ["/home/user/workspace/project/file1.txt", "/home/user/workspace/project/file2.txt"]
	FilePaths []string `json:"filePaths" binding:"required"`
}

// CheckIgnoreFileResponse represents the response for checking ignore file
// @Description 检查忽略文件的响应数据
type CheckIgnoreFileResponse struct {
	// 响应代码
	// example: 0
	Code string `json:"code"`

	// 是否成功
	// example: true
	Success bool `json:"success"`

	// 是否应该忽略
	// example: false
	Ignore bool `json:"ignore"`

	// 错误信息
	// example: ""
	Message string `json:"message"`
}

// WorkspaceEvent represents a single workspace event
// @Description 工作区事件数据
type WorkspaceEvent struct {
	// 事件类型
	// example: add_file
	// enum: open_workspace,close_workspace,add_file,modify_file,delete_file,rename_file
	EventType string `json:"eventType" binding:"required"`

	// 事件发生时间
	// example: 2025-07-18 16:01:00
	EventTime string `json:"eventTime" binding:"required"`

	// 源文件路径
	// example: G:\projects\codebase-indexer\main.go
	SourcePath string `json:"sourcePath" binding:"required"`

	// 目标文件路径（重命名或移动时使用）
	// example: G:\projects\codebase-indexer\new_main.go
	TargetPath string `json:"targetPath"`
}

// PublishEventsRequest represents the request for publishing workspace events
// @Description 发布工作区事件的请求参数
type PublishEventsRequest struct {
	// 工作空间路径
	// required: true
	// example: G:\projects\codebase-indexer
	Workspace string `json:"workspace" binding:"required"`

	// 事件数据列表
	// required: true
	Data []WorkspaceEvent `json:"data" binding:"required"`
}

// PublishEventsResponse represents the response for publishing workspace events
// @Description 发布工作区事件的响应数据
type PublishEventsResponse struct {
	// 响应代码
	// example: 0
	Code string `json:"code"`

	// 是否成功
	// example: true
	Success bool `json:"success"`

	// 响应消息
	// example: ok
	Message string `json:"message"`

	// 处理的事件数量
	// example: 1
	Data int `json:"data"`
}

// TriggerIndexRequest represents the request for triggering index build
// @Description 触发索引构建的请求参数
type TriggerIndexRequest struct {
	// 工作空间路径
	// required: true
	// example: G:\projects\codebase-indexer
	Workspace string `json:"workspace" binding:"required"`

	// 索引类型
	// required: true
	// enum: codegraph,embedding,all
	// example: codegraph
	Type string `json:"type" binding:"required"`
}

// TriggerIndexResponse represents the response for triggering index build
// @Description 触发索引构建的响应数据
type TriggerIndexResponse struct {
	// 响应代码
	// example: 0
	Code string `json:"code"`

	// 是否成功
	// example: true
	Success bool `json:"success"`

	// 响应消息
	// example: ok
	Message string `json:"message"`

	// 触发的任务ID
	// example: 1
	Data int `json:"data"`
}

// IndexStatus represents the status of a specific index type
// @Description 索引状态信息
type IndexStatus struct {
	// 状态
	// example: running
	// enum: pending,running,success,failed
	Status string `json:"status"`

	Process float32 `json:"process"`

	// 总文件数
	// example: 100
	TotalFiles int `json:"totalFiles"`

	// 成功处理的文件数
	// example: 10
	TotalSucceed int `json:"totalSucceed"`

	// 处理失败的文件数
	// example: 10
	TotalFailed int `json:"totalFailed"`

	// 总块数（仅embedding索引）
	// example: 1000
	TotalChunks int `json:"totalChunks,omitempty"`

	FailedReason string `json:"failedReason"`

	FailedFiles []string `json:"failedFiles,omitempty"`

	ProcessTs int64 `json:"processTs"`
}

type IndexStatusData struct {
	Embedding IndexStatus `json:"embedding"`
	Codegraph IndexStatus `json:"codegraph"`
}

// IndexStatusResponse represents the response for querying index status
// @Description 索引状态查询的响应数据
type IndexStatusResponse struct {
	// 响应代码
	// example: 0
	Code string `json:"code"`

	// 响应消息
	// example: ok
	Message string `json:"message"`

	// 项目索引状态数据
	Data IndexStatusData `json:"data"`
}

// IndexStatusQuery represents the query parameters for index status
// @Description 索引状态查询参数
type IndexStatusQuery struct {
	// 客户端ID
	// required: true
	// example: 111a
	ClientId string `form:"clientId" binding:"required"`

	// 工作空间路径
	// required: true
	// example: g:\projects\codebase-indexer
	Workspace string `form:"workspace" binding:"required"`
}

// IndexSwitchQuery represents the query parameters for index switch
// @Description 索引功能开关查询参数
type IndexSwitchQuery struct {
	Workspace string `form:"workspace" binding:"required"`
	// 开关状态
	// required: true
	// example: on
	// enum: on,off
	// default: off
	Switch string `form:"switch" binding:"required,oneof=on off"`
}

// IndexSwitchResponse represents the response for index switch
// @Description 索引功能开关响应数据
type IndexSwitchResponse struct {
	// 响应代码
	// example: 0
	Code string `json:"code"`

	// 是否成功
	// example: true
	Success bool `json:"success"`

	// 响应消息
	// example: ok
	Message string `json:"message"`

	// 当前开关状态
	// example: true
	Data bool `json:"data"`
}
