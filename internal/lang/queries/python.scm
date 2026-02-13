;; Class definitions
(class_definition
  name: (identifier) @name) @definition.class

;; Function/method definitions
(function_definition
  name: (identifier) @name) @definition.function

;; Function and method calls
(call
  function: [
    (identifier) @name
    (attribute
      attribute: (identifier) @name)
  ]) @reference.call

;; Import references: from x import y
(import_from_statement
  name: (dotted_name
    (identifier) @name)) @reference.import

;; Import references: import x
(import_statement
  name: (dotted_name
    (identifier) @name)) @reference.import
