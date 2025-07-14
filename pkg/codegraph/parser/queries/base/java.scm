(package_declaration
  (scoped_identifier) @package.name
  ) @package

(import_declaration
  (scoped_identifier
    name: (identifier)
    ) @import.name
  ) @import


;; Class declarations
(class_declaration
  name: (identifier) @name) @definition.class

;; Interface declarations
(interface_declaration
  name: (identifier) @name) @definition.interface


;; Method declarations
(method_declaration
  name: (identifier) @definition.method.name
  parameters: (formal_parameters) @definition.method.parameters
  ) @definition.method

;; Constructor declarations
(constructor_declaration
  name: (identifier) @definition.constructor.name
  parameters: (formal_parameters) @definition.constructor.parameters

  ) @definition.constructor


;; Enum declarations
(enum_declaration
  name: (identifier) @name) @definition.enum

;; Field declarations
(field_declaration
  declarator: (variable_declarator
                name: (identifier) @name)) @definition.field

;; Constant field declarations (static final)
(field_declaration
  (modifiers
    "static"
    "final")
  (variable_declarator
    name: (identifier) @constant.name)) @constant

;; Enum constants
(enum_constant
  name: (identifier) @enum_constant.name) @enum_constant

;; Type parameters
(type_parameters
  (type_parameter) @type_parameters.name) @type_parameters

;; Annotation declarations
(annotation_type_declaration
  name: (identifier) @definition.annotation.name) @definition.annotation

;; 注解调用
(marker_annotation
  name: (identifier) @annotation.name
  ) @annotation

;; 局部变量
(local_variable_declaration
  declarator: (variable_declarator
                name: (identifier) @local_variable.name
                )
  ) @local_variable

;; 方法调用
(method_invocation
  object: (_) @call.method.owner
  name: (identifier) @call.method.name
  arguments: (argument_list) @call.method.arguments
  ) @call.method