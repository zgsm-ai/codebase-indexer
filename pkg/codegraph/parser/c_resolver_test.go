package parser

import (
	"context"
	"fmt"
	"testing"

	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"

	"github.com/stretchr/testify/assert"
)

func TestCResolver(t *testing.T) {
	logger := initLogger()                // 如果有日志初始化
	parser := NewSourceFileParser(logger) // 假设有类似 Java 的解析器
	res, err := parser.Parse(context.Background(), &types.SourceFile{
		Path:    "testdata/java/test.java",
		Content: readFile("testdata/java/test.java"),
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)
	for _, elem := range res.Elements {
		if !resolver.IsValidElement(elem) {
			fmt.Printf("📦 Type: %s | Name: %s | Path: %s\n", elem.GetType(), elem.GetName(), elem.GetPath())
			fmt.Printf("🔍 Range: %v | Scope: %s\n", elem.GetRange(), elem.GetScope())
			fmt.Println("--------------------------------------------------")
		}

		//fmt.Printf("📦 Type: %s | Name: %s | Path: %s\n", elem.GetType(), elem.GetName(), elem.GetPath())
		//fmt.Printf("🔍 Range: %v | Scope: %s\n", elem.GetRange(), elem.GetScope())
		//fmt.Println("--------------------------------------------------")

	}
	for _, elem := range res.Imports {
		fmt.Printf("📦 Type: %s | Name: %s | Path: %s\n", elem.GetType(), elem.GetName(), elem.GetPath())
		fmt.Printf("🔍 Range: %v | Scope: %s\n", elem.GetRange(), elem.GetScope())
		fmt.Println("--------------------------------------------------")
		if !resolver.IsValidElement(elem) {
			fmt.Println("Error: ")
		}
	}
}
func TestCResolver_ResolveImport(t *testing.T) {
	logger := initLogger()                // 如果有日志初始化
	parser := NewSourceFileParser(logger) // 假设有类似 Java 的解析器

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantImports []string
		description string
	}{
		{
			name: "标准库和自定义头文件、条件包含、系统特定头文件、第三方库、包含保护、条件编译、错误处理、时间处理",
			sourceFile: &types.SourceFile{
				Path:    "pkg/codegraph/parser/testdata/c/testImport.c",
				Content: readFile("testdata/c/testImport.c"),
			},
			wantImports: []string{
				"<stdio.h>",
				"<stdlib.h>",
				"<string.h>",
				"<math.h>",
				"\"myheader.h\"",
				"\"utils.h\"",
				"\"project_config.h\"",
				"\"main_module.h\"",
				"<assert.h>",
				"<unistd.h>",
				"<sys/types.h>",
				"<sys/socket.h>",
				"<netinet/in.h>",
				"<curl/curl.h>",
				"\"config.h\"",
				"<windows.h>",
				"<pthread.h>",
				"<errno.h>",
				"<signal.h>",
				"<time.h>",
			},
			description: "测试C语言各种#include导入的解析，包括标准库、自定义头文件、条件包含、系统特定、第三方库、包含保护、条件编译、错误处理和时间处理等情况",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)
			// fmt.Println("res",res)
			if err == nil {
				// 验证导入解析
				fmt.Println(len(res.Imports))
				for _, importItem := range res.Imports {
					fmt.Printf("Import: %s\n", importItem.GetName())
					assert.NotEmpty(t, importItem.GetName())
					assert.Equal(t, types.ElementTypeImport, importItem.GetType())
					assert.Contains(t, tt.wantImports, importItem.GetName())
				}
			}
		})
	}
}
func TestCResolver_ResolveFunction(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantFuncs   []resolver.Declaration
		description string
	}{
		{
			name: "testFunc.c 全部函数声明解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/c/testFunc.c",
				Content: readFile("testdata/c/testFunc.c"),
			},
			wantErr: nil,
			wantFuncs: []resolver.Declaration{
				// 基本类型
				{Name: "func1", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func2", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func3", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func4", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func5", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func6", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func7", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func8", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func9", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 带参数的函数
				{Name: "func10", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "a", Type: []string{types.PrimitiveType}}}},
				{Name: "func11", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "b", Type: []string{types.PrimitiveType}}}},
				{Name: "func12", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "c", Type: []string{types.PrimitiveType}}}},
				{Name: "func13", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}, {Name: "y", Type: []string{types.PrimitiveType}}}},
				{Name: "func14", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "a", Type: []string{types.PrimitiveType}}, {Name: "b", Type: []string{types.PrimitiveType}}, {Name: "c", Type: []string{types.PrimitiveType}}}},

				// 无参数但明确指定void
				{Name: "func15", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "", Type: []string{types.PrimitiveType}}}},
				{Name: "func16", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "", Type: []string{types.PrimitiveType}}}},

				// 复杂返回值类型
				{Name: "func17", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func18", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func19", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func20", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func21", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func22", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func23", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 复杂参数类型
				{Name: "func24", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ptr", Type: []string{types.PrimitiveType}}}},
				{Name: "func25", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "str", Type: []string{types.PrimitiveType}}}},
				{Name: "func26", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "arr", Type: []string{types.PrimitiveType}}}},
				{Name: "func27", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "a", Type: []string{types.PrimitiveType}}}},
				{Name: "func28", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "str", Type: []string{types.PrimitiveType}}}},
				{Name: "func29", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}}},

				// 指针参数组合
				{Name: "func30", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "a", Type: []string{types.PrimitiveType}}, {Name: "b", Type: []string{types.PrimitiveType}}}},
				{Name: "func31", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}, {Name: "y", Type: []string{types.PrimitiveType}}, {Name: "z", Type: []string{types.PrimitiveType}}}},
				{Name: "func32", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "src", Type: []string{types.PrimitiveType}}, {Name: "dest", Type: []string{types.PrimitiveType}}}},

				// 数组参数
				{Name: "func33", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "arr", Type: []string{types.PrimitiveType}}}},
				{Name: "func34", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "str", Type: []string{types.PrimitiveType}}}},
				{Name: "func35", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "matrix", Type: []string{types.PrimitiveType}}}},
				{Name: "func36", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "arr", Type: []string{types.PrimitiveType}}}},
				{Name: "func37", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "buffer", Type: []string{types.PrimitiveType}}}},

				// 多维数组参数
				{Name: "func38", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "matrix", Type: []string{types.PrimitiveType}}}},
				{Name: "func39", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "cube", Type: []string{types.PrimitiveType}}}},
				{Name: "func40", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "tensor", Type: []string{types.PrimitiveType}}}},

				// 结构体参数
				{Name: "func41", ReturnType: []string{"Point"}, Parameters: []resolver.Parameter{{Name: "p", Type: []string{"Point"}}}},
				{Name: "func42", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "p", Type: []string{"Point"}}}},
				{Name: "func43", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "a", Type: []string{"Point"}}, {Name: "b", Type: []string{"Point"}}}},

				// 枚举参数
				{Name: "func44", ReturnType: []string{"Color"}, Parameters: []resolver.Parameter{{Name: "c", Type: []string{"Color"}}}},
				{Name: "func45", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "c", Type: []string{"Color"}}}},

				// 联合体参数
				{Name: "func46", ReturnType: []string{"Data"}, Parameters: []resolver.Parameter{{Name: "d", Type: []string{"Data"}}}},
				{Name: "func47", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "d", Type: []string{"Data"}}}},

				// 函数指针参数
				{Name: "func48", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "callback", Type: []string{types.PrimitiveType}}}},
				{Name: "func49", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "handler", Type: []string{types.PrimitiveType}}}},
				{Name: "func50", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}, {Name: "compare", Type: []string{types.PrimitiveType}}}},

				// 复杂函数指针参数
				{Name: "func51", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "callbacks", Type: []string{types.PrimitiveType}}}},
				{Name: "func52", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "handlers", Type: []string{types.PrimitiveType}}}},
				// {Name: "func53", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}}},

				// 可变参数函数
				{Name: "func54", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "count", Type: []string{types.PrimitiveType}}}},
				{Name: "func55", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "format", Type: []string{types.PrimitiveType}}}},

				// 复杂组合
				{Name: "func56", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ptr", Type: []string{types.PrimitiveType}}, {Name: "strings", Type: []string{types.PrimitiveType}}, {Name: "vptr", Type: []string{types.PrimitiveType}}}},
				{Name: "func57", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "points", Type: []string{"Point"}}, {Name: "colors", Type: []string{"Color"}}, {Name: "data", Type: []string{"Data"}}}},

				// 嵌套指针
				{Name: "func58", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				// {Name: "func59", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func60", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ptr", Type: []string{types.PrimitiveType}}}},

				// 限定符组合
				{Name: "func61", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ptr", Type: []string{types.PrimitiveType}}}},
				{Name: "func62", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}}},
				{Name: "func63", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "str", Type: []string{types.PrimitiveType}}}},

				// 复杂返回值和参数组合
				// {Name: "func64", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}, {Name: "callback", Type: []string{types.PrimitiveType}}}},
				// {Name: "func65", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "op", Type: []string{types.PrimitiveType}}}},
				// {Name: "func66", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "arr", Type: []string{types.PrimitiveType}}}},

				// 长参数列表
				{Name: "func67", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "a1", Type: []string{types.PrimitiveType}}, {Name: "a2", Type: []string{types.PrimitiveType}}, {Name: "a3", Type: []string{types.PrimitiveType}}, {Name: "a4", Type: []string{types.PrimitiveType}}, {Name: "a5", Type: []string{types.PrimitiveType}}, {Name: "a6", Type: []string{types.PrimitiveType}}, {Name: "a7", Type: []string{types.PrimitiveType}}, {Name: "a8", Type: []string{types.PrimitiveType}}}},
				{Name: "func68", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "c1", Type: []string{types.PrimitiveType}}, {Name: "c2", Type: []string{types.PrimitiveType}}, {Name: "c3", Type: []string{types.PrimitiveType}}, {Name: "c4", Type: []string{types.PrimitiveType}}, {Name: "c5", Type: []string{types.PrimitiveType}}, {Name: "c6", Type: []string{types.PrimitiveType}}, {Name: "c7", Type: []string{types.PrimitiveType}}, {Name: "c8", Type: []string{types.PrimitiveType}}, {Name: "c9", Type: []string{types.PrimitiveType}}}},

				// 混合复杂类型
				{Name: "func69", ReturnType: []string{"Rectangle"}, Parameters: []resolver.Parameter{{Name: "points", Type: []string{"Point"}}, {Name: "count", Type: []string{types.PrimitiveType}}}},
				{Name: "func70", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "rect", Type: []string{"Rectangle"}}, {Name: "color", Type: []string{"Color"}}}},

				// 函数声明中的typedef使用
				{Name: "func71", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "cmp", Type: []string{"Comparator"}}}},
				{Name: "func72", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "h", Type: []string{"Handler"}}}},
				{Name: "func73", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "comparators", Type: []string{"Comparator"}}, {Name: "count", Type: []string{types.PrimitiveType}}}},

				// 内联函数声明（C99）
				{Name: "func74", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}}},
				{Name: "func75", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}}},

				// 存储类说明符
				{Name: "func76", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}}},
				{Name: "func77", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "x", Type: []string{types.PrimitiveType}}}},

				// 完整复杂示例
				{Name: "func78", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "points", Type: []string{"Point"}}, {Name: "colors", Type: []string{"Color"}}, {Name: "callbacks", Type: []string{types.PrimitiveType}}}},

				// 函数指针数组作为参数
				{Name: "func79", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "func_array", Type: []string{types.PrimitiveType}}}},
				{Name: "func80", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "handlers", Type: []string{types.PrimitiveType}}}},

				// 返回函数指针的函数
				// {Name: "func81", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "choice", Type: []string{types.PrimitiveType}}}},
				// {Name: "func82", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "type", Type: []string{types.PrimitiveType}}}},

				// // 极其复杂的嵌套声明
				// {Name: "func83", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "param", Type: []string{types.PrimitiveType}}}},

				// 使用预定义类型
				{Name: "func84", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "len", Type: []string{types.PrimitiveType}}}},
				{Name: "func85", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "offset", Type: []string{types.PrimitiveType}}}},
				{Name: "func86", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ch", Type: []string{types.PrimitiveType}}}},

				// 布尔类型（C99）
				{Name: "func87", ReturnType: []string{"_Bool"}, Parameters: []resolver.Parameter{{Name: "flag", Type: []string{"_Bool"}}}},
				{Name: "func88", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "condition", Type: []string{types.PrimitiveType}}}},

				// 空指针常量参数
				{Name: "func89", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ptr", Type: []string{types.PrimitiveType}}}},
				{Name: "func90", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "data", Type: []string{types.PrimitiveType}}}},

				// 字符串字面量相关
				{Name: "func91", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "str", Type: []string{types.PrimitiveType}}}},
				{Name: "func92", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "buffer", Type: []string{types.PrimitiveType}}, {Name: "size", Type: []string{types.PrimitiveType}}}},

				// 数学相关类型
				{Name: "func93", ReturnType: []string{"intmax_t"}, Parameters: []resolver.Parameter{{Name: "value", Type: []string{"intmax_t"}}}},
				{Name: "func94", ReturnType: []string{"uintmax_t"}, Parameters: []resolver.Parameter{{Name: "value", Type: []string{"uintmax_t"}}}},
				{Name: "func95", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ptr", Type: []string{types.PrimitiveType}}}},

				// 文件操作相关类型
				{Name: "func96", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "filename", Type: []string{types.PrimitiveType}}}},
				{Name: "func97", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "stream", Type: []string{types.PrimitiveType}}}},

				// 信号处理相关
				// {Name: "func98", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "sig", Type: []string{types.PrimitiveType}}, {Name: "handler", Type: []string{types.PrimitiveType}}}},

				// 时间相关类型
				{Name: "func99", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "timer", Type: []string{types.PrimitiveType}}}},
				{Name: "func100", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "clk", Type: []string{types.PrimitiveType}}}},

				// 本地化相关
				{Name: "func101", ReturnType: []string{"locale_t"}, Parameters: []resolver.Parameter{{Name: "locale", Type: []string{"locale_t"}}}},

				// 多线程相关类型
				{Name: "func102", ReturnType: []string{"thrd_t"}, Parameters: []resolver.Parameter{{Name: "thread", Type: []string{"thrd_t"}}}},
				{Name: "func103", ReturnType: []string{"mtx_t"}, Parameters: []resolver.Parameter{{Name: "mutex", Type: []string{"mtx_t"}}}},

				// 原子类型（C11）
				{Name: "func104", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "value", Type: []string{types.PrimitiveType}}}},
				{Name: "func105", ReturnType: []string{"atomic_int"}, Parameters: []resolver.Parameter{{Name: "aint", Type: []string{"atomic_int"}}}},

				// 泛型相关（C11）
				// {Name: "func106", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "n", Type: []string{types.PrimitiveType}}}},

				// 可选的数组参数标记
				{Name: "func107", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "arr", Type: []string{types.PrimitiveType}}}},
				{Name: "func108", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "buffer", Type: []string{types.PrimitiveType}}}},

				// 复杂的VLA参数
				{Name: "func109", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "rows", Type: []string{types.PrimitiveType}}, {Name: "cols", Type: []string{types.PrimitiveType}}, {Name: "matrix", Type: []string{types.PrimitiveType}}}},
				{Name: "func110", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "n", Type: []string{types.PrimitiveType}}, {Name: "arr", Type: []string{types.PrimitiveType}}}},

				// 限定符的复杂组合
				{Name: "func111", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "ptr", Type: []string{types.PrimitiveType}}}},
				{Name: "func112", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "argv", Type: []string{types.PrimitiveType}}}},

				// 函数参数中的匿名结构体
				{Name: "func113", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "point", Type: []string{types.PrimitiveType}}}},
				{Name: "func114", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "data", Type: []string{types.PrimitiveType}}}},

				// 嵌套的匿名类型
				{Name: "func115", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{{Name: "nested", Type: []string{types.PrimitiveType}}}},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 1. 收集所有函数（不考虑重载，直接用名字做唯一键）
				funcMap := make(map[string]*resolver.Declaration)
				for _, element := range res.Elements {
					if fn, ok := element.(*resolver.Function); ok {
						funcMap[fn.Declaration.Name] = &fn.Declaration
					}
				}
				// 2. 逐个比较每个期望的函数
				for _, wantFunc := range tt.wantFuncs {
					actualFunc, exists := funcMap[wantFunc.Name]
					assert.True(t, exists, "未找到函数: %s", wantFunc.Name)
					if exists {
						assert.Equal(t, wantFunc.ReturnType, actualFunc.ReturnType,
							"函数 %s 的返回值类型不匹配，期望 %v，实际 %v",
							wantFunc.Name, wantFunc.ReturnType, actualFunc.ReturnType)
						assert.Equal(t, len(wantFunc.Parameters), len(actualFunc.Parameters),
							"函数 %s 的参数数量不匹配，期望 %d，实际 %d",
							wantFunc.Name, len(wantFunc.Parameters), len(actualFunc.Parameters))
						for i, wantParam := range wantFunc.Parameters {
							assert.Equal(t, wantParam.Type, actualFunc.Parameters[i].Type,
								"函数 %s 的第 %d 个参数类型不匹配，期望 %v，实际 %v",
								wantFunc.Name, i+1, wantParam.Type, actualFunc.Parameters[i].Type)
						}
					}
				}
			}
		})
	}
}

