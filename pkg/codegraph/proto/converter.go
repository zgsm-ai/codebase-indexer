package proto

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ElementTypeToProto 将 types.ElementType 转换为 codegraphpb.ElementType
func ElementTypeToProto(t types.ElementType) codegraphpb.ElementType {
	switch t {
	case types.ElementTypeFunction, types.ElementTypeFunctionName, types.ElementTypeFunctionDeclaration:
		return codegraphpb.ElementType_FUNCTION
	case types.ElementTypeMethod, types.ElementTypeMethodName:
		return codegraphpb.ElementType_METHOD
	case types.ElementTypeMethodCall, types.ElementTypeFunctionCall, types.ElementTypeCallName:
		return codegraphpb.ElementType_CALL
	case types.ElementTypeReference:
		return codegraphpb.ElementType_REFERENCE
	case types.ElementTypeClass, types.ElementTypeClassName:
		return codegraphpb.ElementType_CLASS
	case types.ElementTypeInterface, types.ElementTypeInterfaceName:
		return codegraphpb.ElementType_INTERFACE
	case types.ElementTypeVariable, types.ElementTypeVariableName, types.ElementTypeLocalVariable,
		types.ElementTypeLocalVariableName, types.ElementTypeGlobalVariable:
		return codegraphpb.ElementType_VARIABLE
	default:
		return codegraphpb.ElementType_UNDEFINED
	}
}

// ElementTypeFromProto 将 codegraphpb.ElementType 转换为 types.ElementType
func ElementTypeFromProto(t codegraphpb.ElementType) types.ElementType {
	switch t {
	case codegraphpb.ElementType_FUNCTION:
		return types.ElementTypeFunction
	case codegraphpb.ElementType_METHOD:
		return types.ElementTypeMethod
	case codegraphpb.ElementType_CALL:
		return types.ElementTypeMethodCall
	case codegraphpb.ElementType_REFERENCE:
		return types.ElementTypeReference
	case codegraphpb.ElementType_CLASS:
		return types.ElementTypeClass
	case codegraphpb.ElementType_INTERFACE:
		return types.ElementTypeInterface
	case codegraphpb.ElementType_VARIABLE:
		return types.ElementTypeVariable
	case codegraphpb.ElementType_UNDEFINED:
		return types.ElementTypeUndefined
	default:
		return types.ElementTypeUndefined
	}
}

//// ElementTypeSliceToProto 将 []types.ElementType 转换为 []codegraphpb.ElementType
//func ElementTypeSliceToProto(elementTypes []types.ElementType) []codegraphpb.ElementType {
//	result := make([]codegraphpb.ElementType, len(elementTypes))
//	for i, t := range elementTypes {
//		result[i] = ElementTypeToProto(t)
//	}
//	return result
//}
//
//// ElementTypeSliceFromProto 将 []codegraphpb.ElementType 转换为 []types.ElementType
//func ElementTypeSliceFromProto(elementTypes []codegraphpb.ElementType) []types.ElementType {
//	result := make([]types.ElementType, len(elementTypes))
//	for i, t := range elementTypes {
//		result[i] = ElementTypeFromProto(t)
//	}
//	return result
//}
//
//// RelationTypeToProto 将 resolver.RelationType 转换为 codegraphpb.RelationType
//func RelationTypeToProto(t resolver.RelationType) codegraphpb.RelationType {
//	switch t {
//	case resolver.RelationTypeUndefined:
//		return codegraphpb.RelationType_RELATION_TYPE_UNDEFINED
//	case resolver.RelationTypeDefinition:
//		return codegraphpb.RelationType_RELATION_TYPE_DEFINITION
//	case resolver.RelationTypeReference:
//		return codegraphpb.RelationType_RELATION_TYPE_REFERENCE
//	case resolver.RelationTypeInherit:
//		return codegraphpb.RelationType_RELATION_TYPE_INHERIT
//	case resolver.RelationTypeImplement:
//		return codegraphpb.RelationType_RELATION_TYPE_IMPLEMENT
//	case resolver.RelationTypeSuperClass:
//		return codegraphpb.RelationType_RELATION_TYPE_SUPER_CLASS
//	case resolver.RelationTypeSuperInterface:
//		return codegraphpb.RelationType_RELATION_TYPE_SUPER_INTERFACE
//	default:
//		return codegraphpb.RelationType_RELATION_TYPE_UNDEFINED
//	}
//}
//
//// RelationTypeFromProto 将 codegraphpb.RelationType 转换为 resolver.RelationType
//func RelationTypeFromProto(t codegraphpb.RelationType) resolver.RelationType {
//	switch t {
//	case codegraphpb.RelationType_RELATION_TYPE_UNDEFINED:
//		return resolver.RelationTypeUndefined
//	case codegraphpb.RelationType_RELATION_TYPE_DEFINITION:
//		return resolver.RelationTypeDefinition
//	case codegraphpb.RelationType_RELATION_TYPE_REFERENCE:
//		return resolver.RelationTypeReference
//	case codegraphpb.RelationType_RELATION_TYPE_INHERIT:
//		return resolver.RelationTypeInherit
//	case codegraphpb.RelationType_RELATION_TYPE_IMPLEMENT:
//		return resolver.RelationTypeImplement
//	case codegraphpb.RelationType_RELATION_TYPE_SUPER_CLASS:
//		return resolver.RelationTypeSuperClass
//	case codegraphpb.RelationType_RELATION_TYPE_SUPER_INTERFACE:
//		return resolver.RelationTypeSuperInterface
//	default:
//		return resolver.RelationTypeUndefined
//	}
//}
//
//// RelationToProto 将 resolver.Relation 转换为 codegraphpb.Relation
//func RelationToProto(r *resolver.Relation) *codegraphpb.Relation {
//	if r == nil {
//		return nil
//	}
//
//	return &codegraphpb.Relation{
//		ElementName:  r.ElementName,
//		ElementPath:  r.ElementPath,
//		Range:        r.Range,
//		RelationType: RelationTypeToProto(r.RelationType),
//	}
//}

