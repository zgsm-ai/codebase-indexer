;; Kotlin structure query
;; Captures class definitions, function definitions, variable declarations, and more

;; Class declarations
(class_declaration
  name: (identifier) @name) @declaration.class

;; Object declarations
(object_declaration
  name: (identifier) @name) @declaration.object

;; Function declarations
(function_declaration
  name: (identifier) @name) @declaration.function

;; Property declarations
(property_declaration
   (identifier) @name) @declaration.property


;; Type alias declarations
(type_alias
   (identifier) @name) @declaration.type_alias

;; Enum class declarations
(enum_entry
  (identifier) @name) @declaration.enum

;; Companion object declarations
(companion_object
    name: (identifier) @name) @declaration.companion

;; Constructor declarations
(class_declaration
    name: (identifier) @name) @declaration.constructor