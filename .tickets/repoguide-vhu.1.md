---
id: repoguide-vhu.1
status: closed
deps: []
links: []
created: 2026-02-13T07:57:31.721488-08:00
type: task
priority: 1
parent: repoguide-vhu
---
# Refactor parse.go to be language-agnostic

Move Python-specific AST logic out of parse.go into lang/python.go. Add function fields to Language struct: FindMethodClass, FindReceiverType, ExtractSignature. Add definition.method capture to captureMap. Change ExtractTags to accept *lang.Language. Move nodeText and collapseWhitespace helpers to lang package. Files: internal/lang/lang.go, internal/lang/python.go (new), internal/parse/parse.go, main.go.


