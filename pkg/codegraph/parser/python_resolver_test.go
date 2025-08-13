package parser

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPythonResolver(t *testing.T) {

}

func TestPythonResolver_ResolveImport(t *testing.T) {

	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantImports []struct {
			name   string
			source string
			alias  string
		}
		wantErr     error
		description string
	}{
		{
			name: "正常导入",
			sourceFile: &types.SourceFile{
				Path:    "testdata/python/testImport.py",
				Content: readFile("testdata/python/testImport.py"),
			},
			wantImports: []struct {
				name   string
				source string
				alias  string
			}{
				{"module", "", ""},
				{"module1", "", ""},
				{"module2", "", ""},
				{"package.module", "", ""},
				{"package.subpackage.module", "", ""},
				{"module", "", "alias"},
				{"module1", "", "alias1"},
				{"module2", "", "alias2"},
				{"package.module", "", "alias"},
				{"name", "module", ""},
				{"name1", "module", ""},
				{"name2", "module", ""},
				{"name", "package.module", ""},
				{"name", "package.subpackage.module", ""},
				{"name", "module", "alias"},
				{"name1", "module", "alias1"},
				{"name2", "module", "alias2"},
				{"name3", "module", "alias3"},
				{"name4", "module", ""},
				{"name5", "module", ""},
				{"*", "module", ""},
				{"defaultdict", "collections", ""},
				{"OrderedDict", "collections", ""},
				{"Counter", "collections", ""},
				{"name", "..module11", ""},
				{"module", "..package12", ""},
				{"name", "..package.module13", ""},
				{"name", "..package.module13", "name1"},
			},
			wantErr:     nil,
			description: "测试正常的Python导入解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 验证导入解析
				// 测试逻辑有问题
				for _, importItem := range res.Imports {
					found := false
					for _, wantImport := range tt.wantImports {
						if wantImport.name == importItem.GetName() && wantImport.source == importItem.Source && wantImport.alias == importItem.Alias {
							found = true
							break
						}
					}
					assert.True(t, found, "from "+importItem.Source+" import "+importItem.GetName()+" as "+importItem.Alias+"导入名称不一致")
				}
			}
		})
	}

}

func TestPythonResolver_ResolveFunction(t *testing.T) {
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
			name: "testFunc.py 全部函数声明解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/python/testFunc.py",
				Content: readFile("testdata/python/testFunc.py"),
			},
			wantErr: nil,
			wantFuncs: []resolver.Declaration{
				// 基本函数
				{Name: "hello", ReturnType: nil, Parameters: []resolver.Parameter{}},
				{Name: "greet", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "name", Type: nil},
				}},
				{Name: "add", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "a", Type: nil},
					{Name: "b", Type: nil},
				}},
				{Name: "greet", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "name", Type: nil},
				}},
				{Name: "connect", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "host", Type: nil},
					{Name: "port", Type: nil},
					{Name: "timeout", Type: nil},
				}},
				{Name: "greet1", ReturnType: []string{"str"}, Parameters: []resolver.Parameter{
					{Name: "name", Type: []string{"str"}},
				}},
				{Name: "process", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "items", Type: []string{"list", "dict", "str", "int"}},
					{Name: "items5", Type: []string{"dict", "str", "int"}},
					{Name: "items6", Type: []string{"str"}},
				}},
				{Name: "log", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "...args", Type: nil},
				}},
				{Name: "config", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "...kwargs", Type: nil},
				}},
				{Name: "func", ReturnType: nil, Parameters: []resolver.Parameter{
					{Name: "a", Type: nil},
					{Name: "...args", Type: nil},
					{Name: "...kwargs", Type: nil},
				}},
				{Name: "great_test", ReturnType: []string{"int"}, Parameters: []resolver.Parameter{
					{Name: "a", Type: nil},
					{Name: "b", Type: []string{"str"}},
					{Name: "c", Type: nil},
					{Name: "d", Type: []string{"int"}},
				}},
				{Name: "add_status", ReturnType: []string{"list", "dict", "str", "int"}, Parameters: []resolver.Parameter{
					{Name: "items", Type: []string{"list", "dict", "str", "int"}},
				}},
				{Name: "f_test", ReturnType: []string{"Foo", "Foo1"}, Parameters: []resolver.Parameter{
					{Name: "a", Type: nil},
				}},
			},
			description: "测试 testFunc.py 中所有函数声明的解析",
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
						funcMap[fn.Declaration.Name] = fn.Declaration
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
						assert.ElementsMatch(t, wantFunc.Parameters, actualFunc.Parameters,
							"函数 %s 的参数不匹配，期望 %v，实际 %v",
							wantFunc.Name, wantFunc.Parameters, actualFunc.Parameters)
					}
				}
			}
		})
	}
}

