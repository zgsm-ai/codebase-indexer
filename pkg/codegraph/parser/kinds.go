package parser

//
//import "codebase-indexer/pkg/codegraph/parser/resolver"
//
//const identifier = "identifier"
//
//// DefinitionNodeType 定义了各语言在 tree-sitter 中的原始节点类型
//var DefinitionNodeType = map[resolver.Language][]string{
//	Go: {
//		"function_declaration", // 函数声明
//		"method_declaration",   // 方法声明
//		"type_declaration",     // 类型声明
//		"struct_type",          // 结构体类型
//		"interface_type",       // 接口类型
//		"var_declaration",      // 变量声明
//		"const_declaration",    // 常量声明
//	},
//	Java: {
//		"class_declaration",       // 类声明
//		"interface_declaration",   // 接口声明
//		"method_declaration",      // 方法声明
//		"constructor_declaration", // 构造函数声明
//		"enum_declaration",        // 枚举声明
//		"field_declaration",       // 字段声明
//		"type_parameter",          // 类型参数(泛型)
//	},
//	Python: {
//		"function_definition",  // 函数定义
//		"class_definition",     // 类定义
//		"decorated_definition", // 装饰器定义
//		"assignment",           // 赋值语句
//	},
//	JavaScript: {
//		"function_declaration", // 函数声明
//		"class_declaration",    // 类声明
//		"method_definition",    // 方法定义
//		"arrow_function",       // 箭头函数
//		"variable_declarator",  // 变量声明
//	},
//	TypeScript: {
//		"function_declaration",   // 函数声明
//		"class_declaration",      // 类声明
//		"interface_declaration",  // 接口声明
//		"type_alias_declaration", // 类型别名声明
//		"enum_declaration",       // 枚举声明
//		"namespace_declaration",  // 命名空间声明
//		"module_declaration",     // 模块声明
//	},
//	TSX: {
//		"function_declaration",     // 函数声明
//		"class_declaration",        // 类声明
//		"jsx_element",              // JSX元素
//		"jsx_self_closing_element", // JSX自闭合元素
//		"jsx_fragment",             // JSX片段
//	},
//	Rust: {
//		"function_item", // 函数项
//		"struct_item",   // 结构体项
//		"enum_item",     // 枚举项
//		"trait_item",    // 特质项
//		"impl_item",     // 实现项
//		"type_item",     // 类型项
//		"const_item",    // 常量项
//		"static_item",   // 静态项
//	},
//	C: {
//		"function_definition", // 函数定义
//		"struct_declaration",  // 结构体声明
//		"union_declaration",   // 联合体声明
//		"enum_declaration",    // 枚举声明
//		"typedef_declaration", // 类型定义
//		"declaration",         // 声明
//	},
//	CPP: {
//		"function_definition",  // 函数定义
//		"class_declaration",    // 类声明
//		"struct_declaration",   // 结构体声明
//		"namespace_definition", // 命名空间定义
//		"template_declaration", // 模板声明
//		"using_declaration",    // using声明
//	},
//	CSharp: {
//		"method_declaration",    // 方法声明
//		"class_declaration",     // 类声明
//		"interface_declaration", // 接口声明
//		"struct_declaration",    // 结构体声明
//		"property_declaration",  // 属性声明
//		"delegate_declaration",  // 委托声明
//		"event_declaration",     // 事件声明
//	},
//	Ruby: {
//		"method",           // 方法
//		"class",            // 类
//		"module",           // 模块
//		"singleton_method", // 单例方法
//		"constant",         // 常量
//	},
//	PHP: {
//		"function_definition",   // 函数定义
//		"method_declaration",    // 方法声明
//		"class_declaration",     // 类声明
//		"interface_declaration", // 接口声明
//		"trait_declaration",     // 特质声明
//		"namespace_definition",  // 命名空间定义
//	},
//	Kotlin: {
//		"class_declaration",     // 类声明
//		"interface_declaration", // 接口声明
//		"object_declaration",    // 对象声明
//		"function_declaration",  // 函数声明
//		"property_declaration",  // 属性声明
//		"type_alias",            // 类型别名
//		"enum_class",            // 枚举类
//		"companion_object",      // 伴生对象
//	},
//	Scala: {
//		"class_definition",  // 类定义
//		"trait_definition",  // 特质定义
//		"object_definition", // 对象定义
//		"def",               // 函数定义
//		"val_definition",    // 值定义
//		"var_definition",    // 变量定义
//		"type_definition",   // 类型定义
//		"case_class",        // 样例类
//	},
//}
