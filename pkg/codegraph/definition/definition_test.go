package definition

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGoFileStructure(t *testing.T) {
	// 测试用的 Go 代码
	code := []byte(`
package test

// TestStruct 是一个测试结构体
type TestStruct struct {
	Identifier string
	Age  int
}

// TestInterface 是一个测试接口
type TestInterface interface {
	GetName() string
	GetAge() int
}

// TestFunc 是一个测试函数
func TestFunc(name string, age int) (string, error) {
	return name, nil
}

// TestMethod 是一个测试方法
func (s *TestStruct) TestMethod() string {
	return s.Identifier
}

// 常量定义
const TestConst = "test"

// 变量定义
var TestVar = "test"
`)

	// 获取 Go 语言配置
	parser, err := NeDefinitionParser()
	assert.NoError(t, err)
	// 解析文件结构
	structure, err := parser.Parse(context.Background(), &types.SourceFile{
		Content: code,
		Path:    "test.go",
	}, ParseOptions{})
	if err != nil {
		t.Fatalf("failed to parse file structure: %v", err)
	}

	// 验证结果
	if len(structure.Definitions) == 0 {
		t.Fatal("no definitions found")
	}
	assert.NotEmpty(t, structure.Path)
	assert.Equal(t, "go", structure.Language)

	// 预期的位置信息 (tree-sitter 使用从0开始的行列号)
	expectedRanges := map[string][]int32{
		"test":          {1, 0, 1, 12},   // line 4: type TestStruct struct {
		"TestStruct":    {4, 0, 7, 1},    // line 4: type TestStruct struct {
		"TestInterface": {10, 0, 13, 1},  // line 10: type TestInterface interface {
		"TestFunc":      {16, 0, 18, 1},  // line 16: func TestFunc(...)
		"TestMethod":    {21, 0, 23, 1},  // line 21: func (s *TestStruct) TestMethod()
		"TestConst":     {26, 0, 26, 24}, // line 26: const TestConst = "test"
		"TestVar":       {29, 0, 29, 20}, // line 29: var TestVar = "test"
	}

	// 验证每个定义
	foundDefs := make(map[string]bool)
	for _, def := range structure.Definitions {
		foundDefs[def.Name] = true

		// 验证类型
		switch def.Name {
		case "TestStruct":
			assert.Equal(t, "declaration.struct", def.Type)
		case "TestInterface":
			assert.Equal(t, "declaration.interface", def.Type)
		case "TestFunc":
			assert.Equal(t, "declaration.function", def.Type)
		case "TestMethod":
			assert.Equal(t, "declaration.method", def.Type)
		case "TestConst":
			assert.Equal(t, "declaration.const", def.Type)
		case "test":
			assert.Equal(t, "package", def.Type)
		case "TestVar":
			assert.Equal(t, "global_variable", def.Type)
		default:
			t.Errorf("unexpected definition: %s", def.Name)
		}

		// 验证位置信息
		expectedRange, ok := expectedRanges[def.Name]
		if !ok {
			t.Errorf("no expected range for definition: %s", def.Name)
			continue
		}

		assert.Equal(t, expectedRange[0], def.Range[0], "wrong start line for %s", def.Name)
		assert.Equal(t, expectedRange[1], def.Range[1], "wrong start column for %s", def.Name)
		assert.Equal(t, expectedRange[2], def.Range[2], "wrong end line for %s", def.Name)
		assert.Equal(t, expectedRange[3], def.Range[3], "wrong end column for %s", def.Name)
	}

	// 确保所有预期的定义都被找到
	for name := range expectedRanges {
		assert.True(t, foundDefs[name], "definition %s was not found", name)
	}
}
