package types

type NodeType string

const ( //
	NodeTypeDefinition     NodeType = "definition"     // 定义节点（根节点）
	NodeTypeUnknown        NodeType = "unknown"        // 未知
	NodeTypeReference      NodeType = "reference"      // 引用关系
	NodeTypeImplementation NodeType = "implementation" // 实现关系（类 -> 接口）
)

type IndexTaskMetrics struct {
	TotalFiles       int
	TotalSymbols     int
	TotalFailedFiles int
	FailedFilePaths  []string
}

// CodeDefinition 代码文件结构
type CodeDefinition struct {
	Path        string
	Language    string
	Definitions []*Definition
}

type Definition struct {
	Name    string
	Type    string
	Path    string
	Range   []int32
	Content []byte
}

type QueryDefinitionOptions struct {
	StartLine   int
	EndLine     int
	Workspace   string
	FilePath    string
	CodeSnippet []byte
}

type QueryReferenceOptions struct {
	Workspace  string
	FilePath   string
	StartLine  int
	EndLine    int
	SymbolName string
}

type RelationNode struct {
	FilePath   string          `json:"FilePath"`
	SymbolName string          `json:"symbolName"`
	Position   Position        `json:"position"`
	Content    string          `json:"content"`
	NodeType   string          `json:"nodeType"`
	Children   []*RelationNode `json:"children"`
}

type CodeGraphSummary struct {
	TotalFiles int `json:"totalFiles"`
}

type Position struct {
	StartLine   int `json:"startLine"`   // 开始行（从1开始）
	StartColumn int `json:"startColumn"` // 开始列（从1开始）
	EndLine     int `json:"endLine"`     // 结束行（从1开始）
	EndColumn   int `json:"endColumn"`   // 结束列（从1开始）
}

// ToPosition 辅助函数：将 ranges 转换为 types.Position
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

// ToRange 辅助函数：将 Position 转换为 range
func ToRange(position Position) []int32 {
	return []int32{int32(position.StartLine) - 1, int32(position.StartColumn) - 1,
		int32(position.EndLine) - 1, int32(position.EndColumn) - 1}
}
