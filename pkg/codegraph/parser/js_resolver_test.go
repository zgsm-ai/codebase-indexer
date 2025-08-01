package parser

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJavaScriptResolver_ResolveImport(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantImports []resolver.Import
		description string
	}{
		{
			name: "默认导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_import_test.js",
				Content: []byte(`import defaultName from 'modules.js';
import * as moduleName from 'modules.js';
import { export1, export2 } from 'modules.js';`),
			},
			wantErr: nil,
			wantImports: []resolver.Import{
				{BaseElement: &resolver.BaseElement{Name: "defaultName", Type: types.ElementTypeImport}, Source: "modules.js"},
				{BaseElement: &resolver.BaseElement{Name: "", Type: types.ElementTypeImport}, Source: "modules.js", Alias: "moduleName"},
				{BaseElement: &resolver.BaseElement{Name: "export1", Type: types.ElementTypeImport}, Source: "modules.js"},
				{BaseElement: &resolver.BaseElement{Name: "export2", Type: types.ElementTypeImport}, Source: "modules.js"},
			},
			description: "测试JavaScript默认导入语法",
		},
		{
			name: "命名导入和别名",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_named_import_test.js",
				Content: []byte(`import { export as ex1 } from 'modules';
import { export1 as ex1, export2 as ex2 } from 'moduls.js';
import defaultName, { export } from './modules';`),
			},
			wantErr: nil,
			wantImports: []resolver.Import{
				{BaseElement: &resolver.BaseElement{Name: "export", Type: types.ElementTypeImport}, Source: "modules", Alias: "ex1"},
				{BaseElement: &resolver.BaseElement{Name: "export1", Type: types.ElementTypeImport}, Source: "moduls.js", Alias: "ex1"},
				{BaseElement: &resolver.BaseElement{Name: "export2", Type: types.ElementTypeImport}, Source: "moduls.js", Alias: "ex2"},
				{BaseElement: &resolver.BaseElement{Name: "defaultName", Type: types.ElementTypeImport}, Source: "./modules"},
				{BaseElement: &resolver.BaseElement{Name: "export", Type: types.ElementTypeImport}, Source: "./modules"},
			},
			description: "测试JavaScript命名导入和别名",
		},
		{
			name: "命名空间导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_namespace_import_test.js",
				Content: []byte(`import * as moduleName from 'modules.js';
import defaultName, * as moduleName from 'modules';`),
			},
			wantErr: nil,
			wantImports: []resolver.Import{
				{BaseElement: &resolver.BaseElement{Name: "", Type: types.ElementTypeImport}, Source: "modules.js", Alias: "moduleName"},
				{BaseElement: &resolver.BaseElement{Name: "defaultName", Type: types.ElementTypeImport}, Source: "modules"},
				{BaseElement: &resolver.BaseElement{Name: "", Type: types.ElementTypeImport}, Source: "modules", Alias: "moduleName"},
			},
			description: "测试JavaScript命名空间导入",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望导入数量: %d, 实际导入数量: %d\n", len(tt.wantImports), len(res.Imports))

				// 验证导入数量精确匹配
				assert.Equal(t, len(tt.wantImports), len(res.Imports), "导入数量不匹配")

				// 创建实际导入的映射
				actualImports := make(map[string]*resolver.Import)
				for i, importItem := range res.Imports {
					// 添加基础字段断言
					assert.NotEmpty(t, importItem.GetPath(), "Import[%d] Path 不能为空", i)
					assert.NotEmpty(t, importItem.GetRange(), "Import[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(importItem.GetRange()), "Import[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, importItem.GetType(), "Import[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(importItem.Scope), "Import[%d] Scope 不能为空", i)

					key := fmt.Sprintf("%s_%s", importItem.GetName(), importItem.Source)
					actualImports[key] = importItem
				}

				// 验证每个期望的导入
				for _, wantImport := range tt.wantImports {
					key := fmt.Sprintf("%s_%s", wantImport.GetName(), wantImport.Source)
					actualImport, exists := actualImports[key]
					assert.True(t, exists, "未找到导入: %s from %s", wantImport.GetName(), wantImport.Source)

					if exists {
						assert.Equal(t, wantImport.GetName(), actualImport.GetName())
						assert.Equal(t, wantImport.Source, actualImport.Source)
						assert.Equal(t, types.ElementTypeImport, actualImport.GetType())
						if wantImport.Alias != "" {
							assert.Equal(t, wantImport.Alias, actualImport.Alias)
						}
					}
				}
			}
		})
	}
}

