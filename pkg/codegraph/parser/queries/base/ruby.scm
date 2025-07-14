;; Ruby structure query
;; Captures method definitions, class definitions, module definitions, and more

;; Method definitions
(method
  name: (identifier) @definition.method.name) @definition.method

;; Class definitions
(class
  name: (constant) @definition.class.name) @definition.class

;; Module definitions
(module
  name: (constant) @module.name) @module

;; Singleton method definitions
(singleton_method
  name: (identifier) @definition.singleton_method.name) @definition.singleton_method


;; Constant assignments
(assignment
  left: (constant) @constant.name) @constant

;; Constant assignments (convention: uppercase names)
(assignment
  left: (identifier) @constant.name
  (#match? @constant.name "^[A-Z][A-Z0-9_]*$")) @constant

;; Module methods
(module
  body: (body_statement
          (method
            name: (identifier) @definition.method.name))) @definition.method

;; Class methods
(class
  body: (body_statement
          (singleton_method
            name: (identifier) @definition.method.name))) @definition.method

;; Instance methods
(class
  body: (body_statement
          (method
            name: (identifier) @name))) @definition.method

;; Attribute accessors
(call
  method: (identifier) @accessor
  (#match? @accessor "^(attr_reader|attr_writer|attr_accessor)$")
  arguments: (argument_list
               (simple_symbol) @definition.attribute.name)) @definition.attribute

;; method call
(call
  method: (identifier) @call.method.name
  arguments: (argument_list) @call.method.arguments
  ) @call.method

