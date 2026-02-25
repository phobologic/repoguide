---
id: repoguide-bpa.8
status: closed
deps: [repoguide-bpa.2, repoguide-bpa.3, repoguide-bpa.5, repoguide-bpa.6, repoguide-bpa.7]
links: []
created: 2026-02-13T07:25:35.928201-08:00
type: task
priority: 1
parent: repoguide-bpa
---
# Wire up CLI and concurrent pipeline (main.go)

Port cli.py to Go. Stdlib flag with aliases: -n/--max-files, -l/--language, --cache, --max-file-size, -V/--version. Version via ldflags. Pipeline: flags→discover→cache check (mtime)→size filter→parse (concurrent)→build graph→rank→select→encode→cache write→print stdout. Concurrent parsing: worker pool sized to GOMAXPROCS, each worker creates own sitter.Parser, fan out via channel. Warnings to stderr. Refactor run() to accept io.Writer+[]string for testability. Integration tests: default root, max-files, empty repo, language filter, cache create/reuse/invalidate, max-file-size.