func TestJavaScriptResolver_ResolveFunction(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name          string
		sourceFile    *types.SourceFile
		wantErr       error
		wantFunctions []resolver.Function
		description   string
	}{
		{
			name: "JavaScript函数声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_functions.js",
				Content: []byte(`
	// 普通函数
	function add(a, b) {
		return a + b;
	}
	
	// 异步函数
	async function fetchData() {
		return await fetch('/api/data');
	}
	
	// 箭头函数
	const multiply = (a, b) => a * b;
	
	// 生成器函数
	function* generator() {
		yield 1;
		yield 2;
	}
	
	// 方法
	const obj = {
		method() {
			return 'Hello';
		}
	};
	`),
			},
			wantErr: nil,
			wantFunctions: []resolver.Function{
				{
					BaseElement: &resolver.BaseElement{Name: "add", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "add",
						Parameters: []resolver.Parameter{
							{Name: "a", Type: []string{}},
							{Name: "b", Type: []string{}},
						},
						ReturnType: nil,
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "fetchData", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name:       "fetchData",
						Parameters: []resolver.Parameter{},
						ReturnType: nil,
						Modifier:   "async",
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "multiply", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "multiply",
						Parameters: []resolver.Parameter{
							{Name: "a", Type: []string{}},
							{Name: "b", Type: []string{}},
						},
						ReturnType: nil,
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "generator", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name:       "generator",
						Parameters: []resolver.Parameter{},
						ReturnType: nil,
						Modifier:   "*",
					},
				},
			},
			description: "测试JavaScript各种函数声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望函数数量: %d\n", len(tt.wantFunctions))

				// 收集所有函数
				var actualFunctions []*resolver.Function
				for _, element := range res.Elements {
					if function, ok := element.(*resolver.Function); ok {
						actualFunctions = append(actualFunctions, function)
					}
				}

				fmt.Printf("实际函数数量: %d\n", len(actualFunctions))

				// 验证函数数量精确匹配
				assert.Equal(t, len(tt.wantFunctions), len(actualFunctions),
					"函数数量不匹配，期望 %d，实际 %d", len(tt.wantFunctions), len(actualFunctions))

				// 创建实际函数的映射
				actualFuncMap := make(map[string]*resolver.Function)
				for i, function := range actualFunctions {
					// 添加基础字段断言
					assert.NotEmpty(t, function.GetName(), "Function[%d] Name 不能为空", i)
					assert.NotEmpty(t, function.GetPath(), "Function[%d] Path 不能为空", i)
					assert.NotEmpty(t, function.GetRange(), "Function[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(function.GetRange()), "Function[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, function.GetType(), "Function[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(function.Scope), "Function[%d] Scope 不能为空", i)

					actualFuncMap[function.GetName()] = function
					fmt.Printf("函数: %s, 参数数量: %d, 修饰符: %s\n",
						function.GetName(), len(function.Parameters), function.Declaration.Modifier)
				}

				// 验证每个期望的函数
				foundCount := 0
				for _, wantFunction := range tt.wantFunctions {
					actualFunction, exists := actualFuncMap[wantFunction.GetName()]
					assert.True(t, exists, "未找到函数: %s", wantFunction.GetName())

					if exists {
						foundCount++
						// 验证函数名称和类型
						assert.Equal(t, wantFunction.GetName(), actualFunction.GetName(),
							"函数名称不匹配")
						assert.Equal(t, types.ElementTypeFunction, actualFunction.GetType(),
							"函数类型不匹配")

						// 验证参数数量
						assert.Equal(t, len(wantFunction.Parameters), len(actualFunction.Parameters),
							"函数 %s 的参数数量不匹配，期望 %d，实际 %d",
							wantFunction.GetName(), len(wantFunction.Parameters), len(actualFunction.Parameters))

						// 验证每个参数
						for i, wantParam := range wantFunction.Parameters {
							if i < len(actualFunction.Parameters) {
								actualParam := actualFunction.Parameters[i]
								assert.Equal(t, wantParam.Name, actualParam.Name,
									"函数 %s 的第 %d 个参数名称不匹配", wantFunction.GetName(), i+1)
							}
						}

						// 验证修饰符
						assert.Equal(t, wantFunction.Declaration.Modifier, actualFunction.Declaration.Modifier,
							"函数 %s 的修饰符不匹配", wantFunction.GetName())
					}
				}

				// 验证找到了所有期望的函数
				assert.Equal(t, len(tt.wantFunctions), foundCount,
					"找到的函数数量不匹配，期望 %d，实际 %d", len(tt.wantFunctions), foundCount)
			}
		})
	}
}

