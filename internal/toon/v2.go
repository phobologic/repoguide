package toon

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/phobologic/repoguide/internal/model"
)

func encodeV2(rm *model.RepoMap, focused bool) string {
	var parts []string
	parts = append(parts, "fmt: repoguide/v2")
	parts = append(parts, fmt.Sprintf("repo: %s", encodeValue(rm.RepoName)))
	parts = append(parts, fmt.Sprintf("root: %s", encodeValue(rm.Root)))

	fileIDs := make(map[string]string, len(rm.Files))
	fileRanks := make(map[string]float64, len(rm.Files))
	fileRows := make([][]string, 0, len(rm.Files))
	for i := range rm.Files {
		fi := &rm.Files[i]
		id := fmt.Sprintf("f%d", i+1)
		fileIDs[fi.Path] = id
		fileRanks[fi.Path] = fi.Rank
		fileRows = append(fileRows, []string{id, formatRank(fi.Rank), fi.Path})
	}
	parts = append(parts, formatTabular("files", []string{"id", "rank", "path"}, fileRows))

	if focused && len(rm.CallSites) > 0 {
		parts = append(parts, encodeSitesV2(rm.CallSites, fileIDs, fileRanks))
	}
	if focused && len(rm.Members) > 0 {
		parts = append(parts, encodeMembersV2(rm.Members))
	}

	definitionRanks := buildDefinitionRanks(rm)
	definitionTables, signatureRows := encodeDefinitionsV2(rm, fileIDs, focused)
	parts = append(parts, definitionTables...)
	if focused && len(signatureRows) > 0 {
		parts = append(parts, formatTabular("sig", []string{"file", "line", "signature"}, signatureRows))
	}

	parts = append(parts, encodeDependenciesV2(rm.Dependencies, fileIDs, fileRanks))
	parts = append(parts, encodeCallsV2(rm.CallEdges, definitionRanks))

	if !focused && len(rm.CallSites) > 0 {
		parts = append(parts, encodeSitesV2(rm.CallSites, fileIDs, fileRanks))
	}
	if !focused && len(rm.Members) > 0 {
		parts = append(parts, encodeMembersV2(rm.Members))
	}

	return strings.Join(parts, "\n")
}

func encodeMembersV2(members []model.Tag) string {
	rows := make([][]string, len(members))
	for i := range members {
		member := &members[i]
		name := member.Name
		if dot := strings.LastIndex(member.Name, "."); dot >= 0 {
			name = member.Name[dot+1:]
		}
		rows[i] = []string{shortKind(member.SymbolKind), fmt.Sprintf("%d", member.Line), name, member.Signature}
	}
	return formatTabular("members", []string{"kind", "line", "name", "signature"}, rows)
}

func encodeSitesV2(sites []model.CallSite, fileIDs map[string]string, fileRanks map[string]float64) string {
	sortedSites := append([]model.CallSite(nil), sites...)
	sort.Slice(sortedSites, func(i, j int) bool {
		leftRank := fileRanks[sortedSites[i].File]
		rightRank := fileRanks[sortedSites[j].File]
		if leftRank != rightRank {
			return leftRank > rightRank
		}
		if sortedSites[i].File != sortedSites[j].File {
			return sortedSites[i].File < sortedSites[j].File
		}
		if sortedSites[i].Line != sortedSites[j].Line {
			return sortedSites[i].Line < sortedSites[j].Line
		}
		if sortedSites[i].Caller != sortedSites[j].Caller {
			return sortedSites[i].Caller < sortedSites[j].Caller
		}
		return sortedSites[i].Callee < sortedSites[j].Callee
	})

	rows := make([][]string, len(sortedSites))
	for i := range sortedSites {
		site := &sortedSites[i]
		rows[i] = []string{fmt.Sprintf("%s->%s@%s:%d", site.Caller, site.Callee, fileIDs[site.File], site.Line)}
	}
	return formatTabular("callsites", []string{"site"}, rows)
}

