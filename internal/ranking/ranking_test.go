package ranking

import (
	"testing"

	"github.com/phobologic/repoguide/internal/model"
)

func makeRepoMap() *model.RepoMap {
	return &model.RepoMap{
		RepoName: "test",
		Root:     "test",
		Files: []model.FileInfo{
			{Path: "a.py", Language: "python", Rank: 0.5},
			{Path: "b.py", Language: "python", Rank: 0.3},
			{Path: "c.py", Language: "python", Rank: 0.2},
		},
		Dependencies: []model.Dependency{
			{Source: "a.py", Target: "b.py", Symbols: []string{"foo"}},
			{Source: "a.py", Target: "c.py", Symbols: []string{"bar"}},
			{Source: "b.py", Target: "c.py", Symbols: []string{"baz"}},
		},
	}
}

// makeFilterRepoMap builds a RepoMap with tags and call edges for filter tests.
//
//	a.go defines Foo and Bar; b.go defines Baz; c.go defines Qux.
//	Call edges: Foo→Baz, Qux→Foo.
//	Deps: a.go→b.go (Baz), c.go→a.go (Foo).
func makeFilterRepoMap() *model.RepoMap {
	return &model.RepoMap{
		RepoName: "test",
		Root:     "test",
		Files: []model.FileInfo{
			{
				Path: "a.go", Language: "go", Rank: 0.5,
				Tags: []model.Tag{
					{Name: "Foo", Kind: model.Definition, SymbolKind: model.Function, Line: 1, File: "a.go"},
					{Name: "Bar", Kind: model.Definition, SymbolKind: model.Function, Line: 5, File: "a.go"},
				},
			},
			{
				Path: "b.go", Language: "go", Rank: 0.3,
				Tags: []model.Tag{
					{Name: "Baz", Kind: model.Definition, SymbolKind: model.Function, Line: 1, File: "b.go"},
				},
			},
			{
				Path: "c.go", Language: "go", Rank: 0.2,
				Tags: []model.Tag{
					{Name: "Qux", Kind: model.Definition, SymbolKind: model.Function, Line: 1, File: "c.go"},
				},
			},
		},
		Dependencies: []model.Dependency{
			{Source: "a.go", Target: "b.go", Symbols: []string{"Baz"}},
			{Source: "c.go", Target: "a.go", Symbols: []string{"Foo"}},
		},
		CallEdges: []model.CallEdge{
			{Caller: "Foo", Callee: "Baz"},
			{Caller: "Qux", Callee: "Foo"},
		},
		CallSites: []model.CallSite{
			{Caller: "Foo", Callee: "Baz", File: "a.go", Line: 10},
			{Caller: "Foo", Callee: "Baz", File: "a.go", Line: 20},
			{Caller: "Qux", Callee: "Foo", File: "c.go", Line: 5},
		},
	}
}

func TestSelectFilesAll(t *testing.T) {
	t.Parallel()

	rm := makeRepoMap()
	got := SelectFiles(rm, 0)
	if got != rm {
		t.Error("maxFiles=0 should return original")
	}

	got = SelectFiles(rm, 5)
	if got != rm {
		t.Error("maxFiles > len should return original")
	}

	got = SelectFiles(rm, 3)
	if got != rm {
		t.Error("maxFiles == len should return original")
	}
}

func TestSelectFilesSubset(t *testing.T) {
	t.Parallel()

	rm := makeRepoMap()
	got := SelectFiles(rm, 2)

	if len(got.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(got.Files))
	}
	if got.Files[0].Path != "a.py" || got.Files[1].Path != "b.py" {
		t.Errorf("expected a.py, b.py; got %s, %s", got.Files[0].Path, got.Files[1].Path)
	}

	// Only a.py→b.py dep should survive (c.py not in selected)
	if len(got.Dependencies) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got.Dependencies))
	}
	if got.Dependencies[0].Source != "a.py" || got.Dependencies[0].Target != "b.py" {
		t.Errorf("unexpected dep: %+v", got.Dependencies[0])
	}
}

func TestSelectFilesOne(t *testing.T) {
	t.Parallel()

	rm := makeRepoMap()
	got := SelectFiles(rm, 1)

	if len(got.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got.Files))
	}
	if len(got.Dependencies) != 0 {
		t.Errorf("expected 0 deps, got %d", len(got.Dependencies))
	}
}

func TestFilterBySymbolMatch(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterBySymbol(rm, "Foo")

	// Foo is in a.go; Foo calls Baz (b.go) and is called by Qux (c.go) — all 3 files included.
	if len(got.Files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(got.Files), fileNames(got))
	}
	// Both call edges touch Foo.
	if len(got.CallEdges) != 2 {
		t.Fatalf("expected 2 call edges, got %d", len(got.CallEdges))
	}
	// All deps touch the expanded file set.
	if len(got.Dependencies) != 2 {
		t.Errorf("expected 2 deps, got %d", len(got.Dependencies))
	}
}

func TestFilterBySymbolNoMatch(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterBySymbol(rm, "NoSuchSymbol")

	if len(got.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(got.Files))
	}
	if len(got.CallEdges) != 0 {
		t.Errorf("expected 0 call edges, got %d", len(got.CallEdges))
	}
	if len(got.Dependencies) != 0 {
		t.Errorf("expected 0 deps, got %d", len(got.Dependencies))
	}
}

