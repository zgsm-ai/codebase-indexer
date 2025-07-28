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
    arguments:(arguments)@import.source
  )
)@import

;;import函数 - 带await的动态导入
(variable_declarator
  name:(identifier) @import.name
  value:(await_expression
    (call_expression
      function:(import)@import.declaration
      arguments:(arguments)@import.source
    )
  )
)@import

;; 函数、变量
(variable_declarator
  name: (identifier) @variable.name
  type: (type_annotation)? @variable.type
  value: (_) @variable.value
) @variable

;;解构变量
(variable_declarator
  name: [(array_pattern 
          (identifier) @variable.name)
          (object_pattern 
          (shorthand_property_identifier_pattern) @variable.name)]
  type: (type_annotation)? @variable.type
  value: (_) @variable.value
) @variable

;;type variable
(type_alias_declaration
  name: (type_identifier) @variable.name
  type_parameters: (type_parameters)? @variable.type
) @variable

;; Function declarations
(function_declaration
  name: (identifier) @name) @definition.function


;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @definition.method

;; Interface declarations
(interface_declaration
  name: (type_identifier) @definition.interface.name
  (extends_type_clause)? @definition.interface.extends
  ) @definition.interface


;; Type declarations（TypeScript 中通常用 type_alias_declaration 表示类型别名）
;; 注：type_declaration 可能不是标准节点，建议统一使用 type_alias_declaration

;; Enum declarations
(enum_declaration
  name: (identifier) @variable.name) @variable


;; Decorator declarations
(decorator
  (identifier) @name) @definition.decorator

;; Abstract class declarations
(class_declaration
  name: (type_identifier) @definition.class.name
  (class_heritage
    (extends_clause (identifier) @definition.class.extends)
    )?
  (implements_clause)? @definition.class.implements
  ) @definition.class

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name) @definition.method

(import_statement) @import_declaration

;; Export type declarations
(export_statement) @export_declaration

;; method call
(call_expression
  function: (_
              (identifier) @call.method.owner
              )
  arguments: (arguments) @call.method.arguments
  ) @call.method

(call_expression
  function: (identifier) @call.function.owner
  arguments: (arguments) @call.function.arguments
  ) @call.function