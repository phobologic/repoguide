;; Class-like definitions
(class_declaration
  name: (type_identifier) @name) @definition.class

(abstract_class_declaration
  name: (type_identifier) @name) @definition.class

(interface_declaration
  name: (type_identifier) @name) @definition.class

(type_alias_declaration
  name: (type_identifier) @name) @definition.class

(enum_declaration
  name: (identifier) @name) @definition.class

;; Class fields
(public_field_definition
  name: [
    (property_identifier) @name
    (private_property_identifier) @name
  ]) @definition.field

;; Interface and object-type fields
(property_signature
  name: [
    (property_identifier) @name
    (private_property_identifier) @name
  ]) @definition.field

;; Class methods
(method_definition
  name: [
    (property_identifier) @name
    (private_property_identifier) @name
  ]) @definition.method

(abstract_method_signature
  name: [
    (property_identifier) @name
    (private_property_identifier) @name
  ]) @definition.method

;; Interface methods
(method_signature
  name: [
    (property_identifier) @name
    (private_property_identifier) @name
  ]) @definition.method

;; Function declarations
(function_declaration
  name: (identifier) @name) @definition.function

(generator_function_declaration
  name: (identifier) @name) @definition.function

;; Import references
(import_statement
  (import_clause
    (identifier) @name)) @reference.import

(import_statement
  (import_clause
    (named_imports
      (import_specifier
        name: (identifier) @name)))) @reference.import

;; Re-export references
(export_statement
  (export_clause
    (export_specifier
      name: (identifier) @name))
  source: (string)) @reference.import

(import_statement
  (import_clause
    (namespace_import
      (identifier) @name))) @reference.import

(import_statement
  (import_require_clause
    (identifier) @name)) @reference.import

;; Function and method calls
(call_expression
  function: [
    (identifier) @name
    (member_expression
      property: [
        (property_identifier) @name
        (private_property_identifier) @name
      ])
    (instantiation_expression
      function: (identifier) @name)
    (instantiation_expression
      function: (member_expression
        property: [
          (property_identifier) @name
          (private_property_identifier) @name
        ]))
  ]) @reference.call
