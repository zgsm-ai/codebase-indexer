package parser

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type SourceFileParser struct {
	logger          logger.Logger
	resolverManager *resolver.ResolverManager
}

func NewSourceFileParser(logger logger.Logger) *SourceFileParser {
	resolveManager := resolver.NewResolverManager()
	return &SourceFileParser{
		logger:          logger,
		resolverManager: resolveManager,
	}
}

func (p *SourceFileParser) Parse(ctx context.Context,
	sourceFile *types.SourceFile) (*FileElementTable, error) {
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

	baseQuery, ok := BaseQueries[langParser.Language]
	if !ok {
		return nil, lang.ErrQueryNotFound
	}
	// TODO baseQuery永远不会关闭，影响？

	captureNames := baseQuery.CaptureNames() // 根据scm文件从上到下排列的
	if len(captureNames) == 0 {
		return nil, fmt.Errorf("tree_sitter base_processor query capture names is empty")
	}

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(baseQuery, tree.RootNode(), content)

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
		elems, err := p.processNode(ctx, langParser.Language, match, captureNames, sourceFile)
		if err != nil {
			p.logger.Debug("tree_sitter base processor processNode error: %v", err)
			continue // 跳过错误的匹配
		}

		for _, element := range elems {
			// 去重，主要针对variable
			if position, ok := visited[element.GetName()]; ok && isSamePosition(position, element.GetRange()) {
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
	return &FileElementTable{
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
	sourceFile *types.SourceFile) ([]resolver.Element, error) {

	if len(match.Captures) == 0 || len(captureNames) == 0 {
		return nil, lang.ErrNoCaptures
	} // root node
	rootIndex := match.Captures[0].Index
	rootCaptureName := captureNames[rootIndex]

	rootElement := newRootElement(rootCaptureName, rootIndex)
	rootElement.SetPath(sourceFile.Path)
	resolvedElements := make([]resolver.Element, 0)

	resolveCtx := &resolver.ResolveContext{
		Language:     language,
		Match:        match,
		CaptureNames: captureNames,
		SourceFile:   sourceFile,
		Logger:       p.logger,
	}
	elements, err := p.resolverManager.Resolve(ctx, rootElement, resolveCtx)
	if err != nil {
		// TODO full_name（import）、 find identifier recur (variable)、parameters/arguments
		p.logger.Debug("parse match err: %v", err)
	}
	resolvedElements = append(resolvedElements, elements...)

	return resolvedElements, nil
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