//// RelationFromProto 将 codegraphpb.Relation 转换为 resolver.Relation
//func RelationFromProto(r *codegraphpb.Relation) *resolver.Relation {
//	if r == nil {
//		return nil
//	}
//
//	return &resolver.Relation{
//		ElementName:  r.GetElementName(),
//		ElementPath:  r.GetElementPath(),
//		Range:        r.GetRange(),
//		RelationType: RelationTypeFromProto(r.GetRelationType()),
//	}
//}

//// RelationSliceToProto 将 []*resolver.Relation 转换为 []*codegraphpb.Relation
//func RelationSliceToProto(relations []*resolver.Relation) []*codegraphpb.Relation {
//	if relations == nil {
//		return nil
//	}
//
//	result := make([]*codegraphpb.Relation, len(relations))
//	for i, r := range relations {
//		result[i] = RelationToProto(r)
//	}
//	return result
//}

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
			Elements:  make([]*codegraphpb.Element, len(ft.Elements)),
			Imports:   make([]*codegraphpb.Import, len(ft.Imports)),
		}
		if ft.Package != nil {
			pft.Package = &codegraphpb.Package{Name: ft.Package.Name, Range: ft.Package.Range}
		}

		for i, imp := range ft.Imports {
			pft.Imports[i] = &codegraphpb.Import{Name: imp.Name, Source: imp.Source,
				Alias: imp.Alias, Range: imp.Range}
		}

		for k, e := range ft.Elements {
			pbe := &codegraphpb.Element{
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

			//for _, r := range e.GetRelations() {
			//	pbe.Relations = append(pbe.Relations, RelationToProto(r))
			//}
			// extra_data
			extraData, err := MarshalExtraData(e)
			if err != nil {
				// TODO remove this debug info
				fmt.Printf("marshal extra data error:%v", err)
			} else {
				pbe.ExtraData = extraData
			}
			pft.Elements[k] = pbe
		}

		protoElementTables[j] = pft
	}
	return protoElementTables
}

//
//// RelationSliceFromProto 将 []*codegraphpb.Relation 转换为 []*resolver.Relation
//func RelationSliceFromProto(relations []*codegraphpb.Relation) []*resolver.Relation {
//	if relations == nil {
//		return nil
//	}
//
//	result := make([]*resolver.Relation, len(relations))
//	for i, r := range relations {
//		result[i] = RelationFromProto(r)
//	}
//	return result
//}

const (
	keyParameters      = "parameters"
	keyReturnType      = "returnType"
	keySuperClasses    = "superClasses"
	keySuperInterfaces = "superInterfaces"
)

func GetParametersFromExtraData(extraData map[string][]byte) (parameters []resolver.Parameter, err error) {
	parametersBytes, ok := extraData[keyParameters]
	if !ok {
		return
	}
	err = json.Unmarshal(parametersBytes, &parameters)
	return
}

func GetReturnTypeFromExtraData(extraData map[string][]byte) (returnType []string, err error) {
	returnTypeBytes, ok := extraData[keyReturnType]
	if !ok {
		return
	}
	err = json.Unmarshal(returnTypeBytes, &returnType)
	return
}

func GetSuperClassesFromExtraData(extraData map[string][]byte) (superClasses []string, err error) {
	superClassesBytes, ok := extraData[keySuperClasses]
	if !ok {
		return
	}
	err = json.Unmarshal(superClassesBytes, &superClasses)
	return
}

