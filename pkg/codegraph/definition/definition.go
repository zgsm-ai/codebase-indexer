package definition

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const name = "name"

// DefParser  用于解析代码结构
type DefParser struct {
}

type ParseOptions struct {
	IncludeContent bool
}

// NewDefinitionParser creates a new generic parser with the given config.
func NewDefinitionParser() *DefParser {
	return &DefParser{}
}

// Parse 解析文件结构，返回结构信息（例如函数、结构体、接口、变量、常量等）
func (s *DefParser) Parse(ctx context.Context, codeFile *types.SourceFile, opts ParseOptions) (*types.CodeDefinition, error) {
	// Extract file extension
	langConf, err := lang.GetSitterParserByFilePath(codeFile.Path)
	if err != nil {
		return nil, err
	}
	query, ok := parser.DefinitionQueries[langConf.Language]
	if !ok {
		return nil, lang.ErrQueryNotFound
	}

	sitterParser := sitter.NewParser()
	defer sitterParser.Close()
	sitterLanguage := langConf.SitterLanguage()
	if err := sitterParser.SetLanguage(sitterLanguage); err != nil {
		return nil, err
	}
	code := codeFile.Content
	tree := sitterParser.Parse(code, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file")
	}
	defer tree.Close()

	// 执行 query，并处理匹配结果
	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), code)

	// 消费 matches，并调用 ProcessStructureMatch 处理匹配结果
	definitions := make([]*types.Definition, 0)
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		def, err := s.ProcessDefinitionNode(m, query, code, opts)
		if err != nil {
			continue // 跳过错误的匹配
		}
		definitions = append(definitions, def)
	}

	// 返回结构信息，包含处理后的定义
	return &types.CodeDefinition{
		Definitions: definitions,
		Path:        codeFile.Path,
		Language:    string(langConf.Language),
	}, nil
}

// ProcessDefinitionNode provides shared functionality for processing structure matches
func (p *DefParser) ProcessDefinitionNode(match *sitter.QueryMatch, query *sitter.Query,
	source []byte, opts ParseOptions) (*types.Definition, error) {
	if len(match.Captures) == 0 {
		return nil, lang.ErrNoCaptures
	}

	// 获取定义节点、名称节点和其他必要节点
	var defNode *sitter.Node
	var nameNode *sitter.Node
	var defType string

	for _, capture := range match.Captures {
		captureName := query.CaptureNames()[capture.Index]
		if captureName == name {
			nameNode = &capture.Node
		} else if defNode == nil { // 使用第一个非 name 的捕获作为定义类型
			defNode = &capture.Node
			defType = captureName
		}
	}

	if defNode == nil || nameNode == nil {
		return nil, lang.ErrMissingNode
	}
	// TODO range 有问题，golang  import (xxx xxx xxx) 捕获的是整体。
	// 获取名称
	nodeName := nameNode.Utf8Text(source)
	if nodeName == "" {
		return nil, fmt.Errorf("no name found for QueryDefinitions")
	}

	// 获取范围
	startPoint := defNode.StartPosition()
	endPoint := defNode.EndPosition()
	startLine := startPoint.Row
	startColumn := startPoint.Column
	endLine := endPoint.Row
	endColumn := endPoint.Column

	var content []byte
	if opts.IncludeContent {
		content = source[defNode.StartByte():defNode.EndByte()]
	}

	return &types.Definition{
		Type:    defType,
		Name:    nodeName,
		Range:   []int32{int32(startLine), int32(startColumn), int32(endLine), int32(endColumn)},
		Content: content,
	}, nil
}
