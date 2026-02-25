---
id: repoguide-bpa.6
status: closed
deps: [repoguide-bpa.4]
links: []
created: 2026-02-13T07:25:22.842731-08:00
type: task
priority: 1
parent: repoguide-bpa
---
# Implement tree-sitter parsing (internal/parse)

Port parsing.py to Go. ExtractTags(absPath, relPath string, language *lang.Language) ([]Tag, error). Capture map: definition.class, definition.function, reference.call, reference.import → (TagKind, SymbolKind). Method detection via parent node traversal (func→block→class_definition, or func→decorated_definition→block→class_definition). Class name extraction for method prefixing. Signature extraction for classes (name+args) and functions (name+params+return). Key diff: node.Content(source) requires []byte threading. Tests with real tree-sitter parsing of temp Python files. This is the most complex module.


