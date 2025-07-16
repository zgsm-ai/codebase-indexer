package types

const (
	EmptyString = ""
	DoubleQuote = "\""
	Comma       = ","
	Identifier  = "identifier"
	Dot         = "."
)

// ElementType 表示代码元素类型，使用字符串字面量作为枚举值
type ElementType string

const (
	ElementTypeNamespace           ElementType = "namespace"
	ElementTypePackage             ElementType = "package"
	ElementTypePackageName         ElementType = "package.name"
	ElementTypeUndefined           ElementType = "undefined"
	ElementTypeImport              ElementType = "import"
	ElementTypeImportName          ElementType = "import.name"
	ElementTypeClass               ElementType = "definition.class"
	ElementTypeClassName           ElementType = "definition.class.name"
	ElementTypeClassExtends        ElementType = "definition.class.extends"
	ElementTypeClassImplements     ElementType = "definition.class.implements"
	ElementTypeInterface           ElementType = "definition.interface"
	ElementTypeStruct              ElementType = "definition.struct"
	ElementTypeEnum                ElementType = "definition.enum"
	ElementTypeUnion               ElementType = "definition.union"
	ElementTypeTrait               ElementType = "definition.trait"
	ElementTypeTypeAlias           ElementType = "definition.type_alias"
	ElementTypeFunction            ElementType = "definition.function"
	ElementTypeMethodCall          ElementType = "call.method"
	ElementTypeFunctionCall        ElementType = "call.function"
	ElementTypeFunctionDeclaration ElementType = "declaration.function"
	ElementTypeMethod              ElementType = "definition.method"
	ElementTypeMethodModifier      ElementType = "definition.method.modifier"
	ElementTypeMethodReturnType    ElementType = "definition.method.return_type"
	ElementTypeMethodName          ElementType = "definition.method.name"
	ElementTypeMethodParameters    ElementType = "definition.method.parameters"
	ElementTypeConstructor         ElementType = "definition.constructor"
	ElementTypeDestructor          ElementType = "definition.destructor"
	ElementTypeGlobalVariable      ElementType = "global_variable"
	ElementTypeLocalVariable       ElementType = "local_variable"
	ElementTypeLocalVariableName   ElementType = "local_variable.name"
	ElementTypeVariable            ElementType = "variable"
	ElementTypeConstant            ElementType = "constant"
	ElementTypeMacro               ElementType = "macro"
	ElementTypeField               ElementType = "definition.field"
	ElementTypeFieldName           ElementType = "definition.field.name"
	ElementTypeFieldType           ElementType = "definition.field.type"
	ElementTypeFieldModifier       ElementType = "definition.field.modifier"
	ElementTypeParameter           ElementType = "definition.parameter"
	ElementTypeComment             ElementType = "comment"
	ElementTypeAnnotation          ElementType = "annotation"
	ElementTypeReference           ElementType = "reference"
)

// TypeMappings 类型映射表 - captureName -> ElementType（使用ElementType字符串值作为键）
var TypeMappings = map[string]ElementType{
	string(ElementTypeNamespace):           ElementTypeNamespace,
	string(ElementTypePackage):             ElementTypePackage,
	string(ElementTypePackageName):         ElementTypePackageName,
	string(ElementTypeUndefined):           ElementTypeUndefined,
	string(ElementTypeImport):              ElementTypeImport,
	string(ElementTypeImportName):          ElementTypeImportName,
	string(ElementTypeClass):               ElementTypeClass,
	string(ElementTypeClassName):           ElementTypeClassName,
	string(ElementTypeClassExtends):        ElementTypeClassExtends,
	string(ElementTypeClassImplements):     ElementTypeClassImplements,
	string(ElementTypeInterface):           ElementTypeInterface,
	string(ElementTypeStruct):              ElementTypeStruct,
	string(ElementTypeEnum):                ElementTypeEnum,
	string(ElementTypeUnion):               ElementTypeUnion,
	string(ElementTypeTrait):               ElementTypeTrait,
	string(ElementTypeTypeAlias):           ElementTypeTypeAlias,
	string(ElementTypeFunction):            ElementTypeFunction,
	string(ElementTypeFunctionCall):        ElementTypeFunctionCall,
	string(ElementTypeFunctionDeclaration): ElementTypeFunctionDeclaration,
	string(ElementTypeMethod):              ElementTypeMethod,
	string(ElementTypeMethodCall):          ElementTypeMethodCall,
	string(ElementTypeConstructor):         ElementTypeConstructor,
	string(ElementTypeDestructor):          ElementTypeDestructor,
	string(ElementTypeGlobalVariable):      ElementTypeGlobalVariable,
	string(ElementTypeLocalVariable):       ElementTypeLocalVariable,
	string(ElementTypeLocalVariableName):   ElementTypeLocalVariableName,
	string(ElementTypeVariable):            ElementTypeVariable,
	string(ElementTypeConstant):            ElementTypeConstant,
	string(ElementTypeMacro):               ElementTypeMacro,
	string(ElementTypeField):               ElementTypeField,
	string(ElementTypeParameter):           ElementTypeParameter,
	string(ElementTypeComment):             ElementTypeComment,
	string(ElementTypeAnnotation):          ElementTypeAnnotation,
}

