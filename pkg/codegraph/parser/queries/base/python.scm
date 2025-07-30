
;; ------------------------------------import------------------------------------
(import_statement
  name: (dotted_name) @import.name
)@import

(import_statement
  name: (aliased_import
          name: (dotted_name) @import.name
          alias: (identifier) @import.alias
        )
) @import

;; Python import_from_statement 语义捕获模式
(import_from_statement
  module_name: [
    (dotted_name) @import.source
    (relative_import) @import.source
  ]?
  [
    ;; 处理通配符导入
    (wildcard_import) @import.name
    ;; 处理普通导入列表
    name: [
      ;; 处理带别名的导入
      (aliased_import
        name: (dotted_name) @import.name
        alias: (identifier) @import.alias)
      ;; 处理不带别名的导入  
      (dotted_name) @import.name
      ;; 处理标识符导入
      (identifier) @import.name
    ]
  ]
) @import

;;------------------------------------function------------------------------------

(function_definition
  name: (identifier) @definition.function.name
  parameters: (parameters) @definition.function.parameters
  return_type: (type)? @definition.function.return_type
)@definition.function


;; -----------------------------------class-----------------------------------
(class_definition
  name: (identifier) @definition.class.name
  superclasses: (argument_list)? @definition.class.extends
)@definition.class


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