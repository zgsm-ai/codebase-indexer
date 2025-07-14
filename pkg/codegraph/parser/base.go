package parser

import (
	"codebase-indexer/pkg/codegraph/parser/resolver"
	"codebase-indexer/pkg/codegraph/parser/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
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
	sourceFile *resolver.SourceFile,
	projectInfo *resolver.ProjectInfo) (*ParsedSource, error) {
	// Extract file extension
	langParser, err := GetTreeSitterParserByFilePath(sourceFile.Path)
	if err != nil {
		return nil, err
	}
	queryScm, ok := BaseQueries[langParser.Language]
	if !ok {
		return nil, ErrQueryNotFound
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

	query, err := sitter.NewQuery(sitterLanguage, queryScm)
	if err != nil && IsRealQueryErr(err) {
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
		element, err := p.processNode(ctx, langParser.Language, match, captureNames, sourceFile, projectInfo)
		if err != nil {
			p.logger.Debug("tree_sitter base processor processNode error: %v", err)
			continue // 跳过错误的匹配
		}
		// 去重，主要针对variable
		if position, ok := visited[element.GetName()]; ok && isSamePosition(position, element.GetRange()) {
			p.logger.Debug("tree_sitter base_processor duplicate element visited: %s, %v",
				element.GetName(), position)
			continue
		}
		visited[element.GetName()] = element.GetRange()
		// package go/java
		if element.GetType() == resolver.ElementTypePackage && sourcePackage == nil {
			sourcePackage = element.(*resolver.Package)
			continue
		}

		// imports
		if element.GetType() == resolver.ElementTypeImport {
			imports = append(imports, element.(*resolver.Import))
			continue
		}

		elements = append(elements, element)
	}
	//TODO 顺序解析，对于使用在前，定义在后的类型，未进行处理，比如函数、方法、全局变量。需要再进行二次解析。

	// 返回结构信息，包含处理后的定义
	return &ParsedSource{
		Path:     sourceFile.Path,
		Package:  sourcePackage,
		Imports:  imports,
		Language: langParser.Language,
		Elements: elements,
	}, nil
}

func (p *SourceFileParser) processNode(
	ctx context.Context,
	language resolver.Language,
	match *sitter.QueryMatch,
	captureNames []string,
	sourceFile *resolver.SourceFile,
	projectInfo *resolver.ProjectInfo) (resolver.Element, error) {

	if len(match.Captures) == 0 || len(captureNames) == 0 {
		return nil, ErrNoCaptures
	} // root node
	rootIndex := match.Captures[0].Index
	rootCaptureName := captureNames[rootIndex]

	rootElement := initRootElement(rootCaptureName)

	rootElement.SetRootIndex(rootIndex)

	for _, capture := range match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			p.logger.Debug("tree_sitter base_processor capture node %s is missing or error",
				capture.Node.Kind())
			continue
		}
		captureName := captureNames[capture.Index] // index not in order

		resolveCtx := &resolver.ResolveContext{
			Language:    language,
			CaptureName: captureName,
			CaptureNode: &capture.Node,
			SourceFile:  sourceFile,
			ProjectInfo: projectInfo,
		}

		if err := p.resolverManager.Resolve(ctx, rootElement, resolveCtx); err != nil {
			// TODO full_name（import）、 find identifier recur (variable)、parameters/arguments
			p.logger.Debug("parse capture node %s err: %v", captureName, err)
		}

	}

	return rootElement, nil
}

// findIdentifierNode 递归遍历语法树节点，查找类型为"identifier"的节点
func findIdentifierNode(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	// 检查当前节点是否为identifier类型
	if node.Kind() == identifier {
		return node
	}

	// 遍历所有子节点
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		// 递归查找子节点中的identifier
		result := findIdentifierNode(child)
		if result != nil {
			return result // 找到则立即返回
		}
	}

	// 未找到identifier节点
	return nil
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