func TestPythonResolver_ResolveClass(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantClasses []struct {
			Name         string
			SuperClasses []string
		}
		description string
	}{
		{
			name: "testClass.py 全部类声明解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/python/testClass.py",
				Content: readFile("testdata/python/testClass.py"),
			},
			wantErr: nil,
			wantClasses: []struct {
				Name         string
				SuperClasses []string
			}{
				// 这里只列举部分，实际可根据 testClass.py 全部补全
				{"Person", nil},
				{"Animal", nil},
				{"Car", nil},
				{"Dog", []string{"Animal"}},
				{"Cat", []string{"Animal"}},
				{"Manager", []string{"Employee"}},
				{"Rectangle", []string{"Shape"}},
				{"FlyingCar", []string{"Car", "Aircraft"}},
				{"StudentTeacher", []string{"Student", "Teacher"}},
				{"WalkerSwimmer", []string{"Walker", "Swimmer"}},
				{"Database", []string{"SingletonMeta"}},
				{"Model2", []string{"BaseModel", "ModelMeta"}},
				{"APIRouter", []string{"BaseRouter", "RouterMeta"}},
				{"Config", []string{"ConfigMeta"}},
				{"User2", []string{"UserMeta"}},
				{"Product2", []string{"ModelMeta"}},
				{"Order2", []string{"BaseModel", "OrderMeta"}},
				{"Payment", []string{"BaseModel", "PaymentMeta"}},
				{"Container", []string{"Generic", "T"}},
				{"Repository", []string{"Generic", "T"}},
				{"Map", []string{"Generic", "K", "V"}},
				{"UserContainer", []string{"Container", "User"}},
				{"ProductRepository", []string{"Repository", "Product"}},
				{"UserList", []string{"List", "User"}},
				{"ProductList", []string{"List", "Product"}},
				{"UserDict", []string{"Dict", "User", "str"}},
				{"ConfigDict", []string{"Dict", "str", "int", "bool", "Union", "str"}},
				{"FlexibleContainer", []string{"Union", "BaseClass", "PaymentMeta", "List", "int", "str", "List"}},
				{"NumberOrString", []string{"Union", "int", "float", "str"}},
				{"ComplexClass", []string{"List", "User", "Dict", "str", "Product", "MetaClass"}},
				{"DataProcessor", []string{"List", "Dict", "Union", "int", "str", "str", "Optional", "Logger", "Cache"}},
				{"AdvancedManager", []string{"List", "Dict", "str", "User", "Permission", "ManagerMeta"}},
				{"User", []string{"Dict", "str", "User"}},      // metaclass=Dict[str, User]，不是继承
				{"Product", []string{"List", "Product"}},       // metaclass=List[Product]
				{"Order", []string{"Union", "TypeA", "TypeB"}}, // metaclass=Union[TypeA, TypeB]
				{"Model", []string{"Model"}},                   // metaclass=django.db.models.Model
				{"Atest", []string{"Foo", "Foo1"}},             // metaclass=mylib.utils.Foo[mylib.utils.Foo1]
				{"Btest", []string{"Foo", "Foo1", "User"}},
				{"Ctest", []string{"Foo", "User"}},
			},
			description: "测试 testClass.py 中所有类声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 1. 收集所有类（用名字做唯一键）
				classMap := make(map[string]*resolver.Class)
				for _, element := range res.Elements {
					if cls, ok := element.(*resolver.Class); ok {
						classMap[cls.BaseElement.Name] = cls
					}
				}
				// 2. 逐个比较每个期望的类
				for _, wantClass := range tt.wantClasses {
					actualClass, exists := classMap[wantClass.Name]
					assert.True(t, exists, "未找到类: %s", wantClass.Name)
					if exists {
						assert.ElementsMatch(t, wantClass.SuperClasses, actualClass.SuperClasses,
							"类 %s 的继承父类不匹配，期望 %v，实际 %v",
							wantClass.Name, wantClass.SuperClasses, actualClass.SuperClasses)
					}
				}
			}
		})
	}
}

