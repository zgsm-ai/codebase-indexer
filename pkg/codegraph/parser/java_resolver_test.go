package parser

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJavaResolver_ResolveImport(t *testing.T) {

	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata")

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		description string
	}{
		{
			name: "正常类导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ImportTest.java",
				Content: []byte(`package com.example;
import java.util.List;
import java.util.ArrayList;
import static java.lang.Math.PI;
import com.example.utils.*;`),
			},
			wantErr:     nil,
			description: "测试正常的Java导入解析",
		},
		{
			name: "包通配符导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/WildcardImportTest.java",
				Content: []byte(`package com.example;
import java.util.*;
import java.io.*;`),
			},
			wantErr:     nil,
			description: "测试包通配符导入解析",
		},
		{
			name: "静态导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/StaticImportTest.java",
				Content: []byte(`package com.example;
import static java.lang.Math.PI;
import static java.lang.Math.abs;
import static java.util.Collections.emptyList;`),
			},
			wantErr:     nil,
			description: "测试静态导入解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 验证导入解析
				fmt.Println(len(res.Imports))
				for _, importItem := range res.Imports {
					fmt.Printf("Import: %s\n", importItem.GetName())
					assert.NotEmpty(t, importItem.GetName())
					assert.Equal(t, types.ElementTypeImport, importItem.GetType())
				}
			}
		})
	}
}

