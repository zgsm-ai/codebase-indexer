package resolver

import "codebase-indexer/pkg/codegraph/types"

// Element 定义所有代码元素的接口
type Element interface {
	GetName() string
	GetType() types.ElementType
	GetRange() []int32
	GetContent() []byte
	GetRootIndex() uint32
	GetRelations() []*Relation
	GetPath() string
	SetPath(path string)
	SetName(name string)
	SetType(et types.ElementType)
	SetRange(range_ []int32)
	SetContent(content []byte)
	SetRelations(relations []*Relation)
}

// BaseElement 提供接口的基础实现，其他类型嵌入该结构体
type BaseElement struct {
	Name             string
	Path             string // 符号所属文件路径
	rootCaptureIndex uint32
	Scope            types.Scope
	Type             types.ElementType
	Content          []byte
	Range            []int32
	Relations        []*Relation // 与该节点有关的节点
}

type Relation struct {
	ElementName  string       // 符号名
	ElementPath  string       // 符号路径
	Range        []int32      // [开始行，开始列，结束行，结束列]
	RelationType RelationType // 符号关系， 定义、引用、继承、实现
}

type RelationType int

const (
	RelationTypeUndefined RelationType = iota
	RelationTypeDefinition
	RelationTypeReference
	RelationTypeInherit
	RelationTypeImplement
	RelationTypeSuperClass
	RelationTypeSuperInterface
)

func NewBaseElement(rootCaptureIndex uint32) *BaseElement {
	return &BaseElement{
		rootCaptureIndex: rootCaptureIndex,
	}
}

func (e *BaseElement) GetName() string            { return e.Name }
func (e *BaseElement) GetType() types.ElementType { return e.Type }
func (e *BaseElement) GetRange() []int32          { return e.Range }
func (e *BaseElement) GetContent() []byte         { return e.Content }
func (e *BaseElement) GetRootIndex() uint32       { return e.rootCaptureIndex }
func (e *BaseElement) GetPath() string {
	return e.Path
}
func (e *BaseElement) SetPath(path string) {
	e.Path = path
}
func (e *BaseElement) SetName(name string) {
	e.Name = name
}
func (e *BaseElement) SetType(et types.ElementType) {
	e.Type = et
}
func (e *BaseElement) SetRange(range_ []int32) {
	e.Range = range_
}

func (e *BaseElement) SetContent(content []byte) {
	e.Content = content
}

func (e *BaseElement) SetRelations(relations []*Relation) {
	e.Relations = relations
}
func (e *BaseElement) GetRelations() []*Relation {
	return e.Relations
}

// Import 表示导入语句
type Import struct {
	*BaseElement
	Source string // from (xxx)
	Alias  string // as (xxx)
}

// Package 表示代码包
type Package struct {
	*BaseElement
}

// Function 表示函数
type Function struct {
	*BaseElement
	Owner string
	Declaration
}

// Method 表示方法
type Method struct {
	*BaseElement
	Owner string
	Declaration
}

// Call 函数调用
type Call struct {
	*BaseElement
	Owner      string
	Parameters []*Parameter
}

// Reference 结构体、类的引用
type Reference struct {
	Owner string // 包名
	*BaseElement
}

// Class 表示类
type Class struct {
	*BaseElement
	SuperClasses    []string
	SuperInterfaces []string
	Fields          []*Field
	Methods         []*Method
}

type Field struct {
	Modifier string
	Name     string
	Type     string
}

type Parameter struct {
	Name string
	Type []string
}

type Interface struct {
	*BaseElement
	SuperInterfaces []string
	Methods         []*Declaration
}

type Declaration struct {
	Modifier   string
	Name       string
	Parameters []Parameter
	ReturnType []string
}

type Variable struct {
	*BaseElement
	VariableType []string
}
