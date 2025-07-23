package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
)

type PythonResolver struct {
}

var _ ElementResolver = &PythonResolver{}

func (py *PythonResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, py, element, rc)
}

func (py *PythonResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	if element.Name == types.EmptyString {
		return nil, fmt.Errorf("import is empty")
	}

	return []Element{element}, nil

}

func (py *PythonResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}
