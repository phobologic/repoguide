---
id: repoguide-vhu.2
status: closed
deps: [repoguide-vhu.1]
links: []
created: 2026-02-13T07:57:34.495769-08:00
type: task
priority: 1
parent: repoguide-vhu
---
# Add Go language support

Register Go in language registry. Create tree-sitter query (internal/lang/queries/go.scm) for type defs, function/method declarations, calls, imports. Implement Go-specific FindReceiverType (extract receiver type from method_declaration) and ExtractSignature (func/method: name(params) result, type: Name). Add parse tests for Go: function, method with receiver, type definition, call, import. Files: internal/lang/golang.go (new), internal/lang/queries/go.scm (new), internal/parse/parse_test.go, internal/lang/lang_test.go, go.mod.


