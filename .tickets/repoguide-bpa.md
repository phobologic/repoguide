---
id: repoguide-bpa
status: closed
deps: []
links: []
created: 2026-02-13T07:24:53.927016-08:00
type: epic
priority: 1
---
# Port sourcecrumb to Go (repoguide)

Rewrite sourcecrumb (Python tree-sitter repo mapper) in Go for performance. Same pipeline: discover → parse → graph → PageRank → select → TOON encode. Key wins: parallel parsing, no GIL, lower overhead. Deps: smacker/go-tree-sitter, go-gitignore. Stdlib flag for CLI, hand-rolled PageRank.


