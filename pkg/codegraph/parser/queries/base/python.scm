(import_statement
  name: (dotted_name)* @import.name
  name: (aliased_import
          name: (dotted_name)* @import.name
          alias: (identifier) @import.alias
          )*
  ) @import

(import_from_statement
  module_name: (dotted_name) @import.module_name
  name: (dotted_name) @import.name
  )  @import


;; Function definitions
(function_definition
  name: (identifier) @definition.function.name) @definition.function

;; Class definitions
(class_definition
  name: (identifier) @definition.class.name) @definition.class

;; Decorated functions
(decorated_definition
  definition: (function_definition
                name: (identifier) @definition.decorated_function.name)) @definition.decorated_function

;; Variable assignments
(assignment
  left: (identifier) @variable.name) @variable


;; Method definitions (inside classes)
(class_definition
  body: (block
          (function_definition
            name: (identifier) @definition.method.name))) @definition.method

;; Type aliases
(assignment
  left: (identifier) @type.name
  right: (call
           function: (identifier)
           (#eq? @type.name "TypeVar"))) @type

;; Enum definitions (Python 3.4+)
(class_definition
  name: (identifier) @definition.enum.name
  superclasses: (argument_list
                  (identifier) @base
                  (#eq? @base "Enum"))) @definition.enum

;; Dataclass definitions
(decorated_definition
  (decorator
    (expression (identifier) @decorator)
    (#eq? @decorator "dataclass"))
  definition: (class_definition
                name: (identifier) @definition.dataclass.name)) @definition.dataclass

;; Protocol definitions
(class_definition
  name: (identifier) @definition.protocol.name
  superclasses: (argument_list
                  (identifier) @base
                  (#eq? @base "Protocol"))
  ) @definition.protocol

;; function call
(call
  function: (identifier) @call.function.name
  arguments: (argument_list) @call.function.arguments
  ) @call.function


;; method call
(call
  function: (attribute
              object: (identifier) @call.method.owner
              attribute: (identifier) @call.method.name
              )
  arguments: (argument_list) @call.method.arguments
  ) @call.method