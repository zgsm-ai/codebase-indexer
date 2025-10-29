// internal/dto/backend.go - 后端API请求和响应数据结构定义
package dto

import "codebase-indexer/pkg/codegraph/types"

// SearchReferenceRequest 关系检索请求
type SearchReferenceRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	FilePath     string `form:"filePath"` // 可选，适配单符号查询
	StartLine    int    `form:"startLine"`
	EndLine      int    `form:"endLine"`
	SymbolName   string `form:"symbolName"`
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

type ReferenceData struct {
	List []*types.RelationNode `json:"list"`
}

// SearchDefinitionRequest 获取定义请求
type SearchDefinitionRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	FilePath     string `form:"filePath"`
	SymbolNames  string `form:"symbolNames"`
	StartLine    int    `form:"startLine,omitempty"`
	EndLine      int    `form:"endLine,omitempty"`
	CodeSnippet  string `form:"codeSnippet,omitempty"`
}

// CallGraphData 代码片段内部元素或单符号的调用链
type CallGraphData struct {
	List []*types.RelationNode `json:"list"`
}

// GetCallGraphRequest 获取函数调用链及其函数定义
type SearchCallGraphRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	FilePath     string `form:"filePath" binding:"required"`
	LineRange    string `form:"lineRange,omitempty"`
	SymbolName   string `form:"symbolName,omitempty"`
	MaxLayer     int    `form:"maxLayer,omitempty"`
}

type ReadCodeSnippetsRequest struct {
	ClientId      string              `json:"clientId" binding:"required"`
	WorkspacePath string              `json:"workspacePath" binding:"required"`
	CodeSnippets  []*CodeSnippetQuery `json:"codeSnippets"  binding:"required,min=1"`
}

type CodeSnippetQuery struct {
	FilePath  string `json:"filePath" binding:"required"`        // 确保文件路径不为空
	StartLine int    `json:"startLine" binding:"required,min=1"` // 行号至少为1
	EndLine   int    `json:"endLine" binding:"required,min=1"`   // 结束行必须大于开始行
}

type CodeSnippetsData struct {
	CodeSnippets []*CodeSnippet `json:"list"`
}

type CodeSnippet struct {
	FilePath  string `json:"filePath"`  // 文件路径
	StartLine int    `json:"startLine"` // 起始行号
	EndLine   int    `json:"endLine"`   // 结束行号
	Content   string `json:"content"`   // 代码片段内容
}

// DefinitionInfo 定义信息
type DefinitionInfo struct {
	FilePath string   `json:"filePath"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Content  string   `json:"content,omitempty"`
	Position Position `json:"position"`
}

type DefinitionData struct {
	List []*DefinitionInfo `json:"list"`
}

// GetFileContentRequest 获取文件内容请求
type GetFileContentRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	FilePath     string `form:"filePath" binding:"required"`
	StartLine    int    `form:"startLine,omitempty"`
	EndLine      int    `form:"endLine,omitempty"`
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

type DirectoryData struct {
	//CodebaseId    string            `json:"codebaseId"`
	//Name          string            `json:"name"`
	RootPath      string            `json:"rootPath"`
	TotalFiles    int               `json:"-"`
	TotalSize     int64             `json:"-"`
	DirectoryTree []*types.TreeNode `json:"directoryTree"`
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

type FileStructureData struct {
	List []*FileStructureInfo `json:"list"`
}

// GetIndexSummaryRequest 获取索引情况请求
type GetIndexSummaryRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
}

// ExportIndexRequest 导出索引请求
type ExportIndexRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
}

// DeleteIndexRequest 删除索引请求
type DeleteIndexRequest struct {
	ClientId     string `form:"clientId" binding:"required"`
	CodebasePath string `form:"codebasePath" binding:"required"`
	IndexType    string `form:"indexType" binding:"required"`
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

// ToPosition 辅助函数：将 ranges 转换为 Position
func ToPosition(ranges []int32) Position {
	if len(ranges) != 3 && len(ranges) != 4 {
		return Position{}
	}
	if len(ranges) == 3 {
		return Position{
			StartLine:   int(ranges[0]) + 1,
			StartColumn: int(ranges[1]) + 1,
			EndLine:     int(ranges[0]) + 1,
			EndColumn:   int(ranges[2]) + 1,
		}
	} else {
		return Position{
			StartLine:   int(ranges[0]) + 1,
			StartColumn: int(ranges[1]) + 1,
			EndLine:     int(ranges[2]) + 1,
			EndColumn:   int(ranges[3]) + 1,
		}
	}

}

const (
	Embedding = "embedding"
	Codegraph = "codegraph"
	All       = "all"
)
