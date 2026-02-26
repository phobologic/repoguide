// Package ranking implements token-budget-aware file selection.
package ranking

import (
	"strings"

	"github.com/phobologic/repoguide/internal/model"
)

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

	// Build set of definition names in selected files to filter call edges.
	selectedDefs := make(map[string]struct{})
	for i := range selected {
		for j := range selected[i].Tags {
			tag := &selected[i].Tags[j]
			if tag.Kind == model.Definition {
				selectedDefs[tag.Name] = struct{}{}
			}
		}
	}

	var callEdges []model.CallEdge
	for i := range rm.CallEdges {
		ce := &rm.CallEdges[i]
		if _, ok := selectedDefs[ce.Caller]; ok {
			callEdges = append(callEdges, *ce)
		}
	}

	var callSites []model.CallSite
	for i := range rm.CallSites {
		cs := &rm.CallSites[i]
		if _, ok := selectedDefs[cs.Caller]; ok {
			callSites = append(callSites, *cs)
		}
	}

	return &model.RepoMap{
		RepoName:     rm.RepoName,
		Root:         rm.Root,
		Files:        selected,
		Dependencies: deps,
		CallEdges:    callEdges,
		CallSites:    callSites,
	}
}

// FilterBySymbol returns a new RepoMap containing only symbols whose name
// contains substr (case-insensitive), the files that define those symbols,
// files that define their direct callers and callees, and the edges that
// connect them.
func FilterBySymbol(rm *model.RepoMap, substr string) *model.RepoMap {
	lower := strings.ToLower(substr)

	// Find matched symbols and their files.
	matchedSymbols := make(map[string]struct{})
	matchedFiles := make(map[string]struct{})
	for i := range rm.Files {
		for j := range rm.Files[i].Tags {
			tag := &rm.Files[i].Tags[j]
			if tag.Kind == model.Definition && strings.Contains(strings.ToLower(tag.Name), lower) {
				matchedSymbols[tag.Name] = struct{}{}
				matchedFiles[rm.Files[i].Path] = struct{}{}
			}
		}
	}

	// Expand to include files that define callers/callees of matched symbols.
	relatedSymbols := make(map[string]struct{})
	for i := range rm.CallEdges {
		ce := &rm.CallEdges[i]
		if _, ok := matchedSymbols[ce.Caller]; ok {
			relatedSymbols[ce.Callee] = struct{}{}
		}
		if _, ok := matchedSymbols[ce.Callee]; ok {
			relatedSymbols[ce.Caller] = struct{}{}
		}
	}
	for i := range rm.Files {
		for j := range rm.Files[i].Tags {
			tag := &rm.Files[i].Tags[j]
			if tag.Kind == model.Definition {
				if _, ok := relatedSymbols[tag.Name]; ok {
					matchedFiles[rm.Files[i].Path] = struct{}{}
				}
			}
		}
	}

	var files []model.FileInfo
	for i := range rm.Files {
		if _, ok := matchedFiles[rm.Files[i].Path]; ok {
			fi := rm.Files[i]
			// Trim tags to only the matched and related definitions so the
			// symbols table stays focused rather than dumping all exports from
			// every matched file.
			var filteredTags []model.Tag
			for j := range fi.Tags {
				tag := &fi.Tags[j]
				if tag.Kind != model.Definition {
					continue
				}
				_, isMatched := matchedSymbols[tag.Name]
				_, isRelated := relatedSymbols[tag.Name]
				if isMatched || isRelated {
					filteredTags = append(filteredTags, *tag)
				}
			}
			fi.Tags = filteredTags
			files = append(files, fi)
		}
	}

	var deps []model.Dependency
	for i := range rm.Dependencies {
		d := &rm.Dependencies[i]
		_, srcOK := matchedFiles[d.Source]
		_, tgtOK := matchedFiles[d.Target]
		if srcOK || tgtOK {
			deps = append(deps, *d)
		}
	}

	var callEdges []model.CallEdge
	for i := range rm.CallEdges {
		ce := &rm.CallEdges[i]
		_, callerOK := matchedSymbols[ce.Caller]
		_, calleeOK := matchedSymbols[ce.Callee]
		if callerOK || calleeOK {
			callEdges = append(callEdges, *ce)
		}
	}

	var callSites []model.CallSite
	for i := range rm.CallSites {
		cs := &rm.CallSites[i]
		_, callerOK := matchedSymbols[cs.Caller]
		_, calleeOK := matchedSymbols[cs.Callee]
		if callerOK || calleeOK {
			callSites = append(callSites, *cs)
		}
	}

	return &model.RepoMap{
		RepoName:     rm.RepoName,
		Root:         rm.Root,
		Files:        files,
		Dependencies: deps,
		CallEdges:    callEdges,
		CallSites:    callSites,
	}
}

// FilterByFile returns a new RepoMap containing only files whose path
// contains substr (case-insensitive), with all dependency edges touching
// those files and call edges from functions defined in those files.
func FilterByFile(rm *model.RepoMap, substr string) *model.RepoMap {
	lower := strings.ToLower(substr)

	matchedFiles := make(map[string]struct{})
	var files []model.FileInfo
	for i := range rm.Files {
		if strings.Contains(strings.ToLower(rm.Files[i].Path), lower) {
			matchedFiles[rm.Files[i].Path] = struct{}{}
			files = append(files, rm.Files[i])
		}
	}

	var deps []model.Dependency
	for i := range rm.Dependencies {
		d := &rm.Dependencies[i]
		_, srcOK := matchedFiles[d.Source]
		_, tgtOK := matchedFiles[d.Target]
		if srcOK || tgtOK {
			deps = append(deps, *d)
		}
	}

	// Build definition-to-file map for call edge filtering.
	defToFile := make(map[string]string)
	for i := range files {
		for j := range files[i].Tags {
			tag := &files[i].Tags[j]
			if tag.Kind == model.Definition {
				defToFile[tag.Name] = files[i].Path
			}
		}
	}

	var callEdges []model.CallEdge
	for i := range rm.CallEdges {
		ce := &rm.CallEdges[i]
		if _, ok := defToFile[ce.Caller]; ok {
			callEdges = append(callEdges, *ce)
		}
	}

	var callSites []model.CallSite
	for i := range rm.CallSites {
		cs := &rm.CallSites[i]
		if _, ok := matchedFiles[cs.File]; ok {
			callSites = append(callSites, *cs)
		}
	}

	return &model.RepoMap{
		RepoName:     rm.RepoName,
		Root:         rm.Root,
		Files:        files,
		Dependencies: deps,
		CallEdges:    callEdges,
		CallSites:    callSites,
	}
}
