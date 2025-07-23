package parser

import (
	"context"
	"fmt"
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
