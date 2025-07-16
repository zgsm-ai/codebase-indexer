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
	ElementTypeUndefined           ElementType = "undefined"
	ElementTypeImport              ElementType = "import"
	ElementTypeClass               ElementType = "definition.class"
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
	ElementTypeConstructor         ElementType = "definition.constructor"
	ElementTypeDestructor          ElementType = "definition.destructor"
	ElementTypeGlobalVariable      ElementType = "global_variable"
	ElementTypeLocalVariable       ElementType = "local_variable"
	ElementTypeVariable            ElementType = "variable"
	ElementTypeConstant            ElementType = "constant"
	ElementTypeMacro               ElementType = "macro"
	ElementTypeField               ElementType = "definition.field"
	ElementTypeParameter           ElementType = "definition.parameter"
	ElementTypeComment             ElementType = "comment"
	ElementTypeAnnotation          ElementType = "annotation"
	ElementTypeReference           ElementType = "reference"
)

// TypeMappings 类型映射表 - captureName -> ElementType（使用ElementType字符串值作为键）
var TypeMappings = map[string]ElementType{
	string(ElementTypeNamespace):           ElementTypeNamespace,
	string(ElementTypePackage):             ElementTypePackage,
	string(ElementTypeUndefined):           ElementTypeUndefined,
	string(ElementTypeImport):              ElementTypeImport,
	string(ElementTypeClass):               ElementTypeClass,
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
