(preproc_include
  path: (system_lib_string)* @import.name
  path: (string_literal)* @import.name
  ) @import

;; Macro definitions
(preproc_def
  name: (identifier) @constant.name
  ) @constant

;; Constant declarations
(declaration
  (type_qualifier) @qualifier
  declarator: (init_declarator
                declarator: (identifier) @constant.name)
  (#eq? @qualifier "const")) @constant


;; extern Variable declarations
(translation_unit
  (declaration
    (storage_class_specifier) @type
    (identifier) @global_extern_variable.name
    (#eq? @type "extern")
    ) @global_extern_variable
  )


;; Variable declarations
(translation_unit
  (declaration
    (_) * @type
    declarator: (init_declarator
                  declarator: (identifier) @global_variable.name)
    (#not-eq? @type "const")
    (#not-eq? @type "extern")
    ) @global_variable
  )

;; Struct declarations
(struct_specifier
  name: (type_identifier) @definition.struct.name) @definition.struct

;; Enum declarations
(enum_specifier
  name: (type_identifier) @definition.enum.name) @definition.enum

;; Union declarations
(union_specifier
  name: (type_identifier) @definition.union.name) @definition.union


(declaration
  declarator: (function_declarator
                declarator: (identifier) @declaration.function.name
                parameters: (parameter_list) @declaration.function.parameters
                )
  ) @declaration.function

;; Function definitions
(function_definition
  declarator: (function_declarator
                declarator: (identifier) @definition.function.name
                parameters: (parameter_list) @definition.function.parameters
                )
  ) @definition.function

;; TODO 去找它的 identifier
;; variable & function  declaration
(declaration
  type: (_) @type
  ) @declaration

;; function_call  TODO ，这里不好确定它的parent，原因是存在嵌套、赋值等。可能得通过代码去递归。
(call_expression
  function: (identifier) @call.funciton.name
  arguments: (argument_list) @call.funciton.arguments
  ) @call.funciton

