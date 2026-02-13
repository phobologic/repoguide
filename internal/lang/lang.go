// Package lang provides a language registry mapping file extensions to
// tree-sitter languages and their embedded query files.
package lang

import (
	"embed"
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

//go:embed queries/*.scm
var queryFS embed.FS

// Language holds tree-sitter configuration for a supported language.
type Language struct {
	Name       string
	Extensions []string
	lang       *sitter.Language
	queryOnce  sync.Once
	query      *sitter.Query
	queryErr   error
}

// GetLanguage returns the tree-sitter Language pointer.
func (l *Language) GetLanguage() *sitter.Language {
	return l.lang
}

// NewParser creates a fresh tree-sitter parser for this language.
// Each goroutine must use its own parser (not thread-safe).
func (l *Language) NewParser() *sitter.Parser {
	p := sitter.NewParser()
	p.SetLanguage(l.lang)
	return p
}

// GetTagQuery returns the compiled tree-sitter query (safe to share across goroutines).
func (l *Language) GetTagQuery() (*sitter.Query, error) {
	l.queryOnce.Do(func() {
		data, err := queryFS.ReadFile(fmt.Sprintf("queries/%s.scm", l.Name))
		if err != nil {
			l.queryErr = fmt.Errorf("reading query file: %w", err)
			return
		}
		q, err := sitter.NewQuery(data, l.lang)
		if err != nil {
			l.queryErr = fmt.Errorf("compiling query: %w", err)
			return
		}
		l.query = q
	})
	return l.query, l.queryErr
}

// Languages maps language names to their configuration.
var Languages = map[string]*Language{
	"python": {
		Name:       "python",
		Extensions: []string{".py"},
		lang:       python.GetLanguage(),
	},
}

// extensionMap maps file extensions to language names.
var extensionMap = buildExtensionMap()

func buildExtensionMap() map[string]string {
	m := make(map[string]string)
	for _, l := range Languages {
		for _, ext := range l.Extensions {
			m[ext] = l.Name
		}
	}
	return m
}

// ForExtension returns the language name for a file extension, or "" if unsupported.
func ForExtension(ext string) string {
	return extensionMap[ext]
}
