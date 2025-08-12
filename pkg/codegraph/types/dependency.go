package types

// RelationType 表示代码元素之间的关系类型
type RelationType int32

const (
	// RelationTypeUndefined 未定义的关系类型
	RelationTypeUndefined RelationType = 0
	// RelationTypeDefinition 定义关系（如变量/函数的定义）
	RelationTypeDefinition RelationType = 1
	// RelationTypeReference 引用关系（如对已定义元素的引用）
	RelationTypeReference RelationType = 2
	// RelationTypeInherit 继承关系（如类继承）
	RelationTypeInherit RelationType = 3
	// RelationTypeImplement 实现关系（如类实现接口）
	RelationTypeImplement RelationType = 4
	// RelationTypeSuperClass 父类关系
	RelationTypeSuperClass RelationType = 5
	// RelationTypeSuperInterface 父接口关系
	RelationTypeSuperInterface RelationType = 6
)

// String 返回关系类型的字符串表示（可选实现，便于日志和调试）
func (rt RelationType) String() string {
	switch rt {
	case RelationTypeUndefined:
		return "UNDEFINED"
	case RelationTypeDefinition:
		return "DEFINITION"
	case RelationTypeReference:
		return "REFERENCE"
	case RelationTypeInherit:
		return "INHERIT"
	case RelationTypeImplement:
		return "IMPLEMENT"
	case RelationTypeSuperClass:
		return "SUPER_CLASS"
	case RelationTypeSuperInterface:
		return "SUPER_INTERFACE"
	default:
		return "UNKNOWN"
	}
}

// Relation 关系定义
type Relation struct {
	ElementName  string
	ElementPath  string
	Range        []int32
	RelationType RelationType
}
