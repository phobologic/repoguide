---
id: repoguide-bpa.5
status: closed
deps: [repoguide-bpa.4]
links: []
created: 2026-02-13T07:25:18.64356-08:00
type: task
priority: 2
parent: repoguide-bpa
---
# Implement file discovery (internal/discover)

Port discovery.py to Go. DiscoverFiles(root string, opts ...Option) ([]FileEntry, error). Git strategy: exec.CommandContext git ls-files --cached --others --exclude-standard with 10s timeout. Fallback: filepath.WalkDir + go-gitignore. SKIP_DIRS as map[string]struct{}. Options: WithLanguage(lang), WithExtraIgnores(patterns). Depends on lang package for extension mapping. Sort results by path. Tests: temp dirs with Python files, gitignore, skip dirs, symlinks, language filter. Dep: sabhiram/go-gitignore.


