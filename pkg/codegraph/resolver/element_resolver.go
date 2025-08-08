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

func resolve(ctx context.Context, b ElementResolver, element Element, rc *ResolveContext) (elems []Element, err error) {
	switch element := element.(type) {
	case *Import:
		elems, err = b.resolveImport(ctx, element, rc)
	case *Package:
		elems, err = b.resolvePackage(ctx, element, rc)
	case *Function:
		elems, err = b.resolveFunction(ctx, element, rc)
	case *Method:
		elems, err = b.resolveMethod(ctx, element, rc)
	case *Class:
		elems, err = b.resolveClass(ctx, element, rc)
	case *Variable:
		elems, err = b.resolveVariable(ctx, element, rc)
	case *Interface:
		elems, err = b.resolveInterface(ctx, element, rc)
	case *Call:
		elems, err = b.resolveCall(ctx, element, rc )
	default:
		rootCap := rc.Match.Captures[0]
		updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
		err = fmt.Errorf("[resolver] unsupported element: type=%s | path=%s | range=%v ",
			element.GetType(), element.GetPath(), element.GetRange())
		return nil, err
	}
	return FilterValidElems(elems, rc.Logger), err
}

// IsValidElement 检查必须字段
func IsValidElement(e Element) bool {
	return e.GetName() != types.EmptyString && e.GetType() != types.EmptyString &&
		e.GetPath() != types.EmptyString && len(e.GetRange()) == 4 && IsValidElementType(e)
}

func IsValidElementType(e Element) bool {
	switch element := e.(type) {
	case *Import:
		return element.Type == types.ElementTypeImport
	case *Package:
		return element.Type == types.ElementTypePackage
	case *Function:
		return element.Type == types.ElementTypeFunction
	case *Method:
		return element.Type == types.ElementTypeMethod
	case *Class:
		return element.Type == types.ElementTypeClass
	case *Variable:
		return element.Type == types.ElementTypeVariable
	case *Interface:
		return element.Type == types.ElementTypeInterface
	case *Call:
		return element.Type == types.ElementTypeFunctionCall ||
			element.Type == types.ElementTypeMethodCall
	case *Reference:
		return element.Type == types.ElementTypeReference
	default:
		return false
	}
}
