package resolver

import (
	"context"
	"fmt"
)

type ElementResolver interface {
	Resolve(ctx context.Context, element Element, rc *ResolveContext) error
	resolveImport(ctx context.Context, element Import, rc *ResolveContext) error
	resolvePackage(ctx context.Context, element Package, rc *ResolveContext) error
	resolveFunction(ctx context.Context, element Function, rc *ResolveContext) error
	resolveMethod(ctx context.Context, element Method, rc *ResolveContext) error
	resolveClass(ctx context.Context, element Class, rc *ResolveContext) error
	resolveVariable(ctx context.Context, element Variable, rc *ResolveContext) error
	resolveInterface(ctx context.Context, element Interface, rc *ResolveContext) error
}

func resolve(ctx context.Context, b ElementResolver, element Element, rc *ResolveContext) error {
	switch element.(type) {
	case Import:
		return b.resolveImport(ctx, element.(Import), rc)
	case Package:
		return b.resolvePackage(ctx, element.(Package), rc)
	case Function:
		return b.resolveFunction(ctx, element.(Function), rc)
	case Method:
		return b.resolveMethod(ctx, element.(Method), rc)
	case Class:
		return b.resolveClass(ctx, element.(Class), rc)
	case Variable:
		return b.resolveVariable(ctx, element.(Variable), rc)
	default:
		return fmt.Errorf("element_resover not supported element %v", element)
	}
}
