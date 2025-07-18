package types

const (
	EmptyString    = ""
	DoubleQuote    = "\""
	Comma          = ","
	Identifier     = "identifier"
	Dot            = "."
	LF             = "\n"
	PackagePrivate = "package-private"
	PublicAbstract = "public abstract"
	ModifierProtected = "protected"
	ModifierPrivate = "private"
	ModifierPublic = "public"
	ModifierDefault = "default"
	PrimitiveType = "primitive_type"
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
	ElementTypeImportAlias         ElementType = "import.alias"
	ElementTypeImportPath          ElementType = "import.path"
	ElementTypeClass               ElementType = "definition.class"
	ElementTypeClassName           ElementType = "definition.class.name"
	ElementTypeClassExtends        ElementType = "definition.class.extends"
	ElementTypeClassImplements     ElementType = "definition.class.implements"
	ElementTypeClassModifiers      ElementType = "definition.class.modifiers"
	ElementTypeInterface           ElementType = "definition.interface"
	ElementTypeInterfaceName       ElementType = "definition.interface.name"
	ElementTypeInterfaceType       ElementType = "definition.interface.type"
	ElementTypeInterfaceExtends    ElementType = "definition.interface.extends"
	ElementTypeInterfaceModifiers  ElementType = "definition.interface.modifiers"
	ElementTypeStruct              ElementType = "definition.struct"
	ElementTypeStructName          ElementType = "definition.struct.name"
	ElementTypeStructType          ElementType = "definition.struct.type"
	ElementTypeEnum                ElementType = "definition.enum"
	ElementTypeUnion               ElementType = "definition.union"
	ElementTypeTrait               ElementType = "definition.trait"
	ElementTypeTypeAlias           ElementType = "definition.type_alias"
	ElementTypeFunction            ElementType = "definition.function"
	ElementTypeFunctionName        ElementType = "definition.function.name"
	ElementTypeFunctionParameters  ElementType = "definition.function.parameters"
	ElementTypeMethodCall          ElementType = "call.method"
	ElementTypeCallArguments       ElementType = "call.method.arguments"
	ElementTypeCallOwner           ElementType = "call.method.owner"
	ElementTypeCallName            ElementType = "call.method.name"
	ElementTypeFunctionCall        ElementType = "call.function"
	ElementTypeFunctionOwner       ElementType = "call.function.owner"
	ElementTypeFunctionArguments   ElementType = "call.function.arguments"
	ElementTypeFunctionDeclaration ElementType = "declaration.function"
	ElementTypeMethod              ElementType = "definition.method"
	ElementTypeMethodModifier      ElementType = "definition.method.modifier"
	ElementTypeMethodReturnType    ElementType = "definition.method.return_type"
	ElementTypeMethodName          ElementType = "definition.method.name"
	ElementTypeMethodParameters    ElementType = "definition.method.parameters"
	ElementTypeMethodReceiver      ElementType = "definition.method.receiver"
	ElementTypeConstructor         ElementType = "definition.constructor"
	ElementTypeDestructor          ElementType = "definition.destructor"
	ElementTypeGlobalVariable      ElementType = "global_variable"
	ElementTypeLocalVariable       ElementType = "local_variable"
	ElementTypeLocalVariableName   ElementType = "local_variable.name"
	ElementTypeLocalVariableType   ElementType = "local_variable.type"	
	ElementTypeLocalVariableValue  ElementType = "local_variable.value"
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
	string(ElementTypeImportAlias):         ElementTypeImportAlias,
	string(ElementTypeImportPath):          ElementTypeImportPath,
	string(ElementTypeClass):               ElementTypeClass,
	string(ElementTypeClassName):           ElementTypeClassName,
	string(ElementTypeInterfaceType):       ElementTypeInterfaceType,
	string(ElementTypeClassExtends):        ElementTypeClassExtends,
	string(ElementTypeClassImplements):     ElementTypeClassImplements,
	string(ElementTypeClassModifiers):      ElementTypeClassModifiers,
	string(ElementTypeInterface):           ElementTypeInterface,
	string(ElementTypeInterfaceName):       ElementTypeInterfaceName,
	string(ElementTypeInterfaceExtends):    ElementTypeInterfaceExtends,
	string(ElementTypeInterfaceModifiers):  ElementTypeInterfaceModifiers,
	string(ElementTypeStruct):              ElementTypeStruct,
	string(ElementTypeStructName):          ElementTypeStructName,
	string(ElementTypeStructType):          ElementTypeStructType,
	string(ElementTypeEnum):                ElementTypeEnum,
	string(ElementTypeUnion):               ElementTypeUnion,
	string(ElementTypeTrait):               ElementTypeTrait,
	string(ElementTypeTypeAlias):           ElementTypeTypeAlias,
	string(ElementTypeFunction):            ElementTypeFunction,
	string(ElementTypeFunctionCall):        ElementTypeFunctionCall,
	string(ElementTypeFunctionOwner):       ElementTypeFunctionOwner,
	string(ElementTypeFunctionArguments):   ElementTypeFunctionArguments,
	string(ElementTypeFunctionDeclaration): ElementTypeFunctionDeclaration,
	string(ElementTypeMethod):              ElementTypeMethod,
	string(ElementTypeMethodCall):          ElementTypeMethodCall,
	string(ElementTypeMethodModifier):      ElementTypeMethodModifier,
	string(ElementTypeMethodReturnType):    ElementTypeMethodReturnType,
	string(ElementTypeMethodName):          ElementTypeMethodName,
	string(ElementTypeMethodParameters):    ElementTypeMethodParameters,
	string(ElementTypeMethodReceiver):      ElementTypeMethodReceiver,
	string(ElementTypeCallArguments):       ElementTypeCallArguments,
	string(ElementTypeCallOwner):           ElementTypeCallOwner,
	string(ElementTypeCallName):            ElementTypeCallName,
	string(ElementTypeConstructor):         ElementTypeConstructor,
	string(ElementTypeDestructor):          ElementTypeDestructor,
	string(ElementTypeGlobalVariable):      ElementTypeGlobalVariable,
	string(ElementTypeLocalVariable):       ElementTypeLocalVariable,
	string(ElementTypeLocalVariableName):   ElementTypeLocalVariableName,
	string(ElementTypeLocalVariableType):   ElementTypeLocalVariableType,
	string(ElementTypeLocalVariableValue):  ElementTypeLocalVariableValue,
	string(ElementTypeVariable):            ElementTypeVariable,
	string(ElementTypeConstant):            ElementTypeConstant,
	string(ElementTypeMacro):               ElementTypeMacro,
	string(ElementTypeField):               ElementTypeField,
	string(ElementTypeFieldName):           ElementTypeFieldName,
	string(ElementTypeFieldType):           ElementTypeFieldType,
	string(ElementTypeFieldModifier):       ElementTypeFieldModifier,
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
	NodeKindMethodElem           NodeKind = "method_elem"
	NodeKindMethodSpec           NodeKind = "method_spec"
	NodeKindFieldList            NodeKind = "field_declaration_list"
	NodeKindField                NodeKind = "field_declaration"
	NodeKindMethod               NodeKind = "method_declaration"
	NodeKindConstructor          NodeKind = "constructor_declaration"
	NodeKindVariableDeclarator   NodeKind = "variable_declarator"
	NodeKindModifier             NodeKind = "modifiers"
	NodeKindIdentifier           NodeKind = "identifier"
	NodeKindFormalParameters     NodeKind = "formal_parameters"
	NodeKindUndefined            NodeKind = "undefined"
	NodeKindFuncLiteral          NodeKind = "func_literal"
	NodeKindSelectorExpression   NodeKind = "selector_expression"
	NodeKindFieldIdentifier      NodeKind = "field_identifier"
	NodeKindArgumentList         NodeKind = "argument_list"
	NodeKindShortVarDeclaration  NodeKind = "short_var_declaration"
	NodeKindCompositeLiteral     NodeKind = "composite_literal"
	NodeKindCallExpression       NodeKind = "call_expression"
	NodeKindParameterList        NodeKind = "parameter_list"
	NodeKindParameterDeclaration NodeKind = "parameter_declaration"
	// 用于接收函数的返回类型和字段的类型
	NodeKindIntegralType         NodeKind = "integral_type"
	NodeKindFloatingPointType    NodeKind = "floating_point_type"
	NodeKindBooleanType          NodeKind = "boolean_type"
	NodeKindCharType             NodeKind = "char_type"
	NodeKindVoidType             NodeKind = "void_type"
	NodeKindArrayType            NodeKind = "array_type"
	NodeKindGenericType          NodeKind = "generic_type"
	NodeKindTypeIdentifier       NodeKind = "type_identifier"
	NodeKindTypeArguments        NodeKind = "type_arguments"
	NodeKindScopedTypeIdentifier NodeKind = "scoped_type_identifier"
	NodeKindWildcard             NodeKind = "wildcard" // 通配符 <? extends MyClass>
)

