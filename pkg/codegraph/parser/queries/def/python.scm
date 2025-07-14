(import_statement ) @import @name

(import_from_statement )  @import_from @name
;; Python structure query
;; Captures function definitions, class definitions, variable declarations, and more

;; Function definitions
(function_definition
  name: (identifier) @name) @definition.function

;; Class definitions
(class_definition
  name: (identifier) @name) @definition.class

;; Decorated functions
(decorated_definition
  definition: (function_definition
                name: (identifier) @name)) @definition.decorated_function

;; Variable assignments
(assignment
  left: (identifier) @name) @variable

;; Constant assignments (uppercase)
(assignment
  left: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @constant

;; Method definitions (inside classes)
(class_definition
  body: (block
          (function_definition
            name: (identifier) @name))) @definition.method

;; Type aliases
(assignment
  left: (identifier) @name
  right: (call
           function: (identifier)
           (#eq? @name "TypeVar"))) @type

;; Enum definitions (Python 3.4+)
(class_definition
  name: (identifier) @name
  superclasses: (argument_list
                  (identifier) @base
                  (#eq? @base "Enum"))) @definition.enum

;; Dataclass definitions
(decorated_definition
  (decorator
    (expression (identifier) @decorator)
    (#eq? @decorator "dataclass"))
  definition: (class_definition
                name: (identifier) @name)) @definition.dataclass

;; Protocol definitions
(class_definition
  name: (identifier) @name
  superclasses: (argument_list
                  (identifier) @base
                  (#eq? @base "Protocol"))
  ) @definition.protocol