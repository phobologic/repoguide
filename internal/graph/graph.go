// Package graph builds a dependency graph and computes PageRank.
package graph

import (
	"math"
	"sort"

	"github.com/phobologic/repoguide/internal/model"
)

// BuildGraph creates dependency edges from cross-file symbol references.
// Returns a list of dependencies suitable for the RepoMap.
func BuildGraph(fileInfos []model.FileInfo) []model.Dependency {
	// Build definition index: symbol name → set of files that define it
	defines := make(map[string]map[string]struct{})
	for i := range fileInfos {
		fi := &fileInfos[i]
		for j := range fi.Tags {
			tag := &fi.Tags[j]
			if tag.Kind == model.Definition {
				if defines[tag.Name] == nil {
					defines[tag.Name] = make(map[string]struct{})
				}
				defines[tag.Name][fi.Path] = struct{}{}
			}
		}
	}

	// Build edges: source → target → list of symbols
	type edgeKey struct{ src, tgt string }
	edgeSymbols := make(map[edgeKey][]string)

	for i := range fileInfos {
		fi := &fileInfos[i]
		for j := range fi.Tags {
			tag := &fi.Tags[j]
			if tag.Kind != model.Reference {
				continue
			}
			defFiles := defines[tag.Name]
			if defFiles == nil {
				continue
			}
			// Iterate in sorted order for determinism
			sorted := sortedKeys(defFiles)
			for _, defFile := range sorted {
				if defFile == fi.Path {
					continue // no self-edges
				}
				key := edgeKey{fi.Path, defFile}
				// Only add symbol if not already present
				if !contains(edgeSymbols[key], tag.Name) {
					edgeSymbols[key] = append(edgeSymbols[key], tag.Name)
				}
			}
		}
	}

	var deps []model.Dependency
	for key, syms := range edgeSymbols {
		deps = append(deps, model.Dependency{
			Source:  key.src,
			Target:  key.tgt,
			Symbols: syms,
		})
	}

	// Sort for deterministic output
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].Source != deps[j].Source {
			return deps[i].Source < deps[j].Source
		}
		return deps[i].Target < deps[j].Target
	})

	return deps
}

// BuildCallGraph builds function-level call edges from the parsed file infos.
// An edge is only included when the callee is a known definition in the repo
// and the caller (Enclosing) is non-empty. Edges are deduplicated and sorted.
func BuildCallGraph(fileInfos []model.FileInfo) []model.CallEdge {
	// Build set of all known definition names.
	knownDefs := make(map[string]struct{})
	for i := range fileInfos {
		for j := range fileInfos[i].Tags {
			tag := &fileInfos[i].Tags[j]
			if tag.Kind == model.Definition {
				knownDefs[tag.Name] = struct{}{}
			}
		}
	}

	type edgeKey struct{ caller, callee string }
	seen := make(map[edgeKey]struct{})

	var edges []model.CallEdge
	for i := range fileInfos {
		for j := range fileInfos[i].Tags {
			tag := &fileInfos[i].Tags[j]
			if tag.Kind != model.Reference || tag.Enclosing == "" {
				continue
			}
			if _, ok := knownDefs[tag.Name]; !ok {
				continue
			}
			key := edgeKey{tag.Enclosing, tag.Name}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			edges = append(edges, model.CallEdge{Caller: tag.Enclosing, Callee: tag.Name})
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Caller != edges[j].Caller {
			return edges[i].Caller < edges[j].Caller
		}
		return edges[i].Callee < edges[j].Callee
	})

	return edges
}

