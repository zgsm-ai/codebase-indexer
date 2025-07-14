;; C++ structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Function definitions
(function_definition
  declarator: (function_declarator
                declarator: (identifier) @name)) @definition.function

;; Class declarations
(class_specifier
  name: (type_identifier) @name) @definition.class

;; Struct declarations
(struct_specifier
  name: (type_identifier) @name) @definition.struct



;; Variable declarations (keep as declaration)
(declaration
  declarator: (init_declarator
                declarator: (identifier) @name)) @variable

;; Member variable declarations (keep as declaration)
(field_declaration
  declarator: (field_identifier) @name) @declaration.field

;; Union declarations
(union_specifier
  name: (type_identifier) @name) @definition.union

;; Enum declarations
(enum_specifier
  name: (type_identifier) @name) @definition.enum

;; Type alias declarations (these are definitions)
(alias_declaration
  name: (type_identifier) @name) @definition.type_alias

;; Typedef declarations
(type_definition
  declarator: (type_identifier) @name) @definition.typedef