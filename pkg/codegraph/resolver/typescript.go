package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
)

type TypeScriptResolver struct {
}

var _ ElementResolver = &TypeScriptResolver{}

func (ts *TypeScriptResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, ts, element, rc)
}

func (ts *TypeScriptResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	if element.Name == types.EmptyString {
		return nil, fmt.Errorf("import is empty")
	}

	return []Element{element}, nil

}

func (ts *TypeScriptResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (ts *TypeScriptResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (ts *TypeScriptResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (ts *TypeScriptResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (ts *TypeScriptResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (ts *TypeScriptResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (ts *TypeScriptResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}
