// repoguide generates a tree-sitter repository map in TOON format.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/phobologic/repoguide/internal/discover"
	"github.com/phobologic/repoguide/internal/graph"
	"github.com/phobologic/repoguide/internal/lang"
	"github.com/phobologic/repoguide/internal/model"
	"github.com/phobologic/repoguide/internal/parse"
	"github.com/phobologic/repoguide/internal/ranking"
	"github.com/phobologic/repoguide/internal/toon"
)

var version = "dev"

const defaultMaxFileSize = 1_000_000 // 1 MB

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("repoguide", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		maxFiles    int
		langs       string
		cachePath   string
		maxFileSize int
		showVersion bool
	)

	fs.IntVar(&maxFiles, "n", 0, "maximum number of files to include")
	fs.IntVar(&maxFiles, "max-files", 0, "maximum number of files to include")
	fs.StringVar(&langs, "l", "", "comma-separated languages to include")
	fs.StringVar(&langs, "langs", "", "comma-separated languages to include")
	fs.StringVar(&cachePath, "cache", "", "cache file path")
	fs.IntVar(&maxFileSize, "max-file-size", defaultMaxFileSize, "skip files larger than this many bytes")
	fs.BoolVar(&showVersion, "V", false, "show version and exit")
	fs.BoolVar(&showVersion, "version", false, "show version and exit")

	if err := fs.Parse(reorderArgs(args)); err != nil {
		return err
	}

	if showVersion {
		_, _ = fmt.Fprintf(stdout, "repoguide %s\n", version)
		return nil
	}

	root := "."
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolving root: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("root path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: not a directory", root)
	}

	var langFilter []string
	if langs != "" {
		for _, name := range strings.Split(langs, ",") {
			name = strings.TrimSpace(name)
			if _, ok := lang.Languages[name]; !ok {
				return fmt.Errorf("unsupported language %q", name)
			}
			langFilter = append(langFilter, name)
		}
	}

	// Discover files
	files, err := discover.Files(root, langFilter)
	if err != nil {
		return fmt.Errorf("discovering files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no parseable files found")
	}

	// Check cache freshness
	if cachePath != "" && cacheIsFresh(cachePath, root, files) {
		data, err := os.ReadFile(cachePath)
		if err == nil {
			_, _ = stdout.Write(data)
			return nil
		}
	}

	// Filter by size
	files = filterBySize(root, files, maxFileSize, stderr)
	if len(files) == 0 {
		return fmt.Errorf("no parseable files found (all exceeded size limit)")
	}

	// Parse files concurrently
	fileInfos := parseFilesConcurrent(root, files, stderr)
	if len(fileInfos) == 0 {
		return fmt.Errorf("no files could be parsed")
	}

	// Build graph and rank
	deps := graph.BuildGraph(fileInfos)
	graph.Rank(fileInfos, deps)

	rm := &model.RepoMap{
		RepoName:     filepath.Base(root),
		Root:         filepath.Base(root),
		Files:        fileInfos,
		Dependencies: deps,
	}

	// Select top N files
	if maxFiles > 0 {
		rm = ranking.SelectFiles(rm, maxFiles)
	}

	// Encode to TOON
	output := toon.Encode(rm)

	// Write cache
	if cachePath != "" {
		_ = os.WriteFile(cachePath, []byte(output+"\n"), 0o644)
	}

	_, _ = fmt.Fprintln(stdout, output)
	return nil
}

func cacheIsFresh(cachePath, root string, files []discover.FileEntry) bool {
	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		return false
	}
	cacheMtime := cacheInfo.ModTime()

	for _, f := range files {
		fi, err := os.Stat(filepath.Join(root, f.Path))
		if err != nil {
			return false
		}
		if !fi.ModTime().Before(cacheMtime) {
			return false
		}
	}
	return true
}

func filterBySize(root string, files []discover.FileEntry, maxSize int, stderr io.Writer) []discover.FileEntry {
	var kept []discover.FileEntry
	for _, f := range files {
		fi, err := os.Stat(filepath.Join(root, f.Path))
		if err != nil {
			kept = append(kept, f) // keep if can't stat
			continue
		}
		if fi.Size() > int64(maxSize) {
			_, _ = fmt.Fprintf(stderr, "Warning: %s: skipped (>%d bytes)\n", f.Path, maxSize)
			continue
		}
		kept = append(kept, f)
	}
	return kept
}

func parseFilesConcurrent(root string, files []discover.FileEntry, stderr io.Writer) []model.FileInfo {
	type result struct {
		index int
		info  model.FileInfo
		ok    bool
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	work := make(chan int, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup
	var stderrMu sync.Mutex

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each goroutine gets its own parser
			parsers := make(map[string]*parserPair)

			for idx := range work {
				f := files[idx]
				pp, ok := parsers[f.Language]
				if !ok {
					l := lang.Languages[f.Language]
					q, err := l.GetTagQuery()
					if err != nil {
						stderrMu.Lock()
						_, _ = fmt.Fprintf(stderr, "Warning: failed to compile query for %s: %v\n", f.Language, err)
						stderrMu.Unlock()
						continue
					}
					pp = &parserPair{lang: l, parser: l.NewParser(), query: q}
					parsers[f.Language] = pp
				}

				absPath := filepath.Join(root, f.Path)
				source, err := os.ReadFile(absPath)
				if err != nil {
					stderrMu.Lock()
					_, _ = fmt.Fprintf(stderr, "Warning: failed to parse %s: %v\n", f.Path, err)
					stderrMu.Unlock()
					continue
				}

				tags := parse.ExtractTags(pp.lang, pp.parser, pp.query, source, f.Path)
				results <- result{
					index: idx,
					info: model.FileInfo{
						Path:     f.Path,
						Language: f.Language,
						Tags:     tags,
					},
					ok: true,
				}
			}
		}()
	}

	for i := range files {
		work <- i
	}
	close(work)

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results in original order
	indexed := make([]model.FileInfo, len(files))
	valid := make([]bool, len(files))
	for r := range results {
		indexed[r.index] = r.info
		valid[r.index] = r.ok
	}

	var fileInfos []model.FileInfo
	for i, v := range valid {
		if v {
			fileInfos = append(fileInfos, indexed[i])
		}
	}

	return fileInfos
}

type parserPair struct {
	lang   *lang.Language
	parser *sitter.Parser
	query  *sitter.Query
}

// flagsWithValue lists flags that take a value argument.
var flagsWithValue = map[string]bool{
	"-n": true, "--n": true,
	"-max-files": true, "--max-files": true,
	"-l": true, "--l": true,
	"-langs": true, "--langs": true,
	"-cache": true, "--cache": true,
	"-max-file-size": true, "--max-file-size": true,
}

// reorderArgs moves positional arguments after all flags so Go's flag package
// can parse them correctly (it stops at the first non-flag arg).
func reorderArgs(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(args[i]) > 0 && args[i][0] == '-' {
			flags = append(flags, args[i])
			if flagsWithValue[args[i]] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}
