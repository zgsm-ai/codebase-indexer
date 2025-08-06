package types

const (
	EmptyString       = ""
	Underline         = "_"
	DoubleQuote       = "\""
	Slash             = "/"
	Comma             = ","
	Colon             = ":"
	Identifier        = "identifier"
	Dot               = "."
	Star              = "*"
	CurrentDir        = "./"
	ParentDir         = "../"
	EmailAt           = "@"
	Space             = " "
	LF                = "\n"
	Hash              = "#"
	SingleQuote       = "'"
	Static            = "static"
	Arrow             = "arrow"
	PackagePrivate    = "package-private"
	PublicAbstract    = "public abstract"
	ModifierProtected = "protected"
	ModifierPrivate   = "private"
	ModifierPublic    = "public"
	ModifierDefault   = "default"
	PrimitiveType     = "primitive_type"
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
	ElementTypeImportSource        ElementType = "import.source"
	ElementTypeClass               ElementType = "definition.class"
	ElementTypeClassName           ElementType = "definition.class.name"
	ElementTypeClassExtends        ElementType = "definition.class.extends"
	ElementTypeClassExtendsName    ElementType = "definition.class.extends.name"
	ElementTypeClassImplements     ElementType = "definition.class.implements"
	ElementTypeClassModifiers      ElementType = "definition.class.modifiers"
	ElementTypeInterface           ElementType = "definition.interface"
	ElementTypeInterfaceName       ElementType = "definition.interface.name"
	ElementTypeInterfaceType       ElementType = "definition.interface.type"
	ElementTypeInterfaceExtends    ElementType = "definition.interface.extends"
	ElementTypeInterfaceModifiers  ElementType = "definition.interface.modifiers"
	ElementTypeStruct              ElementType = "definition.struct"
	ElementTypeStructName          ElementType = "definition.struct.name"
	ElementTypeStructExtends       ElementType = "definition.struct.extends"
	ElementTypeStructType          ElementType = "definition.struct.type"
	ElementTypeEnum                ElementType = "definition.enum"
	ElementTypeEnumName            ElementType = "definition.enum.name"
	ElementTypeEnumImplements      ElementType = "definition.enum.implements"
	ElementTypeEnumModifiers       ElementType = "definition.enum.modifiers"
	ElementTypeUnion               ElementType = "definition.union"
	ElementTypeUnionName           ElementType = "definition.union.name"
	ElementTypeTypedefName         ElementType = "definition.type.alias"
	ElementTypeTypedefType         ElementType = "definition.type.original_type"
	ElementTypeTypedef             ElementType = "definition.type"
	ElementTypeTrait               ElementType = "definition.trait"
	ElementTypeEnumConstant        ElementType = "definition.enum.constant"
	ElementTypeEnumConstantName    ElementType = "definition.enum.constant.name"
	ElementTypeEnumConstantValue   ElementType = "definition.enum.constant.value"
	ElementTypeTypeAlias           ElementType = "definition.type_alias"
	ElementTypeFunction            ElementType = "definition.function"
	ElementTypeFunctionName        ElementType = "definition.function.name"
	ElementTypeFunctionReturnType  ElementType = "definition.function.return_type"
	ElementTypeFunctionParameters  ElementType = "definition.function.parameters"
	ElementTypeMethodCall          ElementType = "call.method"
	ElementTypeCallArguments       ElementType = "call.method.arguments"
	ElementTypeCallOwner           ElementType = "call.method.owner"
	ElementTypeCallName            ElementType = "call.method.name"
	ElementTypeFunctionCall        ElementType = "call.function"
	ElementTypeFunctionCallName    ElementType = "call.function.name"
	ElementTypeFunctionOwner       ElementType = "call.function.owner"
	ElementTypeFunctionArguments   ElementType = "call.function.arguments"
	ElementTypeStructCall          ElementType = "call.struct"
	ElementTypeStructCallType      ElementType = "call.struct.type"
	ElementTypeFunctionDeclaration ElementType = "declaration.function"
	ElementTypeMethod              ElementType = "definition.method"
	ElementTypeMethodModifier      ElementType = "definition.method.modifier"
	ElementTypeMethodReturnType    ElementType = "definition.method.return_type"
	ElementTypeMethodName          ElementType = "definition.method.name"
	ElementTypeMethodOwner         ElementType = "definition.method.owner"
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
	ElementTypeVariableName        ElementType = "variable.name"
	ElementTypeVariableValue       ElementType = "variable.value"
	ElementTypeVariableType        ElementType = "variable.type"
	ElementTypeConstant            ElementType = "constant"
	ElementTypeMacro               ElementType = "macro"
	ElementTypeField               ElementType = "definition.field"
	ElementTypeFieldName           ElementType = "definition.field.name"
	ElementTypeFieldType           ElementType = "definition.field.type"
	ElementTypeFieldValue          ElementType = "definition.field.value"
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
	string(ElementTypeImportSource):        ElementTypeImportSource,
	string(ElementTypeClass):               ElementTypeClass,
	string(ElementTypeClassName):           ElementTypeClassName,
	string(ElementTypeInterfaceType):       ElementTypeInterfaceType,
	string(ElementTypeClassExtends):        ElementTypeClassExtends,
	string(ElementTypeClassExtendsName):    ElementTypeClassExtendsName,
	string(ElementTypeClassImplements):     ElementTypeClassImplements,
	string(ElementTypeClassModifiers):      ElementTypeClassModifiers,
	string(ElementTypeInterface):           ElementTypeInterface,
	string(ElementTypeInterfaceName):       ElementTypeInterfaceName,
	string(ElementTypeInterfaceExtends):    ElementTypeInterfaceExtends,
	string(ElementTypeInterfaceModifiers):  ElementTypeInterfaceModifiers,
	string(ElementTypeStruct):              ElementTypeStruct,
	string(ElementTypeStructName):          ElementTypeStructName,
	string(ElementTypeStructType):          ElementTypeStructType,
	string(ElementTypeStructExtends):       ElementTypeStructExtends,
	string(ElementTypeEnum):                ElementTypeEnum,
	string(ElementTypeEnumName):            ElementTypeEnumName,
	string(ElementTypeEnumImplements):      ElementTypeEnumImplements,
	string(ElementTypeEnumModifiers):       ElementTypeEnumModifiers,
	string(ElementTypeEnumConstant):        ElementTypeEnumConstant,
	string(ElementTypeEnumConstantName):    ElementTypeEnumConstantName,
	string(ElementTypeEnumConstantValue):   ElementTypeEnumConstantValue,
	string(ElementTypeUnion):               ElementTypeUnion,
	string(ElementTypeUnionName):           ElementTypeUnionName,
	string(ElementTypeTypedefName):         ElementTypeTypedefName,
	string(ElementTypeTypedefType):         ElementTypeTypedefType,
	string(ElementTypeTypedef):             ElementTypeTypedef,
	string(ElementTypeTrait):               ElementTypeTrait,
	string(ElementTypeTypeAlias):           ElementTypeTypeAlias,
	string(ElementTypeFunction):            ElementTypeFunction,
	string(ElementTypeFunctionName):        ElementTypeFunctionName,
	string(ElementTypeFunctionParameters):  ElementTypeFunctionParameters,
	string(ElementTypeFunctionReturnType):  ElementTypeFunctionReturnType,
	string(ElementTypeFunctionCall):        ElementTypeFunctionCall,
	string(ElementTypeFunctionCallName):    ElementTypeFunctionCallName,
	string(ElementTypeFunctionOwner):       ElementTypeFunctionOwner,
	string(ElementTypeFunctionArguments):   ElementTypeFunctionArguments,
	string(ElementTypeFunctionDeclaration): ElementTypeFunctionDeclaration,
	string(ElementTypeMethod):              ElementTypeMethod,
	string(ElementTypeMethodCall):          ElementTypeMethodCall,
	string(ElementTypeMethodModifier):      ElementTypeMethodModifier,
	string(ElementTypeMethodReturnType):    ElementTypeMethodReturnType,
	string(ElementTypeMethodName):          ElementTypeMethodName,
	string(ElementTypeMethodOwner):         ElementTypeMethodOwner,
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
	string(ElementTypeVariableName):        ElementTypeVariableName,
	string(ElementTypeVariableValue):       ElementTypeVariableValue,
	string(ElementTypeVariableType):        ElementTypeVariableType,
	string(ElementTypeConstant):            ElementTypeConstant,
	string(ElementTypeMacro):               ElementTypeMacro,
	string(ElementTypeField):               ElementTypeField,
	string(ElementTypeFieldName):           ElementTypeFieldName,
	string(ElementTypeFieldType):           ElementTypeFieldType,
	string(ElementTypeFieldValue):          ElementTypeFieldValue,
	string(ElementTypeFieldModifier):       ElementTypeFieldModifier,
	string(ElementTypeParameter):           ElementTypeParameter,
	string(ElementTypeComment):             ElementTypeComment,
	string(ElementTypeAnnotation):          ElementTypeAnnotation,
	string(ElementTypeStructCall):          ElementTypeStructCall,
	string(ElementTypeStructCallType):      ElementTypeStructCallType,
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
	Path    string
	Content []byte
}
type NodeKind string

