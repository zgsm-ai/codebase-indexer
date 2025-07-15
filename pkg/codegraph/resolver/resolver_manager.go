package resolver

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/project"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

type ResolveContext struct {
	Language     lang.Language
	Match        *treesitter.QueryMatch
	CaptureNames []string // 通过Match.Capture.Index获取captureName
	SourceFile   *types.SourceFile
	ProjectInfo  *project.ProjectInfo
}

// 解析器管理器
type ResolverManager struct {
	resolvers map[lang.Language]ElementResolver
}

// 新建解析器管理器
func NewResolverManager() *ResolverManager {
	manager := &ResolverManager{
		resolvers: make(map[lang.Language]ElementResolver),
	}

	manager.register(lang.Java, &JavaResolver{})
	manager.register(lang.Python, &PythonResolver{})
	manager.register(lang.Go, &GoResolver{})
	manager.register(lang.C, &CppResolver{})
	manager.register(lang.CPP, &CppResolver{})
	manager.register(lang.JavaScript, &JavaScriptResolver{})
	manager.register(lang.TypeScript, &JavaScriptResolver{})

	return manager

}

// 注册解析器
func (rm *ResolverManager) register(language lang.Language, resolver ElementResolver) {
	rm.resolvers[language] = resolver
}

func (rm *ResolverManager) Resolve(
	ctx context.Context,
	element Element,
	rc *ResolveContext) ([]Element, error) {

	r, ok := rm.resolvers[rc.Language]
	if !ok {
		return nil, fmt.Errorf("element_resolver unsupported language: %s", rc.Language)
	}
	return r.Resolve(ctx, element, rc)
}
