---
id: repoguide-vhu.4
status: closed
deps: [repoguide-vhu.2, repoguide-vhu.3]
links: []
created: 2026-02-13T07:57:39.286388-08:00
type: task
priority: 2
parent: repoguide-vhu
---
# Replace --language with --langs multi-language filter

Replace -l/--language single-language flag with -l/--langs accepting comma-separated languages (e.g. -l go,ruby). Update discover.Files to accept []string instead of single string. Empty list means all supported languages. Validate each language exists in lang.Languages. Update flagsWithValue map. Files: main.go, internal/discover/discover.go, internal/discover/discover_test.go.


