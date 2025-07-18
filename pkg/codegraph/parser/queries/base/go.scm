(package_clause
  (package_identifier) @package.name
  ) @package

;; TODO 双引号需要去掉
(import_declaration
  (import_spec_list
    (import_spec
      name: (package_identifier) * @import.alias
      path: (interpreted_string_literal) @import.path
      )
    ) *

  (import_spec
    name: (package_identifier) * @import.alias
    path: (interpreted_string_literal) @import.path
    ) *
  ) @import

;; function
(function_declaration
  name: (identifier) @definition.function.name
  parameters: (parameter_list) @definition.function.parameters
  ) @definition.function

;; 全局变量声明 - 直接捕获标识符节点
(source_file
  (var_declaration
    (var_spec
      name: (identifier) @global_variable
      type: (_)? @global_variable.type
      value: (_)? @global_variable.value
    )
  )
)

;; 函数内的变量声明 - 直接捕获标识符节点
(block
  (var_declaration
    (var_spec
      name: (identifier) @variable
      type: (_)? @variable.type
      value: (_)? @variable.value
    )
  )
)

;; var块中的多变量声明
(var_declaration
  (var_spec_list
    (var_spec
      name: (identifier) @variable
      type: (_)? @variable.type
      value: (expression_list
               (_) @variable.value
             )*
    )
  )
)

;;短变量
(block
  (short_var_declaration
    left: (expression_list
            (identifier) @local_variable
          )
    right: (expression_list
            (_) @local_variable.value
          )
    )
)


;; method
(method_declaration
  receiver: (parameter_list
              (parameter_declaration
                name: (identifier)*
                type: (type_identifier) @definition.method.owner
                )
              )
  name: (field_identifier) @definition.method.name
  parameters: (parameter_list) @definition.method.parameters
  ) @definition.method

(type_declaration (type_spec name: (type_identifier) @definition.interface.name type: (interface_type) @definition.interface.type)) @definition.interface

(type_declaration (type_spec name: (type_identifier) @definition.struct.name type: (struct_type) @definition.struct.type)) @definition.struct

(type_declaration (type_spec name: (type_identifier) @definition.type_alias.name type: (type_identifier))) @definition.type_alias

;; 常量声明 - 直接捕获标识符节点

;;全局常量
(source_file
  (const_declaration
    (const_spec
      name: (identifier) @global_variable
      value: (_)? @global_variable.value
    )
  )
)

;;局部常量
(block
  (const_declaration
    (const_spec
      name: (identifier) @constant
      value: (_)? @constant.value
    )
  )
)


;; function/method_call
(call_expression
  arguments: (argument_list) @call.function.arguments
  ) @call.function