func TestPythonResolver_ResolveMethod(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantMethods []resolver.Method
		description string
	}{
		{
			name: "testMethod.py 全部方法声明解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/python/testMethod.py",
				Content: readFile("testdata/python/testMethod.py"),
			},
			wantErr: nil,
			wantMethods: []resolver.Method{
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "simple_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_params",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "a", Type: nil},
							{Name: "b", Type: nil},
							{Name: "c", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_defaults",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "a", Type: nil},
							{Name: "b", Type: nil},
							{Name: "c", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "typed_method",
						ReturnType: []string{"bool"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "x", Type: []string{"int"}},
							{Name: "y", Type: []string{"str"}},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "complex_typed_method",
						ReturnType: []string{"tuple", "int", "str"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "data", Type: []string{"list", "dict", "str", "int"}},
							{Name: "callback", Type: []string{"callable"}},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "variadic_args",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "...args", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "variadic_kwargs",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "...kwargs", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "mixed_params",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "a", Type: nil},
							{Name: "b", Type: nil},
							{Name: "...args", Type: nil},
							{Name: "...kwargs", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "keyword_only",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "x", Type: nil},
							{Name: "y", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "positional_only",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "a", Type: nil},
							{Name: "b", Type: nil},
							{Name: "c", Type: nil},
							{Name: "d", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "documented_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "x", Type: nil},
							{Name: "y", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "generator_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "async_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "complex_body_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "x", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_exception",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_context",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_comprehension",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "items", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_ternary",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "x", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_loops",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "items", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_assert",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "value", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_complex_return",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "recursive_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "n", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__special_method__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "_private_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_underscore_end_",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_complex_control_flow",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "items", Type: nil},
						},
					},
					Owner: "TestClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "static_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{},
					},
					Owner: "StaticMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "static_method_with_params",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "a", Type: nil},
							{Name: "b", Type: nil},
						},
					},
					Owner: "StaticMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "typed_static_method",
						ReturnType: []string{"bool"},
						Parameters: []resolver.Parameter{
							{Name: "x", Type: []string{"int"}},
							{Name: "y", Type: []string{"str"}},
						},
					},
					Owner: "StaticMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "class_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "cls", Type: nil},
						},
					},
					Owner: "ClassMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "class_method_with_params",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "cls", Type: nil},
							{Name: "name", Type: nil},
						},
					},
					Owner: "ClassMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "computed_property1",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "PropertyClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "computed_property2",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "value", Type: nil},
						},
					},
					Owner: "PropertyClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "computed_property3",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "PropertyClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "decorated_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "DecoratedMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "decorated_method_with_args",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "DecoratedMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "cached_property",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "DecoratedMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "abstract_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "AbstractClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "abstract_method_with_params",
						ReturnType: []string{"str"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "x", Type: []string{"int"}},
						},
					},
					Owner: "AbstractClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "concrete_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "AbstractClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "parent_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ParentClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "overridden_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ParentClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "overridden_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ChildClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "extended_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ChildClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "generic_method",
						ReturnType: []string{"T"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "item", Type: []string{"T"}},
						},
					},
					Owner: "TypedClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_typing_annotations",
						ReturnType: []string{"List", "str"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "items", Type: []string{"List", "int"}},
							{Name: "mapping", Type: []string{"Dict", "str", "int"}},
							{Name: "optional_value", Type: []string{"Optional", "str"}},
							{Name: "union_value", Type: []string{"Union", "int", "str"}},
							{Name: "callback", Type: []string{"Callable", "int", "str"}},
						},
					},
					Owner: "TypedClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__init__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "value", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__str__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__repr__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__len__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__getitem__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "key", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__setitem__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "key", Type: nil},
							{Name: "value", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__call__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "...args", Type: nil},
							{Name: "...kwargs", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__enter__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__exit__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "exc_type", Type: nil},
							{Name: "exc_val", Type: nil},
							{Name: "exc_tb", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__iter__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__next__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__add__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "other", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__eq__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "other", Type: nil},
						},
					},
					Owner: "MagicMethodClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "inner_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "InnerClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "inner_static_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{},
					},
					Owner: "InnerClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "outer_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "OuterClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_unpacking",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "UnpackingClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_walrus",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "items", Type: nil},
						},
					},
					Owner: "WalrusClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_a",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "Base1",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_b",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "Base2",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "combined_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MultiInheritanceClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "access_class_var",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ClassVariableClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "modify_class_var",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "new_value", Type: nil},
						},
					},
					Owner: "ClassVariableClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__init__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "InstanceVariableClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "access_instance_var",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "InstanceVariableClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "modify_instance_var",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "new_value", Type: nil},
						},
					},
					Owner: "InstanceVariableClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_global",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ScopeClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "outer_with_nonlocal",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ScopeClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_complex_defaults",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "data", Type: nil},
							{Name: "callback", Type: nil},
							{Name: "config", Type: nil},
						},
					},
					Owner: "DefaultValuesClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_match",
						ReturnType: []string{"str"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "value", Type: nil},
						},
					},
					Owner: "MatchClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "greet",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "DataClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "is_adult",
						ReturnType: []string{"bool"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "DataClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "expensive_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "n", Type: nil},
						},
					},
					Owner: "CacheClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "multi_decorated_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{},
					},
					Owner: "MultiDecoratorClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "complex_control_flow",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "items", Type: nil},
						},
					},
					Owner: "ControlFlowClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "async_generator_method",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "AsyncGeneratorClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__init__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ContextManagerClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__enter__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ContextManagerClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__exit__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "exc_type", Type: nil},
							{Name: "exc_val", Type: nil},
							{Name: "exc_tb", Type: nil},
						},
					},
					Owner: "ContextManagerClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_with_context_manager",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "ContextManagerClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__init__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "DescriptorClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__get__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "obj", Type: nil},
							{Name: "objtype", Type: nil},
						},
					},
					Owner: "CustomDescriptor",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__set__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "obj", Type: nil},
							{Name: "value", Type: nil},
						},
					},
					Owner: "CustomDescriptor",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__delete__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "obj", Type: nil},
						},
					},
					Owner: "CustomDescriptor",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "use_descriptor",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "DescriptorClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__new__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "cls", Type: nil},
							{Name: "name", Type: nil},
							{Name: "bases", Type: nil},
							{Name: "attrs", Type: nil},
						},
					},
					Owner: "MetaClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "method_using_meta_attr",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
						},
					},
					Owner: "MetaClassUser",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "__init__",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "...args", Type: nil},
							{Name: "...kwargs", Type: nil},
						},
					},
					Owner: "ComplexClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "class_property",
						ReturnType: nil,
						Parameters: []resolver.Parameter{
							{Name: "cls", Type: nil},
						},
					},
					Owner: "ComplexClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "cached_static_method",
						ReturnType: []string{"int"},
						Parameters: []resolver.Parameter{
							{Name: "x", Type: []string{"int"}},
						},
					},
					Owner: "ComplexClass",
				},
				{
					BaseElement: nil,
					Declaration: &resolver.Declaration{
						Name:       "complex_method",
						ReturnType: []string{"Union", "str", "int"},
						Parameters: []resolver.Parameter{
							{Name: "self", Type: nil},
							{Name: "required_param", Type: []string{"str"}},
							{Name: "optional_param", Type: []string{"int"}},
							{Name: "...args", Type: []string{"int"}},
							{Name: "keyword_only", Type: []string{"bool"}},
							{Name: "...kwargs", Type: []string{"dict"}},
							{Name: "optional_param1", Type: []string{"Foo", "Foo1", "Foo"}},
						},
					},
					Owner: "ComplexClass",
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
				// 1. 收集所有函数（不考虑重载，直接用名字做唯一键）
				methodMap := make(map[string]*resolver.Method)
				for _, element := range res.Elements {
					if fn, ok := element.(*resolver.Method); ok {
						// 用方法名和Owner联合做key，避免重复
						key := fn.Declaration.Name + "@" + fn.Owner
						// 如果已经存在同名同Owner的方法，则跳过，保证唯一
						if _, exists := methodMap[key]; !exists {
							methodMap[key] = fn
						}
					}
				}
				// 2. 逐个比较每个期望的函数
				for _, wantMethod := range tt.wantMethods {
					key := wantMethod.Declaration.Name + "@" + wantMethod.Owner
					actualMethod, exists := methodMap[key]
					assert.True(t, exists, "未找到方法: %s::%s", wantMethod.Owner, wantMethod.Declaration.Name)
					if exists {
						assert.Equal(t, wantMethod.Declaration.ReturnType, actualMethod.Declaration.ReturnType,
							"方法 %s 的返回值类型不匹配，期望 %v，实际 %v",
							wantMethod.Declaration.Name, wantMethod.Declaration.ReturnType, actualMethod.Declaration.ReturnType)
						// 递归比较参数数组及其内部元素（包括Type字段的slice）
						assert.Equal(t, len(wantMethod.Declaration.Parameters), len(actualMethod.Declaration.Parameters),
							"方法 %s 的参数数量不匹配，期望 %d，实际 %d",
							wantMethod.Declaration.Name, len(wantMethod.Declaration.Parameters), len(actualMethod.Declaration.Parameters))
						// 参数顺序保证不了，只能无序比对
						wantParams := wantMethod.Declaration.Parameters
						actualParams := actualMethod.Declaration.Parameters

						assert.Equal(t, len(wantParams), len(actualParams),
							"方法 %s 的参数数量不匹配，期望 %d，实际 %d",
							wantMethod.Declaration.Name, len(wantParams), len(actualParams))

						// 构建map，key为参数名，值为Type
						wantParamMap := make(map[string][]string)
						for _, p := range wantParams {
							wantParamMap[p.Name] = p.Type
						}
						actualParamMap := make(map[string][]string)
						for _, p := range actualParams {
							actualParamMap[p.Name] = p.Type
						}

						for name, wantType := range wantParamMap {
							actualType, ok := actualParamMap[name]
							assert.True(t, ok, "方法 %s 缺少参数: %s", wantMethod.Declaration.Name, name)
							if ok {
								assert.ElementsMatch(t, wantType, actualType,
									"方法 %s 的参数 %s 的类型不匹配，期望 %v，实际 %v",
									wantMethod.Declaration.Name, name, wantType, actualType)
							}
						}
						assert.Equal(t, wantMethod.Owner, actualMethod.Owner,
							"方法 %s 的Owner不匹配，期望 %v，实际 %v",
							wantMethod.Declaration.Name, wantMethod.Owner, actualMethod.Owner)
					}
				}
			}
		})
	}
}

