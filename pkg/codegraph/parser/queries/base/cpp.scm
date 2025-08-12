(preproc_include
  path: (system_lib_string) @import.name
) @import

(preproc_include
  path: (string_literal) @import.name
) @import

(using_declaration
  (identifier) @import.name
  ) @import



;; ----------------------------类的声明-------------------------------
(class_specifier
  name: (type_identifier) @definition.class.name
  (base_class_clause)? @definition.class.extends
  body:(_)
) @definition.class


;; Struct declarations
(struct_specifier
  name: (type_identifier) @definition.struct.name
  (base_class_clause)? @definition.struct.extends
  body:(_)
)@definition.struct

;; Union declarations
(union_specifier
  name: (type_identifier) @definition.union.name
  ;;做占位，用于区分声明和定义
  body: (_)
) @definition.union

;; Enum declarations - treat enum name as class
(enum_specifier
  name: (type_identifier) @definition.enum.name
  (#not-match? @definition.enum.name "^$")
  ;;做占位，用于区分声明和定义
  body: (_)
)@definition.enum

(type_definition
  type: (_) @definition.typedef.name
  declarator: [
    ;; 基本类型别名 (如: typedef int MyInt;)
    (type_identifier) @definition.typedef.alias
    
    ;; 指针类型 (如: typedef int* IntPtr;)
    (pointer_declarator
      declarator: (type_identifier) @definition.typedef.alias)
    
    ;; 数组类型 (如: typedef int IntArray[10];)
    (array_declarator
      declarator: (type_identifier) @definition.typedef.alias)
    
    ;; 函数类型 (如: typedef int MyFunc(int);)
    (function_declarator
      declarator: (type_identifier) @definition.typedef.alias
      parameters: (parameter_list))
    
    ;; 简单函数指针 (如: typedef int (*FuncPtr)(int, int);)
    (function_declarator
      declarator: (parenthesized_declarator
        (pointer_declarator
          declarator: (type_identifier) @definition.typedef.alias))
      parameters: (parameter_list))
    
    ;; 复杂函数指针 
    (function_declarator
      declarator: (parenthesized_declarator
        (pointer_declarator
          declarator: (type_identifier) @definition.typedef.alias)))
    
    ;; 多层嵌套的指针/数组组合
    (pointer_declarator
      declarator: (array_declarator
        declarator: (type_identifier) @definition.typedef.alias))
    
    (array_declarator
      declarator: (pointer_declarator
        declarator: (type_identifier) @definition.typedef.alias))
    
    ;; 其他可能的复杂声明符
    (parenthesized_declarator
      (type_identifier) @definition.typedef.alias)
  ]
) @definition.typedef

;; Type alias declarations (these are definitions)
(alias_declaration
  name: (type_identifier) @definition.type_alias.alias
  type: (_) @definition.type_alias.name
) @definition.type_alias

;; namespace Math {}
(namespace_definition
  name: (namespace_identifier) @namespace.name
  ) @namespace



;; ------------------------------变量声明----------------------------------
;; Variable declarations (keep as declaration)
;; int x = 42;
(declaration
  type: (_) @variable.type
  declarator: [
    ;; 有默认值：init_declarator 结构
    (init_declarator
      declarator: (identifier) @variable.name
      value: (_)? @variable.value)
    ;; 没有默认值：裸 declarator（identifier）
    (identifier) @variable.name
  ]
  (#not-match? @variable.name "^$") ;; 专门针对嵌套类的情况
) @variable

;; 指针类型
(declaration
  type: (_) @variable.type
  declarator: [
    ;; 有默认值
    (init_declarator
      declarator: (pointer_declarator
                    declarator: (identifier) @variable.name)
      value: (_)? @variable.value)

    ;; 没有默认值
    (pointer_declarator
      declarator: (identifier) @variable.name)
  ]
  (#not-match? @variable "^$")
) @variable
;; 引用类型
(declaration
  type: (_)@variable.type
  declarator: (init_declarator
    declarator: (reference_declarator
      (identifier) @variable.name)
    value: (_) @variable.value)
) @variable

;; char buf[4] = "abc";
(declaration
  type: (_) @variable.type
  declarator: [
    ; 情形 A：有初始化
    (init_declarator
      declarator: (array_declarator
        declarator: (identifier) @variable.name)
      value: (_)? @variable.value)
    ; 情形 B：没有初始化
    (array_declarator
      declarator: (identifier) @variable.name)
  ]) @variable


;; ------------------------字段声明--------------------------------
(field_declaration
  type: (_) @definition.field.type
  declarator: [
    (field_identifier) @definition.field.name
    (reference_declarator (field_identifier) @definition.field.name)
    (pointer_declarator (field_identifier) @definition.field.name)
    (array_declarator declarator: (field_identifier) @definition.field.name)
  ]
  (#not-match? @definition.field.name "^$") ;; 用于针对捕获到嵌套类、结构体等异常情况
  default_value: (_) @definition.field.value ?
) @definition.field

;; Enum constants - treat enum values as fields  
(enumerator 
  name: (identifier) @definition.enum.constant.name
  value: (_)? @definition.enum.constant.value
)@definition.enum.constant


;;-----------------------函数/方法定义----------------------------
;; 返回值不带指针和引用的基础函数定义
(function_definition
  type: (_) @definition.function.return_type
  declarator: [
    ;; 直接函数声明符（如：void func14(...)）
    (function_declarator
      declarator: (identifier) @definition.function.name
      parameters: (parameter_list) @definition.function.parameters
    )
    ;; 指针函数声明符（如：int *func(...)）
    (pointer_declarator
      declarator: (function_declarator
        declarator: (identifier) @definition.function.name
        parameters: (parameter_list) @definition.function.parameters
      )
    )
    ;; 双指针函数声明符（如：int **func(...)）
    (pointer_declarator
      declarator: (pointer_declarator
        declarator: (function_declarator
          declarator: (identifier) @definition.function.name
          parameters: (parameter_list) @definition.function.parameters
        )
      )
    )
  ]
) @definition.function

;; 返回值带引用的函数定义
(function_definition
  type: (_) @definition.function.return_type
  declarator: (reference_declarator                    ;; 引用修饰
    (function_declarator
      declarator: (identifier) @definition.function.name
      parameters: (parameter_list) @definition.function.parameters
    )
  ) @definition.function.reference
) @definition.function

;; 方法的定义
(function_definition
  type: (_) @definition.method.return_type
  declarator: (function_declarator
                declarator: (field_identifier) @definition.method.name
                parameters: (parameter_list) @definition.method.parameters
                (type_qualifier)? @definition.method.qualifiers
              )
) @definition.method


;; -----------------------------方法/函数调用-----------------------------
;; TODO 对象.方法 对象->方法
(call_expression
  function: (
              field_expression
              argument: (_) @call.method.owner
              field: (field_identifier) @call.method.name
              )
  arguments: (argument_list) @call.method.arguments
) @call.method

;; add(a,b)
(call_expression
  function: (identifier) @call.function.name
  arguments: (argument_list) @call.function.arguments
  )@call.function

; 匹配 std::max<int>(...) 或 ::ns::foo<T>(...)
(call_expression
  ; function 可能是 qualified_identifier
  function: (qualified_identifier
    scope: (_) @call.function.owner    ; <- 命名空间 / 类名
    name: (_) @call.function.name)
  arguments: (argument_list) @call.function.arguments
) @call.function

;; max<int>(1,2)
;;(call_expression
;;  function: (template_function 
;;  name: (_) @call.function.name)
;;  arguments: (argument_list) @call.function.arguments
;;) @call.function

;; auto c = foo<int>(42, "hello");
(call_expression
  function: (template_function
    name: (_) @call.template.name
    arguments: (template_argument_list) @call.template.args)
  arguments: (argument_list) 
) @call.template


(new_expression
  type: (qualified_identifier
           scope: (namespace_identifier) @call.new.owner
           name: (type_identifier) @call.new.type
         ) 
  arguments: (argument_list)? @call.new.args
) @call.new
(new_expression
  type: (type_identifier) @call.new.type                  
  arguments: (argument_list)? @call.new.args
) @call.new
