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
		_, _ = fmt.Fprintf(stderr, `Usage: repoguide init [flags] [path-to-CLAUDE.md]

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

	existing, readErr := os.ReadFile(path)
	updated := applySection(string(existing), section)

	if dryRun {
		_, _ = fmt.Fprint(stdout, updated)
		return nil
	}

	if updated == string(existing) {
		_, _ = fmt.Fprintf(stderr, "%s is already up to date\n", path)
		return nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	if readErr != nil {
		_, _ = fmt.Fprintf(stderr, "created %s\n", path)
	} else {
		_, _ = fmt.Fprintf(stderr, "updated %s\n", path)
	}
	return nil
}

// generateSection returns the full sentinel-wrapped repoguide documentation block.
func generateSection() string {
	body := `## repoguide — Repository Map

Run ` + "`repoguide`" + ` via the Bash tool at the start of any task. It produces a ranked
map of files, symbols, and dependencies that replaces the need for broad initial
exploration.

**Availability:** Run ` + "`repoguide --cache .repoguide-cache`" + ` directly. If the
command fails (not installed or no supported files found), skip using the output
and proceed without it. Place ` + "`.repoguide-cache`" + ` at the project root and add it
to ` + "`.gitignore`" + `.

**When to run it:** Run it directly via Bash before launching any subagents —
even in plan mode, where it counts as a permitted read-only action. Do not send
Explore agents to discover structure that repoguide already provides.

**Sharing with subagents:** If subagents are needed after running repoguide,
include the ranked file list and relevant symbols in their prompt so they do not
re-explore what you already have.

**Run it:**
` + "```" + `bash
repoguide                                    # current directory, all languages
repoguide /path/to/repo                      # explicit path
repoguide -l go,typescript                   # filter by language
repoguide -n 20                              # limit to top 20 files (large repos)
repoguide --cache .repoguide-cache           # cache output (fast on repeat runs)
repoguide --cache .repoguide-cache /repo     # cache + explicit path

repoguide --with-tests                       # include test files (excluded by default)
repoguide --symbol BuildGraph                # focused: symbol + its callers/callees
repoguide --file internal/auth               # focused: symbols and deps for a path
repoguide --symbol Handle --file server      # focused: combine filters (AND)
` + "```" + `

**Caching:** Filter flags (` + "`--symbol`" + `, ` + "`--file`" + `) bypass the cache read
but the full output is still written to cache, so subsequent full runs stay fast.

**All flags:** ` + "`repoguide --help`" + `

**How to use the output — two phases:**

**Phase 1 — run once at session start.** The full map orients you to the codebase.
Read the ` + "`files`" + ` table top-to-bottom: PageRank order shows which files are most
central. Start reading there, not from directory listings or arbitrary guesses.

**Phase 2 — run ` + "`--symbol`" + ` or ` + "`--file`" + ` for specific mid-task lookups** (rules 2–7
below). These bypass the cache read but write back to it, so full runs stay fast.

**Rules:**

1. **Read files in ranked order.** The ` + "`files`" + ` table is sorted by PageRank
   (most central first). Do not start from directory listings.

2. **Before running Grep to find where an exported name is used, run
   ` + "`repoguide --symbol <name>`" + ` first.** If the name is indexed, ` + "`--symbol`" + `
   returns structured file+line data you can feed directly to
   ` + "`Read(offset=N, limit=10)`" + `. Fall back to Grep only if ` + "`--symbol`" + ` returns no
   results or the name is unexported/not a definition. **Never pipe
   ` + "`--symbol`" + ` or ` + "`--file`" + ` output through ` + "`head`" + ` or ` + "`tail`" + `** — the complete
   output is the value; truncating it loses callsites.

3. **` + "`--symbol`" + ` returns call sites AND import sites for a name.** The ` + "`callsites`" + `
   table includes every function-call occurrence (` + "`caller`" + ` = calling function) and
   every file-level import (` + "`caller`" + ` = ` + "`<import>`" + `), each with exact file+line.
   The ` + "`dependencies`" + ` table shows which files import the file that defines the
   symbol. Together these answer both "who calls this?" and "who imports this?".
   Use the line numbers directly with ` + "`Read(offset=N, limit=10)`" + ` rather than
   scanning raw Grep output.

4. **To find all files that depend on a module or path, use ` + "`--file <path>`" + `.** The
   ` + "`dependencies`" + ` table in the output lists every importer: ` + "`source`" + ` imports
   ` + "`target`" + `. Example: ` + "`repoguide --file loom/ai/stubs`" + ` shows every file that
   imports from that module. Use this for "I'm replacing X — who depends on it?"
   instead of Grep.

5. **Use the ` + "`callsites`" + ` table for precise file navigation.** Focused queries
   (` + "`--symbol`" + ` or ` + "`--file`" + `) include a ` + "`callsites[N]{caller,callee,file,line}`" + ` table
   with the exact line of every call occurrence. Use those line numbers for
   ` + "`Read(offset=N, limit=10)`" + ` instead of scanning from a rough offset.

6. **Use the ` + "`symbols`" + ` table as a lookup index, not a scanning surface.** It
   lists every exported definition with file and line. Look up a name you already
   know — don't scroll through it searching for something.

7. **Use ` + "`--file`" + ` when focused on a subsystem.** ` + "`repoguide --file internal/auth`" + `
   gives all symbols and dependencies for that path without full-map noise.
   Combine with ` + "`--symbol`" + ` (AND semantics) when a name appears across packages.

8. **Fall back to Grep for: string literal searches, unexported names (repoguide
   only indexes exported definitions), or searching within a file you've already
   identified.**

9. **Re-run after large structural changes.** The map is a snapshot. If you've
   added new files or significantly restructured imports, re-run to refresh it.`

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
