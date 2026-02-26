// Package model defines core data structures for repoguide.
package model

// TagKind indicates whether a tag is a definition or a reference.
type TagKind string

const (
	Definition TagKind = "def"
	Reference  TagKind = "ref"
)

// SymbolKind indicates the syntactic kind of a symbol.
type SymbolKind string

const (
	Class    SymbolKind = "class"
	Field    SymbolKind = "field"
	Function SymbolKind = "function"
	Method   SymbolKind = "method"
	Module   SymbolKind = "module"
)

// Tag represents a single symbol occurrence extracted from source code.
type Tag struct {
	Name       string
	Kind       TagKind
	SymbolKind SymbolKind
	Line       int
	File       string
	Signature  string
	Enclosing  string // qualified name of enclosing func/method for reference tags; "" if top-level
}

// FileInfo holds metadata and extracted tags for a single source file.
type FileInfo struct {
	Path     string
	Language string
	Tags     []Tag
	Rank     float64
}

// Dependency represents an edge in the dependency graph:
// Source references symbols defined in Target.
type Dependency struct {
	Source  string
	Target  string
	Symbols []string
}

// CallEdge represents a function-level call: Caller calls Callee.
// Both names are the qualified symbol names as they appear in definitions
// (e.g., "Server.Handle", "greet").
type CallEdge struct {
	Caller string
	Callee string
}

// CallSite records a specific call occurrence with its source location.
type CallSite struct {
	Caller string
	Callee string
	File   string
	Line   int
}

// RepoMap is the complete analyzed repository map, ready for serialization.
type RepoMap struct {
	RepoName     string
	Root         string
	Files        []FileInfo
	Dependencies []Dependency
	CallEdges    []CallEdge
	CallSites    []CallSite
	// Members holds field/method tags for focused --symbol --members queries.
	// Empty in full-map mode.
	Members []Tag
}
