;; Class definitions
(class
  name: (constant) @name) @definition.class

;; Module definitions
(module
  name: (constant) @name) @definition.class

;; Method definitions
(method
  name: (identifier) @name) @definition.function

;; Singleton method definitions (class methods like def self.foo)
(singleton_method
  name: (identifier) @name) @definition.function

;; attr_accessor / attr_reader / attr_writer :name
(call
  method: (identifier) @_attr_method
  arguments: (argument_list
    (simple_symbol) @name)) @definition.field
(#match? @_attr_method "^attr_(accessor|reader|writer)$")

;; Method calls
(call
  method: (identifier) @name) @reference.call
