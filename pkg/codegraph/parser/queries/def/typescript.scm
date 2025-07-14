;; let/const declarations
(lexical_declaration
  (variable_declarator
    name: (identifier) @name )
  ) @declaration.let

;; Function declarations
(function_declaration
  name: (identifier) @name) @declaration.function

;; Function expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name )
  ) @declaration.variable


;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @definition.method

;; Interface declarations
(interface_declaration
  name: (type_identifier) @name) @declaration.interface

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @declaration.type_alias

;; Type declarations（TypeScript 中通常用 type_alias_declaration 表示类型别名）
;; 注：type_declaration 可能不是标准节点，建议统一使用 type_alias_declaration

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @declaration.enum


;; Decorator declarations
(decorator
  (identifier) @name) @declaration.decorator

;; Abstract class declarations（修正祖先节点判断逻辑）
(class_declaration
  name: (type_identifier) @name ) @declaration.class

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name ) @definition.method

(import_statement ) @import_declaration

;; Export type declarations
(export_statement ) @export_declaration