func TestJavaResolver_ResolveClass(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata")

	sourceFile := &types.SourceFile{
		Path:    "testdata/com/example/test/TestClass.java",
		Content: readFile("testdata/com/example/test/TestClass.java"),
	}

	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 1. 平铺输出所有类的详细信息
	fmt.Println("\n【所有类详细信息】")
	for _, element := range res.Elements {
		cls, ok := element.(*resolver.Class)
		if !ok {
			continue
		}
		fmt.Printf("类名: %s\n", cls.GetName())
		fmt.Printf("  作用域: %v\n", cls.BaseElement.Scope)
		if len(cls.SuperClasses) > 0 {
			fmt.Printf("  父类: %v\n", cls.SuperClasses)
		} else {
			fmt.Printf("  父类: 无\n")
		}
		if len(cls.SuperInterfaces) > 0 {
			fmt.Printf("  实现接口: %v\n", cls.SuperInterfaces)
		} else {
			fmt.Printf("  实现接口: 无\n")
		}
		if len(cls.Fields) > 0 {
			fmt.Println("  字段:")
			for _, field := range cls.Fields {
				fmt.Printf("    %s %s %s\n", field.Modifier, field.Type, field.Name)
			}
		} else {
			fmt.Println("  字段: 无")
		}
		if len(cls.Methods) > 0 {
			fmt.Println("  方法:")
			for _, method := range cls.Methods {
				fmt.Printf("    %s %s %s(", method.Declaration.Modifier, method.Declaration.ReturnType, method.Declaration.Name)
				for i, param := range method.Declaration.Parameters {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%s %s", param.Type, param.Name)
				}
				fmt.Println(")")
			}
		} else {
			fmt.Println("  方法: 无")
		}
		fmt.Println("--------------------------------")
	}

	// 2. 断言所有类的结构和内容
	// 期望类信息
	expectedClasses := map[string]struct {
		Scope        types.Scope
		SuperClasses []string
		SuperIfaces  []string
		Fields       []resolver.Field
		Methods      []string // 只断言方法名
	}{
		"ReportGenerator": {
			Scope:        types.ScopePackage,
			SuperClasses: nil,
			SuperIfaces:  []string{"Printable", "Savable"},
			Fields: []resolver.Field{
				{Modifier: "private", Type: "int", Name: "reportId"},
				{Modifier: "protected", Type: "String", Name: "title"},
				{Modifier: "public static final", Type: "String", Name: "VERSION"},
				{Modifier: types.PackagePrivate, Type: "int", Name: "a"},
				{Modifier: types.PackagePrivate, Type: "int", Name: "b"},
				{Modifier: types.PackagePrivate, Type: "int", Name: "c"},
			},
			Methods: []string{"print", "save", "ReportGenerator"},
		},
		"ReportDetails": {
			Scope:        types.ScopeProject,
			SuperClasses: nil,
			SuperIfaces:  nil,
			Fields: []resolver.Field{
				{Modifier: types.PackagePrivate, Type: "boolean", Name: "verified"},
			},
			Methods: []string{"verify"},
		},
		"InternalReview": {
			Scope:        types.ScopeClass,
			SuperClasses: nil,
			SuperIfaces:  nil,
			Fields: []resolver.Field{
				{Modifier: types.PackagePrivate, Type: "char", Name: "level"},
			},
			Methods: nil,
		},
		"ReportMetadata": {
			Scope:        types.ScopePackage,
			SuperClasses: nil,
			SuperIfaces:  nil,
			Fields: []resolver.Field{
				{Modifier: types.PackagePrivate, Type: "long", Name: "createdAt"},
			},
			Methods: []string{"describe"},
		},
		"User": {
			Scope:        types.ScopePackage,
			SuperClasses: nil,
			SuperIfaces:  nil,
			Fields: []resolver.Field{
				{Modifier: "protected", Type: "String", Name: "username"},
				{Modifier: "public", Type: "int", Name: "age"},
			},
			Methods: []string{"login"},
		},
		"FinancialReport": {
			Scope:        types.ScopeProject, // 若有 types.ScopePublic 则用，否则用 ScopePackage
			SuperClasses: []string{"User"},
			SuperIfaces:  []string{"Printable", "Savable"},
			Fields: []resolver.Field{
				{Modifier: "public", Type: "List<String>", Name: "authors"},
				{Modifier: "protected", Type: "Map<String, Double>", Name: "monthlyRevenue"},
				{Modifier: "private final", Type: "ReportGenerator", Name: "generator"},
				{Modifier: types.PackagePrivate, Type: "List<? extends Number>", Name: "statistics"},
				{Modifier: types.PackagePrivate, Type: "ReportGenerator[]", Name: "reports"},
			},
			Methods: []string{"FinancialReport", "main", "print", "save", "prepareData", "calculateProfit"},
		},
	}

	// 遍历所有期望类，逐一断言
	for className, want := range expectedClasses {
		found := false
		for _, element := range res.Elements {
			cls, ok := element.(*resolver.Class)
			if !ok {
				continue
			}
			if cls.GetName() != className {
				continue
			}
			found = true
			assert.Equal(t, want.Scope, cls.BaseElement.Scope, "类 %s 作用域不匹配", className)
			// 父类
			if want.SuperClasses != nil {
				assert.Equal(t, want.SuperClasses, cls.SuperClasses, "类 %s 父类不匹配", className)
			}
			// 接口
			if want.SuperIfaces != nil {
				assert.Equal(t, want.SuperIfaces, cls.SuperInterfaces, "类 %s 实现接口不匹配", className)
			}
			// 字段
			if want.Fields != nil {
				assert.Len(t, cls.Fields, len(want.Fields), "类 %s 字段数量不匹配", className)
				actualFields := make(map[string]*resolver.Field)
				for _, field := range cls.Fields {
					actualFields[field.Name] = field
				}

				for _, wantField := range want.Fields {
					actualField, exists := actualFields[wantField.Name]

					assert.True(t, exists, "类 %s 未找到字段: %s", className, wantField.Name)
					if exists {
						assert.Equal(t, wantField.Modifier, actualField.Modifier, "类 %s 字段 %s 修饰符不匹配", className, wantField.Name)
						assert.Equal(t, wantField.Type, actualField.Type, "类 %s 字段 %s 类型不匹配", className, wantField.Name)
					}
				}
			}
			// 方法
			if want.Methods != nil {
				actualMethods := make(map[string]bool)
				for _, method := range cls.Methods {
					actualMethods[method.Declaration.Name] = true
				}
				for _, wantMethod := range want.Methods {
					assert.True(t, actualMethods[wantMethod], "类 %s 未找到方法: %s", className, wantMethod)
				}
			}
			break
		}
		assert.True(t, found, "未找到类: %s", className)
	}
}

