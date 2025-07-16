package resolver

import "codebase-indexer/pkg/codegraph/types"

// Element 定义所有代码元素的接口
type Element interface {
	GetName() string
	GetType() types.ElementType
	GetRange() []int32
	GetContent() []byte
	GetRootIndex() uint32
	SetName(name string)
	SetType(et types.ElementType)
	SetRange(range_ []int32)
	SetContent(content []byte)
}

// BaseElement 提供接口的基础实现，其他类型嵌入该结构体
type BaseElement struct {
	Name             string
	rootCaptureIndex uint32
	Scope            types.Scope
	Type             types.ElementType
	Content          []byte
	Range            []int32
}

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

// Import 表示导入语句
type Import struct {
	*BaseElement
	Source    string   // from (xxx)
	Alias     string   // as (xxx)
	FilePaths []string // 相对于项目root的路径（排除标准库/第三方包）
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

type Call struct {
	*BaseElement
	Owner      string
	Parameters []*Parameter
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
	Type string
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
	ReturnType string
}

type Variable struct {
	*BaseElement
}
