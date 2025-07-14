;; JavaScript structure query
;; Captures function definitions, class definitions, variable declarations, and more

;; Function declarations
(function_declaration
  name: (identifier) @name) @declaration.function

;; Function expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (function_expression))) @declaration.function_expression

;; Arrow functions
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (arrow_function))) @declaration.arrow_function

;; Class declarations
(class_declaration
  name: (identifier) @name) @declaration.class

;; Class expressions
(variable_declaration
  (variable_declarator
    name: (identifier) @name
    value: (class))) @declaration.class

;; Method definitions (inside classes)
(method_definition
  name: (property_identifier) @name) @definition.method

(program
  (_
    (variable_declarator) @global_variable
    )
  )

;; Object properties
(pair
  key: (property_identifier) @name) @declaration.property

;; Export declarations
(export_statement
  declaration: (function_declaration
                 name: (identifier) @name)) @declaration.export_function

;; Export named declarations
(export_statement
  (export_clause
    (export_specifier
      name: (identifier) @name))) @declaration.export_statement