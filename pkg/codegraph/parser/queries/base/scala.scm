;; Scala structure query
;; Captures various Scala code structures including:
;; - Class definitions
;; - Object declarations
;; - Trait definitions
;; - Method definitions
;; - Type aliases
;; - Enum declarations

;; Class definitions
(class_definition
  name: (identifier) @definition.class.name) @definition.class

;; Object declarations
(object_definition
  name: (identifier) @definition.object.name) @definition.object

;; Trait definitions
(trait_definition
  name: (identifier) @definition.trait.name) @definition.trait

;; Method definitions
(function_definition
  name: (identifier) @definition.method.name) @definition.method

;; Type alias definitions
(type_definition
  name: (identifier) @definition.type.name) @definition.type

;; Enum definitions (Scala 3)
(enum_definition
  name: (identifier) @definition.enum.name) @definition.enum

;; Value definitions (val)
(val_definition
  pattern: (identifier) @constant.name) @constant

;; Variable definitions (var)
(var_definition
  pattern: (identifier) @variable.name) @variable

;; Package object definitions
(package_object
  name: (identifier) @package_object.name) @package_object