(preproc_include
  "#include" @name
  ) @include

(preproc_def) @macro @name

;; Constant declarations
(translation_unit
  (declaration
    (type_qualifier) @qualifier
    declarator: (init_declarator
                  declarator: (identifier) @name)
    (#eq? @qualifier "const")) @const
  )

;; extern Variable declarations
(translation_unit
  (declaration
    (storage_class_specifier) @type
    (identifier) @name
    (#eq? @type "extern")
    ) @global_extern_variable
  )

;; Variable declarations
(translation_unit
  (declaration
    (_) * @type
    declarator: (init_declarator
                  declarator: (identifier) @name)
    (#not-eq? @type "const")
    (#not-eq? @type "extern")
    ) @global_variable
)


;; Function definitions
(function_definition
  declarator: (function_declarator
                declarator: (identifier) @name)) @definition.function

(declaration
  declarator: (function_declarator
                declarator: (identifier) @name)
  ) @declaration.function


;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @declaration.struct

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @declaration.enum

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @declaration.union
