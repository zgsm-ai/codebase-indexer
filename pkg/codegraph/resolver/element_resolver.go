package resolver

import (
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
}

func resolve(ctx context.Context, b ElementResolver, element Element, rc *ResolveContext) ([]Element, error) {
	switch element.(type) {
	case *Import:
		return b.resolveImport(ctx, element.(*Import), rc)
	case *Package:
		return b.resolvePackage(ctx, element.(*Package), rc)
	case *Function:
		return b.resolveFunction(ctx, element.(*Function), rc)
	case *Method:
		return b.resolveMethod(ctx, element.(*Method), rc)
	case *Class:
		return b.resolveClass(ctx, element.(*Class), rc)
	case *Variable:
		return b.resolveVariable(ctx, element.(*Variable), rc)
	case *Interface:
		return b.resolveInterface(ctx, element.(*Interface), rc)
	default:
		return nil, fmt.Errorf("element_resover not supported element %v", element)
	}
}
