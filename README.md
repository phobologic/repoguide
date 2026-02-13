# repoguide

A fast, tree-sitter-based repository mapper that generates compact
[TOON](https://github.com/nicois/toon)-format output for LLM context windows.

repoguide analyzes a codebase to extract symbols (classes, functions, methods),
build a dependency graph between files, rank files by importance using PageRank,
and output a structured summary — all designed to fit efficiently in an LLM's
context window.

## Installation

Requires Go 1.25+ and a C compiler (for tree-sitter CGo bindings).

```bash
go install github.com/phobologic/repoguide@latest
```

Or build from source:

```bash
git clone https://github.com/phobologic/repoguide.git
cd repoguide
go build -o repoguide .
```

Set the version at build time:

```bash
go build -ldflags "-X main.version=1.0.0" -o repoguide .
```

## Usage

```
repoguide [FLAGS] [ROOT]
```

`ROOT` defaults to the current directory.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--max-files N` | `-n` | Maximum number of files to include in output |
| `--language NAME` | `-l` | Restrict to a specific language (e.g., `python`) |
| `--cache PATH` | | Cache file; reuse if newer than all source files |
| `--max-file-size N` | | Skip files larger than N bytes (default: 1MB) |
| `--version` | `-V` | Show version and exit |

### Examples

Map the current directory:

```bash
repoguide .
```

Top 10 most important files:

```bash
repoguide -n 10 /path/to/project
```

With caching (skips re-analysis if no files changed):

```bash
repoguide --cache /tmp/project.cache /path/to/project
```

## Output Format

repoguide outputs [TOON](https://github.com/nicois/toon) (Token-Oriented
Object Notation), a compact format optimized for LLM context windows.

```toon
repo: myproject
root: myproject
files[3]{path,language,rank}:
  myproject/models.py,python,0.2615
  myproject/utils.py,python,0.1155
  myproject/main.py,python,0.0590
symbols[5]{file,name,kind,line,signature}:
  myproject/models.py,User,class,10,User(Base)
  myproject/models.py,User.__init__,method,12,__init__(self, name: str) -> None
  myproject/utils.py,helper,function,1,helper(x: int) -> str
  myproject/main.py,main,function,5,main() -> None
  myproject/main.py,run,function,10,run(config: Config)
dependencies[1]{source,target,symbols}:
  myproject/main.py,myproject/models.py,User
```

### Sections

- **files** — Discovered files ranked by PageRank importance (most important first)
- **symbols** — All definitions (classes, functions, methods) with signatures and line numbers
- **dependencies** — Cross-file edges showing which files reference symbols from other files

## How It Works

1. **Discover** — Finds source files via `git ls-files` (falls back to directory walk with `.gitignore` support)
2. **Parse** — Extracts symbols using tree-sitter queries (classes, functions, methods, imports, calls)
3. **Graph** — Builds a dependency graph from cross-file symbol references
4. **Rank** — Applies PageRank to identify the most important files
5. **Select** — Takes the top N files (if `--max-files` is set)
6. **Encode** — Outputs the result in TOON format

Parsing runs concurrently across all available CPU cores for fast analysis of
large codebases.

## Supported Languages

| Language | Extensions |
|----------|------------|
| Python | `.py` |

Additional languages can be added by providing a tree-sitter grammar and a
`.scm` query file in `internal/lang/queries/`.

## Development

```bash
go test ./...              # Run all tests
go build -o repoguide .   # Build binary
```

## License

MIT
