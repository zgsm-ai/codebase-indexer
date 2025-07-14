

(lexical_declaration (variable_declarator name: (identifier) @name ) ) @declaration.let

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
  name: (type_identifier) @name) @definition.interface

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @name) @declaration.type_alias


;; Enum declarations
(enum_declaration
  name: (identifier) @name) @declaration.enum


;; Decorator declarations
(decorator
  (identifier) @name) @declaration.decorator

;; Abstract class declarations
(class_declaration
  name: (type_identifier) @name ) @declaration.class

;; Abstract method declarations
(method_definition
  name: (property_identifier) @name
  ) @definition.method


;; Conditional type declarations
(conditional_type
  left: (type) @name
  ) @declaration.conditional_type

;; Import type declarations
(import_statement ) @import_statement @name

;; Export type declarations
(export_statement ) @export_statement @name



;; JSX Element declarations (custom components)
(jsx_element
  open_tag: (jsx_opening_element
              name: (identifier) @name)) @definition.jsx_element

;; JSX Self-closing elements
(jsx_self_closing_element
  name: (identifier) @name) @definition.jsx_element



;; JSX Namespace components
(jsx_element
  open_tag: (jsx_opening_element
              name: (member_expression
                      object: (identifier) @namespace
                      property: (property_identifier) @name))) @definition.jsx_element

;; JSX Props interface declarations
(interface_declaration
  name: (type_identifier) @name
  (#match? @name "^.*Props$")) @definition.interface


;; React Component type declarations
(type_alias_declaration
  name: (type_identifier) @name
  value: (union_type
           (type_identifier) @react
           (#eq? @react "React"))) @definition.type