func TestJavaResolver_ResolveVariable(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata")

	testCases := []struct {
		name          string
		sourceFile    *types.SourceFile
		wantErr       error
		wantVariables []resolver.Variable
		description   string
	}{
		{
			name: "TestVar.java 全变量类型校验",
			sourceFile: &types.SourceFile{
				Path:    "testdata/com/example/test/TestVar.java",
				Content: readFile("testdata/com/example/test/TestVar.java"),
			},
			wantErr: nil,
			wantVariables: []resolver.Variable{
				{BaseElement: &resolver.BaseElement{Name: "number", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "price", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "isActive", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "initial", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "array", Type: types.ElementTypeLocalVariable}, VariableType: []string{"MyClass"}},
				{BaseElement: &resolver.BaseElement{Name: "name", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "age", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "tags", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "scoreMap", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "person", Type: types.ElementTypeLocalVariable}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "numbers", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "names", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "people", Type: types.ElementTypeLocalVariable}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "personList", Type: types.ElementTypeLocalVariable}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "personMap", Type: types.ElementTypeLocalVariable}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "task", Type: types.ElementTypeLocalVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "wildcardList", Type: types.ElementTypeLocalVariable}, VariableType: []string{"Person"}},
			},
			description: "测试 TestVar.java 中所有变量的类型解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("--------------------------------\n")
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望变量数量: %d\n", len(tt.wantVariables))

				// 收集所有变量
				var actualVariables []*resolver.Variable
				for _, element := range res.Elements {
					if variable, ok := element.(*resolver.Variable); ok {
						actualVariables = append(actualVariables, variable)
						fmt.Printf("变量: %s, Type: %s, VariableType: %s\n", variable.GetName(), variable.GetType(), variable.VariableType)
					}
				}

				fmt.Printf("实际变量数量: %d\n", len(actualVariables))

				// 验证变量数量
				assert.Len(t, actualVariables, len(tt.wantVariables),
					"变量数量不匹配，期望 %d，实际 %d", len(tt.wantVariables), len(actualVariables))

				// 创建实际变量的映射
				actualVarMap := make(map[string]*resolver.Variable)
				for _, variable := range actualVariables {
					actualVarMap[variable.GetName()] = variable
				}

				// 逐个比较每个期望的变量
				for _, wantVariable := range tt.wantVariables {
					actualVariable, exists := actualVarMap[wantVariable.GetName()]
					assert.True(t, exists, "未找到变量: %s", wantVariable.GetName())

					if exists {
						// 验证变量名称
						assert.Equal(t, wantVariable.GetName(), actualVariable.GetName(),
							"变量名称不匹配，期望 %s，实际 %s",
							wantVariable.GetName(), actualVariable.GetName())

						// 验证变量类型
						assert.Equal(t, wantVariable.GetType(), actualVariable.GetType(),
							"变量类型不匹配，期望 %s，实际 %s",
							wantVariable.GetType(), actualVariable.GetType())

						// 验证变量的 VariableType 字段
						assert.Equal(t, wantVariable.VariableType, actualVariable.VariableType,
							"变量 %s 的VariableType不匹配，期望 %s，实际 %s",
							wantVariable.GetName(), wantVariable.VariableType, actualVariable.VariableType)
					}
				}
			}
		})
	}
}