func TestCResolver_ResolveStruct(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantStructs []resolver.Class
		description string
	}{
		{
			name: "testStruct.c 全部结构体声明解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/c/testStruct.c",
				Content: readFile("testdata/c/testStruct.c"),
			},
			wantErr: nil,
			wantStructs: []resolver.Class{
				// 基本结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Student",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "Student1",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "Person",
					},
				},
				// 嵌套结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Address",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "Employee",
					},
				},
				// 位域结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Permission",
					},
				},
				// 自引用结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "ListNode",
					},
				},
				// 复杂嵌套结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Date",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "Time",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "DateTime",
					},
				},
				// 联合体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Data",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "MixedData",
					},
				},
				// 函数指针结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "MathOps",
					},
				},
				// 数组成员结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Student",
					},
				},
				// 指针数组结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Database",
					},
				},
				// 长整型和无符号类型
				{
					BaseElement: &resolver.BaseElement{
						Name: "FileHeader",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "Status",
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "Priority",
					},
				},
				// 枚举成员结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Task",
					},
				},
				// 柔性数组成员（C99）
				{
					BaseElement: &resolver.BaseElement{
						Name: "Packet",
					},
				},
				// 复杂的数据结构
				{
					BaseElement: &resolver.BaseElement{
						Name: "TreeNode",
					},
				},
				// 多层嵌套
				{
					BaseElement: &resolver.BaseElement{
						Name: "University",
					},
				},
				// 匿名结构体成员
				{
					BaseElement: &resolver.BaseElement{
						Name: "Config",
					},
				},
				// 复杂指针结构体
				{
					BaseElement: &resolver.BaseElement{
						Name: "Callback",
					},
				},
			},
			description: "测试C语言各种结构体声明的解析，包括基本结构体、嵌套结构体、位域、自引用、联合体、函数指针、数组、枚举成员、柔性数组、复杂嵌套等情况",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 1. 收集所有结构体（直接用名字做唯一键）
				structMap := make(map[string]*resolver.Class)
				for _, element := range res.Elements {
					if class, ok := element.(*resolver.Class); ok {
						structMap[class.Name] = class
					}
				}

				// 2. 逐个比较每个期望的结构体
				for _, wantStruct := range tt.wantStructs {
					actualStruct, exists := structMap[wantStruct.Name]
					assert.True(t, exists, "未找到结构体: %s", wantStruct.Name)
					if exists {
						assert.NotNil(t, actualStruct.BaseElement.Name,
							"结构体 %s 的名称为空",
							wantStruct.Name)
						assert.NotNil(t, actualStruct.BaseElement.Scope,
							"结构体 %s 的作用域为空",
							wantStruct.Name)
						assert.NotNil(t, actualStruct.BaseElement.Type,
							"结构体 %s 的类型为空",
							wantStruct.Name)
						assert.NotNil(t, actualStruct.BaseElement.Range,
							"结构体 %s 的范围为空",
							wantStruct.Name)

					}
				}
			}
		})
	}
}

