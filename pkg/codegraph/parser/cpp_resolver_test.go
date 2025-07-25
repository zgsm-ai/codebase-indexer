package parser

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
)

func TestCPPResolver(t *testing.T) {

}
func TestCPPResolver_ResolveImport(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.CPP, "pkg/codegraph/parser/testdata")

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		description string
	}{
		{
			name: "普通头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ImportTest.cpp",
				Content: []byte(`#include "test.h"
#include "utils/helper.hpp"
`),
			},
			wantErr:     nil,
			description: "测试普通C++头文件导入",
		},
		{
			name: "系统头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/SystemImportTest.cpp",
				Content: []byte(`#include <vector>
#include <string>
`),
			},
			wantErr:     nil,
			description: "测试系统头文件导入",
		},
		{
			name: "相对路径头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/RelativeImportTest.cpp",
				Content: []byte(`#include "./local.hpp"
#include "../common.hpp"
`),
			},
			wantErr:     nil,
			description: "测试相对路径头文件导入",
		},
		{
			name: "嵌套目录头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/NestedImportTest.cpp",
				Content: []byte(`#include "nested/dir/deep.hpp"
`),
			},
			wantErr:     nil,
			description: "测试嵌套目录头文件导入",
		},
		{
			name: "混合引号和尖括号",
			sourceFile: &types.SourceFile{
				Path: "testdata/MixedImportTest.cpp",
				Content: []byte(`#include "test.h"
#include <map>
`),
			},
			wantErr:     nil,
			description: "测试混合引号和尖括号导入",
		},
		{
			name: "using声明导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/UsingImportTest.cpp",
				Content: []byte(`using namespace std;
using std::vector;
using myns::MyClass;
using myns::MyClass2;
`),
			},
			wantErr:     nil,
			description: "测试using声明导入",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			if tt.wantErr != nil {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
			assert.NotNil(t, res)
			if err == nil {
				for _, importItem := range res.Imports {
					fmt.Printf("Import: %s\n", importItem.GetName())
					assert.NotEmpty(t, importItem.GetName())
					assert.Equal(t, types.ElementTypeImport, importItem.GetType())
				}
			}
		})
	}
}

