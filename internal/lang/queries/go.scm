;; Type definitions (struct, interface, type alias)
(type_spec
  name: (type_identifier) @name) @definition.class

;; Struct fields
(type_spec
  (struct_type
    (field_declaration_list
      (field_declaration
        name: (field_identifier) @name) @definition.field)))

;; Interface methods
(type_spec
  (interface_type
    (method_elem
      name: (field_identifier) @name) @definition.field))

;; Function declarations
(function_declaration
  name: (identifier) @name) @definition.function

;; Method declarations (with receiver)
(method_declaration
  name: (field_identifier) @name) @definition.method

;; Function and method calls
(call_expression
  function: [
    (identifier) @name
    (selector_expression
      field: (field_identifier) @name)
  ]) @reference.call

;; Import paths
(import_spec
  path: (interpreted_string_literal) @name) @reference.import
