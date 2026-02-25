package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	sentinelStart = "<!-- repoguide:start -->"
	sentinelEnd   = "<!-- repoguide:end -->"
)

// runInit implements the `repoguide init` subcommand, which writes (or updates)
// a repoguide usage section in a CLAUDE.md file.
func runInit(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("repoguide init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var dryRun bool
	fs.BoolVar(&dryRun, "dry-run", false, "print what would be written without modifying the file")

	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage: repoguide init [flags] [path-to-CLAUDE.md]

Write a repoguide usage section to a CLAUDE.md file. The section is wrapped in
sentinel comments so it can be updated in place on subsequent runs without
touching surrounding content. Creates the file if it does not exist.

path-to-CLAUDE.md defaults to ./CLAUDE.md.

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	section := generateSection()

	// --dry-run with no path: just print the section itself.
	if dryRun && fs.NArg() == 0 {
		_, _ = fmt.Fprintln(stdout, section)
		return nil
	}

	path := "CLAUDE.md"
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}

	existing, _ := os.ReadFile(path)
	updated := applySection(string(existing), section)

	if dryRun {
		_, _ = fmt.Fprint(stdout, updated)
		return nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	_, _ = fmt.Fprintf(stderr, "wrote repoguide section to %s\n", path)
	return nil
}

// generateSection returns the full sentinel-wrapped repoguide documentation block.
func generateSection() string {
	body := `## repoguide — Repository Map

Run ` + "`repoguide`" + ` via the Bash tool at the start of any task on an unfamiliar
codebase. It produces a ranked map of files, symbols, and dependencies that
replaces the need for broad initial exploration.

**Availability:** Check with ` + "`repoguide --version`" + ` first; skip gracefully if
not found.

**Run it:**
` + "```" + `bash
repoguide                                    # current directory, all languages
repoguide /path/to/repo                      # explicit path
repoguide -l go,typescript                   # filter by language
repoguide -n 20                              # limit to top 20 files (large repos)
repoguide --cache .repoguide-cache           # cache output (fast on repeat runs)
repoguide --cache .repoguide-cache /repo     # cache + explicit path
` + "```" + `

**Caching:** Use ` + "`--cache <file>`" + ` to avoid re-parsing on every call — essential
for large repos. Add the cache file to ` + "`.gitignore`" + `. A conventional path is
` + "`.repoguide-cache`" + `.

**All flags:** ` + "`repoguide --help`" + `

**How to use the output — follow these rules:**

1. **Read files in ranked order.** The ` + "`files`" + ` table is sorted by PageRank
   (most central first). Read from the top down — do not start with arbitrary
   files or directory listings.

2. **Use ` + "`symbols`" + ` instead of Grep to find definitions.** Before running any
   search for a function, class, or method, check the ` + "`symbols`" + ` table. It
   lists every exported definition with file and line number.

3. **Use ` + "`dependencies`" + ` to trace call chains.** Before reading a file to
   understand what it calls or imports, check the ` + "`dependencies`" + ` table first.

4. **Only fall back to Glob/Grep for things repoguide cannot answer** — e.g.,
   finding all usages of a symbol, or searching within a file you've already
   identified.`

	return sentinelStart + "\n" + body + "\n" + sentinelEnd
}

// applySection inserts section into content, replacing an existing sentinel
// block if present or appending if not. It is a pure function for easy testing.
func applySection(content, section string) string {
	start := strings.Index(content, sentinelStart)
	end := strings.Index(content, sentinelEnd)

	if start >= 0 && end > start {
		return content[:start] + section + content[end+len(sentinelEnd):]
	}

	// Append, ensuring a blank line separator.
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + section + "\n"
}