func TestCPPResolver_ResolveFunction(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.CPP, "pkg/codegraph/parser/testdata")

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantFuncs   []resolver.Declaration
		description string
	}{
		{
			name: "testfunc.cpp 全部函数声明解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/testfunc.cpp",
				Content: readFile("testdata/testfunc.cpp"),
			},
			wantErr: nil,
			wantFuncs: []resolver.Declaration{
				// 基本类型
				{Name: "getInt", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "doNothing", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "getFloat", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 指针和引用
				{Name: "getBuffer", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "count", Type: []string{types.PrimitiveType}},
				}},
				{Name: "getNameRef1", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "getNameRef2", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 标准模板容器
				{Name: "getVector", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "getMap", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 嵌套模板类型
				{Name: "getComplexMap", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 自定义模板类型
				{Name: "getBox", ReturnType: []string{"Box"}, Parameters: []resolver.Parameter{}},
				{Name: "getBoxOfVector", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{
					Name:       "getComplexMap1",
					ReturnType: []string{types.PrimitiveType},
					Parameters: []resolver.Parameter{
						{Name: "simpleMap", Type: []string{types.PrimitiveType}},
						{Name: "names", Type: []string{types.PrimitiveType}},
						{Name: "key", Type: []string{types.PrimitiveType}},
						{Name: "count", Type: []string{types.PrimitiveType}},
					},
				},

				// pair 和 tuple 类型
				{Name: "getPair", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "getTuple", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "count", Type: []string{types.PrimitiveType}}, // 有默认值，断言类型即可
				}},

				// auto 和 decltype
				{Name: "getAutoValue", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				// {Name: "getAnotherInt", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 带默认参数和命名空间返回值
				{Name: "getNames", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "count", Type: []string{types.PrimitiveType}}, // 有默认值，断言类型即可
				}},

				// 带 const 和 noexcept 的返回值
				{Name: "getConstVector", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},

				// 你补充的 15+ 个函数
				{Name: "func0", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func1", ReturnType: []string{"MyClass"}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{"MyStruct"}},
					{Name: "arg2", Type: []string{types.PrimitiveType}},
				}},
				// 泛型函数模板参数名可用T，参数名可用arg1、arg2
				{Name: "func2", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func3", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
					{Name: "arg2", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func4", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func5", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func6", ReturnType: []string{"MyStruct"}, Parameters: []resolver.Parameter{}},
				{Name: "func7", ReturnType: []string{"MyClass"}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{"MyClass"}},
					{Name: "arg2", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func8", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func9", ReturnType: []string{"MyClass"}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
					{Name: "arg2", Type: []string{"MyClass"}},
				}},
				{Name: "func10", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func11", ReturnType: []string{"MyClass"}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{"MyStruct"}},
				}},
				{Name: "func12", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
					{Name: "arg2", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func13", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func14", ReturnType: []string{"MyClass"}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{"MyClass"}},
					{Name: "arg2", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func15", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{}},
				{Name: "func16", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
					{Name: "arg2", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func17", ReturnType: []string{"MyStruct"}, Parameters: []resolver.Parameter{}},
				{Name: "func18", ReturnType: []string{types.PrimitiveType}, Parameters: []resolver.Parameter{
					{Name: "arg1", Type: []string{types.PrimitiveType}},
				}},
				{Name: "func19", ReturnType: []string{"MyClass"}, Parameters: []resolver.Parameter{}},
			},
			description: "测试 testfunc.cpp 中所有函数声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
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

func TestCPPResolver_ResolveMethod(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.CPP, "pkg/codegraph/parser/testdata")

	sourceFile := &types.SourceFile{
		Path:    "testdata/testmethod.cpp",
		Content: readFile("testdata/testmethod.cpp"),
	}

	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 收集所有方法
	methodMap := make(map[string]*resolver.Method)
	for _, element := range res.Elements {
		if m, ok := element.(*resolver.Method); ok {
			methodMap[m.Declaration.Name] = m
		}
	}

	// 断言部分典型方法
	type wantMethod struct {
		Name       string
		ReturnType []string
		ParamTypes [][]string
		Owner      string
		IsStatic   bool
	}
	cases := []wantMethod{
		// PersonStruct
		{
			Name:       "getAddressMap",
			ReturnType: []string{"Address"}, // 实际应为 map<string, vector<Address>>
			ParamTypes: [][]string{},
			Owner:      "PersonStruct",
			IsStatic:   false,
		},
		{
			Name:       "getJobList",
			ReturnType: []string{"Job"}, // list<map<string, Job>>
			ParamTypes: [][]string{},
			Owner:      "PersonStruct",
			IsStatic:   false,
		},
		{
			Name:       "getNestedInts",
			ReturnType: []string{types.PrimitiveType}, // vector<list<int>>
			ParamTypes: [][]string{},
			Owner:      "PersonStruct",
			IsStatic:   false,
		},
		{
			Name:       "sayHello",
			ReturnType: []string{types.PrimitiveType}, // void
			ParamTypes: [][]string{},
			Owner:      "PersonStruct",
			IsStatic:   false,
		},
		{
			Name:       "setAge",
			ReturnType: []string{types.PrimitiveType},     // void
			ParamTypes: [][]string{{types.PrimitiveType}}, // int newAge = 30
			Owner:      "PersonStruct",
			IsStatic:   false,
		},
		{
			Name:       "setNameAndAddress",
			ReturnType: []string{types.PrimitiveType},                  // void
			ParamTypes: [][]string{{types.PrimitiveType}, {"Address"}}, // const std::string&, const Address&
			Owner:      "PersonStruct",
			IsStatic:   false,
		},
		{
			Name:       "updateInfo",
			ReturnType: []string{types.PrimitiveType},                                                   // void
			ParamTypes: [][]string{{types.PrimitiveType}, {types.PrimitiveType}, {types.PrimitiveType}}, // const vector<int>&, int, double
			Owner:      "PersonStruct",
			IsStatic:   false,
		},

		// Job
		{
			Name:       "setSalary",
			ReturnType: []string{types.PrimitiveType},     // void
			ParamTypes: [][]string{{types.PrimitiveType}}, // double s
			Owner:      "Job",
			IsStatic:   false,
		},
		{
			Name:       "getTitle",
			ReturnType: []string{types.PrimitiveType}, // string
			ParamTypes: [][]string{},
			Owner:      "Job",
			IsStatic:   false,
		},
		{
			Name:       "getSalary",
			ReturnType: []string{types.PrimitiveType}, // double
			ParamTypes: [][]string{},
			Owner:      "Job",
			IsStatic:   false,
		},

		// PersonClass
		{
			Name:       "getAddresses",
			ReturnType: []string{"Address"}, // vector<list<Address>>
			ParamTypes: [][]string{},
			Owner:      "PersonClass",
			IsStatic:   false,
		},
		{
			Name:       "getJobMap",
			ReturnType: []string{"Job"}, // map<string, map<string, Job>>
			ParamTypes: [][]string{},
			Owner:      "PersonClass",
			IsStatic:   false,
		},
		{
			Name:       "getNestedDoubles",
			ReturnType: []string{types.PrimitiveType}, // list<vector<double>>
			ParamTypes: [][]string{},
			Owner:      "PersonClass",
			IsStatic:   false,
		},
		{
			Name:       "greet",
			ReturnType: []string{types.PrimitiveType}, // void
			ParamTypes: [][]string{},
			Owner:      "PersonClass",
			IsStatic:   false,
		},
		{
			Name:       "setHeight",
			ReturnType: []string{types.PrimitiveType},     // void
			ParamTypes: [][]string{{types.PrimitiveType}}, // double newHeight = 170.5
			Owner:      "PersonClass",
			IsStatic:   false,
		},
		{
			Name:       "setJobAndAge",
			ReturnType: []string{types.PrimitiveType},              // void
			ParamTypes: [][]string{{"Job"}, {types.PrimitiveType}}, // const Job&, int age = 25
			Owner:      "PersonClass",
			IsStatic:   false,
		},
		{
			Name:       "updateStats",
			ReturnType: []string{types.PrimitiveType},                                                   // void
			ParamTypes: [][]string{{types.PrimitiveType}, {types.PrimitiveType}, {types.PrimitiveType}}, // const list<int>&, int, double
			Owner:      "PersonClass",
			IsStatic:   false,
		},
	}

	for _, c := range cases {
		m, ok := methodMap[c.Name]
		assert.True(t, ok, "未找到方法: %s", c.Name)
		if ok {
			assert.Equal(t, c.ReturnType, m.Declaration.ReturnType, "方法 %s 返回值类型不符", c.Name)
			assert.Equal(t, len(c.ParamTypes), len(m.Declaration.Parameters), "方法 %s 参数数量不符", c.Name)
			for i, wantParamType := range c.ParamTypes {
				assert.Equal(t, wantParamType, m.Declaration.Parameters[i].Type, "方法 %s 第%d个参数类型不符", c.Name, i+1)
			}
			assert.Contains(t, m.Owner, c.Owner, "方法 %s 所属类不符", c.Name)
			if c.IsStatic {
				assert.Contains(t, m.Declaration.Modifier, "static", "方法 %s 应为static", c.Name)
			}
		}
	}
}

func TestCPPResolver_ResolveCall(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.CPP, "pkg/codegraph/parser/testdata")

	sourceFile := &types.SourceFile{
		Path:    "testdata/testcall.cpp",
		Content: readFile("testdata/testcall.cpp"),
	}
	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Collect all function calls
	callMap := make(map[string]*resolver.Call)
	for _, element := range res.Elements {
		if call, ok := element.(*resolver.Call); ok {
			callMap[call.GetName()] = call
		}
	}

	// Test cases for different types of function calls
	testCases := []struct {
		name          string
		expectedOwner string
		paramCount    int
	}{
		{"freeFunction", "", 3},          // Free function call (int, double, char)
		{"nsFunction", "MyNamespace", 3}, // Namespace function call (int, int, int)
		{"memberFunction", "obj", 2},     // Object member function call (int, double)
		{"memberFunction1", "ptr", 3},     // Pointer member function call (int, double)
		{"staticFunction", "MyClass", 2}, // Static member function call (int, int)
		{"templatedFunction", "", 4},     // Template function call (4 args via generic lambda)
		{"lambda", "", 3},                // Lambda function call (int, int, int)
		{"fp", "", 3},                    // Function pointer call (int, double, char)
		{"obj", "", 4},                     // Function object call (int, int, int, int)
		{"append", "str", 2},             // Method chaining (first call) (const char*, size_t)
		{"at", "", 1},                    // Method chaining (second call) (size_t)
	}

	// Verify each expected function call
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Call_%s", tc.name), func(t *testing.T) {
			// For each test case, there might be multiple calls with the same name
			// (like memberFunction is called twice), so we need to find at least one match
			found := false
			for _, call := range callMap {
				if call.GetName() == tc.name {
					// If owner is specified, it must match
					if tc.expectedOwner != "" && !strings.Contains(call.Owner, tc.expectedOwner) {
						continue
					}
					// Verify parameter count
					assert.Equal(t, tc.paramCount, len(call.Parameters),
						"Call %s should have %d parameters", tc.name, tc.paramCount)
					found = true
					break
				}
			}
			assert.True(t, found, "Call to %s not found", tc.name)
		})
	}
}

