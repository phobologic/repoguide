package toon

import (
	"fmt"
	"strings"

	"github.com/phobologic/repoguide/internal/model"
)

func encodeV1(rm *model.RepoMap, focused bool) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("repo: %s", encodeValue(rm.RepoName)))
	parts = append(parts, fmt.Sprintf("root: %s", encodeValue(rm.Root)))

	var fileRows [][]string
	for i := range rm.Files {
		fi := &rm.Files[i]
		fileRows = append(fileRows, []string{
			fi.Path,
			fi.Language,
			fmt.Sprintf("%.4f", fi.Rank),
		})
	}
	parts = append(parts, formatTabular("files", []string{"path", "language", "rank"}, fileRows))

	if focused && len(rm.CallSites) > 0 {
		parts = append(parts, encodeSites(rm.CallSites))
	}
	if focused && len(rm.Members) > 0 {
		parts = append(parts, encodeMembers(rm.Members))
	}

	var symbolRows [][]string
	for i := range rm.Files {
		fi := &rm.Files[i]
		for j := range fi.Tags {
			tag := &fi.Tags[j]
			if tag.Kind == model.Definition {
				symbolRows = append(symbolRows, []string{
					fi.Path,
					tag.Name,
					string(tag.SymbolKind),
					fmt.Sprintf("%d", tag.Line),
					tag.Signature,
				})
			}
		}
	}
	parts = append(parts, formatTabular("symbols", []string{"file", "name", "kind", "line", "signature"}, symbolRows))

	var depRows [][]string
	for i := range rm.Dependencies {
		dep := &rm.Dependencies[i]
		depRows = append(depRows, []string{dep.Source, dep.Target, strings.Join(dep.Symbols, " ")})
	}
	parts = append(parts, formatTabular("dependencies", []string{"source", "target", "symbols"}, depRows))

	var callRows [][]string
	for i := range rm.CallEdges {
		callEdge := &rm.CallEdges[i]
		callRows = append(callRows, []string{callEdge.Caller, callEdge.Callee})
	}
	parts = append(parts, formatTabular("calls", []string{"caller", "callee"}, callRows))

	if !focused && len(rm.CallSites) > 0 {
		parts = append(parts, encodeSites(rm.CallSites))
	}
	if !focused && len(rm.Members) > 0 {
		parts = append(parts, encodeMembers(rm.Members))
	}

	return strings.Join(parts, "\n")
}

func encodeMembers(members []model.Tag) string {
	rows := make([][]string, len(members))
	for i := range members {
		member := &members[i]
		name := member.Name
		if dot := strings.LastIndex(member.Name, "."); dot >= 0 {
			name = member.Name[dot+1:]
		}
		rows[i] = []string{name, string(member.SymbolKind), fmt.Sprintf("%d", member.Line), member.Signature}
	}
	return formatTabular("members", []string{"name", "kind", "line", "signature"}, rows)
}

func encodeSites(sites []model.CallSite) string {
	rows := make([][]string, len(sites))
	for i := range sites {
		site := &sites[i]
		rows[i] = []string{site.Caller, site.Callee, site.File, fmt.Sprintf("%d", site.Line)}
	}
	return formatTabular("callsites", []string{"caller", "callee", "file", "line"}, rows)
}