func TestCResolver_ResolveVariable(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	testCases := []struct {
		name          string
		sourceFile    *types.SourceFile
		wantErr       error
		wantVariables []resolver.Variable
		description   string
	}{
		{
			name: "testVar.c 全部变量和字段解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/c/testVar.c",
				Content: readFile("testdata/c/testVar.c"),
			},
			wantErr: nil,
			wantVariables: []resolver.Variable{
				// 枚举值
				{BaseElement: &resolver.BaseElement{Name: "RED"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "GREEN"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "BLUE"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "ACTIVE"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "INACTIVE"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "PENDING"}, VariableType: []string{types.PrimitiveType}},

				// 联合体成员
				{BaseElement: &resolver.BaseElement{Name: "i"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "f"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "str"}, VariableType: []string{types.PrimitiveType}},

				// Point结构体成员
				{BaseElement: &resolver.BaseElement{Name: "x"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "y"}, VariableType: []string{types.PrimitiveType}},

				// Person结构体成员
				{BaseElement: &resolver.BaseElement{Name: "age"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "height"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "weight"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "gender"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "id"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "name"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "scores"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "grades"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "email"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "data_ptr"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "location"}, VariableType: []string{"Point"}},
				{BaseElement: &resolver.BaseElement{Name: "favorite_color"}, VariableType: []string{"Color"}},
				{BaseElement: &resolver.BaseElement{Name: "status"}, VariableType: []string{"Status"}},
				{BaseElement: &resolver.BaseElement{Name: "extra_info"}, VariableType: []string{"Data"}},

				// 全局基本类型变量
				{BaseElement: &resolver.BaseElement{Name: "a"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "b"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "c"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "x"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "y"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "d1"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "d2"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "ch1"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "ch2"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "uid1"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "uid2"}, VariableType: []string{types.PrimitiveType}},

				// 全局数组
				{BaseElement: &resolver.BaseElement{Name: "arr"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "matrix"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "str"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "values"}, VariableType: []string{types.PrimitiveType}},

				// 全局指针
				{BaseElement: &resolver.BaseElement{Name: "ptr1"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "ptr2"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "str_ptr"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "float_ptr"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "void_ptr"}, VariableType: []string{types.PrimitiveType}},

				// 全局结构体变量
				{BaseElement: &resolver.BaseElement{Name: "person1"}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "person2"}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "person_array"}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "person_ptr"}, VariableType: []string{"Person"}},

				// 全局联合体变量
				{BaseElement: &resolver.BaseElement{Name: "data1"}, VariableType: []string{"Data"}},
				{BaseElement: &resolver.BaseElement{Name: "data2"}, VariableType: []string{"Data"}},

				// 全局枚举变量
				{BaseElement: &resolver.BaseElement{Name: "color1"}, VariableType: []string{"Color"}},
				{BaseElement: &resolver.BaseElement{Name: "status1"}, VariableType: []string{"Status"}},

				// 静态变量
				{BaseElement: &resolver.BaseElement{Name: "static_var"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "static_point"}, VariableType: []string{"Point"}},

				// const变量
				{BaseElement: &resolver.BaseElement{Name: "const_int"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "const_float"}, VariableType: []string{types.PrimitiveType}},

				// 局部变量
				{BaseElement: &resolver.BaseElement{Name: "local_int"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "local_int2"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "local_arr"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "local_person"}, VariableType: []string{"Person"}},

				// 指针数组
				{BaseElement: &resolver.BaseElement{Name: "values"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "ptrs"}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "ptrs2"}, VariableType: []string{types.PrimitiveType}},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 遍历期望的变量，断言解析结果中存在同名变量且类型一致
				varMap := make(map[string]*resolver.Variable)
				for _, element := range res.Elements {

					if v, ok := element.(*resolver.Variable); ok {
						varMap[v.BaseElement.Name] = v
					}
				}
				for _, wantVar := range tt.wantVariables {
					actualVar, exists := varMap[wantVar.BaseElement.Name]
					assert.True(t, exists, "未找到变量: %s", wantVar.BaseElement.Name)
					if exists {
						assert.Equal(t, wantVar.VariableType, actualVar.VariableType,
							"变量 %s 的类型不匹配，期望 %v，实际 %v",
							wantVar.BaseElement.Name, wantVar.VariableType, actualVar.VariableType)
						assert.True(t, resolver.IsValidElement(actualVar))
					}
				}
			}
		})
	}
}

