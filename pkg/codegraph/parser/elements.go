package parser

import (
	"codebase-indexer/pkg/codegraph/resolver"
)

type ParsedSource struct {
	Path     string
	Package  *resolver.Package
	Imports  []*resolver.Import
	Language resolver.Language
	Elements []resolver.Element
}

func newRootElement(elementTypeValue string, rootIndex uint32) resolver.Element {
	elementType := resolver.ToElementType(elementTypeValue)
	base := resolver.NewBaseElement(rootIndex)
	switch elementType {
	case resolver.ElementTypePackage:
		base.Type = resolver.ElementTypePackage
		return &resolver.Package{BaseElement: base}
	case resolver.ElementTypeImport:
		base.Type = resolver.ElementTypeImport
		return &resolver.Import{BaseElement: base}
	case resolver.ElementTypeFunction:
		base.Type = resolver.ElementTypeFunction
		return &resolver.Function{BaseElement: base}
	case resolver.ElementTypeClass:
		base.Type = resolver.ElementTypeClass
		return &resolver.Class{BaseElement: base}
	case resolver.ElementTypeMethod:
		base.Type = resolver.ElementTypeMethod
		return &resolver.Method{BaseElement: base}
	case resolver.ElementTypeFunctionCall:
		base.Type = resolver.ElementTypeFunctionCall
		return &resolver.Call{BaseElement: base}
	case resolver.ElementTypeMethodCall:
		base.Type = resolver.ElementTypeMethodCall
		return &resolver.Call{BaseElement: base}
	case resolver.ElementTypeInterface:
		base.Type = resolver.ElementTypeInterface
		return &resolver.Interface{BaseElement: base}
	default:
		base.Type = resolver.ElementTypeUndefined
		return base
	}
}
