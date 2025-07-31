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
					{Name: "...", Type: nil},
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

func TestPythonResolver_ResolveClass(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantClasses []struct {
			Name        string
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
				Name        string
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
				{"Container", []string{"Generic","T"}},
				{"Repository", []string{"Generic","T"}},
				{"Map", []string{"Generic","K","V"}},
				{"UserContainer", []string{"Container","User"}},
				{"ProductRepository", []string{"Repository","Product"}},
				{"UserList", []string{"List","User"}},
				{"ProductList", []string{"List","Product"}},
				{"UserDict", []string{"Dict","User","str"}},
				{"ConfigDict", []string{"Dict","str","int","bool","Union","str"}},
				{"FlexibleContainer", []string{"Union", "BaseClass","PaymentMeta","List","int","str","List"}},
				{"NumberOrString", []string{"Union","int","float","str"}},
				{"ComplexClass", []string{"List","User","Dict","str","Product","MetaClass"}},
				{"DataProcessor", []string{"List","Dict","Union","int","str","str","Optional","Logger","Cache"}},
				{"AdvancedManager", []string{"List", "Dict","str","User","Permission","ManagerMeta"}},
				{"User", []string{"Dict","str","User"}}, // metaclass=Dict[str, User]，不是继承
				{"Product", []string{"List","Product"}}, // metaclass=List[Product]
				{"Order", []string{"Union","TypeA","TypeB"}}, // metaclass=Union[TypeA, TypeB]
				{"Model", []string{"Model"}}, // metaclass=django.db.models.Model
				{"Atest", []string{"Foo","Foo1"}}, // metaclass=mylib.utils.Foo[mylib.utils.Foo1]
				{"Btest", []string{"Foo","Foo1","User"}},
				{"Ctest", []string{"Foo","User"}},
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