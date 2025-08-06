package proto

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"time"
)

// ElementTypeToProto 将 types.ElementType 转换为 codegraphpb.ElementType
func ElementTypeToProto(t types.ElementType) codegraphpb.ElementType {
	switch t {
	case types.ElementTypeFunction, types.ElementTypeFunctionName, types.ElementTypeFunctionDeclaration:
		return codegraphpb.ElementType_ELEMENT_TYPE_FUNCTION
	case types.ElementTypeMethod, types.ElementTypeMethodName:
		return codegraphpb.ElementType_ELEMENT_TYPE_METHOD
	case types.ElementTypeMethodCall, types.ElementTypeFunctionCall, types.ElementTypeCallName:
		return codegraphpb.ElementType_ELEMENT_TYPE_CALL
	case types.ElementTypeReference:
		return codegraphpb.ElementType_ELEMENT_TYPE_REFERENCE
	case types.ElementTypeClass, types.ElementTypeClassName:
		return codegraphpb.ElementType_ELEMENT_TYPE_CLASS
	case types.ElementTypeInterface, types.ElementTypeInterfaceName:
		return codegraphpb.ElementType_ELEMENT_TYPE_INTERFACE
	case types.ElementTypeVariable, types.ElementTypeVariableName, types.ElementTypeLocalVariable,
		types.ElementTypeLocalVariableName, types.ElementTypeGlobalVariable:
		return codegraphpb.ElementType_ELEMENT_TYPE_VARIABLE
	default:
		return codegraphpb.ElementType_ELEMENT_TYPE_UNDEFINED
	}
}

// ElementTypeFromProto 将 codegraphpb.ElementType 转换为 types.ElementType
func ElementTypeFromProto(t codegraphpb.ElementType) types.ElementType {
	switch t {
	case codegraphpb.ElementType_ELEMENT_TYPE_FUNCTION:
		return types.ElementTypeFunction
	case codegraphpb.ElementType_ELEMENT_TYPE_METHOD:
		return types.ElementTypeMethod
	case codegraphpb.ElementType_ELEMENT_TYPE_CALL:
		return types.ElementTypeMethodCall
	case codegraphpb.ElementType_ELEMENT_TYPE_REFERENCE:
		return types.ElementTypeReference
	case codegraphpb.ElementType_ELEMENT_TYPE_CLASS:
		return types.ElementTypeClass
	case codegraphpb.ElementType_ELEMENT_TYPE_INTERFACE:
		return types.ElementTypeInterface
	case codegraphpb.ElementType_ELEMENT_TYPE_VARIABLE:
		return types.ElementTypeVariable
	case codegraphpb.ElementType_ELEMENT_TYPE_UNDEFINED:
		return types.ElementTypeUndefined
	default:
		return types.ElementTypeUndefined
	}
}

// ElementTypeSliceToProto 将 []types.ElementType 转换为 []codegraphpb.ElementType
func ElementTypeSliceToProto(elementTypes []types.ElementType) []codegraphpb.ElementType {
	result := make([]codegraphpb.ElementType, len(elementTypes))
	for i, t := range elementTypes {
		result[i] = ElementTypeToProto(t)
	}
	return result
}

// ElementTypeSliceFromProto 将 []codegraphpb.ElementType 转换为 []types.ElementType
func ElementTypeSliceFromProto(elementTypes []codegraphpb.ElementType) []types.ElementType {
	result := make([]types.ElementType, len(elementTypes))
	for i, t := range elementTypes {
		result[i] = ElementTypeFromProto(t)
	}
	return result
}

// RelationTypeToProto 将 resolver.RelationType 转换为 codegraphpb.RelationType
func RelationTypeToProto(t resolver.RelationType) codegraphpb.RelationType {
	switch t {
	case resolver.RelationTypeUndefined:
		return codegraphpb.RelationType_RELATION_TYPE_UNDEFINED
	case resolver.RelationTypeDefinition:
		return codegraphpb.RelationType_RELATION_TYPE_DEFINITION
	case resolver.RelationTypeReference:
		return codegraphpb.RelationType_RELATION_TYPE_REFERENCE
	case resolver.RelationTypeInherit:
		return codegraphpb.RelationType_RELATION_TYPE_INHERIT
	case resolver.RelationTypeImplement:
		return codegraphpb.RelationType_RELATION_TYPE_IMPLEMENT
	case resolver.RelationTypeSuperClass:
		return codegraphpb.RelationType_RELATION_TYPE_SUPER_CLASS
	case resolver.RelationTypeSuperInterface:
		return codegraphpb.RelationType_RELATION_TYPE_SUPER_INTERFACE
	default:
		return codegraphpb.RelationType_RELATION_TYPE_UNDEFINED
	}
}