func encodeDefinitionsV2(rm *model.RepoMap, fileIDs map[string]string, includeSignatures bool) ([]string, [][]string) {
	tables := map[string][][]string{"c": nil, "f": nil, "m": nil, "fld": nil}
	var signatureRows [][]string

	for i := range rm.Files {
		fi := &rm.Files[i]
		fileID := fileIDs[fi.Path]
		for j := range fi.Tags {
			tag := &fi.Tags[j]
			if tag.Kind != model.Definition {
				continue
			}
			key := shortKind(tag.SymbolKind)
			if _, ok := tables[key]; !ok {
				key = "f"
			}
			tables[key] = append(tables[key], []string{fileID, fmt.Sprintf("%d", tag.Line), tag.Name})
			if includeSignatures && tag.Signature != "" {
				signatureRows = append(signatureRows, []string{fileID, fmt.Sprintf("%d", tag.Line), tag.Signature})
			}
		}
	}

	encoded := []string{
		formatTabular("defs.c", []string{"file", "line", "name"}, tables["c"]),
		formatTabular("defs.f", []string{"file", "line", "name"}, tables["f"]),
		formatTabular("defs.m", []string{"file", "line", "name"}, tables["m"]),
		formatTabular("defs.fld", []string{"file", "line", "name"}, tables["fld"]),
	}

	return encoded, signatureRows
}

func encodeDependenciesV2(dependencies []model.Dependency, fileIDs map[string]string, fileRanks map[string]float64) string {
	sortedDeps := append([]model.Dependency(nil), dependencies...)
	sort.Slice(sortedDeps, func(i, j int) bool {
		leftWeight := maxFloat(fileRanks[sortedDeps[i].Source], fileRanks[sortedDeps[i].Target])
		rightWeight := maxFloat(fileRanks[sortedDeps[j].Source], fileRanks[sortedDeps[j].Target])
		if leftWeight != rightWeight {
			return leftWeight > rightWeight
		}
		leftSource := fileRanks[sortedDeps[i].Source]
		rightSource := fileRanks[sortedDeps[j].Source]
		if leftSource != rightSource {
			return leftSource > rightSource
		}
		if sortedDeps[i].Source != sortedDeps[j].Source {
			return sortedDeps[i].Source < sortedDeps[j].Source
		}
		return sortedDeps[i].Target < sortedDeps[j].Target
	})

	rows := make([][]string, len(sortedDeps))
	for i := range sortedDeps {
		dep := &sortedDeps[i]
		symbols := append([]string(nil), dep.Symbols...)
		sort.Strings(symbols)
		rows[i] = []string{fmt.Sprintf("%s->%s", fileIDs[dep.Source], fileIDs[dep.Target]), strings.Join(symbols, "|")}
	}
	return formatTabular("deps", []string{"edge", "symbols"}, rows)
}

func encodeCallsV2(callEdges []model.CallEdge, definitionRanks map[string]float64) string {
	sortedCalls := append([]model.CallEdge(nil), callEdges...)
	sort.Slice(sortedCalls, func(i, j int) bool {
		leftWeight := maxFloat(definitionRanks[sortedCalls[i].Caller], definitionRanks[sortedCalls[i].Callee])
		rightWeight := maxFloat(definitionRanks[sortedCalls[j].Caller], definitionRanks[sortedCalls[j].Callee])
		if leftWeight != rightWeight {
			return leftWeight > rightWeight
		}
		if sortedCalls[i].Caller != sortedCalls[j].Caller {
			return sortedCalls[i].Caller < sortedCalls[j].Caller
		}
		return sortedCalls[i].Callee < sortedCalls[j].Callee
	})

	rows := make([][]string, len(sortedCalls))
	for i := range sortedCalls {
		call := &sortedCalls[i]
		rows[i] = []string{fmt.Sprintf("%s->%s", call.Caller, call.Callee)}
	}
	return formatTabular("calls", []string{"edge"}, rows)
}

func buildDefinitionRanks(rm *model.RepoMap) map[string]float64 {
	ranks := make(map[string]float64)
	for i := range rm.Files {
		fi := &rm.Files[i]
		for j := range fi.Tags {
			tag := &fi.Tags[j]
			if tag.Kind == model.Definition {
				ranks[tag.Name] = fi.Rank
			}
		}
	}
	return ranks
}

func shortKind(kind model.SymbolKind) string {
	switch kind {
	case model.Class:
		return "c"
	case model.Function:
		return "f"
	case model.Method:
		return "m"
	case model.Field:
		return "fld"
	default:
		return string(kind)
	}
}

func formatRank(rank float64) string {
	formatted := strconv.FormatFloat(rank, 'f', 3, 64)
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	if formatted == "" {
		return "0"
	}
	return formatted
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
