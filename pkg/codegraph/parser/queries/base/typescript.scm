(import_statement
  (import_clause
    (identifier) @import.name
    ) *
  (import_clause
    (named_imports
      (import_specifier
        name: (identifier)? @import.name
        alias: (identifier) * @import.alias
        )
      )
    ) *
  (import_clause
    (namespace_import
      (identifier) @import.alias
    )
  ) *
  source: (string)* @import.source
  ) @import

;;import函数
(variable_declarator
  name:(identifier) @import.name
  (call_expression
    function:(import)@import.declaration
    arguments:(arguments(string)@import.source)
  )
)@import

;;import函数 - 带await的动态导入
(variable_declarator
  name:(identifier) @import.name
  value:(await_expression
    (call_expression
      function:(import)@import.declaration
      arguments:(arguments(string)@import.source)
    )
  )
)@import

(variable_declarator
  name:(identifier)@import.name
  value:(arrow_function
    body:(call_expression
      function:(import) @import.declaration
      arguments:(arguments
        (string)@import.source
      )
    )
  )
)@import


;;-----------------------------变量定义--------------------------

;; 函数
(variable_declarator
  name: (identifier) @variable.name
  type: (type_annotation)? @variable.type
) @variable

;; Enum Assignment
(enum_assignment
  name: (property_identifier) @definition.enum.constant.name
  value: (_)? @definition.enum.constant.value
  ) @definition.enum.constant

;;解构变量
(variable_declarator
  name: [(array_pattern 
          (identifier) @variable.name)
          (object_pattern 
          (shorthand_property_identifier_pattern) @variable.name
          )]
  type: (type_annotation)? @variable.type
) @variable

;;type variable
(type_alias_declaration
  name: (type_identifier) @variable.name
  type_parameters: (type_parameters)? @variable.type
) @variable

;; Enum declarations
(enum_declaration
  name: (identifier) @definition.enum.constant.name
  body: (_)
  ) @definition.enum.constant

;;-----------------------------函数定义--------------------------

;; Function declarations
(function_declaration
  name: (identifier) @definition.function.name
  parameters: (formal_parameters)? @definition.function.parameters
  return_type:(type_annotation)? @definition.function.return_type
  ) @definition.function

;; Generator declaration
(generator_function_declaration
  name: (identifier) @definition.function.name
  parameters: (formal_parameters)? @definition.function.parameters
  return_type:(type_annotation)? @definition.function.return_type
  ) @definition.function

;;箭头函数
(variable_declarator
  name:(identifier)@definition.function.name
  value:(arrow_function
    [
      parameter:(identifier) @definition.function.parameters
      parameters:(formal_parameters) @definition.function.parameters 
    ]
    return_type:(type_annotation)? @definition.function.return_type
  )
)@definition.function

;;-----------------------------方法定义--------------------------

;; 类方法
(method_definition
  (accessibility_modifier)? @definition.method.modifier
  name: (property_identifier) @definition.method.name
  parameters: (formal_parameters)? @definition.method.parameters
  return_type:(type_annotation)?@definition.method.return_type
  ) @definition.method

;;-----------------------------接口声明--------------------------
;; Interface declarations
(interface_declaration
  name: (type_identifier) @definition.interface.name
  (extends_type_clause)? @definition.interface.extends
  ) @definition.interface

;;-----------------------------类声明--------------------------

;; Abstract class declarations
(abstract_class_declaration
  name: (type_identifier) @definition.class.name
  (class_heritage
    (extends_clause (identifier) @definition.class.extends)
    )?
  (implements_clause)? @definition.class.implements
  ) @definition.class

;;class declarations
(class_declaration
  name: (type_identifier) @definition.class.name
  (class_heritage
    (extends_clause (identifier) @definition.class.extends)
    )?
  (implements_clause)? @definition.class.implements
  ) @definition.class

;;-----------------------------方法调用--------------------------
;; method call
(call_expression
  function: (member_expression) @call.method.owner
  arguments: (arguments) @call.method.arguments
  ) @call.method

(call_expression
  function: (identifier) @call.function.owner
  arguments: (arguments) @call.function.arguments
  ) @call.function

(new_expression
  constructor:[(member_expression)(identifier)@call.struct]
)