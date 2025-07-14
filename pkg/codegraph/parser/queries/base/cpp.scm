(preproc_include
  path: (system_lib_string)* @import.name
  path: (string_literal)* @import.name
  ) @import

(using_declaration
  (identifier) @import.name
  ) @import

(namespace_definition
  name: (namespace_identifier) @namespace.name
  ) @namespace

;; Class declarations
(class_specifier
  name: (type_identifier) @definition.class.name) @definition.class

;; Struct declarations
(struct_specifier
  name: (type_identifier) @definition.struct.name) @definition.struct


;; Variable declarations (keep as declaration)
(declaration
  declarator: (init_declarator
                declarator: (identifier) @variable.name)) @variable

;; Member variable declarations (keep as declaration)
(field_declaration
  declarator: (field_identifier) @definition.field.name) @definition.field

;; Union declarations
(union_specifier
  name: (type_identifier) @definition.union.name) @definition.union

;; Enum declarations
(enum_specifier
  name: (type_identifier) @efinition.enum.name) @definition.enum

;; Type alias declarations (these are definitions)
(alias_declaration
  name: (type_identifier) @definition.type_alias.name) @definition.type_alias

;; Typedef declarations
(type_definition
  declarator: (type_identifier) @definition.typedef.name) @definition.typedef

(declaration
  declarator: (function_declarator
                declarator: (identifier) @declaration.function.name
                parameters: (parameter_list) @declaration.function.parameters
                )
  ) @declaration.function



(function_definition
  declarator: (function_declarator
                declarator: (identifier) @definition.function.name
                parameters: (parameter_list) @definition.function.parameters
                )) @definition.function

;; TODO 对象.方法
(call_expression
  function: (
              field_expression
              argument: (identifier) @call.method.owner
              field: (field_identifier) @call.method.name
              )
  arguments: (argument_list) @call.method.arguments
  ) @call.method

;; 函数调用
(call_expression
  function: (qualified_identifier
              scope: (namespace_identifier) @call.function.namespace
              name: (identifier) @call.function.name
              )
  arguments: (argument_list) @call.function.arguments
  ) @call.function