func TestJavaScriptResolver_ResolveVariable(t *testing.T) {
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
			name: "JavaScript变量声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_variables.js",
				Content: []byte(`
	// 变量声明
	var globalVar = 'global';
	let blockVar = 'block';
	const constant = 42;
	
	// 解构赋值
	const { name, age } = person;
	const [first, second] = array;
	
	// 对象
	const person = {
		name: 'Alice',
		age: 30,
	};
	
	// 数组
	const arr = [1, 'two', true];
	`),
			},
			wantErr: nil,
			wantVariables: []resolver.Variable{
				{BaseElement: &resolver.BaseElement{Name: "globalVar", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "blockVar", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "constant", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "name", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "age", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "first", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "second", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "person", Type: types.ElementTypeVariable}, VariableType: nil},
				{BaseElement: &resolver.BaseElement{Name: "arr", Type: types.ElementTypeVariable}, VariableType: nil},
			},
			description: "测试JavaScript各种变量声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望变量数量: %d\n", len(tt.wantVariables))

				// 收集所有变量
				var actualVariables []*resolver.Variable
				for _, element := range res.Elements {
					if variable, ok := element.(*resolver.Variable); ok {
						actualVariables = append(actualVariables, variable)
					}
				}

				fmt.Printf("实际变量数量: %d\n", len(actualVariables))

				// 验证变量数量精确匹配
				assert.Equal(t, len(tt.wantVariables), len(actualVariables),
					"变量数量不匹配，期望 %d，实际 %d", len(tt.wantVariables), len(actualVariables))

				// 创建实际变量的映射
				actualVarMap := make(map[string]*resolver.Variable)
				for i, variable := range actualVariables {
					// 添加基础字段断言
					assert.NotEmpty(t, variable.GetName(), "Variable[%d] Name 不能为空", i)
					assert.NotEmpty(t, variable.GetPath(), "Variable[%d] Path 不能为空", i)
					assert.NotEmpty(t, variable.GetRange(), "Variable[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(variable.GetRange()), "Variable[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, variable.GetType(), "Variable[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(variable.Scope), "Variable[%d] Scope 不能为空", i)

					actualVarMap[variable.GetName()] = variable
					fmt.Printf("变量: %s, Type: %s, VariableType: %v\n",
						variable.GetName(), variable.GetType(), variable.VariableType)
				}

				// 验证每个期望的变量
				foundCount := 0
				for _, wantVariable := range tt.wantVariables {
					actualVariable, exists := actualVarMap[wantVariable.GetName()]
					assert.True(t, exists, "未找到变量: %s", wantVariable.GetName())

					if exists {
						foundCount++
						// 验证变量名称和类型
						assert.Equal(t, wantVariable.GetName(), actualVariable.GetName(),
							"变量名称不匹配")
						assert.Equal(t, types.ElementTypeVariable, actualVariable.GetType(),
							"变量类型不匹配")

						// 验证变量的具体类型（JavaScript通常为空）
						assert.Equal(t, wantVariable.VariableType, actualVariable.VariableType,
							"变量 %s 的VariableType不匹配，期望 %v，实际 %v",
							wantVariable.GetName(), wantVariable.VariableType, actualVariable.VariableType)
					}
				}

				// 验证找到了所有期望的变量
				assert.Equal(t, len(tt.wantVariables), foundCount,
					"找到的变量数量不匹配，期望 %d，实际 %d", len(tt.wantVariables), foundCount)
			}
		})
	}
}