func TestJavaResolver_ResolveInterface(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata")

	testCases := []struct {
		name          string
		sourceFile    *types.SourceFile
		wantErr       error
		wantIfaceName string
		wantMethods   []resolver.Declaration // 使用完整的 Declaration 结构
		description   string
	}{
		{
			name: "简单接口声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/SimpleInterfaceTest.java",
				Content: []byte(`package com.example;
public interface SimpleInterface {
    void doSomething();
    int getValue();
}`),
			},
			wantErr:       nil,
			wantIfaceName: "SimpleInterface",
			wantMethods: []resolver.Declaration{
				{
					Modifier:   "public abstract",
					Name:       "doSomething",
					ReturnType: []string{types.PrimitiveType},
					Parameters: []resolver.Parameter{},
				},
				{
					Modifier:   "public abstract",
					Name:       "getValue",
					ReturnType: []string{types.PrimitiveType},
					Parameters: []resolver.Parameter{},
				},
			},
			description: "测试简单接口声明解析",
		},
		{
			name: "Printable接口声明",
			sourceFile: &types.SourceFile{
				Path:    "testdata/com/example/test/TestClass.java",
				Content: readFile("testdata/com/example/test/TestClass.java"),
			},
			wantErr:       nil,
			wantIfaceName: "Printable",
			wantMethods: []resolver.Declaration{
				{
					Modifier:   "public abstract",
					Name:       "print",
					ReturnType: []string{types.PrimitiveType},
					Parameters: []resolver.Parameter{},
				},
			},
			description: "测试Printable接口声明解析",
		},
		{
			name: "Savable接口声明",
			sourceFile: &types.SourceFile{
				Path:    "testdata/com/example/test/TestClass.java",
				Content: readFile("testdata/com/example/test/TestClass.java"),
			},
			wantErr:       nil,
			wantIfaceName: "Savable",
			wantMethods: []resolver.Declaration{
				{
					Modifier:   "public abstract",
					Name:       "save",
					ReturnType: []string{types.PrimitiveType},
					Parameters: []resolver.Parameter{
						{Name: "destination", Type: []string{types.PrimitiveType}},
					},
				},
			},
			description: "测试Savable接口声明解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 1. 收集所有接口
				ifaceMap := make(map[string]*resolver.Interface)
				for _, element := range res.Elements {
					if iface, ok := element.(*resolver.Interface); ok {
						ifaceMap[iface.GetName()] = iface
					}
				}

				// 2. 查找目标接口
				iface, exists := ifaceMap[tt.wantIfaceName]
				assert.True(t, exists, "未找到接口类型: %s", tt.wantIfaceName)
				if exists {
					assert.Equal(t, types.ElementTypeInterface, iface.GetType())

					// 验证方法数量
					assert.Equal(t, len(tt.wantMethods), len(iface.Methods),
						"方法数量不匹配，期望 %d，实际 %d", len(tt.wantMethods), len(iface.Methods))

					// 创建实际方法的映射，用于比较
					actualMethods := make(map[string]*resolver.Declaration)
					for _, method := range iface.Methods {
						actualMethods[method.Name] = method
					}

					// 逐个比较每个期望的方法
					for _, wantMethod := range tt.wantMethods {
						actualMethod, exists := actualMethods[wantMethod.Name]
						assert.True(t, exists, "未找到方法: %s", wantMethod.Name)

						if exists {
							// 比较修饰符
							assert.Equal(t, wantMethod.Modifier, actualMethod.Modifier,
								"方法 %s 的修饰符不匹配，期望 %s，实际 %s",
								wantMethod.Name, wantMethod.Modifier, actualMethod.Modifier)

							// 比较返回值类型
							assert.Equal(t, wantMethod.ReturnType, actualMethod.ReturnType,
								"方法 %s 的返回值类型不匹配，期望 %s，实际 %s",
								wantMethod.Name, wantMethod.ReturnType, actualMethod.ReturnType)

							// 比较参数数量
							assert.Equal(t, len(wantMethod.Parameters), len(actualMethod.Parameters),
								"方法 %s 的参数数量不匹配，期望 %d，实际 %d",
								wantMethod.Name, len(wantMethod.Parameters), len(actualMethod.Parameters))

							// 比较参数详情
							for i, wantParam := range wantMethod.Parameters {
								if i < len(actualMethod.Parameters) {
									actualParam := actualMethod.Parameters[i]
									assert.Equal(t, wantParam.Name, actualParam.Name,
										"方法 %s 的第 %d 个参数名称不匹配，期望 %s，实际 %s",
										wantMethod.Name, i+1, wantParam.Name, actualParam.Name)

									assert.Equal(t, wantParam.Type, actualParam.Type,
										"方法 %s 的第 %d 个参数类型不匹配，期望 %s，实际 %s",
										wantMethod.Name, i+1, wantParam.Type, actualParam.Type)
								}
							}
						}
					}
				}
			}
		})
	}
}