var NodeKindMappings = map[string]NodeKind{
	string(NodeKindField):                NodeKindField,
	string(NodeKindMethod):               NodeKindMethod,
	string(NodeKindConstructor):          NodeKindConstructor,
	string(NodeKindUndefined):            NodeKindUndefined,
	string(NodeKindVariableDeclarator):   NodeKindVariableDeclarator,
	string(NodeKindModifier):             NodeKindModifier,
	string(NodeKindIdentifier):           NodeKindIdentifier,
	string(NodeKindFormalParameters):     NodeKindFormalParameters,
	string(NodeKindMethodElem):           NodeKindMethodElem,
	string(NodeKindMethodSpec):           NodeKindMethodSpec,
	string(NodeKindFieldList):            NodeKindFieldList,
	string(NodeKindFuncLiteral):          NodeKindFuncLiteral,
	string(NodeKindSelectorExpression):   NodeKindSelectorExpression,
	string(NodeKindFieldIdentifier):      NodeKindFieldIdentifier,
	string(NodeKindArgumentList):         NodeKindArgumentList,
	string(NodeKindShortVarDeclaration):  NodeKindShortVarDeclaration,
	string(NodeKindCompositeLiteral):     NodeKindCompositeLiteral,
	string(NodeKindCallExpression):       NodeKindCallExpression,
	string(NodeKindParameterList):        NodeKindParameterList,
	string(NodeKindParameterDeclaration): NodeKindParameterDeclaration,

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
	string(NodeKindTypeArguments):        NodeKindTypeArguments,
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
	NodeKindConstructor:          {},
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

// NodeKindTypeMap 定义节点类型到类型字符串的映射
var NodeKindTypeMap = map[string]string{
	"identifier":                 "unknown",
	"int_literal":                "int",
	"float_literal":              "float64",
	"interpreted_string_literal": "string",
	"raw_string_literal":         "string",
	"true":                       "bool",
	"false":                      "bool",
	"nil":                        "nil",
	"selector_expression":        "selector",
	"call_expression":            "function_result",
	"binary_expression":          "expression",
	"unary_expression":           "expression",
	"array_literal":              "array/slice",
	"slice_literal":              "array/slice",
	"map_literal":                "map",
	"composite_literal":          "struct",
}

func IsTypeNode(kind NodeKind) bool {
	_, exists := NodeKindTypeMappings[kind]
	return exists
}

// GetNodeTypeString 根据节点类型返回对应的类型字符串
func GetNodeTypeString(nodeKind string, value string) string {
	if typeStr, exists := NodeKindTypeMap[nodeKind]; exists {
		return typeStr
	}
	return nodeKind
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