// BuildCallSites returns all individual call and import occurrences with source
// locations. Unlike BuildCallGraph, it does not deduplicate: if a function calls
// another three times, three CallSite entries are returned. Module-level import
// references (where no enclosing function exists) are included with Caller set to
// "<import>". Intended for focused (--symbol / --file) queries where precise line
// numbers matter.
func BuildCallSites(fileInfos []model.FileInfo) []model.CallSite {
	// Build set of all known definition names.
	knownDefs := make(map[string]struct{})
	for i := range fileInfos {
		for j := range fileInfos[i].Tags {
			tag := &fileInfos[i].Tags[j]
			if tag.Kind == model.Definition {
				knownDefs[tag.Name] = struct{}{}
			}
		}
	}

	var sites []model.CallSite
	for i := range fileInfos {
		for j := range fileInfos[i].Tags {
			tag := &fileInfos[i].Tags[j]
			if tag.Kind != model.Reference {
				continue
			}
			if _, ok := knownDefs[tag.Name]; !ok {
				continue
			}
			caller := tag.Enclosing
			if caller == "" {
				caller = "<import>"
			}
			sites = append(sites, model.CallSite{
				Caller: caller,
				Callee: tag.Name,
				File:   fileInfos[i].Path,
				Line:   tag.Line,
			})
		}
	}

	sort.Slice(sites, func(i, j int) bool {
		if sites[i].Caller != sites[j].Caller {
			return sites[i].Caller < sites[j].Caller
		}
		if sites[i].Callee != sites[j].Callee {
			return sites[i].Callee < sites[j].Callee
		}
		if sites[i].File != sites[j].File {
			return sites[i].File < sites[j].File
		}
		return sites[i].Line < sites[j].Line
	})

	return sites
}

// Rank applies PageRank to file_infos and sorts them by rank descending.
func Rank(fileInfos []model.FileInfo, deps []model.Dependency) {
	if len(fileInfos) == 0 {
		return
	}

	if len(deps) == 0 {
		uniform := 1.0 / float64(len(fileInfos))
		for i := range fileInfos {
			fileInfos[i].Rank = uniform
		}
		return
	}

	// Build adjacency for PageRank
	// Edge from source to target means source references target.
	// Count edges per (source, target) pair.
	outEdges := make(map[string][]string) // node → list of targets (with repeats for multi-edges)
	outDegree := make(map[string]int)     // total out-edges per node
	nodes := make(map[string]struct{})

	for i := range fileInfos {
		nodes[fileInfos[i].Path] = struct{}{}
	}

	for _, d := range deps {
		// Each symbol is an edge
		for range d.Symbols {
			outEdges[d.Source] = append(outEdges[d.Source], d.Target)
			outDegree[d.Source]++
		}
	}

	ranks := pageRank(nodes, outEdges, outDegree, 0.85, 100, 1e-6)

	for i := range fileInfos {
		fileInfos[i].Rank = ranks[fileInfos[i].Path]
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].Rank > fileInfos[j].Rank
	})
}

func pageRank(
	nodes map[string]struct{},
	outEdges map[string][]string,
	outDegree map[string]int,
	alpha float64,
	maxIter int,
	tol float64,
) map[string]float64 {
	n := len(nodes)
	if n == 0 {
		return nil
	}

	rank := make(map[string]float64, n)
	initial := 1.0 / float64(n)
	for node := range nodes {
		rank[node] = initial
	}

	teleport := (1.0 - alpha) / float64(n)

	for iter := 0; iter < maxIter; iter++ {
		newRank := make(map[string]float64, n)

		// Dangling node contribution (nodes with no outgoing edges)
		var danglingSum float64
		for node := range nodes {
			if outDegree[node] == 0 {
				danglingSum += rank[node]
			}
		}
		danglingContrib := alpha * danglingSum / float64(n)

		for node := range nodes {
			newRank[node] = teleport + danglingContrib
		}

		// Distribute rank through edges
		for src, targets := range outEdges {
			deg := float64(outDegree[src])
			contrib := alpha * rank[src] / deg
			for _, tgt := range targets {
				newRank[tgt] += contrib
			}
		}

		// Check convergence
		var diff float64
		for node := range nodes {
			diff += math.Abs(newRank[node] - rank[node])
		}

		rank = newRank

		if diff < tol {
			break
		}
	}

	return rank
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
