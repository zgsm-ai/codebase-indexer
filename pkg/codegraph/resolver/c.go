package resolver

import (
	"context"
	"fmt"
)

type CResolver struct {
}

var _ ElementResolver = &CResolver{}

func (r *CResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, r, element, rc)
}

func (r *CResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}

func (r *CResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}

func (r *CResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}

func (r *CResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}

func (r *CResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}

func (r *CResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}

func (r *CResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}

func (r *CResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("using the C++ parser for processing")
}
