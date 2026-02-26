package graph

import (
	"math"
	"testing"

	"github.com/phobologic/repoguide/internal/model"
)

func TestBuildGraphCrossFileRef(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{
			Path:     "a.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "foo", Kind: model.Reference, SymbolKind: model.Function},
			},
		},
		{
			Path:     "b.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "foo", Kind: model.Definition, SymbolKind: model.Function},
			},
		},
	}

	deps := BuildGraph(fileInfos)
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if deps[0].Source != "a.py" || deps[0].Target != "b.py" {
		t.Errorf("dep: %+v", deps[0])
	}
	if len(deps[0].Symbols) != 1 || deps[0].Symbols[0] != "foo" {
		t.Errorf("symbols: %v", deps[0].Symbols)
	}
}

func TestBuildGraphNoSelfEdge(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{
			Path:     "a.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "foo", Kind: model.Definition, SymbolKind: model.Function},
				{Name: "foo", Kind: model.Reference, SymbolKind: model.Function},
			},
		},
	}

	deps := BuildGraph(fileInfos)
	if len(deps) != 0 {
		t.Errorf("expected 0 deps (no self-edges), got %d", len(deps))
	}
}

func TestBuildGraphNoDefs(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{
			Path:     "a.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "foo", Kind: model.Reference, SymbolKind: model.Function},
			},
		},
	}

	deps := BuildGraph(fileInfos)
	if len(deps) != 0 {
		t.Errorf("expected 0 deps (unresolved ref), got %d", len(deps))
	}
}

func TestRankUniform(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{Path: "a.py"},
		{Path: "b.py"},
		{Path: "c.py"},
	}

	Rank(fileInfos, nil)

	expected := 1.0 / 3.0
	for _, fi := range fileInfos {
		if math.Abs(fi.Rank-expected) > 1e-9 {
			t.Errorf("%s rank = %f, want %f", fi.Path, fi.Rank, expected)
		}
	}
}

func TestRankWithEdges(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{Path: "a.py"},
		{Path: "b.py"},
		{Path: "c.py"},
	}

	deps := []model.Dependency{
		{Source: "a.py", Target: "b.py", Symbols: []string{"x"}},
		{Source: "c.py", Target: "b.py", Symbols: []string{"y"}},
	}

	Rank(fileInfos, deps)

	// b.py should have highest rank (referenced by both a and c)
	if fileInfos[0].Path != "b.py" {
		t.Errorf("expected b.py first, got %s", fileInfos[0].Path)
	}

	// Ranks should sum to ~1.0
	var sum float64
	for _, fi := range fileInfos {
		sum += fi.Rank
	}
	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("ranks sum to %f, expected ~1.0", sum)
	}

	// b.py should rank higher than a.py and c.py
	if fileInfos[0].Rank <= fileInfos[1].Rank {
		t.Errorf("b.py rank (%f) should be > second file rank (%f)",
			fileInfos[0].Rank, fileInfos[1].Rank)
	}
}

func TestRankEmpty(t *testing.T) {
	t.Parallel()
	Rank(nil, nil) // should not panic
}

func TestBuildCallGraph(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{
			Path:     "a.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "bar", Kind: model.Definition, SymbolKind: model.Function},
				{Name: "foo", Kind: model.Definition, SymbolKind: model.Function},
				// foo calls bar (in-repo, should be included)
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo"},
				// foo calls external (not in-repo, should be excluded)
				{Name: "print", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo"},
				// top-level call (no enclosing, should be excluded)
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: ""},
			},
		},
	}

	edges := BuildCallGraph(fileInfos)
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d: %+v", len(edges), edges)
	}
	if edges[0].Caller != "foo" || edges[0].Callee != "bar" {
		t.Errorf("unexpected edge: %+v", edges[0])
	}
}

func TestBuildCallGraphDeduplication(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{
			Path:     "a.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "bar", Kind: model.Definition, SymbolKind: model.Function},
				{Name: "foo", Kind: model.Definition, SymbolKind: model.Function},
				// foo calls bar multiple times — should produce only one edge
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo"},
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo"},
			},
		},
	}

	edges := BuildCallGraph(fileInfos)
	if len(edges) != 1 {
		t.Errorf("expected 1 deduplicated edge, got %d: %+v", len(edges), edges)
	}
}

func TestBuildCallGraphSorting(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{
			Path:     "a.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "bar", Kind: model.Definition, SymbolKind: model.Function},
				{Name: "baz", Kind: model.Definition, SymbolKind: model.Function},
				{Name: "foo", Kind: model.Definition, SymbolKind: model.Function},
				{Name: "baz", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo"},
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo"},
			},
		},
	}

	edges := BuildCallGraph(fileInfos)
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d: %+v", len(edges), edges)
	}
	// Should be sorted: foo,bar before foo,baz
	if edges[0].Callee != "bar" || edges[1].Callee != "baz" {
		t.Errorf("unexpected order: %+v", edges)
	}
}

func TestBuildCallGraphEmpty(t *testing.T) {
	t.Parallel()
	edges := BuildCallGraph(nil)
	if edges != nil {
		t.Errorf("expected nil, got %v", edges)
	}
}

func TestBuildCallSites(t *testing.T) {
	t.Parallel()

	fileInfos := []model.FileInfo{
		{
			Path:     "a.py",
			Language: "python",
			Tags: []model.Tag{
				{Name: "bar", Kind: model.Definition, SymbolKind: model.Function},
				{Name: "foo", Kind: model.Definition, SymbolKind: model.Function},
				// foo calls bar twice at different lines — both sites should appear
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo", Line: 10},
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo", Line: 20},
				// top-level call (no enclosing) — excluded
				{Name: "bar", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "", Line: 5},
				// external call — excluded
				{Name: "print", Kind: model.Reference, SymbolKind: model.Function, Enclosing: "foo", Line: 15},
			},
		},
	}

	sites := BuildCallSites(fileInfos)
	if len(sites) != 2 {
		t.Fatalf("expected 2 call sites (no deduplication), got %d: %+v", len(sites), sites)
	}
	for _, s := range sites {
		if s.Caller != "foo" || s.Callee != "bar" || s.File != "a.py" {
			t.Errorf("unexpected call site: %+v", s)
		}
	}
	if sites[0].Line != 10 || sites[1].Line != 20 {
		t.Errorf("expected lines 10 and 20, got %d and %d", sites[0].Line, sites[1].Line)
	}
}

func TestBuildCallSitesEmpty(t *testing.T) {
	t.Parallel()
	sites := BuildCallSites(nil)
	if sites != nil {
		t.Errorf("expected nil, got %v", sites)
	}
}