func TestCResolver_ResolveCall(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantCalls   []resolver.Call
		description string
	}{
		{
			name: "testCall.c 全部函数调用解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/c/testCall.c",
				Content: readFile("testdata/c/testCall.c"),
			},
			wantErr: nil,
			wantCalls: []resolver.Call{
				// 0个参数的函数调用
				{
					BaseElement: &resolver.BaseElement{
						Name: "initialize_system",
					},
					Parameters: []*resolver.Parameter{},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "get_default_config",
					},
					Parameters: []*resolver.Parameter{},
				},

				// 1个参数的函数调用
				{
					BaseElement: &resolver.BaseElement{
						Name: "free",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "parse_raw_data",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
					},
				},

				// 2个参数的函数调用
				{
					BaseElement: &resolver.BaseElement{
						Name: "fmax",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "run_with_logger",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "process_and_validate",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
					},
				},

				// 3个参数的函数调用
				{
					BaseElement: &resolver.BaseElement{
						Name: "log_student",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
						{}, // 参数3
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "get_timestamp_and_status",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
						{}, // 参数3
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "create_student",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
						{}, // 参数3
					},
				},

				// 4个参数的函数调用
				{
					BaseElement: &resolver.BaseElement{
						Name: "compute_weighted_average",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
						{}, // 参数3
						{}, // 参数4
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "custom_log",
					},
					Parameters: []*resolver.Parameter{
						{}, // 参数1
						{}, // 参数2
						{}, // 参数3
						{}, // 参数4
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 遍历期望的函数调用，断言解析结果中存在同名调用且参数个数一致
				callMap := make(map[string][]*resolver.Call)
				for _, element := range res.Elements {
					if c, ok := element.(*resolver.Call); ok {
						callMap[c.BaseElement.Name] = append(callMap[c.BaseElement.Name], c)
					}
				}

				// 验证每个期望的函数调用
				for _, expectedCall := range tt.wantCalls {
					expectedName := expectedCall.BaseElement.Name
					expectedParamCount := len(expectedCall.Parameters)

					calls, exists := callMap[expectedName]
					assert.True(t, exists, "期望找到函数调用: %s", expectedName)

					if exists {
						// 检查是否有至少一个调用的参数个数匹配
						found := false
						for _, call := range calls {
							fmt.Println("name", call.GetName())
							fmt.Println("LEN", len(call.Parameters))
							fmt.Println("--------------------------------")
							if len(call.Parameters) == expectedParamCount {
								found = true
								break
							}
						}
						assert.True(t, found, "函数 %s 应该有 %d 个参数的调用", expectedName, expectedParamCount)
					}
				}
			}
		})
	}
}