const (
	NodeKindMethodElem                         NodeKind = "method_elem"
	NodeKindMethodSpec                         NodeKind = "method_spec"
	NodeKindFieldList                          NodeKind = "field_declaration_list"
	NodeKindField                              NodeKind = "field_declaration"
	NodeKindMethod                             NodeKind = "method_declaration"
	NodeKindMethodDefinition                   NodeKind = "method_definition"
	NodeKindFieldDefinition                    NodeKind = "field_definition"
	NodeKindConstructor                        NodeKind = "constructor_declaration"
	NodeKindVariableDeclarator                 NodeKind = "variable_declarator"
	NodeKindLexicalDeclaration                 NodeKind = "lexical_declaration"
	NodeKindVariableDeclaration                NodeKind = "variable_declaration"
	NodeKindModifier                           NodeKind = "modifiers"
	NodeKindIdentifier                         NodeKind = "identifier"
	NodeKindKeywordArgument                    NodeKind = "keyword_argument"
	NodeKindSubscript                          NodeKind = "subscript"
	NodeKindAttribute                          NodeKind = "attribute"
	NodeKindType                               NodeKind = "type"
	NodeKindListSplatPattern                   NodeKind = "list_splat_pattern"
	NodeKindDictSplatPattern                   NodeKind = "dictionary_splat_pattern"
	NodeKindDefaultParameter                   NodeKind = "default_parameter"
	NodeKindTypedParameter                     NodeKind = "typed_parameter"
	NodeKindTypedDefaultParameter              NodeKind = "typed_default_parameter"
	NodeKindFormalParameters                   NodeKind = "formal_parameters"
	NodeKindUndefined                          NodeKind = "undefined"
	NodeKindFuncLiteral                        NodeKind = "func_literal"
	NodeKindSelectorExpression                 NodeKind = "selector_expression"
	NodeKindFieldIdentifier                    NodeKind = "field_identifier"
	NodeKindArgumentList                       NodeKind = "argument_list"
	NodeKindShortVarDeclaration                NodeKind = "short_var_declaration"
	NodeKindCompositeLiteral                   NodeKind = "composite_literal"
	NodeKindCallExpression                     NodeKind = "call_expression"
	NodeKindAwaitExpression                    NodeKind = "await_expression"
	NodeKindParameterList                      NodeKind = "parameter_list"
	NodeKindParameterDeclaration               NodeKind = "parameter_declaration"
	NodeKindTypeElem                           NodeKind = "type_elem"
	NodeKindClassBody                          NodeKind = "class_body"
	NodeKindPropertyIdentifier                 NodeKind = "property_identifier"
	NodeKindPrivatePropertyIdentifier          NodeKind = "private_property_identifier"
	NodeKindArrowFunction                      NodeKind = "arrow_function"
	NodeKindMemberExpression                   NodeKind = "member_expression"
	NodeKindNewExpression                      NodeKind = "new_expression"
	NodeKindObject                             NodeKind = "object"
	NodeKindArrayPattern                       NodeKind = "array_pattern"
	NodeKindObjectPattern                      NodeKind = "object_pattern"
	NodeKindShorthandPropertyIdentifierPattern NodeKind = "shorthand_property_identifier_pattern"
	NodeKindString                             NodeKind = "string"
	NodeKindPair                               NodeKind = "pair"
	NodeKindAccessibilityModifier              NodeKind = "accessibility_modifier"
	NodeKindTypeAnnotation                     NodeKind = "type_annotation"
	NodeKindPublicFieldDefinition              NodeKind = "public_field_definition"
	NodeKindRequiredParameter                  NodeKind = "required_parameter"
	NodeKindRestParameter                      NodeKind = "rest_parameter"
	NodeKindOptionalParameter                  NodeKind = "optional_parameter"
	NodeKindMethodSignature                    NodeKind = "method_signature"
	NodeKindQualifiedType                      NodeKind = "qualified_type"
	NodeKindPairPattern                        NodeKind = "pair_pattern"
	NodeKindRestPattern                        NodeKind = "rest_pattern"
	NodeKindParenthesizedExpression            NodeKind = "parenthesized_expression"
	NodeKindExportStatement                    NodeKind = "export_statement"
	NodeKindPropertySignature                  NodeKind = "property_signature"
	NodeKindFunctionType                       NodeKind = "function_type"
	NodeKindSliceType                          NodeKind = "slice_type"
	NodeKindPointType                          NodeKind = "pointer_type"
	NodeKindStructType                         NodeKind = "struct_type"
	NodeKindChannelType                        NodeKind = "channel_type"
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
	NodeKindWildcard             NodeKind = "wildcard"             // 通配符 <? extends MyClass>
	NodeKindPrimitiveType        NodeKind = "primitive_type"       // c/cpp基础类型都由这个接收
	NodeKindQualifiedIdentifier  NodeKind = "qualified_identifier" // c/cpp 复合类型 Outer::Inner

	// 用于查找方法所属的类
	NodeKindClassDeclaration     NodeKind = "class_declaration"
	NodeKindInterfaceDeclaration NodeKind = "interface_declaration"
	NodeKindEnumDeclaration      NodeKind = "enum_declaration"
	NodeKindClassSpecifier       NodeKind = "class_specifier"
	NodeKindStructSpecifier      NodeKind = "struct_specifier"
	NodeKindAccessSpecifier      NodeKind = "access_specifier"
	NodeKindTypeList             NodeKind = "type_list"
	NodeKindBaseClassClause      NodeKind = "base_class_clause"

	// 用于判断变量是否是局部变量
	NodeKindFunctionDeclaration NodeKind = "function_declaration"
	NodeKindMethodDeclaration   NodeKind = "method_declaration"
)

