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
//
// When withMembers is true and a matched symbol is a class/struct, the members
// table of the returned RepoMap is populated with that class's field tags.
// If no top-level definitions match, withMembers triggers a fallback search
// over member names (the unqualified part after ".").
func FilterBySymbol(rm *model.RepoMap, substr string, withMembers bool) *model.RepoMap {
	lower := strings.ToLower(substr)

	// Find matched symbols and their files, excluding field tags from the primary
	// symbol match (fields are handled separately via the members mechanism).
	matchedSymbols := make(map[string]struct{})
	matchedFiles := make(map[string]struct{})
	for i := range rm.Files {
		for j := range rm.Files[i].Tags {
			tag := &rm.Files[i].Tags[j]
			if tag.Kind == model.Definition && tag.SymbolKind != model.Field &&
				strings.Contains(strings.ToLower(tag.Name), lower) {
				matchedSymbols[tag.Name] = struct{}{}
				matchedFiles[rm.Files[i].Path] = struct{}{}
			}
		}
	}

	// Member fallback: if no top-level defs matched and withMembers is requested,
	// search field tags whose unqualified name (part after ".") contains substr.
	// Include the owning class in matched symbols for context.
	if withMembers && len(matchedSymbols) == 0 {
		for i := range rm.Files {
			for j := range rm.Files[i].Tags {
				tag := &rm.Files[i].Tags[j]
				if tag.Kind != model.Definition || tag.SymbolKind != model.Field {
					continue
				}
				unqualified := tag.Name
				if dot := strings.LastIndex(tag.Name, "."); dot >= 0 {
					unqualified = tag.Name[dot+1:]
				}
				if strings.Contains(strings.ToLower(unqualified), lower) {
					matchedSymbols[tag.Name] = struct{}{}
					matchedFiles[rm.Files[i].Path] = struct{}{}
				}
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
			// every matched file. Field tags are never shown in the symbols
			// table — they appear in the members table instead.
			var filteredTags []model.Tag
			for j := range fi.Tags {
				tag := &fi.Tags[j]
				if tag.Kind != model.Definition || tag.SymbolKind == model.Field {
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

	// Collect members when requested.
	var members []model.Tag
	if withMembers {
		// Phase A: for each matched class symbol, include all its field tags.
		for i := range rm.Files {
			for j := range rm.Files[i].Tags {
				tag := &rm.Files[i].Tags[j]
				if tag.Kind != model.Definition || tag.SymbolKind != model.Field {
					continue
				}
				// Check if the owning type (prefix before ".") is a matched class.
				dot := strings.LastIndex(tag.Name, ".")
				if dot < 0 {
					continue
				}
				ownerName := tag.Name[:dot]
				if _, ok := matchedSymbols[ownerName]; ok {
					members = append(members, *tag)
				}
			}
		}
		// Phase B: for fallback-matched field tags (field names directly in
		// matchedSymbols), include them if not already added via Phase A.
		if len(members) == 0 {
			for i := range rm.Files {
				for j := range rm.Files[i].Tags {
					tag := &rm.Files[i].Tags[j]
					if tag.Kind != model.Definition || tag.SymbolKind != model.Field {
						continue
					}
					if _, ok := matchedSymbols[tag.Name]; ok {
						members = append(members, *tag)
					}
				}
			}
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
		Members:      members,
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
