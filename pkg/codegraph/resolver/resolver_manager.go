package resolver

import (
	"context"
	"fmt"
)

// 解析器管理器
type ResolverManager struct {
	resolvers map[Language]ElementResolver
}

// 新建解析器管理器
func NewResolverManager() *ResolverManager {
	manager := &ResolverManager{
		resolvers: make(map[Language]ElementResolver),
	}

	manager.register(Java, &JavaResolver{})
	manager.register(Python, &PythonResolver{})
	manager.register(Go, &GoResolver{})
	manager.register(C, &CppResolver{})
	manager.register(CPP, &CppResolver{})
	manager.register(JavaScript, &JavaScriptResolver{})
	manager.register(TypeScript, &JavaScriptResolver{})

	return manager

}

// 注册解析器
func (rm *ResolverManager) register(language Language, resolver ElementResolver) {
	rm.resolvers[language] = resolver
}

func (rm *ResolverManager) Resolve(
	ctx context.Context,
	element Element,
	rc *ResolveContext) error {

	r, ok := rm.resolvers[rc.Language]
	if !ok {
		return fmt.Errorf("resolver unsupported language: %s", rc.Language)
	}
	return r.Resolve(ctx, element, rc)
}
