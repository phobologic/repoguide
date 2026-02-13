// Package lang provides a language registry mapping file extensions to
// tree-sitter languages and their embedded query files.
package lang

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/phobologic/repoguide/internal/model"
)

//go:embed queries/*.scm
var queryFS embed.FS

var whitespaceRe = regexp.MustCompile(`\s+`)

// Language holds tree-sitter configuration for a supported language.
type Language struct {
	Name       string
	Extensions []string
	lang       *sitter.Language
	queryOnce  sync.Once
	query      *sitter.Query
	queryErr   error

	// FindMethodClass returns the enclosing class name if a @definition.function
	// is actually a method (Python/Ruby style). Returns "" if not a method.
	FindMethodClass func(node *sitter.Node, source []byte) string

	// FindReceiverType returns the receiver type name for a @definition.method
	// node (Go style). Returns "" if not applicable.
	FindReceiverType func(node *sitter.Node, source []byte) string

	// ExtractSignature returns a signature string for a definition node.
	ExtractSignature func(node *sitter.Node, kind model.SymbolKind, source []byte) string
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
// Populated by init() functions in per-language files.
var Languages = map[string]*Language{}

// extensionMap is built lazily after all init() functions have run.
var extensionMap map[string]string
var extensionOnce sync.Once

func getExtensionMap() map[string]string {
	extensionOnce.Do(func() {
		extensionMap = make(map[string]string)
		for _, l := range Languages {
			for _, ext := range l.Extensions {
				extensionMap[ext] = l.Name
			}
		}
	})
	return extensionMap
}

// ForExtension returns the language name for a file extension, or "" if unsupported.
func ForExtension(ext string) string {
	return getExtensionMap()[ext]
}

// NodeText returns the source text of a tree-sitter node.
func NodeText(node *sitter.Node, source []byte) string {
	return string(source[node.StartByte():node.EndByte()])
}

// CollapseWhitespace replaces runs of whitespace with a single space and trims.
func CollapseWhitespace(s string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(s, " "))
}