func TestCPPResolver_ResolveVariable(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.CPP, "pkg/codegraph/parser/testdata")

	sourceFile := &types.SourceFile{
		Path:    "testdata/testvar.cpp",
		Content: readFile("testdata/testvar.cpp"),
	}
	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 收集所有变量
	variableMap := make(map[string]*resolver.Variable)
	for _, element := range res.Elements {
		if v, ok := element.(*resolver.Variable); ok {
			variableMap[v.BaseElement.Name] = v
		}
	}

	// 断言部分典型变量
	type wantVariable struct {
		Name string
		Type []string
	}
	typicalVars := []wantVariable{
		{Name: "value", Type: []string{types.PrimitiveType}},
		{Name: "a", Type: []string{types.PrimitiveType}},
		{Name: "b", Type: []string{types.PrimitiveType}},
		{Name: "c",Type: []string{types.PrimitiveType}},
		{Name: "raw_ptr", Type: []string{types.PrimitiveType}},
		{Name: "raw_ptr2", Type: []string{types.PrimitiveType}},
		{Name: "ref_a", Type: []string{types.PrimitiveType}},
		{Name: "name_ref", Type: []string{types.PrimitiveType}},
		{Name: "text", Type: []string{types.PrimitiveType}},
		{Name: "greeting", Type: []string{types.PrimitiveType}},
		{Name: "pt", Type: []string{"Point"}},
		{Name: "pt_init", Type: []string{"Point"}},
		{Name: "counter", Type: []string{types.PrimitiveType}},
		{Name: "dirty_flag", Type: []string{types.PrimitiveType}},
		{Name: "version", Type: []string{types.PrimitiveType}},
		{Name: "nums", Type: []string{types.PrimitiveType}},
		{Name: "nums_init", Type: []string{types.PrimitiveType}},
		{Name: "vec", Type: []string{types.PrimitiveType}},
		{Name: "vec_init", Type: []string{types.PrimitiveType}},
		{Name: "flag", Type: []string{types.PrimitiveType}},
		{Name: "radius", Type: []string{types.PrimitiveType}},
		{Name: "x", Type: []string{types.PrimitiveType}},
		{Name: "dummy", Type: []string{types.PrimitiveType}},
		{Name: "y", Type: []string{types.PrimitiveType}},
		{Name: "local_a", Type: []string{types.PrimitiveType}},
		{Name: "local_b", Type: []string{types.PrimitiveType}},
		{Name: "local_c", Type: []string{types.PrimitiveType}},
		{Name: "local_d", Type: []string{types.PrimitiveType}},
		{Name: "local_const", Type: []string{types.PrimitiveType}},
		{Name: "local_volatile_flag", Type: []string{types.PrimitiveType}},
		{Name: "local_ptr", Type: []string{types.PrimitiveType}},
		{Name: "local_cstr", Type: []string{types.PrimitiveType}},
		{Name: "local_float_ptr", Type: []string{types.PrimitiveType}},
		{Name: "local_ref", Type: []string{types.PrimitiveType}},
		{Name: "local_str_ref", Type: []string{types.PrimitiveType}},
		{Name: "local_arr", Type: []string{types.PrimitiveType}},
		{Name: "local_ptr2", Type: []string{types.PrimitiveType}},
		{Name: "local_ptr3", Type: []string{types.PrimitiveType}},
		{Name: "local_ref2", Type: []string{types.PrimitiveType}},
		{Name: "local_arr_init", Type: []string{types.PrimitiveType}},
		{Name: "local_chars", Type: []string{types.PrimitiveType}},
		{Name: "local_name", Type: []string{types.PrimitiveType}},
		{Name: "local_vec", Type: []string{types.PrimitiveType}},
		{Name: "data", Type: []string{"shapes_ns::ShapeData"}},
		{Name: "data_ptr", Type: []string{"shapes_ns::ShapeData"}},
		{Name: "w", Type: []string{"Widget"}},
		{Name: "w_ptr", Type: []string{"Widget"}},
		{Name: "shape", Type: []string{"IShape"}},
		{Name: "auto_int", Type: []string{types.PrimitiveType}},
		{Name: "auto_str", Type: []string{types.PrimitiveType}},
		{Name: "auto_vec_ref", Type: []string{types.PrimitiveType}},
		{Name: "loop_i", Type: []string{types.PrimitiveType}},
		{Name: "loop_j", Type: []string{types.PrimitiveType}},
		{Name: "loop_k", Type: []string{types.PrimitiveType}},
		{Name: "loop_u", Type: []string{types.PrimitiveType}},
		{Name: "loop_v", Type: []string{types.PrimitiveType}},
		{Name: "temp_pt", Type: []string{"TempPoint"}},	
	}

	for _, want := range typicalVars {
		v, ok := variableMap[want.Name]
		assert.True(t, ok, "变量 %s 未被解析", want.Name)
		if ok {
			assert.Equal(t, want.Type, v.VariableType, "变量 %s 类型不符", want.Name)
		}
	}
}




func TestCPPResolver_ResolveStruct(t *testing.T) {
	param := `

		struct Vec2 { float x, y; };
	`

	reStruct := regexp.MustCompile(`struct\s+(\w+)\s*\{`)
	matches := reStruct.FindAllStringSubmatch(param, -1)

	for _, match := range matches {
		// match[0] 是整个匹配（struct Point {...}），match[1] 是结构体名
		fmt.Println("Struct name:", match[1])
	}
}
