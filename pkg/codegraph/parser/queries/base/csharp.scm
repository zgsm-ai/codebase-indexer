(using_directive
  (identifier)* @import.name
  (qualified_name name: (identifier)@import.name )*
  ) @import

(namespace_declaration
  name: (identifier) @namespace.name
  ) @namespace


;; Method definitio
(method_declaration
  name: (identifier) @definition.method.name) @definition.method

;; Class declarations
(class_declaration
  name: (identifier) @definition.class.name) @definition.class

;; Interface declarations
(interface_declaration
  name: (identifier) @definition.interface.name) @definition.interface

;; Struct declarations
(struct_declaration
  name: (identifier) @definition.struct.name) @definition.struct

;; Property declarations
(property_declaration
  name: (identifier) @definition.property.name) @definition.property

;; Delegate declarations
(delegate_declaration
  name: (identifier) @definition.delegate.name) @definition.delegate

;; Event declarations
(event_declaration
  name: (identifier) @definition.event.name) @definition.event

;; Constructor declarations
(constructor_declaration
  name: (identifier) @definition.constructor.name) @definition.constructor

;; Destructor declarations
(destructor_declaration
  name: (identifier) @definition.destructor.name) @definition.destructor

;; Enum declarations
(enum_declaration
  name: (identifier) @definition.enum.name) @definition.enum

;; Field declarations
(field_declaration
  (variable_declaration
    (variable_declarator
      name: (identifier) @definition.field.name))) @definition.field

;; Indexer declarations
(indexer_declaration
  type: (identifier) @definition.indexer.name) @definition.indexer

;; Operator declarations
(operator_declaration
  type: (identifier) @definition.operator.name) @definition.operator

;; Type parameter declarations
(type_parameter
  name: (identifier) @definition.type_parameter.name) @definition.type_parameter

;; Record definitions (C# 9.0+)
(record_declaration
  name: (identifier) @definition.record.name) @definition.record

;; Local function definitions
(local_function_statement
  name: (identifier) @definition.local_function.name) @definition.local_function

;; Conversion operator definitions
(conversion_operator_declaration
  type: (identifier) @definition.conversion_operator.name) @definition.conversion_operator

;; 方法调用
(invocation_expression
  function: (member_access_expression
              expression: (identifier) @call.method.owner
              name: (identifier) @call.method.name
              )
  arguments: (argument_list) @call.method.arguments
  )