func TestPythonResolver_ResolveVariable(t *testing.T) {
	// 参考 TestPythonResolver_ResolveMethod 实现变量解析的测试
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantVars    []resolver.Variable
		description string
	}{
		{
			name: "testVar.py 全部变量声明解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/python/testVar.py",
				Content: readFile("testdata/python/testVar.py"),
			},
			wantErr: nil,
			wantVars: []resolver.Variable{
				{
					BaseElement: &resolver.BaseElement{
						Name: "user",
					},
					VariableType: []string{"User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "config",
					},
					VariableType: []string{"Config"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name",
					},
					VariableType: []string{"str"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "optional_user",
					},
					VariableType: []string{"Optional", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "items",
					},
					VariableType: []string{"List", "Item"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "mapping",
					},
					VariableType: []string{"Dict", "str", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "nested",
					},
					VariableType: []string{"List", "Dict", "str", "Response"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "complex_nested",
					},
					VariableType: []string{"Dict", "Category", "List", "Item"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name1",
					},
					VariableType: []string{"Foo"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name2",
					},
					VariableType: []string{"Foo"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name3",
					},
					VariableType: []string{"Foo", "Foo1"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name4",
					},
					VariableType: []string{"Foo", "Foo1"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name5",
					},
					VariableType: []string{"Container", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name6",
					},
					VariableType: []string{"Container", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name7",
					},
					VariableType: []string{"Container", "List", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name8",
					},
					VariableType: []string{"Dict", "str", "Response", "Item"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name9",
					},
					VariableType: []string{"Foo", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "name10",
					},
					VariableType: []string{"Foo", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "container",
					},
					VariableType: []string{"Container", "str"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "container",
					},
					VariableType: []string{"Container", "int"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "nested_container",
					},
					VariableType: []string{"Container", "List", "str"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "complex_container",
					},
					VariableType: []string{"Container", "Dict", "str", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "pair",
					},
					VariableType: []string{"Pair", "User", "Product"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "complex_var",
					},
					VariableType: []string{"Processor", "UserRequest", "UserResponse", "User"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "mixed_var",
					},
					VariableType: []string{"Optional", "Dict", "str", "List", "Item", "Settings"},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "multi_qualified",
					},
					VariableType: []string{"Union", "User", "Admin", "Guest"},
				},
			},
			description: "测试 testVar.py 中所有变量声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 1. 收集所有变量（用名字做唯一键，允许同名变量多次声明）
				varMap := make(map[string][]*resolver.Variable)
				for _, element := range res.Elements {
					if v, ok := element.(*resolver.Variable); ok {
						varMap[v.BaseElement.Name] = append(varMap[v.BaseElement.Name], v)
					}
				}
				// 2. 逐个比较每个期望的变量
				for _, wantVar := range tt.wantVars {
					actualVars, exists := varMap[wantVar.BaseElement.Name]
					assert.True(t, exists, "未找到变量: %s", wantVar.BaseElement.Name)
					if exists {
						// 变量可能有多次声明（如container），找到与期望类型匹配的那一个
						var matched *resolver.Variable
						for _, v := range actualVars {
							if assert.ObjectsAreEqual(wantVar.VariableType, v.VariableType) {
								matched = v
								break
							}
						}
						assert.NotNil(t, matched, "变量 %s 未找到匹配的类型: %v", wantVar.BaseElement.Name, wantVar.VariableType)
						if matched != nil {
							assert.ElementsMatch(t, wantVar.VariableType, matched.VariableType,
								"变量 %s 的类型不匹配，期望 %v，实际 %v",
								wantVar.BaseElement.Name, wantVar.VariableType, matched.VariableType)
							assert.Equal(t, wantVar.BaseElement.Scope, matched.BaseElement.Scope,
								"变量 %s 的Owner不匹配，期望 %v，实际 %v",
								wantVar.BaseElement.Name, wantVar.BaseElement.Scope, matched.BaseElement.Scope)
						}
					}
				}
			}
		})
	}
}

