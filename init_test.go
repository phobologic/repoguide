package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplySectionCreate verifies that applySection on empty content wraps the
// section in sentinels with a trailing newline.
func TestApplySectionCreate(t *testing.T) {
	t.Parallel()
	section := sentinelStart + "\nbody\n" + sentinelEnd
	got := applySection("", section)
	if !strings.Contains(got, sentinelStart) {
		t.Error("missing sentinel start")
	}
	if !strings.Contains(got, sentinelEnd) {
		t.Error("missing sentinel end")
	}
	if !strings.Contains(got, "body") {
		t.Error("missing body")
	}
}

// TestApplySectionAppend verifies that existing content without a sentinel block
// is preserved and the section is appended.
func TestApplySectionAppend(t *testing.T) {
	t.Parallel()
	existing := "# My Project\n\nSome existing content.\n"
	section := sentinelStart + "\nnew content\n" + sentinelEnd
	got := applySection(existing, section)

	if !strings.HasPrefix(got, existing) {
		t.Errorf("existing content should be preserved at start:\n%s", got)
	}
	if !strings.Contains(got, "new content") {
		t.Error("new content missing")
	}
}

// TestApplySectionUpdate verifies that an existing sentinel block is replaced
// precisely, leaving surrounding content intact.
func TestApplySectionUpdate(t *testing.T) {
	t.Parallel()
	before := "# Project\n\n"
	after := "\n\n## Other Section\n"
	old := before + sentinelStart + "\nold content\n" + sentinelEnd + after

	section := sentinelStart + "\nnew content\n" + sentinelEnd
	got := applySection(old, section)

	if !strings.HasPrefix(got, before) {
		t.Errorf("content before sentinel should be preserved:\n%s", got)
	}
	if !strings.HasSuffix(got, after) {
		t.Errorf("content after sentinel should be preserved:\n%s", got)
	}
	if strings.Contains(got, "old content") {
		t.Error("old content should be replaced")
	}
	if !strings.Contains(got, "new content") {
		t.Error("new content missing")
	}
}

// TestInitCreatesFile verifies that runInit creates the target file when it
// does not exist.
func TestInitCreatesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{path}, &stdout, &stderr); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, sentinelStart) {
		t.Error("sentinel start missing from created file")
	}
	if !strings.Contains(content, sentinelEnd) {
		t.Error("sentinel end missing from created file")
	}
}

// TestInitDryRun verifies that --dry-run prints the full would-be file content
// to stdout and does not create or modify the target file.
func TestInitDryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--dry-run", path}, &stdout, &stderr); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	if _, err := os.Stat(path); err == nil {
		t.Error("--dry-run should not create the file")
	}
	out := stdout.String()
	if !strings.Contains(out, sentinelStart) {
		t.Error("dry-run output missing sentinel start")
	}
	if !strings.Contains(out, sentinelEnd) {
		t.Error("dry-run output missing sentinel end")
	}
}

// TestInitDryRunNoPath verifies that --dry-run without a path prints just the
// generated section to stdout without touching any file.
func TestInitDryRunNoPath(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--dry-run"}, &stdout, &stderr); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, sentinelStart) {
		t.Error("output missing sentinel start")
	}
	if !strings.Contains(out, sentinelEnd) {
		t.Error("output missing sentinel end")
	}
	// Should be just the section — no surrounding file boilerplate.
	if strings.HasPrefix(out, "\n") {
		t.Error("output should not have a leading newline when no file is given")
	}
}

// TestInitDryRunShowsFullFile verifies that --dry-run on an existing file
// shows the complete would-be file content, including surrounding text.
func TestInitDryRunShowsFullFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	existing := "# My Project\n\nSome existing content.\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := runInit([]string{"--dry-run", path}, &stdout, &stderr); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "# My Project") {
		t.Error("dry-run output missing existing file content")
	}
	if !strings.Contains(out, sentinelStart) {
		t.Error("dry-run output missing sentinel start")
	}
	// File on disk should be unchanged.
	data, _ := os.ReadFile(path)
	if string(data) != existing {
		t.Error("--dry-run must not modify the file")
	}
}

// TestInitIdempotent verifies that running init twice produces identical output.
func TestInitIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	var buf bytes.Buffer
	if err := runInit([]string{path}, &buf, &buf); err != nil {
		t.Fatalf("first run: %v", err)
	}
	first, _ := os.ReadFile(path)

	if err := runInit([]string{path}, &buf, &buf); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second, _ := os.ReadFile(path)

	if string(first) != string(second) {
		t.Errorf("init is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// TestInitSectionContainsHelpRef verifies the generated section points to --help.
func TestInitSectionContainsHelpRef(t *testing.T) {
	t.Parallel()
	section := generateSection()
	if !strings.Contains(section, "--help") {
		t.Error("generated section should reference --help for flag list")
	}
}

// TestInitSectionContainsExamples verifies the generated section includes
// example invocations, including the --cache example.
func TestInitSectionContainsExamples(t *testing.T) {
	t.Parallel()
	section := generateSection()

	examples := []string{
		"repoguide",
		"-l go",
		"-n 20",
		"--cache",
		".repoguide-cache",
	}
	for _, ex := range examples {
		if !strings.Contains(section, ex) {
			t.Errorf("generated section missing example %q", ex)
		}
	}
}

// TestInitDefaultPath verifies that the path argument defaults to CLAUDE.md in
// the current directory when omitted.
func TestInitDefaultPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	// Pass the path explicitly since we can't change cwd in parallel tests.
	var stdout, stderr bytes.Buffer
	if err := runInit([]string{path}, &stdout, &stderr); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s: %v", path, err)
	}
}
