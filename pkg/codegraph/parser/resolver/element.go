package resolver

// Element 定义所有代码元素的接口
type Element interface {
	GetName() string
	GetType() ElementType
	GetRange() []int32
	SetContent(content []byte)
	SetRootIndex(rootIndex uint32)
}

// BaseElement 提供接口的基础实现，其他类型嵌入该结构体
type BaseElement struct {
	Name             string
	rootCaptureIndex uint32
	Type             ElementType
	Content          []byte
	Range            []int32
}

func (e *BaseElement) GetName() string      { return e.Name }
func (e *BaseElement) GetType() ElementType { return e.Type }
func (e *BaseElement) GetRange() []int32    { return e.Range }

func (e *BaseElement) SetContent(content []byte) {
	e.Content = content
}
func (e *BaseElement) SetRootIndex(rootCaptureIndex uint32) {
	e.rootCaptureIndex = rootCaptureIndex
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
	Fields  []*Field
	Methods []*Method
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
	Methods []*Declaration
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
