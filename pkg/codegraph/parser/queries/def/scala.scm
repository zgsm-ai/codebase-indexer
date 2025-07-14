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
  name: (identifier) @name) @definition.class

;; Object declarations
(object_definition
  name: (identifier) @name) @definition.object

;; Trait definitions
(trait_definition
  name: (identifier) @name) @definition.trait

;; Method definitions
(function_definition
  name: (identifier) @name) @definition.method

;; Type alias definitions
(type_definition
  name: (identifier) @name) @definition.type

;; Enum definitions (Scala 3)
(enum_definition
  name: (identifier) @name) @definition.enum

;; Value definitions (val)
(val_definition
  pattern: (identifier) @name) @definition.val

;; Variable definitions (var)
(var_definition
  pattern: (identifier) @name) @definition.var

;; Package object definitions
(package_object
  name: (identifier) @name) @package_object