func TestJavaScriptResolver_ResolveClass(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantClasses []resolver.Class
		description string
	}{
		{
			name: "JavaScript类声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_classes.js",
				Content: []byte(`
	// 类声明
	class Animal {
		constructor(name) {
			this.name = name;
		}
		
		speak() {
			console.log(this.name + ' makes a sound');
		}
	}
	
	// 继承
	class Dog extends Animal {
		constructor(name, breed) {
			super(name);
			this.breed = breed;
		}
		
		speak() {
			super.speak();
			console.log('Woof!');
		}
		
		// 静态方法
		static create(name, breed) {
			return new Dog(name, breed);
		}
	}
	`),
			},
			wantErr: nil,
			wantClasses: []resolver.Class{
				{
					BaseElement: &resolver.BaseElement{Name: "Animal", Type: types.ElementTypeClass},
					Fields:      []*resolver.Field{},
					Methods: []*resolver.Method{
						{Declaration: resolver.Declaration{Name: "constructor", Modifier: "public"}},
						{Declaration: resolver.Declaration{Name: "speak", Modifier: "public"}},
					},
				},
				{
					BaseElement:  &resolver.BaseElement{Name: "Dog", Type: types.ElementTypeClass},
					SuperClasses: []string{"Animal"},
					Fields:       []*resolver.Field{},
					Methods: []*resolver.Method{
						{Declaration: resolver.Declaration{Name: "constructor", Modifier: "public"}},
						{Declaration: resolver.Declaration{Name: "speak", Modifier: "public"}},
						{Declaration: resolver.Declaration{Name: "create", Modifier: "static public"}},
					},
				},
			},
			description: "测试JavaScript各种类声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望类数量: %d\n", len(tt.wantClasses))

				// 收集所有类
				var actualClasses []*resolver.Class
				for _, element := range res.Elements {
					if class, ok := element.(*resolver.Class); ok {
						actualClasses = append(actualClasses, class)
					}
				}

				fmt.Printf("实际类数量: %d\n", len(actualClasses))

				// 验证类数量精确匹配
				assert.Equal(t, len(tt.wantClasses), len(actualClasses),
					"类数量不匹配，期望 %d，实际 %d", len(tt.wantClasses), len(actualClasses))

				// 创建实际类的映射
				actualClassMap := make(map[string]*resolver.Class)
				for i, class := range actualClasses {
					// 添加基础字段断言
					assert.NotEmpty(t, class.GetName(), "Class[%d] Name 不能为空", i)
					assert.NotEmpty(t, class.GetPath(), "Class[%d] Path 不能为空", i)
					assert.NotEmpty(t, class.GetRange(), "Class[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(class.GetRange()), "Class[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, class.GetType(), "Class[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(class.Scope), "Class[%d] Scope 不能为空", i)

					actualClassMap[class.GetName()] = class
					fmt.Printf("类: %s, 字段数量: %d, 方法数量: %d, 继承: %v\n",
						class.GetName(), len(class.Fields), len(class.Methods), class.SuperClasses)
				}

				// 验证每个期望的类
				foundCount := 0
				for _, wantClass := range tt.wantClasses {
					actualClass, exists := actualClassMap[wantClass.GetName()]
					assert.True(t, exists, "未找到类: %s", wantClass.GetName())

					if exists {
						foundCount++
						// 验证类名称和类型
						assert.Equal(t, wantClass.GetName(), actualClass.GetName(),
							"类名称不匹配")
						assert.Equal(t, types.ElementTypeClass, actualClass.GetType(),
							"类类型不匹配")

						// 验证继承关系
						assert.Equal(t, len(wantClass.SuperClasses), len(actualClass.SuperClasses),
							"类 %s 的继承数量不匹配，期望 %d，实际 %d",
							wantClass.GetName(), len(wantClass.SuperClasses), len(actualClass.SuperClasses))
						for i, expectedSuper := range wantClass.SuperClasses {
							if i < len(actualClass.SuperClasses) {
								assert.Equal(t, expectedSuper, actualClass.SuperClasses[i],
									"类 %s 的第 %d 个继承类不匹配", wantClass.GetName(), i+1)
							}
						}

						// 验证方法数量和详情
						assert.Equal(t, len(wantClass.Methods), len(actualClass.Methods),
							"类 %s 的方法数量不匹配，期望 %d，实际 %d",
							wantClass.GetName(), len(wantClass.Methods), len(actualClass.Methods))

						// 创建实际方法的映射
						actualMethodMap := make(map[string]*resolver.Method)
						for _, method := range actualClass.Methods {
							actualMethodMap[method.Declaration.Name] = method
						}

						// 验证每个期望的方法
						for _, wantMethod := range wantClass.Methods {
							actualMethod, methodExists := actualMethodMap[wantMethod.Declaration.Name]
							assert.True(t, methodExists, "类 %s 中未找到方法: %s",
								wantClass.GetName(), wantMethod.Declaration.Name)

							if methodExists {
								assert.Equal(t, wantMethod.Declaration.Name, actualMethod.Declaration.Name,
									"类 %s 的方法名称不匹配", wantClass.GetName())
								assert.Equal(t, wantMethod.Declaration.Modifier, actualMethod.Declaration.Modifier,
									"类 %s 的方法 %s 修饰符不匹配", wantClass.GetName(), wantMethod.Declaration.Name)
							}
						}
					}
				}

				// 验证找到了所有期望的类
				assert.Equal(t, len(tt.wantClasses), foundCount,
					"找到的类数量不匹配，期望 %d，实际 %d", len(tt.wantClasses), foundCount)
			}
		})
	}
}

