;; ------------------------- import/package-------------------------
(package_declaration
  (scoped_identifier) @package.name
  ) @package

(import_declaration
  (scoped_identifier
    name: (identifier)
    ) @import.name
  ) @import

;; -------------------------Class declarations-------------------------

;; Class declarations
(class_declaration
  (modifiers)? @definition.class.modifiers
  name: (identifier) @definition.class.name
  (superclass (_) @definition.class.extends)?
  (super_interfaces
    (type_list) @definition.class.implements
  )?
) @definition.class


;; Enum declarations -> class
(enum_declaration
  (modifiers)? @definition.enum.modifiers
  name: (identifier) @definition.enum.name
  (super_interfaces
    (type_list) @definition.enum.implements
  )?
) @definition.enum

;; --------------------------------Interface declarations--------------------------------
;; Interface declarations
(interface_declaration
  (modifiers)? @definition.interface.modifiers
  name: (identifier) @definition.interface.name
  (extends_interfaces
    (type_list) @definition.interface.extends
  )?
) @definition.interface


;; ---------------------------------method declaration---------------------------------
(method_declaration
  (modifiers)? @definition.method.modifier
  type: (_) @definition.method.return_type
  name: (identifier) @definition.method.name
  parameters: (formal_parameters) @definition.method.parameters
) @definition.method

;; Constructor declarations
(constructor_declaration
  name: (identifier) @definition.constructor.name
  parameters: (formal_parameters) @definition.constructor.parameters
  ) @definition.constructor



;; --------------------------------Field/Variable declaration--------------------------------
;; enum_constant declarations -> field
(enum_constant
  name: (identifier) @definition.enum.constant.name
  )@definition.enum.constant

;; Field declarations
;; private int adminId = -1, moderatorId;
(field_declaration
  type: (_) @definition.field.type
  declarator: (variable_declarator
    name: (identifier) @definition.field.name
    value: (_)? @definition.field.value
  )
) @definition.field

;; 局部变量
(local_variable_declaration
  type: (_) @local_variable.type
  declarator: (variable_declarator
    name: (identifier) @local_variable.name
    value: (_)? @local_variable.value
  )
) @local_variable

;; ----------------------------------------other---------------------------------------
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


;; -------------------------------- Initializer/Assignment expression --------------------------------
;; 方法调用
(method_invocation
  object: (_) @call.method.owner
  name: (identifier) @call.method.name
  arguments: (argument_list) @call.method.arguments
  ) @call.method