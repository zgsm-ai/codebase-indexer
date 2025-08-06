(preproc_include
  path: (system_lib_string) @import.name
) @import

(preproc_include
  path: (string_literal) @import.name
) @import



;; ---------------------------------- Class Declaration ----------------------------------
;; Struct declarations
(struct_specifier
  name: (type_identifier) @definition.struct.name
  body:(_)
) @definition.struct

;; Enum declarations
(enum_specifier
  name: (type_identifier) @definition.enum.name
  body:(_)
) @definition.enum

;; Union declarations
(union_specifier
  name: (type_identifier) @definition.union.name
  ;;做占位，用于区分声明和定义
  body: (field_declaration_list)@body
) @definition.union

(type_definition
  type: (_) @definition.typedef.name
  declarator: (_) @definition.typedef.alias
)@definition.typedef



;; -------------------------------- Function Declarations --------------------------------
;; Function declarations
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



;; ------------------------ Variable/Field Declaration --------------------------------
(declaration
  type: (_) @variable.type
  declarator:
    ;; ==== 有初始化值 ====
    (init_declarator
      declarator: [
        (identifier) @variable.name
        (pointer_declarator
          declarator: (identifier) @variable.name)
        (array_declarator
          declarator: (identifier) @variable.name)
        (array_declarator
          declarator: (array_declarator
            declarator: (identifier) @variable.name))
        (pointer_declarator
          declarator: (array_declarator
            declarator: (identifier) @variable.name))
      ]
      value: (_) @variable.value)
) @variable

;; 无初始化值的变量声明
(declaration
  type: type: (_) @variable.type
  declarator: [
    (identifier) @variable.name
    (pointer_declarator
      declarator: (identifier) @variable.name)
    (array_declarator
      declarator: (identifier) @variable.name)
    (array_declarator
      declarator: (array_declarator
        declarator: (identifier) @variable.name))
    (pointer_declarator
      declarator: (array_declarator
        declarator: (identifier) @variable.name))
  ]
) @variable


;; ------------------------ Field Declaration --------------------------------
(field_declaration
  type: (_) @definition.field.type
  declarator: [
    (field_identifier) @definition.field.name
    (pointer_declarator
      declarator: (field_identifier) @definition.field.name)
    (array_declarator
      declarator: (field_identifier) @definition.field.name)
    (array_declarator
      declarator: (array_declarator
        declarator: (identifier) @definition.field.name))
    (pointer_declarator
      declarator: (array_declarator
        declarator: (identifier) @definition.field.name))
  ]
) @definition.field

;; ------------------------ Enum Constant --------------------------------
(enumerator
  name: (identifier) @definition.enum.constant.name
  value: (_)? @definition.enum.constant.value
) @definition.enum.constant



;; ------------------------------ Call Expression ------------------------------
(call_expression
  function: (identifier) @call.function.name
  arguments: (argument_list) @call.function.arguments
  )@call.function

;; struct MyStruct a = (struct MyStruct){.x = 1, .y = 2};
(compound_literal_expression
  type: (type_descriptor
    type: (_) @call.compound.type)
) @call.compound