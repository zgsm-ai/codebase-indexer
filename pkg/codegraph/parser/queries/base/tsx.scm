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



;; Export type declarations
(export_statement) @export

(lexical_declaration (variable_declarator name: (identifier) @varibale.name)) @varibale

;; Function declarations
(function_declaration
  name: (identifier) @definition.function.name) @definition.function

;; Function expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @variable.name)
  ) @variable


;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @definition.method.name) @definition.method

;; Interface declarations
(interface_declaration
  name: (type_identifier) @definition.interface.name) @definition.interface

;; Type alias declarations
(type_alias_declaration
  name: (type_identifier) @definition.type_alias.name) @definition.type_alias


;; Enum declarations
(enum_declaration
  name: (identifier) @definition.enum.name) @definition.enum


;; Decorator declarations
(decorator
  (identifier) @definition.decorator.name) @definition.decorator

;; Abstract class declarations
(class_declaration
  name: (type_identifier) @definition.class.name) @definition.class

;; Conditional type declarations
(conditional_type
  left: (type) @name
  ) @definition.conditional_type



;; JSX Element declarations (custom components)
(jsx_element
  open_tag: (jsx_opening_element
              name: (identifier) @definition.jsx_element.name)) @definition.jsx_element

;; JSX Self-closing elements
(jsx_self_closing_element
  name: (identifier) @efinition.jsx_element.name) @definition.jsx_element



;; JSX Namespace components
(jsx_element
  open_tag: (jsx_opening_element
              name: (member_expression
                      object: (identifier) @definition.jsx_element.namespace
                      property: (property_identifier) @definition.jsx_element.name))) @definition.jsx_element


;; React Component type declarations
(type_alias_declaration
  name: (type_identifier) @definition.type.name
  value: (union_type
           (type_identifier) @react
           (#eq? @react "React"))) @definition.type