var NodeKindMappings = map[string]NodeKind{
	string(NodeKindField):                              NodeKindField,
	string(NodeKindMethod):                             NodeKindMethod,
	string(NodeKindMethodDefinition):                   NodeKindMethodDefinition,
	string(NodeKindFieldDefinition):                    NodeKindFieldDefinition,
	string(NodeKindClassBody):                          NodeKindClassBody,
	string(NodeKindConstructor):                        NodeKindConstructor,
	string(NodeKindUndefined):                          NodeKindUndefined,
	string(NodeKindVariableDeclarator):                 NodeKindVariableDeclarator,
	string(NodeKindLexicalDeclaration):                 NodeKindLexicalDeclaration,
	string(NodeKindVariableDeclaration):                NodeKindVariableDeclaration,
	string(NodeKindModifier):                           NodeKindModifier,
	string(NodeKindIdentifier):                         NodeKindIdentifier,
	string(NodeKindType):                               NodeKindType,
	string(NodeKindKeywordArgument):                    NodeKindKeywordArgument,
	string(NodeKindSubscript):                          NodeKindSubscript,
	string(NodeKindAttribute):                          NodeKindAttribute,
	string(NodeKindListSplatPattern):                   NodeKindListSplatPattern,
	string(NodeKindDictSplatPattern):                   NodeKindDictSplatPattern,
	string(NodeKindDefaultParameter):                   NodeKindDefaultParameter,
	string(NodeKindTypedParameter):                     NodeKindTypedParameter,
	string(NodeKindTypedDefaultParameter):              NodeKindTypedDefaultParameter,
	string(NodeKindFormalParameters):                   NodeKindFormalParameters,
	string(NodeKindMethodElem):                         NodeKindMethodElem,
	string(NodeKindMethodSpec):                         NodeKindMethodSpec,
	string(NodeKindFieldList):                          NodeKindFieldList,
	string(NodeKindFuncLiteral):                        NodeKindFuncLiteral,
	string(NodeKindSelectorExpression):                 NodeKindSelectorExpression,
	string(NodeKindFieldIdentifier):                    NodeKindFieldIdentifier,
	string(NodeKindArgumentList):                       NodeKindArgumentList,
	string(NodeKindShortVarDeclaration):                NodeKindShortVarDeclaration,
	string(NodeKindCompositeLiteral):                   NodeKindCompositeLiteral,
	string(NodeKindCallExpression):                     NodeKindCallExpression,
	string(NodeKindAwaitExpression):                    NodeKindAwaitExpression,
	string(NodeKindParameterList):                      NodeKindParameterList,
	string(NodeKindParameterDeclaration):               NodeKindParameterDeclaration,
	string(NodeKindPropertyIdentifier):                 NodeKindPropertyIdentifier,
	string(NodeKindPrivatePropertyIdentifier):          NodeKindPrivatePropertyIdentifier,
	string(NodeKindArrowFunction):                      NodeKindArrowFunction,
	string(NodeKindMemberExpression):                   NodeKindMemberExpression,
	string(NodeKindNewExpression):                      NodeKindNewExpression,
	string(NodeKindObject):                             NodeKindObject,
	string(NodeKindArrayPattern):                       NodeKindArrayPattern,
	string(NodeKindObjectPattern):                      NodeKindObjectPattern,
	string(NodeKindShorthandPropertyIdentifierPattern): NodeKindShorthandPropertyIdentifierPattern,
	string(NodeKindString):                             NodeKindString,
	string(NodeKindPair):                               NodeKindPair,
	string(NodeKindAccessibilityModifier):              NodeKindAccessibilityModifier,
	string(NodeKindTypeAnnotation):                     NodeKindTypeAnnotation,
	string(NodeKindPublicFieldDefinition):              NodeKindPublicFieldDefinition,
	string(NodeKindRequiredParameter):                  NodeKindRequiredParameter,
	string(NodeKindRestParameter):                      NodeKindRestParameter,
	string(NodeKindOptionalParameter):                  NodeKindOptionalParameter,
	string(NodeKindMethodSignature):                    NodeKindMethodSignature,
	string(NodeKindQualifiedType):                      NodeKindQualifiedType,
	string(NodeKindTypeElem):                           NodeKindTypeElem,
	string(NodeKindPairPattern):                        NodeKindPairPattern,
	string(NodeKindRestPattern):                        NodeKindRestPattern,
	string(NodeKindParenthesizedExpression):            NodeKindParenthesizedExpression,
	string(NodeKindExportStatement):                    NodeKindExportStatement,
	string(NodeKindPropertySignature):                  NodeKindPropertySignature,
	string(NodeKindFunctionType):                       NodeKindFunctionType,
	string(NodeKindSliceType):                          NodeKindSliceType,
	string(NodeKindPointType):                          NodeKindPointType,
	string(NodeKindStructType):                         NodeKindStructType,
	string(NodeKindChannelType):                        NodeKindChannelType,
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

	// 用于查找方法所属的类
	string(NodeKindClassDeclaration):     NodeKindClassDeclaration,
	string(NodeKindInterfaceDeclaration): NodeKindInterfaceDeclaration,
	string(NodeKindEnumDeclaration):      NodeKindEnumDeclaration,
	string(NodeKindClassSpecifier):       NodeKindClassSpecifier,
	string(NodeKindStructSpecifier):      NodeKindStructSpecifier,
	string(NodeKindAccessSpecifier):      NodeKindAccessSpecifier,
	string(NodeKindQualifiedIdentifier):  NodeKindQualifiedIdentifier,
	string(NodeKindTypeList):             NodeKindTypeList,
	string(NodeKindBaseClassClause):      NodeKindBaseClassClause,
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
	NodeKindPrimitiveType:        {},
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