func TestJavaResolver_ResolveLocalVariableValue(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata")
	sourceFile := &types.SourceFile{
		Path: "testdata/com/example/test/TestClass.java",
		Content: []byte(`
			package com.example.test;
			public class TestClass {
				public void test() {
					int a = 1;
					String b = "hello";
					double c = 3.14;
					Person p = new Person("Alice", 30);
					Map<String, Integer> map = new HashMap<>();
					Set<Double> set = new HashSet<>();
					Runnable localRunnable = new Runnable() {
						@Override
						public void run() {
							System.out.println("Inner Runnable");
						}
					};
				}
			}
		`),
	}
	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.ErrorIs(t, err, nil)
	assert.NotNil(t, res)

	// 期望的变量名和类型
	expected := map[string]types.ElementType{
		"a":             types.ElementTypeLocalVariable,
		"b":             types.ElementTypeLocalVariable,
		"c":             types.ElementTypeLocalVariable,
		"p":             types.ElementTypeLocalVariable,
		"map":           types.ElementTypeLocalVariable,
		"set":           types.ElementTypeLocalVariable,
		"localRunnable": types.ElementTypeLocalVariable,
	}

	found := map[string]bool{}
	cnt := 0
	for _, element := range res.Elements {
		if variable, ok := element.(*resolver.Variable); ok {
			cnt += 1
			name := variable.GetName()
			typ := variable.GetType()
			fmt.Println("name:", name, "typ:", typ)
			if wantType, ok := expected[name]; ok {
				assert.Equal(t, wantType, typ, "变量 %s 类型不匹配", name)
				found[name] = true
			}
		}
	}
	// 检查所有期望变量都被找到
	for name := range expected {
		assert.True(t, found[name], "未找到变量: %s", name)
	}
	if cnt != len(expected) {
		t.Errorf("变量数量不匹配，期望 %d，实际 %d", len(expected), cnt)
	}
}

