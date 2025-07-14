(function_declaration
  name: (identifier) @name) @declaration.function

(method_declaration
  name: (field_identifier) @name) @declaration.method

(type_declaration (type_spec name: (type_identifier) @name type: (interface_type))) @declaration.interface

(type_declaration (type_spec name: (type_identifier) @name type: (struct_type))) @declaration.struct

(type_declaration (type_spec name: (type_identifier) @name type: (type_identifier))) @declaration.type_alias

(source_file (var_declaration (var_spec name: (identifier) @name)) @global_variable )

(source_file (const_declaration (const_spec name: (identifier) @name)) @declaration.const)

