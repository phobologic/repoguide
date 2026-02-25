---
id: repoguide-bpa.3
status: closed
deps: [repoguide-bpa.1]
links: []
created: 2026-02-13T07:25:09.057764-08:00
type: task
priority: 2
parent: repoguide-bpa
---
# Implement file selection (internal/ranking)

Port ranking.py select_files to Go. SelectFiles(rm RepoMap, maxFiles int) (RepoMap, error). Returns new RepoMap with top maxFiles entries. Filters dependencies to only include edges between selected files. Error on maxFiles < 1. Tests: limits count, no limit returns all, filters deps, max > total, zero/negative errors.