func TestFilterBySymbolCaseInsensitive(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterBySymbol(rm, "foo") // lowercase matches "Foo"

	if len(got.Files) == 0 {
		t.Fatal("expected matches for lowercase 'foo'")
	}
	// Should include a.go (defines Foo).
	found := false
	for _, f := range got.Files {
		if f.Path == "a.go" {
			found = true
		}
	}
	if !found {
		t.Error("a.go should be in results")
	}
}

func TestFilterBySymbolSubstring(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	// "ba" matches both "Bar" (a.go) and "Baz" (b.go).
	got := FilterBySymbol(rm, "ba")

	if len(got.Files) < 2 {
		t.Fatalf("expected at least 2 files for 'ba', got %d: %v", len(got.Files), fileNames(got))
	}
}

func TestFilterBySymbolCallExpansion(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	// Filter for Baz (defined in b.go). Foo calls Baz, so a.go should be included.
	got := FilterBySymbol(rm, "Baz")

	paths := make(map[string]bool)
	for _, f := range got.Files {
		paths[f.Path] = true
	}
	if !paths["b.go"] {
		t.Error("b.go (defines Baz) must be included")
	}
	if !paths["a.go"] {
		t.Error("a.go (defines Foo which calls Baz) must be included via call expansion")
	}
}

func TestFilterBySymbolDepsEitherEndpoint(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	// Filter for Baz (b.go). a.go→b.go dep should be included even though a.go
	// is included only via expansion (its caller Foo calls Baz).
	got := FilterBySymbol(rm, "Baz")

	found := false
	for _, d := range got.Dependencies {
		if d.Source == "a.go" && d.Target == "b.go" {
			found = true
		}
	}
	if !found {
		t.Error("a.go→b.go dependency should be included")
	}
}

func TestFilterByFileMatch(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterByFile(rm, "a.go")

	if len(got.Files) != 1 || got.Files[0].Path != "a.go" {
		t.Fatalf("expected only a.go, got %v", fileNames(got))
	}
	// Both deps touch a.go (a.go→b.go and c.go→a.go).
	if len(got.Dependencies) != 2 {
		t.Errorf("expected 2 deps (both touch a.go), got %d", len(got.Dependencies))
	}
	// Call edges from functions defined in a.go: Foo→Baz.
	if len(got.CallEdges) != 1 {
		t.Fatalf("expected 1 call edge, got %d", len(got.CallEdges))
	}
	if got.CallEdges[0].Caller != "Foo" || got.CallEdges[0].Callee != "Baz" {
		t.Errorf("unexpected call edge: %+v", got.CallEdges[0])
	}
}

func TestFilterByFileNoMatch(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterByFile(rm, "no_such_file.go")

	if len(got.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(got.Files))
	}
}

func TestFilterByFileCaseInsensitive(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterByFile(rm, "A.GO") // uppercase; should still match a.go

	if len(got.Files) == 0 {
		t.Error("expected match for uppercase 'A.GO'")
	}
}

func TestFilterByFileSubstring(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterByFile(rm, ".go") // matches all three files

	if len(got.Files) != 3 {
		t.Fatalf("expected 3 files for '.go', got %d", len(got.Files))
	}
}

func fileNames(rm *model.RepoMap) []string {
	names := make([]string, len(rm.Files))
	for i, f := range rm.Files {
		names[i] = f.Path
	}
	return names
}

// TestFilterBySymbolCallSites verifies that FilterBySymbol propagates CallSites
// matching the target symbol on either end.
func TestFilterBySymbolCallSites(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterBySymbol(rm, "Foo")

	// Foo is caller in Foo→Baz (lines 10, 20) and callee in Qux→Foo (line 5)
	if len(got.CallSites) != 3 {
		t.Fatalf("expected 3 call sites, got %d: %+v", len(got.CallSites), got.CallSites)
	}
}

// TestFilterBySymbolCallSitesNoMatch verifies that CallSites are empty when
// the matched symbol has no call site entries.
func TestFilterBySymbolCallSitesNoMatch(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	// Bar has no call edges or sites in the fixture.
	got := FilterBySymbol(rm, "Bar")

	if len(got.CallSites) != 0 {
		t.Fatalf("expected 0 call sites, got %d: %+v", len(got.CallSites), got.CallSites)
	}
}

// TestFilterByFileCallSites verifies that FilterByFile propagates CallSites
// whose File field matches the filtered path.
func TestFilterByFileCallSites(t *testing.T) {
	t.Parallel()

	rm := makeFilterRepoMap()
	got := FilterByFile(rm, "a.go")

	// Sites with File=="a.go": Foo→Baz at lines 10 and 20
	if len(got.CallSites) != 2 {
		t.Fatalf("expected 2 call sites for a.go, got %d: %+v", len(got.CallSites), got.CallSites)
	}
	for _, cs := range got.CallSites {
		if cs.File != "a.go" {
			t.Errorf("unexpected file in call site: %+v", cs)
		}
	}
}
