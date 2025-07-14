;; Kotlin structure query
;; Captures class definitions, function definitions, variable definitions, and more

(package_header
  (qualified_identifier) @package.name
  ) @package

(import
  (qualified_identifier) @import.name
  ) @import

;; Class definitions
(class_declaration
  name: (identifier) @definition.class.name) @definition.class

;; Object definitions
(object_declaration
  name: (identifier) @definition.object.name) @definition.object

;; Function definitions
(function_declaration
  name: (identifier) @definition.function.name) @definition.function

;; Property definitions
(property_declaration
  (identifier) @definition.property.name) @definition.property


;; Type alias definitions
(type_alias
  (identifier) @definition.type_alias.name) @definition.type_alias

;; Enum class definitions
(enum_entry
  (identifier) @definition.enum.name) @definition.enum

;; Companion object definitions
(companion_object
  name: (identifier) @definition.companion.name) @definition.companion

;; Constructor definitions
(class_declaration
  name: (identifier) @definition.constructor.name) @definition.constructor

;; TODO
(call_expression

  ) @call.method