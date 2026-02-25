---
id: repoguide-bpa.1
status: closed
deps: []
links: []
created: 2026-02-13T07:25:02.146163-08:00
type: task
priority: 2
parent: repoguide-bpa
---
# Define core data model (internal/model)

Create internal/model/model.go with: TagKind/SymbolKind as typed string constants (def/ref, class/function/method/module). Tag struct (Name, Kind, SymbolKind, Line, File string, Signature). FileInfo struct (Path, Language string, Tags []Tag, Rank float64). Dependency struct (Source, Target string, Symbols []string). RepoMap struct (RepoName, Root string, Files []FileInfo, Dependencies []Dependency). All paths as string (Go convention). No logic, just types.


