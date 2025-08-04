(package_clause
  (package_identifier) @package.name
  ) @package

;; TODO 双引号需要去掉
(import_declaration
  (import_spec_list
    (import_spec
      name: [(package_identifier)(dot)] * @import.alias
      path: (interpreted_string_literal) @import.path
      )@import
    ) 
  ) 

(import_declaration
  (import_spec
    name: [(package_identifier)(dot)] @import.alias
    path: (interpreted_string_literal) @import.path
    )@import
  )

;; function
(function_declaration
  name: (identifier) @definition.function.name
  parameters: (parameter_list) @definition.function.parameters
  result:(parameter_list)? @definition.function.return_type
  ) @definition.function

;;-----------------------------变量定义--------------------------

;; 全局变量声明 - 直接捕获标识符节点
(source_file
  (var_declaration
    (var_spec
      name: (identifier) @global_variable
      type: (_)? @global_variable.type
    )
  )
)

;; 函数内的变量声明 - 直接捕获标识符节点
(block
  (var_declaration
    (var_spec
      name: (identifier) @variable
      type: (_)? @variable.type
    )
  )
)

;; var块中的多变量声明
(var_declaration
  (var_spec_list
    (var_spec
      name: (identifier) @variable
      type: (_)? @variable.type
    )
  )
)

;;短变量
(short_var_declaration
  left: (expression_list
          (identifier) @local_variable)
)

(short_var_declaration
  left: (expression_list
          (unary_expression
            operand: (identifier) @local_variable))
)

;;全局常量
(source_file
  (const_declaration
    (const_spec
      name: (identifier) @global_variable
      type: (_)? @global_variable.type
    )
  )
)

;;局部常量
(block
  (const_declaration
    (const_spec
      name: (identifier) @constant
      type: (_)? @constant.type
    )
  )
)

;;-----------------------------函数定义--------------------------

;; method
(method_declaration
  receiver: (parameter_list
              (parameter_declaration
                name: (identifier)*
                type: [(type_identifier) @definition.method.owner (pointer_type (type_identifier) @definition.method.owner)]
                )
              )
  name: (field_identifier) @definition.method.name
  parameters: (parameter_list) @definition.method.parameters
  result:(parameter_list)? @definition.function.return_type
  ) @definition.method

(type_declaration (type_spec name: (type_identifier) @definition.interface.name type: (interface_type) @definition.interface.type)) @definition.interface

(type_declaration (type_spec name: (type_identifier) @definition.struct.name type: (struct_type) @definition.struct.type)) @definition.struct

(type_declaration (type_spec name: (type_identifier) @variable type: (type_identifier) @variable.type))

;; 常量声明 - 直接捕获标识符节点


;; function/method_call
(call_expression
  function:[(selector_expression)(identifier)(parenthesized_expression)]@call.function.field
  arguments: (argument_list) @call.function.arguments
  ) @call.function



;;------------------------------------右值--------------------------
;;右边非基础类型赋值走call
(expression_list
  [(composite_literal
    type: [(type_identifier) (qualified_type)] @call.struct
  )

  (unary_expression
  operand:(composite_literal
    type: [(type_identifier) (selector_expression)] @call.struct
  )
 )

 (identifier) @call.struct

 (type_conversion_expression
  type:(generic_type
    type: (type_identifier) @call.struct
    type_arguments: (type_arguments) @call.struct.type
  )
 )
 
  (type_assertion_expression
    operand:(identifier) @call.struct
    type:(type_identifier) @call.struct.type
  )
 ]
)




