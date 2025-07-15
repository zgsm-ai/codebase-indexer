package parser

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/project"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"strings"
)

type SourceFileParser struct {
	logger          logger.Logger
	resolverManager *resolver.ResolverManager
}

func NewSourceFileParser() *SourceFileParser {
	resolveManager := resolver.NewResolverManager()
	return &SourceFileParser{
		resolverManager: resolveManager,
	}
}

func (p *SourceFileParser) Parse(ctx context.Context,
	sourceFile *types.SourceFile,
	projectInfo *project.ProjectInfo) (*FileSymbolTable, error) {
	// Extract file extension
	langParser, err := lang.GetSitterParserByFilePath(sourceFile.Path)
	if err != nil {
		return nil, err
	}

	sitterParser := sitter.NewParser()
	sitterLanguage := langParser.SitterLanguage()
	if err := sitterParser.SetLanguage(sitterLanguage); err != nil {
		return nil, err
	}

	content := sourceFile.Content
	tree := sitterParser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file: %s", sourceFile.Path)
	}

	defer tree.Close()

	queryScm, ok := BaseQueries[langParser.Language]
	if !ok {
		return nil, lang.ErrQueryNotFound
	}

	query, err := sitter.NewQuery(sitterLanguage, queryScm)
	if err != nil && lang.IsRealQueryErr(err) {
		return nil, err
	}
	defer query.Close()

	captureNames := query.CaptureNames() // 根据scm文件从上到下排列的

	if len(captureNames) == 0 {
		return nil, fmt.Errorf("tree_sitter base_processor query capture names is empty")
	}

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), content)

	// 消费 matches，并调用 ProcessStructureMatch 处理匹配结果
	// elementName->elementPosition
	var visited = make(map[string][]int32)
	var sourcePackage *resolver.Package
	var imports []*resolver.Import
	elements := make([]resolver.Element, 0)
	for {
		// 统一的上下文取消检测函数
		if err = utils.CheckContextCanceled(ctx); err != nil {
			return nil, fmt.Errorf("tree_sitter base processor context canceled: %v", err)
		}

		match := matches.Next()
		if match == nil {
			break
		}
		// TODO Parent 、Children 关系处理。比如变量定义在函数中，函数定义在类中。
		elems, err := p.processNode(ctx, langParser.Language, match, captureNames, sourceFile, projectInfo)
		if err != nil {
			p.logger.Debug("tree_sitter base processor processNode error: %v", err)
			continue // 跳过错误的匹配
		}

		for _, element := range elems {
			// 去重，主要针对variable
			if position, ok := visited[element.GetName()]; ok && isSamePosition(position, element.GetRange()) {
				p.logger.Debug("tree_sitter base_processor duplicate element visited: %s, %v",
					element.GetName(), position)
				continue
			}
			visited[element.GetName()] = element.GetRange()
			// package go/java
			if element.GetType() == types.ElementTypePackage && sourcePackage == nil {
				sourcePackage = element.(*resolver.Package)
				continue
			}

			// imports
			if element.GetType() == types.ElementTypeImport {
				imports = append(imports, element.(*resolver.Import))
				continue
			}

			elements = append(elements, element)
		}

	}
	//TODO 顺序解析，对于使用在前，定义在后的类型，未进行处理，比如函数、方法、全局变量。需要再进行二次解析。

	// 返回结构信息，包含处理后的定义
	return &FileSymbolTable{
		Path:     sourceFile.Path,
		Package:  sourcePackage,
		Imports:  imports,
		Language: langParser.Language,
		Elements: elements,
	}, nil
}

func (p *SourceFileParser) processNode(
	ctx context.Context,
	language lang.Language,
	match *sitter.QueryMatch,
	captureNames []string,
	sourceFile *types.SourceFile,
	projectInfo *project.ProjectInfo) ([]resolver.Element, error) {

	if len(match.Captures) == 0 || len(captureNames) == 0 {
		return nil, lang.ErrNoCaptures
	} // root node
	rootIndex := match.Captures[0].Index
	rootCaptureName := captureNames[rootIndex]

	rootElement := newRootElement(rootCaptureName, rootIndex)

	resolvedElements := make([]resolver.Element, 0)
	for _, capture := range match.Captures {
		node := capture.Node
		if node.IsMissing() || node.IsError() {
			p.logger.Debug("tree_sitter base_processor capture node %s is missing or error",
				node.Kind())
			continue
		}
		captureName := captureNames[capture.Index] // index not in order

		p.updateRootElement(rootElement, &capture, captureName, sourceFile.Content)

		resolveCtx := &resolver.ResolveContext{
			Language:    language,
			CaptureName: captureName,
			CaptureNode: &node,
			SourceFile:  sourceFile,
			ProjectInfo: projectInfo,
		}

		elements, err := p.resolverManager.Resolve(ctx, rootElement, resolveCtx)
		if err != nil {
			// TODO full_name（import）、 find identifier recur (variable)、parameters/arguments
			p.logger.Debug("parse capture node %s err: %v", captureName, err)
		}
		resolvedElements = append(resolvedElements, elements...)
	}

	return resolvedElements, nil
}

func (p *SourceFileParser) updateRootElement(
	rootElement resolver.Element,
	capture *sitter.QueryCapture,
	captureName string,
	content []byte) {
	node := capture.Node
	// 设置range
	if capture.Index == rootElement.GetRootIndex() { // root capture: @package @function @class etc
		// rootNode
		rootElement.SetRange([]int32{
			int32(node.StartPosition().Row),
			int32(node.StartPosition().Column),
			int32(node.StartPosition().Row),
			int32(node.StartPosition().Column),
		})
	}

	// 设置name TODO 这里这里去掉，在 resolve中处理名字
	if rootElement.GetName() == types.EmptyString && IsElementNameCapture(rootElement.GetType(), captureName) {
		// 取root节点的name，比如definition.function.name
		// 获取名称 ,go import 带双引号
		name := strings.ReplaceAll(node.Utf8Text(content), types.DoubleQuote, types.EmptyString)
		if name == types.EmptyString {
			// TODO 日志
			fmt.Printf("tree_sitter base_processor name_node %s %v name not found", captureName, rootElement.GetRange())
		}
		rootElement.SetName(name)
	}
}

func isSamePosition(source []int32, target []int32) bool {
	if len(source) != len(target) {
		return false
	}
	for i := range source {
		if source[i] != target[i] {
			return false
		}
	}
	return true
}
