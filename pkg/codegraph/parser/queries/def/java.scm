;; Java structure query
;; Captures class definitions, interface definitions, method definitions, and more

;; Class declarations
(class_declaration
  name: (identifier) @name) @declaration.class

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @declaration.interface

;; Method declarations
(method_declaration
  name: (identifier) @name) @declaration.method

;; Constructor declarations
(constructor_declaration
  name: (identifier) @name) @declaration.constructor

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @declaration.enum

;; Field declarations
(field_declaration
  declarator: (variable_declarator
                name: (identifier) @name)) @declaration.field

;; Constant field declarations (static final)
(field_declaration
  (modifiers
    "static"
    "final")
  (variable_declarator
    name: (identifier) @name)) @declaration.constant

;; Enum constants
(enum_constant
  name: (identifier) @name) @declaration.enum_constant

;; Type parameters
(type_parameters
  (type_parameter) @type_parameter) @declaration.type_parameters

;; Annotation declarations
(annotation_type_declaration
  name: (identifier) @name) @declaration.annotation_type