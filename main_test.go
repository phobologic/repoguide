package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func createSampleRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, dir, "models.py", `class User:
    def __init__(self, name: str) -> None:
        self.name = name
`)
	writeTestFile(t, dir, "main.py", `from models import User

def greet(user: User) -> str:
    return f"Hello, {user.name}"
`)
	return dir
}

func TestRunBasic(t *testing.T) {
	t.Parallel()
	dir := createSampleRepo(t)

	var stdout, stderr bytes.Buffer
	err := run([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "repo:") {
		t.Error("missing repo: header")
	}
	if !strings.Contains(out, "files[2]") {
		t.Errorf("expected 2 files, got:\n%s", out)
	}
	if !strings.Contains(out, "models.py") {
		t.Error("missing models.py")
	}
	if !strings.Contains(out, "main.py") {
		t.Error("missing main.py")
	}
}

func TestRunMaxFiles(t *testing.T) {
	t.Parallel()
	dir := createSampleRepo(t)

	var stdout, stderr bytes.Buffer
	err := run([]string{"-n", "1", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "files[1]") {
		t.Errorf("expected 1 file, got:\n%s", out)
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := run([]string{"-V"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "repoguide") {
		t.Errorf("version output: %q", stdout.String())
	}
}

func TestRunNoFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "readme.txt", "nothing here")

	var stdout, stderr bytes.Buffer
	err := run([]string{dir}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no parseable files")
	}
	if !strings.Contains(err.Error(), "no parseable files") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := run([]string{"-l", "rust", t.TempDir()}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunCache(t *testing.T) {
	t.Parallel()
	dir := createSampleRepo(t)
	cachePath := filepath.Join(t.TempDir(), "test.cache")

	var stdout1, stderr1 bytes.Buffer
	err := run([]string{"--cache", cachePath, dir}, &stdout1, &stderr1)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Cache file should exist
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache not created: %v", err)
	}

	// Second run should use cache
	var stdout2, stderr2 bytes.Buffer
	err = run([]string{"--cache", cachePath, dir}, &stdout2, &stderr2)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	// Output should be the same (cache has trailing newline, stdout has trailing newline from Fprintln)
	if stdout1.String() != stdout2.String() {
		t.Errorf("cache mismatch:\nfirst:\n%s\nsecond:\n%s", stdout1.String(), stdout2.String())
	}
}

func TestRunSymbols(t *testing.T) {
	t.Parallel()
	dir := createSampleRepo(t)

	var stdout, stderr bytes.Buffer
	err := run([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	out := stdout.String()
	// Check for class definition
	if !strings.Contains(out, "User,class") {
		t.Error("missing User class definition")
	}
	// Check for method
	if !strings.Contains(out, "User.__init__,method") {
		t.Error("missing User.__init__ method")
	}
	// Check for function
	if !strings.Contains(out, "greet,function") {
		t.Error("missing greet function")
	}
}

func TestRunDependencies(t *testing.T) {
	t.Parallel()
	dir := createSampleRepo(t)

	var stdout, stderr bytes.Buffer
	err := run([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	out := stdout.String()
	// main.py references User from models.py
	if !strings.Contains(out, "main.py,models.py,User") {
		t.Errorf("missing dependency main.py → models.py:\n%s", out)
	}
}

func TestRunNotADirectory(t *testing.T) {
	t.Parallel()
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := run([]string{f}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for non-directory")
	}
}

func TestRunMaxFileSize(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "small.py", "x = 1")
	writeTestFile(t, dir, "big.py", strings.Repeat("x = 1\n", 200))

	var stdout, stderr bytes.Buffer
	err := run([]string{"--max-file-size", "100", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "small.py") {
		t.Error("missing small.py")
	}
	if strings.Contains(out, "big.py") {
		t.Error("big.py should be filtered out")
	}
	if !strings.Contains(stderr.String(), "Warning") {
		t.Error("expected warning about skipped file")
	}
}

func TestReorderArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"flags first", []string{"-n", "5", "."}, []string{"-n", "5", "."}},
		{"positional first", []string{".", "-n", "5"}, []string{"-n", "5", "."}},
		{"mixed", []string{"-l", "python", ".", "-n", "5"}, []string{"-l", "python", "-n", "5", "."}},
		{"multi-lang", []string{"-l", "go,ruby", "."}, []string{"-l", "go,ruby", "."}},
		{"no flags", []string{"."}, []string{"."}},
		{"no args", nil, nil},
		{"bool flag", []string{"-V"}, []string{"-V"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := reorderArgs(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len: got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q (full: %v)", i, got[i], tt.want[i], got)
					break
				}
			}
		})
	}
}
