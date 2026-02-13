// Package ranking implements token-budget-aware file selection.
package ranking

import "github.com/phobologic/repoguide/internal/model"

// SelectFiles returns a new RepoMap with only the top-ranked files.
// If maxFiles is <= 0 or >= len(files), all files are returned.
func SelectFiles(rm *model.RepoMap, maxFiles int) *model.RepoMap {
	if maxFiles <= 0 || maxFiles >= len(rm.Files) {
		return rm
	}

	selected := rm.Files[:maxFiles]
	selectedPaths := make(map[string]struct{}, maxFiles)
	for i := range selected {
		selectedPaths[selected[i].Path] = struct{}{}
	}

	var deps []model.Dependency
	for i := range rm.Dependencies {
		d := &rm.Dependencies[i]
		_, srcOK := selectedPaths[d.Source]
		_, tgtOK := selectedPaths[d.Target]
		if srcOK && tgtOK {
			deps = append(deps, *d)
		}
	}

	return &model.RepoMap{
		RepoName:     rm.RepoName,
		Root:         rm.Root,
		Files:        selected,
		Dependencies: deps,
	}
}
