// Package discover finds parseable source files in a repository.
package discover

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/phobologic/repoguide/internal/lang"
)

// FileEntry represents a discovered source file.
type FileEntry struct {
	Path     string // Relative to repo root
	Language string
}

var skipDirs = map[string]struct{}{
	"__pycache__":   {},
	"node_modules":  {},
	".git":          {},
	".hg":           {},
	".svn":          {},
	"venv":          {},
	".venv":         {},
	"env":           {},
	".env":          {},
	"build":         {},
	"dist":          {},
	".tox":          {},
	".mypy_cache":   {},
	".ruff_cache":   {},
	".pytest_cache": {},
	"egg-info":      {},
}

// Files discovers parseable source files under root.
// If languages is non-empty, only files matching one of the listed languages are returned.
func Files(root string, languages []string) ([]FileEntry, error) {
	langSet := make(map[string]struct{}, len(languages))
	for _, l := range languages {
		langSet[l] = struct{}{}
	}
	gitFiles := gitLsFiles(root)
	var gi *ignore.GitIgnore
	if gitFiles == nil {
		gi = loadGitignore(root)
	}

	var results []FileEntry

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		name := d.Name()

		if d.IsDir() {
			if path == root {
				return nil
			}
			if _, skip := skipDirs[name]; skip || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(name, ".") {
			return nil
		}

		// Skip symlinks
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		if gitFiles != nil {
			if _, ok := gitFiles[rel]; !ok {
				return nil
			}
		} else if gi != nil && gi.MatchesPath(rel) {
			return nil
		}

		ext := filepath.Ext(name)
		langName := lang.ForExtension(ext)
		if langName == "" {
			return nil
		}

		if len(langSet) > 0 {
			if _, ok := langSet[langName]; !ok {
				return nil
			}
		}

		results = append(results, FileEntry{Path: rel, Language: langName})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results, nil
}

func gitLsFiles(root string) map[string]struct{} {
	gitDir := filepath.Join(root, ".git")
	info, err := os.Stat(gitDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	files := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line != "" {
			files[line] = struct{}{}
		}
	}
	return files
}

func loadGitignore(root string) *ignore.GitIgnore {
	path := filepath.Join(root, ".gitignore")
	gi, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return nil
	}
	return gi
}
