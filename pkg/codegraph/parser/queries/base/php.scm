;; PHP structure query
;; Captures function definitions, class definitions, interface definitions, and more

;; Function definitions
(function_definition
  name: (name) @definition.function.name) @definition.function

;; Method declarations
(method_declaration
  name: (name) @definition.method.name) @definition.method

;; Class declarations
(class_declaration
  name: (name) @definition.class.name) @definition.class

;; Interface declarations
(interface_declaration
  name: (name) @definition.interface.name) @definition.interface

;; Trait declarations
(trait_declaration
  name: (name) @definition.trait.name) @definition.trait

;; Namespace definitions
(namespace_definition
  name: (namespace_name) @namespace.name) @namespace

;; Property declarations
(property_declaration
  (property_element
    (variable_name) @definition.property.name)) @definition.property

;; Constant declarations
(const_declaration
  (const_element (name) @definition.constant.name)) @definition.constant

;; Variable declarations
(static_variable_declaration
  (variable_name) @definition.static_variable.name) @definition.static_variable

;; Type alias declarations (using)
(use_declaration
  (name) @using.name) @using

;; Enum declarations (PHP 8.1+)
(enum_declaration
  name: (name) @definition.enum.name) @definition.enum

(function_call_expression
  function: (name) @call.function.name
  arguments: (arguments) @call.function.arguments
  ) @call.function
