;; PHP structure query
;; Captures function definitions, class definitions, interface definitions, and more

;; Function definitions
(function_definition
  name: (name) @name) @definition.function

;; Method declarations
(method_declaration
  name: (name) @name) @declaration.method

;; Class declarations
(class_declaration
  name: (name) @name) @declaration.class

;; Interface declarations
(interface_declaration
  name: (name) @name) @declaration.interface

;; Trait declarations
(trait_declaration
  name: (name) @name) @declaration.trait

;; Namespace definitions
(namespace_definition
  name: (namespace_name) @name) @definition.namespace

;; Property declarations
(property_declaration
  (property_element
    (variable_name) @name)) @declaration.property

;; Constant declarations
(const_declaration
  (const_element (name) @name)) @declaration.constant

;; Variable declarations
(static_variable_declaration
  (variable_name) @name) @declaration.static_variable

;; Type alias declarations (using)
(use_declaration
  (name) @name) @declaration.using

;; Enum declarations (PHP 8.1+)
(enum_declaration
  name: (name) @name) @declaration.enum