type Scope string

const (
	ScopeBlock    Scope = "block"
	ScopeFunction Scope = "function"
	ScopeClass    Scope = "class"
	ScopeFile     Scope = "file"
	ScopePackage  Scope = "package"
	ScopeProject  Scope = "project"
)

type SourceFile struct {
	ClientId     string
	CodebasePath string
	CodebaseName string
	Name         string
	Path         string
	Content      []byte
}
type NodeKind string

const (
	NodeKindField              NodeKind = "field_declaration"
	NodeKindMethod             NodeKind = "method_declaration"
	NodeKindVariableDeclarator NodeKind = "variable_declarator"
	NodeKindModifier           NodeKind = "modifiers"
	NodeKindIdentifier         NodeKind = "identifier"
	NodeKindFormalParameters   NodeKind = "formal_parameters"
	NodeKindUndefined          NodeKind = "undefined"

	// 用于接收函数的返回类型和字段的类型
	NodeKindIntegralType         NodeKind = "integral_type"
	NodeKindFloatingPointType    NodeKind = "floating_point_type"
	NodeKindBooleanType          NodeKind = "boolean_type"
	NodeKindCharType             NodeKind = "char_type"
	NodeKindVoidType             NodeKind = "void_type"
	NodeKindArrayType            NodeKind = "array_type"
	NodeKindGenericType          NodeKind = "generic_type"
	NodeKindTypeIdentifier       NodeKind = "type_identifier"
	NodeKindScopedTypeIdentifier NodeKind = "scoped_type_identifier"
	NodeKindWildcard             NodeKind = "wildcard"
)

var NodeKindMappings = map[string]NodeKind{
	string(NodeKindField):              NodeKindField,
	string(NodeKindMethod):             NodeKindMethod,
	string(NodeKindUndefined):          NodeKindUndefined,
	string(NodeKindVariableDeclarator): NodeKindVariableDeclarator,
	string(NodeKindModifier):           NodeKindModifier,
	string(NodeKindIdentifier):         NodeKindIdentifier,
	string(NodeKindFormalParameters):   NodeKindFormalParameters,

	// 用于接收函数的返回类型和字段的类型
	string(NodeKindIntegralType):         NodeKindIntegralType,
	string(NodeKindFloatingPointType):    NodeKindFloatingPointType,
	string(NodeKindBooleanType):          NodeKindBooleanType,
	string(NodeKindCharType):             NodeKindCharType,
	string(NodeKindVoidType):             NodeKindVoidType,
	string(NodeKindArrayType):            NodeKindArrayType,
	string(NodeKindGenericType):          NodeKindGenericType,
	string(NodeKindTypeIdentifier):       NodeKindTypeIdentifier,
	string(NodeKindScopedTypeIdentifier): NodeKindScopedTypeIdentifier,
	string(NodeKindWildcard):             NodeKindWildcard,
}

// 用于接收函数的返回类型和字段的类型
var NodeKindTypeMappings = map[NodeKind]struct{}{
	NodeKindIntegralType:         {},
	NodeKindFloatingPointType:    {},
	NodeKindBooleanType:          {},
	NodeKindCharType:             {},
	NodeKindVoidType:             {},
	NodeKindArrayType:            {},
	NodeKindGenericType:          {},
	NodeKindTypeIdentifier:       {},
	NodeKindScopedTypeIdentifier: {},
	NodeKindWildcard:             {},
}

func ToNodeKind(kind string) NodeKind {
	if kind == EmptyString {
		return NodeKindUndefined
	}
	if nk, exists := NodeKindMappings[kind]; exists {
		return nk
	}
	return NodeKindUndefined
}
func IsTypeNode(kind NodeKind) bool {
	_, exists := NodeKindTypeMappings[kind]
	return exists
}

// ToElementType 将字符串映射为ElementType
func ToElementType(captureName string) ElementType {
	if captureName == EmptyString {
		return ElementTypeUndefined
	}
	if et, exists := TypeMappings[captureName]; exists {
		return et
	}
	return ElementTypeUndefined
}
