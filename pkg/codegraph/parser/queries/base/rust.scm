(use_declaration
  argument: (scoped_identifier
              name: (identifier) ) * @use.name
  argument: (scoped_use_list
              list: (use_list
                      (self)*
                      (identifier)
                      (self)*
                      )) *@use.name
  ) @use

;; Rust structure query
;; Captures function definitions, struct definitions, trait definitions, and more

;; Function definitions
(function_item
  name: (identifier) @definition.function.name) @definition.function

;; Struct definitions
(struct_item
  name: (type_identifier) @definition.struct.name) @definition.struct

;; Enum definitions
(enum_item
  name: (type_identifier) @definition.enum.name) @definition.enum

;; Trait definitions
(trait_item
  name: (type_identifier) @definition.trait.name) @definition.trait

;; Implementation blocks
(impl_item
  trait: (type_identifier) @definition.impl_item.name) @definition.impl_item

;; Type definitions
(type_item
  name: (type_identifier) @definition.type_item.name) @definition.type_item

;; Constant definitions
(const_item
  name: (identifier) @constant.name) @constant

;; Static definitions
(static_item
  name: (identifier) @static_item.name) @static_item

;; Module declarations
(mod_item
  name: (identifier) @module.name) @module

;; Macro definitions
(macro_definition
  name: (identifier) @macro.name) @macro

(let_declaration
  pattern: (identifier) @local_variable.name
  ) @local_variable

(call_expression
  function: (identifier) @call.function.name
  arguments: (arguments) @call.function.arguments
  ) @call.function
