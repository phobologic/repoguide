---
id: repoguide-bpa.4
status: closed
deps: [repoguide-bpa.1]
links: []
created: 2026-02-13T07:25:17.470555-08:00
type: task
priority: 2
parent: repoguide-bpa
---
# Implement language registry (internal/lang)

Port languages.py to Go. Language struct: Name, Extensions, tsLang *sitter.Language, query *sitter.Query. Registry maps: name→Language, ext→name. go:embed queries/*.scm for query files. Query compiled once at init (safe to share). NewParser() creates fresh parser per call (NOT thread-safe). ForExtension(ext) and Get(name) lookups. Copy python.scm verbatim from sourcecrumb. Deps: smacker/go-tree-sitter, smacker/go-tree-sitter/python.