func TestJavaResolver_ResolveMethod(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata")

	sourceFile := &types.SourceFile{
		Path:    "testdata/com/example/test/TestClass.java",
		Content: readFile("testdata/com/example/test/TestClass.java"),
	}

	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 期望方法信息（可根据需要补充更多方法）
	expectedMethods := map[string]struct {
		ReturnType []string
		Parameters []resolver.Parameter
	}{
		// ReportGenerator
		"print": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{},
		},
		"save": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "destination", Type: []string{types.PrimitiveType}},
			},
		},
		"verify": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{},
		},
		"describe": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{},
		},
		// User
		"login": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{},
		},
		// FinancialReport
		"main": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "args", Type: []string{types.PrimitiveType}},
			},
		},
		"prepareData": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{},
		},
		"calculateProfit": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "revenue", Type: []string{types.PrimitiveType}},
				{Name: "cost", Type: []string{types.PrimitiveType}},
			},
		},
		"getReportMap": {
			ReturnType: []string{"ReportGenerator"},
			Parameters: []resolver.Parameter{
				{Name: "users", Type: []string{"User"}},
				{Name: "revenueMap", Type: []string{types.PrimitiveType}},
				{Name: "year", Type: []string{types.PrimitiveType}},
			},
		},
		"getAllReports": {
			ReturnType: []string{"FinancialReport"},
			Parameters: []resolver.Parameter{},
		},
		"buildComplexStructure": {
			ReturnType: []string{"ReportGenerator"},
			Parameters: []resolver.Parameter{
				{Name: "count", Type: []string{types.PrimitiveType}},
				{Name: "names", Type: []string{types.PrimitiveType}},
				{Name: "generator", Type: []string{"ReportGenerator"}},
			},
		},
		"processUsers": {
			ReturnType: []string{"User"},
			Parameters: []resolver.Parameter{
				{Name: "userArray", Type: []string{"User"}},
			},
		},
		"getUserArray": {
			ReturnType: []string{"User"},
			Parameters: []resolver.Parameter{
				{Name: "size", Type: []string{types.PrimitiveType}},
			},
		},
		"getUserList": {
			ReturnType: []string{"User"},
			Parameters: []resolver.Parameter{
				{Name: "users", Type: []string{"User"}},
			},
		},
		"transformMatrix": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "matrix", Type: []string{types.PrimitiveType}},
			},
		},
		"filterList": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "list", Type: []string{types.PrimitiveType}},
				{Name: "threshold", Type: []string{types.PrimitiveType}},
			},
		},
		"groupUsersByPrefix": {
			ReturnType: []string{"User"},
			Parameters: []resolver.Parameter{
				{Name: "prefix", Type: []string{types.PrimitiveType}},
				{Name: "users", Type: []string{"User"}},
			},
		},
		"createTask": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "message", Type: []string{types.PrimitiveType}},
			},
		},
		"factorial": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "n", Type: []string{types.PrimitiveType}},
			},
		},
		"checkReportId": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "id", Type: []string{types.PrimitiveType}},
			},
		},
		"printAllReports": {
			ReturnType: []string{types.PrimitiveType},
			Parameters: []resolver.Parameter{
				{Name: "reports", Type: []string{"ReportGenerator"}},
			},
		},
		"findUserByName": {
			ReturnType: []string{"User"},
			Parameters: []resolver.Parameter{
				{Name: "users", Type: []string{"User"}},
				{Name: "name", Type: []string{types.PrimitiveType}},
			},
		},
	}

	// 收集所有方法
	methodMap := make(map[string]*resolver.Declaration)
	for _, element := range res.Elements {
		if method, ok := element.(*resolver.Method); ok {
			fmt.Println("method.Declaration.Name", method.Declaration.Name)
			methodMap[method.Declaration.Name] = &method.Declaration
		}
	}

	// 断言每个期望方法
	for methodName, want := range expectedMethods {
		actual, exists := methodMap[methodName]
		assert.True(t, exists, "未找到方法: %s", methodName)
		if exists {
			assert.Equal(t, want.ReturnType, actual.ReturnType, "方法 %s 返回值类型不匹配", methodName)
			assert.Equal(t, len(want.Parameters), len(actual.Parameters), "方法 %s 参数数量不匹配", methodName)
			for i, wantParam := range want.Parameters {
				assert.Equal(t, wantParam.Type, actual.Parameters[i].Type, "方法 %s 的第 %d 个参数类型不匹配", methodName, i+1)
				// 参数名可选断言，如需严格可加上
				// assert.Equal(t, wantParam.Name, actual.Parameters[i].Name, "方法 %s 的第 %d 个参数名不匹配", methodName, i+1)
			}
		}
	}
}

