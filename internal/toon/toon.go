// Package toon implements TOON (Token-Oriented Object Notation) encoding.
package toon

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/phobologic/repoguide/internal/model"
)

var (
	needsQuoting = regexp.MustCompile(`[,:"\\{}\[\]]`)
	looksNumeric = regexp.MustCompile(`^-?(?:0|[1-9]\d*)(?:\.\d+)?$`)
	keywords     = map[string]struct{}{
		"true":  {},
		"false": {},
		"null":  {},
	}
)

// Encode converts a RepoMap into TOON format.
func Encode(rm *model.RepoMap) string {
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
		d := &rm.Dependencies[i]
		depRows = append(depRows, []string{
			d.Source,
			d.Target,
			strings.Join(d.Symbols, " "),
		})
	}
	parts = append(parts, formatTabular("dependencies", []string{"source", "target", "symbols"}, depRows))

	var callRows [][]string
	for i := range rm.CallEdges {
		ce := &rm.CallEdges[i]
		callRows = append(callRows, []string{ce.Caller, ce.Callee})
	}
	parts = append(parts, formatTabular("calls", []string{"caller", "callee"}, callRows))

	if len(rm.CallSites) > 0 {
		var siteRows [][]string
		for i := range rm.CallSites {
			cs := &rm.CallSites[i]
			siteRows = append(siteRows, []string{
				cs.Caller,
				cs.Callee,
				cs.File,
				fmt.Sprintf("%d", cs.Line),
			})
		}
		parts = append(parts, formatTabular("callsites", []string{"caller", "callee", "file", "line"}, siteRows))
	}

	return strings.Join(parts, "\n")
}

func formatTabular(name string, columns []string, rows [][]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s[%d]{%s}:", name, len(rows), strings.Join(columns, ","))
	for _, row := range rows {
		encoded := make([]string, len(row))
		for i, cell := range row {
			encoded[i] = encodeValue(cell)
		}
		fmt.Fprintf(&b, "\n  %s", strings.Join(encoded, ","))
	}
	return b.String()
}

func encodeValue(value string) string {
	if value == "" {
		return `""`
	}

	if value != strings.TrimSpace(value) {
		return quote(value)
	}

	if strings.ContainsAny(value, "\n\r\t") {
		return quote(value)
	}

	if _, ok := keywords[strings.ToLower(value)]; ok {
		return quote(value)
	}

	if looksNumeric.MatchString(value) {
		return value
	}

	if needsQuoting.MatchString(value) {
		return quote(value)
	}

	if strings.HasPrefix(value, "-") {
		return quote(value)
	}

	return value
}

func quote(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	escaped = strings.ReplaceAll(escaped, "\r", `\r`)
	escaped = strings.ReplaceAll(escaped, "\t", `\t`)
	return `"` + escaped + `"`
}
