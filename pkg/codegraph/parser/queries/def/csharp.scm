;; C# structure query
;; Captures method definitions, class definitions, interface definitions, and more

;; Method definitio
(method_declaration
  name: (identifier) @name) @definition.method

;; Class declarations
(class_declaration
  name: (identifier) @name) @declaration.class

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @declaration.interface

;; Struct declarations
(struct_declaration
  name: (identifier) @name) @declaration.struct

;; Property declarations
(property_declaration
  name: (identifier) @name) @declaration.property

;; Delegate declarations
(delegate_declaration
  name: (identifier) @name) @declaration.delegate

;; Event declarations
(event_declaration
  name: (identifier) @name) @declaration.event

;; Constructor declarations
(constructor_declaration
  name: (identifier) @name) @declaration.constructor

;; Destructor declarations
(destructor_declaration
  name: (identifier) @name) @declaration.destructor

;; Enum declarations
(enum_declaration
  name: (identifier) @name) @declaration.enum

;; Field declarations
(field_declaration
  (variable_declaration
    (variable_declarator
      name: (identifier) @name))) @declaration.field

;; Indexer declarations
(indexer_declaration
  type: (identifier) @name) @declaration.indexer

;; Operator declarations
(operator_declaration
  type: (identifier) @name) @declaration.operator

;; Type parameter declarations
(type_parameter
  name: (identifier) @name) @declaration.type_parameter

;; Record definitions (C# 9.0+)
(record_declaration
  name: (identifier) @name) @declaration.record

;; Local function definitions
(local_function_statement
  name: (identifier) @name) @definition.local_function

;; Conversion operator definitions
(conversion_operator_declaration
  type: (identifier) @name) @declaration.conversion_operator