func TestJavaResolver_AllResolveMethods(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata")

	source := []byte(`
		package com.example.test;

		import java.util.List;
		import java.util.Map;
		import java.util.Set;
		import static java.lang.Math.PI;

		public class Base {
			protected int id;
		}

		public interface Named {
			String getName();
			
		}

		public interface Ageable {
			int getAge();
		}

		// 继承Base，实现Named接口
		public class Person extends Base implements Named {
			private String name;
			private int age;
			private List<String> tags;
			private Map<String, List<Integer>> scores;
			private Set<? extends Number> numbers;
			private static final double PI_VALUE = PI;
			public Person(String name, int age) {
				this.name = name;
				this.age = age;
			}
			public String getName() { return name; }
		}

		// 既继承Base又实现多个接口
		public class Student extends Base implements Named, Ageable {
			private String name;
			private int age;
			private List<String[]> matrix;
			public Student(String name, int age) {
				this.name = name;
				this.age = age;
			}
			public String getName() { return name; }
			public int getAge() { return age; }
		}

		// 只实现接口
		public class Teacher implements Named {
			private String name;
			public Teacher(String name) { this.name = name; }
			public String getName() { return name; }
		}

		public class TestClass {
			public void test() {
				int a = 1;
				Person p = new Person("Alice", 30);
				Student s = new Student("Bob", 20);
				Teacher t = new Teacher("Tom");
				double pi = PI;
				List<String> list = null;
				Map<String, Integer> map = null;
				Set<Double> set = null;
				List<String[]> matrix = null;
				sayHello();
			}
			public void sayHello() {}
		}
	`)

	sourceFile := &types.SourceFile{
		Path:    "testdata/com/example/test/AllTest.java",
		Content: source,
	}

	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.ErrorIs(t, err, nil)
	assert.NotNil(t, res)

	// 1. 包
	assert.NotNil(t, res.Package)
	fmt.Printf("【包】%s\n", res.Package.GetName())
	assert.Equal(t, "com.example.test", res.Package.GetName())

	// 2. 导入
	assert.NotNil(t, res.Imports)
	fmt.Printf("【导入】数量: %d\n", len(res.Imports))
	for _, ipt := range res.Imports {
		fmt.Printf("  导入: %s\n", ipt.GetName())
	}
	importNames := map[string]bool{}
	for _, ipt := range res.Imports {
		importNames[ipt.GetName()] = true
	}
	assert.True(t, importNames["java.util.List"])
	assert.True(t, importNames["java.util.Map"])
	assert.True(t, importNames["java.util.Set"])
	assert.True(t, importNames["java.lang.Math.PI"])

	// 3. 类
	for _, element := range res.Elements {
		if cls, ok := element.(*resolver.Class); ok {
			fmt.Printf("【类】%s, 字段: %d, 方法: %d, 继承: %v, 实现: %v\n",
				cls.GetName(), len(cls.Fields), len(cls.Methods), cls.SuperClasses, cls.SuperInterfaces)
			for _, field := range cls.Fields {
				fmt.Printf("  字段: %s %s %s\n", field.Modifier, field.Type, field.Name)
			}
			for _, method := range cls.Methods {
				fmt.Printf("  方法: %s %s %s(%v)\n", method.Declaration.Modifier, method.Declaration.ReturnType, method.Declaration.Name, method.Declaration.Parameters)
			}
		}
	}

	// 4. 接口
	for _, element := range res.Elements {
		if iface, ok := element.(*resolver.Interface); ok {
			fmt.Printf("【接口】%s, 方法: %d\n", iface.GetName(), len(iface.Methods))
			for _, method := range iface.Methods {
				fmt.Printf("  方法: %s %s %s(%v)\n", method.Modifier, method.ReturnType, method.Name, method.Parameters)
			}
		}
	}

	// 5. 变量
	for _, element := range res.Elements {
		if variable, ok := element.(*resolver.Variable); ok {
			fmt.Printf("【变量】%s, 类型: %s\n", variable.GetName(), variable.GetType())
		}
	}

	// 6. 方法调用
	for _, element := range res.Elements {
		if call, ok := element.(*resolver.Call); ok {
			fmt.Printf("【方法调用】%s, 所属: %s\n", call.GetName(), call.Owner)
			assert.Equal(t, types.ElementTypeMethodCall, call.GetType())
		}
	}
}
