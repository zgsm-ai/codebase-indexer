;; Ruby structure query
;; Captures method definitions, class definitions, module definitions, and more

;; Method definitions
(method
  name: (identifier) @name) @method

;; Class definitions
(class
  name: (constant) @name) @class

;; Module definitions
(module
  name: (constant) @name) @module

;; Singleton method definitions
(singleton_method
  name: (identifier) @name) @singleton_method


;; Constant assignments
(assignment
  left: (constant) @name) @constant

;; Constant assignments (convention: uppercase names)
(assignment
  left: (identifier) @name
  (#match? @name "^[A-Z][A-Z0-9_]*$")) @constant

;; Module methods
(module
  body: (body_statement
          (method
            name: (identifier) @name))) @method

;; Class methods
(class
  body: (body_statement
          (singleton_method
            name: (identifier) @name))) @method

;; Instance methods
(class
  body: (body_statement
          (method
            name: (identifier) @name))) @method

;; Attribute accessors
(call
  method: (identifier) @accessor
  (#match? @accessor "^(attr_reader|attr_writer|attr_accessor)$")
  arguments: (argument_list
               (simple_symbol) @name)) @attribute