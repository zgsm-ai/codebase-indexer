(package_clause
  (package_identifier) @package.name
  ) @package

;; TODO 双引号需要去掉
(import_declaration
  (import_spec_list
    (import_spec
      name: (package_identifier) * @import.alias
      path: (interpreted_string_literal) @import.name
      )
    ) *

  (import_spec
    name: (package_identifier) * @import.alias
    path: (interpreted_string_literal) @import.name
    ) *
  ) @import

;; function
(function_declaration
  name: (identifier) @definition.function.name
  parameters: (parameter_list) @definition.function.parameters
  ) @definition.function

(source_file (var_declaration (var_spec name: (identifier) @global_variable.name)) @global_variable)

(var_declaration (var_spec name: (identifier) @variable.name)) @variable

;; 多个局部变量，逗号分割,正常不会超过10个
(short_var_declaration
  left: (expression_list
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          (identifier)* @local_variable.name
          )
  ) @local_variable

;; method
(method_declaration
  receiver: (parameter_list
              (parameter_declaration
                name: (identifier)*
                type: (type_identifier) @definition.method.owner
                )
              )
  name: (field_identifier) @definition.method.name
  parameters: (parameter_list) @definition.method.parameters
  ) @definition.method

(type_declaration (type_spec name: (type_identifier) @definition.interface.name type: (interface_type))) @definition.interface

(type_declaration (type_spec name: (type_identifier) @definition.struct.name type: (struct_type))) @definition.struct

(type_declaration (type_spec name: (type_identifier) @definition.type_alias.name type: (type_identifier))) @definition.type_alias


(source_file (const_declaration (const_spec name: (identifier) @constant.name)) @constant)


;; function/method_call
(call_expression
  function: (selector_expression
              operand: (identifier) @call.function.owner
              field: (field_identifier) @call.function.name
              )
  arguments: (argument_list) @call.function.arguments
  ) @call.function