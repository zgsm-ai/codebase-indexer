(import_statement
  (import_clause
    (identifier) @import.name
    ) *
  (import_clause
    (named_imports
      (import_specifier
        name: (identifier) @import.name
        alias: (identifier) * @import.alias
        )
      )
    ) *
  source: (string)* @import.source

  ) @import


;; Function declarations
(function_declaration
  name: (identifier) @definition.function.name
  parameters: (formal_parameters) @definition.function.parameters

  ) @definition.function

;; 全局变量
(program
  (_
    (variable_declarator
      name: (identifier) @global_variable.name
      ) @global_variable
    )
  )

;; 函数、变量

(variable_declarator
  name: (identifier) @variable.name
  ) @variable



;; Object properties
(pair
  key: (property_identifier) @definition.property.name) @definition.property

;; Export declarations
(export_statement
  declaration: (function_declaration
                 name: (identifier) @definition.export_function.name)) @definition.export_function

;; Export named declarations
(export_statement
  (export_clause
    (export_specifier
      name: (identifier) @definition.export_statement.name))) @definition.export_statement

;; 函数调用
(call_expression
  function: (member_expression
              object: (identifier) @call.function.owner
              property: (property_identifier) @call.function.name
              )
  arguments: (arguments) @call.function.arguments
  ) @call.function