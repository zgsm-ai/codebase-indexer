package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
)

type ElementResolver interface {
	Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error)
	resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error)
	resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error)
	resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error)
	resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error)
	resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error)
	resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error)
	resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error)
	resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error)
}

func resolve(ctx context.Context, b ElementResolver, element Element, rc *ResolveContext) ([]Element, error) {
	switch element := element.(type) {
	case *Import:
		return b.resolveImport(ctx, element, rc)
	case *Package:
		return b.resolvePackage(ctx, element, rc)
	case *Function:
		return b.resolveFunction(ctx, element, rc)
	case *Method:
		return b.resolveMethod(ctx, element, rc)
	case *Class:
		return b.resolveClass(ctx, element, rc)
	case *Variable:
		return b.resolveVariable(ctx, element, rc)
	case *Interface:
		return b.resolveInterface(ctx, element, rc)
	case *Call:
		return b.resolveCall(ctx, element, rc)
	default:
		fmt.Println("element_resover not supported element", element.GetType(), "Range:", element.GetRange())
		return nil, fmt.Errorf("element_resover not supported element %v", element)
	}
}

// IsValidElement 检查必须字段
func IsValidElement(e Element) bool {
	return e.GetName() != types.EmptyString && e.GetType() != types.EmptyString &&
		e.GetPath() != types.EmptyString && len(e.GetRange()) == 4
}
