
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
;; 枚举和类不区分



;; ------------------------------------method-----------------------------------
;; 带装饰器
(class_definition
  body: (block
    (decorated_definition
      (decorator) 
      definition: (function_definition
        name: (identifier) @definition.method.name
        parameters: (parameters) @definition.method.parameters
        return_type: (type)? @definition.method.return_type
      )@definition.method
    )
  )
) 

;; 无装饰器
(class_definition
  body: (block
    (function_definition
      name: (identifier) @definition.method.name
      parameters: (parameters) @definition.method.parameters
      return_type: (type)? @definition.method.return_type
    )@definition.method
  )
) 

;; ----------------------------------Variable-------------------------------------
;; Variable assignments
(assignment
  left: (identifier) @variable.name
  type: (type) @variable.type
 )@variable


;; ---------------------------------Call---------------------------------
(call
  function: (attribute
              object: (identifier) 
              attribute: (identifier) @call.function.name
            )
  arguments: (argument_list) @call.function.arguments
) @call.function

;; 对象方法调用，object 是 attribute
(call
  function: (attribute
              object: (attribute) 
              attribute: (identifier) @call.function.name
            )
  arguments: (argument_list) @call.function.arguments
) @call.function

;; 对象方法调用，object 是 call
(call
  function: (attribute
              object: (call) 
              attribute: (identifier) @call.function.name
            )
  arguments: (argument_list) @call.function.arguments
) @call.function

;; 下标调用
(call
  function: (subscript
              value: (identifier) @call.function.name
              subscript: (identifier) 
            )
  arguments: (argument_list) @call.function.arguments
) @call.function

;; 普通函数调用
(call
  function: (identifier) @call.function.name
  arguments: (argument_list) @call.function.arguments
) @call.function


;; Type aliases
(assignment
  left: (identifier) @type.name
  right: (call
           function: (identifier)
           (#eq? @type.name "TypeVar"))) @type








