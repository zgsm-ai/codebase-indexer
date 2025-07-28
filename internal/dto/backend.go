// internal/dto/backend.go - 后端API请求和响应数据结构定义
package dto

// SearchRelationRequest 关系检索请求
type SearchRelationRequest struct {
	ClientId       string `form:"clientId" binding:"required"`
	CodebasePath   string `form:"codebasePath" binding:"required"`
	FilePath       string `form:"filePath" binding:"required"`
	StartLine      int    `form:"startLine" binding:"required"`
	StartColumn    int    `form:"startColumn" binding:"required"`
	EndLine        int    `form:"endLine" binding:"required"`
	EndColumn      int    `form:"endColumn" binding:"required"`
	SymbolName     string `form:"symbolName,omitempty"`
	IncludeContent bool   `form:"includeContent,omitempty"`
	MaxLayer       int    `form:"maxLayer,omitempty"`
}

// RelationNode 关系节点
type RelationNode struct {
	Content  string         `json:"content,omitempty"`
	NodeType string         `json:"nodeType"`
	FilePath string         `json:"filePath"`
	Position Position       `json:"position"`
	Children []RelationNode `json:"children"`
}

// Position 位置信息
type Position struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

// SearchRelationResponse 关系检索响应
type SearchRelationResponse struct {
	Code    int          `json:"code"`
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    RelationData `json:"data"`
}

type RelationData struct {
	List []RelationNode `json:"list"`
}

// SearchDefinitionRequest 获取定义请求
type SearchDefinitionRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	FilePath     string `form:"filePath" binding:"required"`
	StartLine    int    `form:"startLine,omitempty"`
	EndLine      int    `form:"endLine,omitempty"`
	CodeSnippet  string `form:"codeSnippet,omitempty"`
}

// DefinitionInfo 定义信息
type DefinitionInfo struct {
	FilePath string   `json:"filePath"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Content  string   `json:"content,omitempty"`
	Position Position `json:"position"`
}

// SearchDefinitionResponse 获取定义响应
type SearchDefinitionResponse struct {
	Code    int            `json:"code"`
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    DefinitionData `json:"data"`
}

type DefinitionData struct {
	List []DefinitionInfo `json:"list"`
}

// GetFileContentRequest 获取文件内容请求
type GetFileContentRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	FilePath     string `form:"filePath" binding:"required"`
	StartLine    int    `form:"startLine,omitempty"`
	EndLine      int    `form:"endLine,omitempty"`
}

// GetFileContentResponse 获取文件内容响应
type GetFileContentResponse struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

// GetCodebaseDirectoryRequest 获取代码库目录树请求
type GetCodebaseDirectoryRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	Depth        int    `form:"depth,omitempty"`
	IncludeFiles bool   `form:"includeFiles,omitempty"`
	SubDir       string `form:"subDir,omitempty"`
}

// DirectoryNode 目录节点
type DirectoryNode struct {
	Name     string          `json:"name"`
	IsDir    bool            `json:"isDir"`
	Path     string          `json:"path"`
	Size     int64           `json:"size,omitempty"`
	Children []DirectoryNode `json:"children,omitempty"`
}

// GetCodebaseDirectoryResponse 获取代码库目录树响应
type GetCodebaseDirectoryResponse struct {
	Code    int           `json:"code"`
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Data    DirectoryData `json:"data"`
}

type DirectoryData struct {
	CodebaseId    string          `json:"codebaseId"`
	Name          string          `json:"name"`
	RootPath      string          `json:"rootPath"`
	TotalFiles    int             `json:"totalFiles"`
	TotalSize     int64           `json:"totalSize"`
	DirectoryTree []DirectoryNode `json:"directoryTree"`
}

// GetFileStructureRequest 获取文件结构请求
type GetFileStructureRequest struct {
	ClientId     string   `form:"clientId" binding:"required"`
	CodebasePath string   `form:"codebasePath" binding:"required"`
	FilePath     string   `form:"filePath" binding:"required"`
	Types        []string `form:"types,omitempty"`
}

// FileStructureInfo 文件结构信息
type FileStructureInfo struct {
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
	Content  string   `json:"content,omitempty"`
}

// GetFileStructureResponse 获取文件结构响应
type GetFileStructureResponse struct {
	Code    int               `json:"code"`
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    FileStructureData `json:"data"`
}

type FileStructureData struct {
	List []FileStructureInfo `json:"list"`
}

// GetIndexSummaryRequest 获取索引情况请求
type GetIndexSummaryRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
}

// IndexSummary 索引摘要
type IndexSummary struct {
	Codegraph CodegraphInfo `json:"codegraph"`
}

// CodegraphInfo 代码关系索引信息
type CodegraphInfo struct {
	Status     string `json:"status"`
	TotalFiles int    `json:"totalFiles"`
}

// GetIndexSummaryResponse 获取索引情况响应
type GetIndexSummaryResponse struct {
	Code    int          `json:"code"`
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    IndexSummary `json:"data"`
}
