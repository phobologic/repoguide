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

;; Method calls
(call
  method: (identifier) @name) @reference.call
