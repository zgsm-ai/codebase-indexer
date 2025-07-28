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
	Code int `json:"code"`

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
