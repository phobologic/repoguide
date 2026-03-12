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

type Format string

const (
	FormatV1 Format = "v1"
	FormatV2 Format = "v2"
)

func ParseFormat(value string) (Format, error) {
	switch Format(value) {
	case FormatV1, FormatV2:
		return Format(value), nil
	default:
		return "", fmt.Errorf("unsupported format %q", value)
	}
}

func CacheMatchesFormat(data string, format Format) bool {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return false
	}

	switch format {
	case FormatV1:
		return strings.HasPrefix(trimmed, "repo:")
	case FormatV2:
		return strings.HasPrefix(trimmed, "fmt: repoguide/v2")
	default:
		return false
	}
}

// Encode converts a RepoMap into TOON format.
// When focused is true (--symbol or --file query), callsites are emitted
// immediately after files so truncation cuts noise rather than the primary
// deliverable.
func Encode(rm *model.RepoMap, focused bool, format Format) string {
	switch format {
	case FormatV1:
		return encodeV1(rm, focused)
	case FormatV2:
		return encodeV2(rm, focused)
	default:
		return encodeV2(rm, focused)
	}
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
