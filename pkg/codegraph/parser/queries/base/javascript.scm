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


;; Function declarations
(function_declaration
  name: (identifier) @definition.function.name
  parameters: (formal_parameters) @definition.function.parameters

  ) @definition.function

;; Generator declaration
(generator_function_declaration
  name: (identifier) @definition.function.name
  parameters: (formal_parameters)? @definition.function.parameters
  ) @definition.function

;; arrow_function declarations
(variable_declarator
  name:(identifier)@definition.function.name
  value:(arrow_function
    [
      parameter:(identifier) @definition.function.parameters
      parameters:(formal_parameters) @definition.function.parameters
    ]
  )
)@definition.function

;; Object properties
(pair
  key: (property_identifier) @definition.function.name
  value:(function_expression
    parameters: (formal_parameters)@definition.function.parameters
  )  
) @definition.function

;; 函数、变量
(variable_declarator
  name: (identifier) @variable.name
  ) @variable

;;解构变量
(variable_declarator
  name: [(array_pattern 
          (identifier) @variable.name)
          (object_pattern 
          (shorthand_property_identifier_pattern) @variable.name)]
  ) @variable


;; Export declarations
(export_statement
  declaration: (function_declaration
                 name: (identifier) @definition.export_function.name)) @definition.export_function

;; Export named declarations
(export_statement
  (export_clause
    (export_specifier
      name: (identifier) @definition.export_statement.name))) @definition.export_statement

;; 类声明 - 匹配所有类
(class_declaration
  name: (identifier) @definition.class.name
  (class_heritage)? @definition.class.extends
) @definition.class

;; 类方法
(method_definition
  name: (property_identifier) @definition.method.name
  parameters: (formal_parameters) @definition.method.parameters) @definition.method


;; 函数调用
(call_expression
  function:  (identifier)@call.function.name
  arguments: (arguments) @call.function.arguments
  ) @call.function

(call_expression
  function: (member_expression)@call.method.name
  arguments: (arguments) @call.method.arguments
  ) @call.method

(new_expression
  constructor:(member_expression)@call.struct
)