// RelationTypeFromProto 将 codegraphpb.RelationType 转换为 resolver.RelationType
func RelationTypeFromProto(t codegraphpb.RelationType) resolver.RelationType {
	switch t {
	case codegraphpb.RelationType_RELATION_TYPE_UNDEFINED:
		return resolver.RelationTypeUndefined
	case codegraphpb.RelationType_RELATION_TYPE_DEFINITION:
		return resolver.RelationTypeDefinition
	case codegraphpb.RelationType_RELATION_TYPE_REFERENCE:
		return resolver.RelationTypeReference
	case codegraphpb.RelationType_RELATION_TYPE_INHERIT:
		return resolver.RelationTypeInherit
	case codegraphpb.RelationType_RELATION_TYPE_IMPLEMENT:
		return resolver.RelationTypeImplement
	case codegraphpb.RelationType_RELATION_TYPE_SUPER_CLASS:
		return resolver.RelationTypeSuperClass
	case codegraphpb.RelationType_RELATION_TYPE_SUPER_INTERFACE:
		return resolver.RelationTypeSuperInterface
	default:
		return resolver.RelationTypeUndefined
	}
}

// RelationToProto 将 resolver.Relation 转换为 codegraphpb.Relation
func RelationToProto(r *resolver.Relation) *codegraphpb.Relation {
	if r == nil {
		return nil
	}

	return &codegraphpb.Relation{
		ElementName:  r.ElementName,
		ElementPath:  r.ElementPath,
		Range:        r.Range,
		RelationType: RelationTypeToProto(r.RelationType),
	}
}

// RelationFromProto 将 codegraphpb.Relation 转换为 resolver.Relation
func RelationFromProto(r *codegraphpb.Relation) *resolver.Relation {
	if r == nil {
		return nil
	}

	return &resolver.Relation{
		ElementName:  r.GetElementName(),
		ElementPath:  r.GetElementPath(),
		Range:        r.GetRange(),
		RelationType: RelationTypeFromProto(r.GetRelationType()),
	}
}

// RelationSliceToProto 将 []*resolver.Relation 转换为 []*codegraphpb.Relation
func RelationSliceToProto(relations []*resolver.Relation) []*codegraphpb.Relation {
	if relations == nil {
		return nil
	}

	result := make([]*codegraphpb.Relation, len(relations))
	for i, r := range relations {
		result[i] = RelationToProto(r)
	}
	return result
}

// FileElementTablesToProto 将 []parser.FileElementTable 转换为 []*codegraphpb.FileElementTable
func FileElementTablesToProto(fileElementTables []*parser.FileElementTable) []*codegraphpb.FileElementTable {
	if len(fileElementTables) == 0 {
		return nil
	}
	protoElementTables := make([]*codegraphpb.FileElementTable, len(fileElementTables))
	for j, ft := range fileElementTables {
		pft := &codegraphpb.FileElementTable{
			Path:      ft.Path,
			Language:  string(ft.Language),
			Timestamp: time.Now().Unix(),
			Elements:  make([]*codegraphpb.BaseElement, len(ft.Elements)),
		}
		for k, e := range ft.Elements {
			pbe := &codegraphpb.BaseElement{
				Name:        e.GetName(),
				ElementType: ElementTypeToProto(e.GetType()),
				Range:       e.GetRange(),
			}
			// 定义：class interface method function variable
			if e.GetType() == types.ElementTypeClass || e.GetType() == types.ElementTypeInterface ||
				e.GetType() == types.ElementTypeMethod || e.GetType() == types.ElementTypeFunction ||
				e.GetType() == types.ElementTypeVariable {
				pbe.IsDefinition = true
			}

			for _, r := range e.GetRelations() {
				pbe.Relations = append(pbe.Relations, RelationToProto(r))
			}
			pft.Elements[k] = pbe
		}

		protoElementTables[j] = pft
	}
	return protoElementTables
}

// RelationSliceFromProto 将 []*codegraphpb.Relation 转换为 []*resolver.Relation
func RelationSliceFromProto(relations []*codegraphpb.Relation) []*resolver.Relation {
	if relations == nil {
		return nil
	}

	result := make([]*resolver.Relation, len(relations))
	for i, r := range relations {
		result[i] = RelationFromProto(r)
	}
	return result
}
