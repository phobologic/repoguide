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