func TestJavaScriptResolver_ResolveMethodCall(t *testing.T) {
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
			name: "JavaScript方法调用",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_method_calls.js",
				Content: []byte(`
	// 函数调用
	console.log('Hello World');
	alert('Warning');
	
	// 对象方法调用
	user.getName();
	person.setAge(25);
	
	// 链式调用
	obj.getData().then().catch();
	
	// 带多个参数的调用
	Math.max(1, 2, 3);
	
	// 数组方法调用
	arr.push(item);
	list.filter(fn);
	`),
			},
			wantErr: nil,
			wantCalls: []resolver.Call{
				{BaseElement: &resolver.BaseElement{Name: "log", Type: types.ElementTypeMethodCall}, Owner: "console"},
				{BaseElement: &resolver.BaseElement{Name: "alert", Type: types.ElementTypeFunctionCall}},
				{BaseElement: &resolver.BaseElement{Name: "getName", Type: types.ElementTypeMethodCall}, Owner: "user"},
				{BaseElement: &resolver.BaseElement{Name: "setAge", Type: types.ElementTypeMethodCall}, Owner: "person"},
				{BaseElement: &resolver.BaseElement{Name: "getData", Type: types.ElementTypeMethodCall}, Owner: "obj"},
				{BaseElement: &resolver.BaseElement{Name: "then", Type: types.ElementTypeMethodCall}, Owner: "obj.getData()"},
				{BaseElement: &resolver.BaseElement{Name: "catch", Type: types.ElementTypeMethodCall}, Owner: "obj.getData().then()"},
				{BaseElement: &resolver.BaseElement{Name: "max", Type: types.ElementTypeMethodCall}, Owner: "Math"},
				{BaseElement: &resolver.BaseElement{Name: "push", Type: types.ElementTypeMethodCall}, Owner: "arr"},
				{BaseElement: &resolver.BaseElement{Name: "filter", Type: types.ElementTypeMethodCall}, Owner: "list"},
			},
			description: "测试JavaScript各种方法调用的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望调用数量: %d\n", len(tt.wantCalls))

				// 收集所有方法调用
				var actualCalls []*resolver.Call
				for _, element := range res.Elements {
					if element.GetType() == types.ElementTypeMethodCall || element.GetType() == types.ElementTypeFunctionCall {
						if call, ok := element.(*resolver.Call); ok {
							actualCalls = append(actualCalls, call)
						}
					}
				}

				fmt.Printf("实际调用数量: %d\n", len(actualCalls))

				// 打印所有找到的调用
				for i, call := range actualCalls {
					fmt.Printf("[%d] 调用: %s, 类型: %s, 所有者: %s\n",
						i+1, call.GetName(), call.GetType(), call.Owner)
				}

				// 验证调用数量精确匹配
				assert.Equal(t, len(tt.wantCalls), len(actualCalls),
					"调用数量不匹配，期望 %d，实际 %d", len(tt.wantCalls), len(actualCalls))

				// 创建实际调用的映射
				actualCallMap := make(map[string]*resolver.Call)
				for i, call := range actualCalls {
					// 添加基础字段断言
					assert.NotEmpty(t, call.GetName(), "Call[%d] Name 不能为空", i)
					assert.NotEmpty(t, call.GetPath(), "Call[%d] Path 不能为空", i)
					assert.NotEmpty(t, call.GetRange(), "Call[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(call.GetRange()), "Call[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, call.GetType(), "Call[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(call.Scope), "Call[%d] Scope 不能为空", i)

					key := fmt.Sprintf("%s_%s_%s", call.GetName(), call.Owner, call.GetType())
					actualCallMap[key] = call
				}

				// 验证每个期望的调用
				foundCount := 0
				for _, wantCall := range tt.wantCalls {
					key := fmt.Sprintf("%s_%s_%s", wantCall.GetName(), wantCall.Owner, wantCall.GetType())
					actualCall, exists := actualCallMap[key]
					assert.True(t, exists, "未找到调用: %s (所有者: %s, 类型: %s)",
						wantCall.GetName(), wantCall.Owner, wantCall.GetType())

					if exists {
						foundCount++
						// 验证调用名称和类型
						assert.Equal(t, wantCall.GetName(), actualCall.GetName(),
							"调用名称不匹配")
						assert.Equal(t, wantCall.GetType(), actualCall.GetType(),
							"调用类型不匹配")
						assert.Equal(t, wantCall.Owner, actualCall.Owner,
							"调用所有者不匹配")
					}
				}

				// 验证找到了所有期望的调用
				assert.Equal(t, len(tt.wantCalls), foundCount,
					"找到的调用数量不匹配，期望 %d，实际 %d", len(tt.wantCalls), foundCount)
			}
		})
	}
}

