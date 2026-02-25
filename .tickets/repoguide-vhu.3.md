---
id: repoguide-vhu.3
status: closed
deps: [repoguide-vhu.1]
links: []
created: 2026-02-13T07:57:37.107465-08:00
type: task
priority: 1
parent: repoguide-vhu
---
# Add Ruby language support

Register Ruby in language registry. Create tree-sitter query (internal/lang/queries/ruby.scm) for class/module defs, method/singleton_method defs, calls. Implement Ruby-specific FindMethodClass (walk parent chain for class/module) and ExtractSignature (method: name(params), class: Name < Super). Add parse tests for Ruby: class, module, method, class method, call, method in class. Files: internal/lang/ruby.go (new), internal/lang/queries/ruby.scm (new), internal/parse/parse_test.go, internal/lang/lang_test.go, go.mod.