func GetSuperInterfacesFromExtraData(extraData map[string][]byte) (superInterfaces []string, err error) {
	superInterfacesBytes, ok := extraData[keySuperInterfaces]
	if !ok {
		return
	}
	err = json.Unmarshal(superInterfacesBytes, &superInterfaces)
	return
}

func MarshalExtraData(element resolver.Element) (map[string][]byte, error) {
	var errs []error
	extraData := make(map[string][]byte)

	switch e := element.(type) {
	case *resolver.Import, *resolver.Package, *resolver.Variable:
		// 无需处理的类型
	case *resolver.Function:
		if len(e.Parameters) > 0 {
			// 处理函数共有的参数和返回类型
			parametersBytes, err := json.Marshal(e.Parameters)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyParameters] = parametersBytes
			}
		}

		if len(e.ReturnType) > 0 {
			returnTypeBytes, err := json.Marshal(e.ReturnType)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyReturnType] = returnTypeBytes
			}
		}

	case *resolver.Method:
		// 处理方法共有的参数和返回类型
		if len(e.Parameters) > 0 {
			parametersBytes, err := json.Marshal(e.Parameters)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyParameters] = parametersBytes
			}
		}

		if len(e.ReturnType) > 0 {
			returnTypeBytes, err := json.Marshal(e.ReturnType)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyReturnType] = returnTypeBytes
			}
		}

	case *resolver.Class:
		if len(e.SuperClasses) > 0 {
			superClassesBytes, err := json.Marshal(e.SuperClasses)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keySuperClasses] = superClassesBytes
			}
		}

		if len(e.SuperInterfaces) > 0 {
			superInterfacesBytes, err := json.Marshal(e.SuperInterfaces)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keySuperInterfaces] = superInterfacesBytes
			}
		}

	case *resolver.Interface:
		if len(e.SuperInterfaces) > 0 {
			superInterfacesBytes, err := json.Marshal(e.SuperInterfaces)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keySuperInterfaces] = superInterfacesBytes
			}
		}

	case *resolver.Call:
		if len(e.Parameters) > 0 {
			parametersBytes, err := json.Marshal(e.Parameters)
			if err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyParameters] = parametersBytes
			}
		}
	}

	return extraData, errors.Join(errs...)
}

func UnMarshalExtraData(element *codegraphpb.Element) (map[string]any, error) {
	var errs []error
	extraData := make(map[string]any)
	extraDataRaw := element.ExtraData

	if len(extraDataRaw) == 0 {
		return extraData, nil
	}

	switch element.ElementType {
	case codegraphpb.ElementType_IMPORT, codegraphpb.ElementType_PACKAGE, codegraphpb.ElementType_VARIABLE:
		// 无需处理的类型
	case codegraphpb.ElementType_FUNCTION, codegraphpb.ElementType_METHOD:
		// 处理函数和方法共有的参数和返回类型
		if parametersBytes, ok := extraDataRaw[keyParameters]; ok {
			var params []resolver.Parameter
			if err := json.Unmarshal(parametersBytes, &params); err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyParameters] = params
			}
		}

		if returnTypeBytes, ok := extraDataRaw[keyReturnType]; ok {
			var returnType []string
			if err := json.Unmarshal(returnTypeBytes, &returnType); err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyReturnType] = returnType
			}
		}

	case codegraphpb.ElementType_CLASS:
		if superClassesBytes, ok := extraDataRaw[keySuperClasses]; ok {
			var superClasses []resolver.Parameter
			if err := json.Unmarshal(superClassesBytes, &superClasses); err != nil {
				errs = append(errs, err)
			} else {
				extraData[keySuperClasses] = superClasses
			}
		}

		if superInterfacesBytes, ok := extraDataRaw[keySuperInterfaces]; ok {
			var superInterfaces []resolver.Parameter
			if err := json.Unmarshal(superInterfacesBytes, &superInterfaces); err != nil {
				errs = append(errs, err)
			} else {
				extraData[keySuperInterfaces] = superInterfaces
			}
		}

	case codegraphpb.ElementType_INTERFACE:
		if superInterfacesBytes, ok := extraDataRaw[keySuperInterfaces]; ok {
			var superInterfaces []resolver.Parameter
			if err := json.Unmarshal(superInterfacesBytes, &superInterfaces); err != nil {
				errs = append(errs, err)
			} else {
				extraData[keySuperInterfaces] = superInterfaces
			}
		}

	case codegraphpb.ElementType_CALL:
		if parametersBytes, ok := extraDataRaw[keyParameters]; ok {
			var params []resolver.Parameter
			if err := json.Unmarshal(parametersBytes, &params); err != nil {
				errs = append(errs, err)
			} else {
				extraData[keyParameters] = params
			}
		}
	}

	return extraData, errors.Join(errs...)
}
