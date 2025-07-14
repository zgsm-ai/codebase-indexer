package parser

import (
	"codebase-indexer/pkg/codegraph/parser/resolver"
	"strings"
)

const (
	dotName       = ".name"
	dotArguments  = ".arguments"
	dotParameters = ".parameters"
	dotOwner      = ".owner"
	dotSource     = ".source"
	dotAlias      = ".alias"
)

//
//// 类型映射表 - captureName -> ElementType
//var TypeMappings = map[string]ElementType{
//	"package":                ElementTypePackage,
//	"namespace":              ElementTypeNamespace,
//	"import":                 ElementTypeImport,
//	"declaration.function":   ElementTypeFunctionDeclaration,
//	"definition.method":      ElementTypeMethod,
//	"call.method":            ElementTypeMethodCall,
//	"definition.function":    ElementTypeFunction,
//	"call.function":          ElementTypeFunctionCall,
//	"definition.class":       ElementTypeClass,
//	"definition.interface":   ElementTypeInterface,
//	"definition.struct":      ElementTypeStruct,
//	"definition.enum":        ElementTypeEnum,
//	"definition.union":       ElementTypeUnion,
//	"definition.trait":       ElementTypeTrait,
//	"definition.type_alias":  ElementTypeTypeAlias,
//	"definition.constructor": ElementTypeConstructor,
//	"definition.destructor":  ElementTypeDestructor,
//	"global_variable":        ElementTypeGlobalVariable,
//	"local_variable":         ElementTypeLocalVariable,
//	"variable":               ElementTypeVariable,
//	"constant":               ElementTypeConstant,
//	"macro":                  ElementTypeMacro,
//	"definition.field":       ElementTypeField,
//	"definition.parameter":   ElementTypeParameter,
//	"comment":                ElementTypeComment,
//	"doc_comment":            ElementTypeDocComment,
//	"annotation":             ElementTypeAnnotation,
//	"undefined":              ElementTypeUndefined,
//}

// toElementType 将字符串映射为ElementType
func toElementType(captureName string) resolver.ElementType {
	if captureName == EmptyString {
		return resolver.ElementTypeUndefined
	}
	if et, exists := TypeMappings[captureName]; exists {
		return et
	}
	return resolver.ElementTypeUndefined
}

// 函数工厂：生成检查字符串是否以特定后缀结尾的函数
func createSuffixChecker(suffix string) func(string) bool {
	return func(captureName string) bool {
		return strings.HasSuffix(captureName, suffix)
	}
}

// 使用工厂函数创建检查器
var (
	isNameCapture       = createSuffixChecker(dotName)
	isParametersCapture = createSuffixChecker(dotParameters)
	isArgumentsCapture  = createSuffixChecker(dotArguments)
	isOwnerCapture      = createSuffixChecker(dotOwner)
	isSourceCapture     = createSuffixChecker(dotSource)
	isAliasCapture      = createSuffixChecker(dotAlias)
)

// 特殊函数（需要额外判断）保留
func isElementNameCapture(elementType resolver.ElementType, captureName string) bool {
	return isNameCapture(captureName) &&
		captureName == string(elementType)+dotName
}
