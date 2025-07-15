package parser

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
)

type FileSymbolTable struct {
	Path     string
	Package  *resolver.Package
	Imports  []*resolver.Import
	Language lang.Language
	Elements []resolver.Element
}

func newRootElement(elementTypeValue string, rootIndex uint32) resolver.Element {
	elementType := types.ToElementType(elementTypeValue)
	base := resolver.NewBaseElement(rootIndex)
	switch elementType {
	case types.ElementTypePackage:
		base.Type = types.ElementTypePackage
		return &resolver.Package{BaseElement: base}
	case types.ElementTypeImport:
		base.Type = types.ElementTypeImport
		return &resolver.Import{BaseElement: base}
	case types.ElementTypeFunction:
		base.Type = types.ElementTypeFunction
		return &resolver.Function{BaseElement: base}
	case types.ElementTypeClass:
		base.Type = types.ElementTypeClass
		return &resolver.Class{BaseElement: base}
	case types.ElementTypeMethod:
		base.Type = types.ElementTypeMethod
		return &resolver.Method{BaseElement: base}
	case types.ElementTypeFunctionCall:
		base.Type = types.ElementTypeFunctionCall
		return &resolver.Call{BaseElement: base}
	case types.ElementTypeMethodCall:
		base.Type = types.ElementTypeMethodCall
		return &resolver.Call{BaseElement: base}
	case types.ElementTypeInterface:
		base.Type = types.ElementTypeInterface
		return &resolver.Interface{BaseElement: base}
	default:
		base.Type = types.ElementTypeUndefined
		return base
	}
}