func TestJavaScriptResolver_ResolveObjectMethod(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	jsCode := `
	// 对象字面量中的方法定义
	const obj = {
		name: 'Test Object',
		
		// 简写方法
		sayHello() {
			return 'Hello, ' + this.name;
		},
		
		// 异步方法
		async fetchData() {
			return await fetch('/api/data');
		},
		
		// 生成器方法
		*generateIds() {
			let id = 1;
			while (true) {
				yield id++;
			}
		},
		
		// getter方法
		get fullName() {
			return this.name + ' (Object)';
		},
		
		// setter方法
		set fullName(value) {
			this.name = value;
		}
	};

	// Vue组件样式的对象
	const component = {
		data() {
			return {
				message: 'Hello Vue'
			}
		},
		
		methods: {
			greet() {
				alert(this.message);
			}
		},
		
		computed: {
			reversedMessage() {
				return this.message.split('').reverse().join('');
			}
		},
		
		created() {
			console.log('Component created');
		},
		
		mounted() {
			console.log('Component mounted');
		}
	};
	`

	sourceFile := &types.SourceFile{
		Path:    "testdata/js_object_methods.js",
		Content: []byte(jsCode),
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 统计对象方法
	methodCount := 0
	fmt.Println("\n对象方法详情:")
	for i, elem := range res.Elements {
		if elem.GetType() == types.ElementTypeMethod {
			methodCount++
			method, ok := elem.(*resolver.Method)
			assert.True(t, ok)

			// 添加基础字段断言
			assert.NotEmpty(t, method.GetName(), "Method[%d] Name 不能为空", i)
			assert.NotEmpty(t, method.GetPath(), "Method[%d] Path 不能为空", i)
			assert.NotEmpty(t, method.GetRange(), "Method[%d] Range 不能为空", i)
			assert.Equal(t, 4, len(method.GetRange()), "Method[%d] Range 应该包含4个元素", i)
			assert.NotEqual(t, types.ElementTypeUndefined, method.GetType(), "Method[%d] Type 不能为 undefined", i)
			assert.NotEmpty(t, string(method.Scope), "Method[%d] Scope 不能为空", i)

			fmt.Printf("[%d] 对象方法: %s\n", methodCount, method.GetName())
			if method.Owner != "" {
				fmt.Printf("  所有者: %s\n", method.Owner)
			}
			if method.Declaration.Modifier != "" {
				fmt.Printf("  修饰符: %s\n", method.Declaration.Modifier)
			}
			fmt.Printf("  参数数量: %d\n", len(method.Parameters))
		}
	}
	fmt.Printf("\n对象方法总数: %d\n", methodCount)

	// 确认是否找到了方法
	if methodCount == 0 {
		fmt.Println("注意：没有解析到对象方法，请检查JavaScript解析器是否正确实现了对象方法解析")
	}
}