func TestPythonResolver_ResolveCall(t *testing.T) {
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
			name: "testCall.py 全部函数调用解析",
			sourceFile: &types.SourceFile{
				Path:    "testdata/python/testCall.py",
				Content: readFile("testdata/python/testCall.py"),
			},
			wantErr: nil,
			wantCalls: []resolver.Call{
				{
					BaseElement: &resolver.BaseElement{
						Name: "database_transaction",
					},
					Parameters: []*resolver.Parameter{
						{
							Name: "users_table",
						},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "json.loads",
					},
					Parameters: []*resolver.Parameter{
						{Name: "data"},
						{Name: "object_hook=CustomObject.from_dict"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "XMLParser.parse",
					},
					Parameters: []*resolver.Parameter{
						{Name: "xml_string"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "YAMLConfig.load",
					},
					Parameters: []*resolver.Parameter{
						{Name: "config.yaml"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "User.objects.get",
					},
					Parameters: []*resolver.Parameter{
						{Name: "id=1"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "User.objects.filter",
					},
					Parameters: []*resolver.Parameter{
						{Name: "age__gte=18"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "DatabaseSession.create",
					},
					Parameters: []*resolver.Parameter{
						{Name: "engine_url"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "AppConfig.Builder",
					},
					Parameters: []*resolver.Parameter{},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "set_debug",
					},
					Parameters: []*resolver.Parameter{
						{Name: "True"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "set_database_url",
					},
					Parameters: []*resolver.Parameter{
						{Name: "..."},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "build",
					},
					Parameters: []*resolver.Parameter{},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "SettingsLoader.load_from_file",
					},
					Parameters: []*resolver.Parameter{
						{Name: "settings.ini"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "CommandLineParser.parse_args",
					},
					Parameters: []*resolver.Parameter{
						{Name: "sys.argv[1:]"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "DataProcessor",
					},
					Parameters: []*resolver.Parameter{
						{Name: "[a(), 2, 3]"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "filter",
					},
					Parameters: []*resolver.Parameter{},
				},
				{
					BaseElement: &resolver.BaseElement{
						Name: "transform",
					},
					Parameters: []*resolver.Parameter{},
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)
			// if err == nil {
			// 	assert.Equal(t, tt.wantCalls, res.Elements)
			// 	for _, call := range tt.wantCalls {
			// 		assert.Contains(t, res.Elements, call)
			// 	}
			// }
		})
	}
}
