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
      ) *
    (namespace_import
      (identifier) @import.name
      ) *
    ) *
  source: (string) * @import.source
  ) @import

;; let/const declarations
(lexical_declaration
  (variable_declarator
    name: (identifier) @name)
  ) @definition.let

;; Function declarations
(function_declaration
  name: (identifier) @name) @definition.function

;; Function expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name)
  ) @definition.variable


;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @definition.method

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @definition.interface

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @definition.type_alias

;; Type declarations（TypeScript 中通常用 type_alias_declaration 表示类型别名）
;; 注：type_declaration 可能不是标准节点，建议统一使用 type_alias_declaration

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @definition.enum


;; Decorator declarations
(decorator
  (identifier) @name) @definition.decorator

;; Abstract class declarations
(class_declaration
  name: (type_identifier) @name) @definition.class

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