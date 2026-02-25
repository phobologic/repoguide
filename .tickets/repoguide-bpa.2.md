---
id: repoguide-bpa.2
status: closed
deps: [repoguide-bpa.1]
links: []
created: 2026-02-13T07:25:06.194588-08:00
type: task
priority: 2
parent: repoguide-bpa
---
# Implement TOON encoder (internal/toon)

Port toon.py to Go. Encode(RepoMap) string as main entry. encodeValue with quoting rules: regexp.MustCompile for needsQuoting/looksNumeric, keyword set (true/false/null). formatTabular for array sections. strings.Builder for perf. Table-driven tests for: plain string, comma, colon, number, empty, keywords, whitespace, quotes, dash prefix, newlines, tabs. Integration test encoding a full RepoMap.


