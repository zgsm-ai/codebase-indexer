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
			wantErr:     nil,
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
			wantErr:     nil,
			description: "测试JavaScript命名导入和别名",
		},
		{
			name: "命名空间导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/js_namespace_import_test.js",
				Content: []byte(`import * as moduleName from 'modules.js';
import defaultName, * as moduleName from 'modules';`),
			},
			wantErr:     nil,
			description: "测试JavaScript命名空间导入",
		},
		{
			name: "副作用导入",
			sourceFile: &types.SourceFile{
				Path:    "testdata/js_side_effect_import_test.js",
				Content: []byte(`import 'modules';`),
			},
			wantErr:     nil,
			description: "测试JavaScript副作用导入",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 验证导入解析
				fmt.Printf("导入数量: %d\n", len(res.Imports))
				for _, importItem := range res.Imports {
					fmt.Printf("Import: %s, Source: %s\n", importItem.GetName(), importItem.Source)
					assert.NotEmpty(t, importItem.Source)
					assert.Equal(t, types.ElementTypeImport, importItem.GetType())
				}
			}
		})
	}
}

func TestJavaScriptResolver_ResolveFunction(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	jsCode := `
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
	`

	sourceFile := &types.SourceFile{
		Path:    "testdata/js_functions.js",
		Content: []byte(jsCode),
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	funcCount := 0
	fmt.Printf("解析到的元素数量: %d\n", len(res.Elements))
	for _, elem := range res.Elements {
		if elem.GetType() == types.ElementTypeFunction {
			funcCount++
			fn, ok := elem.(*resolver.Function)
			assert.True(t, ok)
			fmt.Printf("函数: %s\n", fn.GetName())
			assert.NotEmpty(t, fn.GetName())
		}
	}
	fmt.Printf("函数数量: %d\n", funcCount)
}

func TestJavaScriptResolver_ResolveVariable(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	jsCode := `
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
	`

	sourceFile := &types.SourceFile{
		Path:    "testdata/js_variables.js",
		Content: []byte(jsCode),
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	varCount := 0
	for _, elem := range res.Elements {
		if elem.GetType() == types.ElementTypeVariable {
			varCount++
			variable, ok := elem.(*resolver.Variable)
			assert.True(t, ok)
			fmt.Printf("变量: %s\n", variable.GetName())
			assert.NotEmpty(t, variable.GetName())
		}
	}
	fmt.Printf("变量数量: %d\n", varCount)
}

func TestJavaScriptResolver_ResolveClass(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	jsCode := `
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
	`

	sourceFile := &types.SourceFile{
		Path:    "testdata/js_classes.js",
		Content: []byte(jsCode),
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	classCount := 0
	for _, elem := range res.Elements {
		if elem.GetType() == types.ElementTypeClass {
			classCount++
			class, ok := elem.(*resolver.Class)
			assert.True(t, ok)
			fmt.Printf("类: %s\n", class.GetName())
			assert.NotEmpty(t, class.GetName())

			// 检查方法
			for _, method := range class.Methods {
				fmt.Printf("  方法: %s\n", method.Declaration.Name)
			}

			// 检查继承关系
			if class.GetName() == "Dog" {
				assert.Contains(t, class.SuperClasses, "Animal")
			}
		}
	}
	fmt.Printf("类数量: %d\n", classCount)
}

func TestJavaScriptResolver_ResolveMethodCall(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	// 使用已有的测试文件
	sourceFile := &types.SourceFile{
		Path:    "testdata/test1.js",
		Content: readFile("testdata/test1.js"),
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	callCount := 0
	fmt.Println("\n方法调用详情:")
	for _, elem := range res.Elements {
		if elem.GetType() == types.ElementTypeMethodCall || elem.GetType() == types.ElementTypeFunctionCall {
			callCount++
			call, ok := elem.(*resolver.Call)
			if ok {
				fmt.Printf("[%d] 方法调用: %s\n", callCount, call.GetName())
				if call.Owner != "" {
					fmt.Printf("  所有者: %s\n", call.Owner)
				}
				if len(call.Parameters) > 0 {
					fmt.Printf("  参数数量: %d\n", len(call.Parameters))
					for i, param := range call.Parameters {
						fmt.Printf("    参数%d: %s\n", i+1, param.Name)
					}
				}
			}
		}
	}
	fmt.Printf("\n方法调用总数: %d\n", callCount)

	// 确保找到了方法调用
	assert.Greater(t, callCount, 0, "应该至少找到一个方法调用")
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
	for _, elem := range res.Elements {
		if elem.GetType() == types.ElementTypeMethod {
			methodCount++
			method, ok := elem.(*resolver.Method)
			assert.True(t, ok)
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

func TestJavaScriptResolver_ResolveComprehensive(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	// 使用已有的测试文件
	sourceFile := &types.SourceFile{
		Path:    "testdata/test1.js",
		Content: readFile("testdata/test1.js"),
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 统计各种类型的元素
	var countByType = make(map[types.ElementType]int)
	for _, elem := range res.Elements {
		countByType[elem.GetType()]++
	}

	// 打印统计结果
	fmt.Println("JavaScript文件解析结果统计:")
	fmt.Printf("导入语句: %d\n", len(res.Imports))
	fmt.Printf("函数: %d\n", countByType[types.ElementTypeFunction])
	fmt.Printf("变量: %d\n", countByType[types.ElementTypeVariable])
	fmt.Printf("类: %d\n", countByType[types.ElementTypeClass])
	fmt.Printf("方法: %d\n", countByType[types.ElementTypeMethod])
	fmt.Printf("方法调用: %d\n", countByType[types.ElementTypeMethodCall])

	// 检查导入语句
	assert.NotEmpty(t, res.Imports)
}
