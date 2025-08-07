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
;;(constructor_declaration
;;  name: (identifier) @definition.constructor.name
;;  parameters: (formal_parameters) @definition.constructor.parameters
;;  ) @definition.constructor



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


;; -------------------------------- Initializer/Assignment expression --------------------------------
;; 方法调用
(method_invocation
  object: (_)? @call.method.owner
  name: (identifier) @call.method.name
  arguments: (argument_list) @call.method.arguments
  ) @call.method


;; Class<java.util.List> clazz2 = java.util.List.class;
(class_literal
  [
    (type_identifier)
    (scoped_type_identifier)
  ] @call.class_literal.type
) @call.class_literal

(class_literal
  (array_type
    element: [
      (type_identifier)
      (scoped_type_identifier)
    ]
  ) @call.class_literal.type
) @call.class_literal


;;  (Object) a
(cast_expression
  type: [
    (type_identifier)
    (scoped_type_identifier)
    (generic_type)
  ] @call.cast.type
) @call.cast

(cast_expression
  type: (array_type
    [  ;; 过滤基础类型，但是没有过滤基础类型类
      (type_identifier)
      (scoped_type_identifier)
      (generic_type)
    ] @call.cast.type)
) @call.cast

;; a instanceof Parent
(instanceof_expression
  right: [
    (scoped_type_identifier)
    (type_identifier)
  ] @call.instanceof.type
) @call.instanceof

(instanceof_expression
  right: (array_type
    element: [
      (type_identifier)
      (scoped_type_identifier)
    ]
  ) @call.instanceof.type
) @call.instanceof

;; new Child()
(object_creation_expression
  type: [
    (scoped_type_identifier)
    (generic_type)
    (type_identifier)
  ] @call.new.type
  arguments: (argument_list) @call.new.args
) @call.new

;; new Dog[3]
(array_creation_expression
  type: [
    (type_identifier)
    (scoped_type_identifier)
  ] @call.new_array.type
) @call.new_array






