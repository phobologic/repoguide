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
	if !strings.Contains(out, "# Repository Map") {
		t.Error("missing agent context header")
	}
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

func TestRunRaw(t *testing.T) {
	t.Parallel()
	dir := createSampleRepo(t)

	var stdout, stderr bytes.Buffer
	err := run([]string{"--raw", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if strings.Contains(out, "# Repository Map") {
		t.Error("--raw should suppress agent context header")
	}
	if !strings.HasPrefix(out, "repo:") {
		t.Errorf("--raw output should start with repo:, got:\n%s", out)
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

	// Cache file should contain raw TOON (no header)
	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("reading cache: %v", err)
	}
	if strings.Contains(string(cacheData), "# Repository Map") {
		t.Error("cache file should not contain agent header")
	}

	// Second run should use cache and include header
	var stdout2, stderr2 bytes.Buffer
	err = run([]string{"--cache", cachePath, dir}, &stdout2, &stderr2)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	if stdout1.String() != stdout2.String() {
		t.Errorf("cache mismatch:\nfirst:\n%s\nsecond:\n%s", stdout1.String(), stdout2.String())
	}

	if !strings.Contains(stdout2.String(), "# Repository Map") {
		t.Error("cached output should include agent header")
	}
}

func TestRunCacheRaw(t *testing.T) {
	t.Parallel()
	dir := createSampleRepo(t)
	cachePath := filepath.Join(t.TempDir(), "test.cache")

	// First run populates cache (with header in output)
	var stdout1, stderr1 bytes.Buffer
	err := run([]string{"--cache", cachePath, dir}, &stdout1, &stderr1)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Second run with --raw should use cache but suppress header
	var stdout2, stderr2 bytes.Buffer
	err = run([]string{"--raw", "--cache", cachePath, dir}, &stdout2, &stderr2)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	out := stdout2.String()
	if strings.Contains(out, "# Repository Map") {
		t.Error("--raw with cache should suppress agent header")
	}
	if !strings.HasPrefix(out, "repo:") {
		t.Errorf("--raw cached output should start with repo:, got:\n%s", out)
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

func TestRunCalls(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "utils.py", `def helper():
    pass
`)
	writeTestFile(t, dir, "main.py", `def greet():
    helper()
`)

	var stdout, stderr bytes.Buffer
	err := run([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "calls[") {
		t.Errorf("missing calls section:\n%s", out)
	}
	if !strings.Contains(out, "greet,helper") {
		t.Errorf("missing greet→helper call edge:\n%s", out)
	}
}

func TestRunSymbolFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "utils.py", `def helper():
    pass
`)
	writeTestFile(t, dir, "main.py", `def greet():
    helper()
`)

	var stdout, stderr bytes.Buffer
	err := run([]string{"--symbol", "helper", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "utils.py") {
		t.Errorf("utils.py (defines helper) should be in output:\n%s", out)
	}
	// greet calls helper, so main.py should also be included via call expansion.
	if !strings.Contains(out, "main.py") {
		t.Errorf("main.py (defines greet which calls helper) should be in output:\n%s", out)
	}
	if !strings.Contains(out, "helper,function") {
		t.Errorf("helper definition should appear in symbols:\n%s", out)
	}
}

func TestRunSymbolFilterNoMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "main.py", "def greet():\n    pass\n")

	var stdout, stderr bytes.Buffer
	err := run([]string{"--symbol", "NoSuchSymbol", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "files[0]") {
		t.Errorf("expected empty files table:\n%s", out)
	}
}

func TestRunFileFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "utils.py", "def helper():\n    pass\n")
	writeTestFile(t, dir, "main.py", "def greet():\n    helper()\n")

	var stdout, stderr bytes.Buffer
	err := run([]string{"--file", "utils", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "utils.py") {
		t.Errorf("utils.py should be in output:\n%s", out)
	}
	if strings.Contains(out, "main.py") && !strings.Contains(out, "dependencies") {
		// main.py only appears if it's in a dep edge, not as a file row
		t.Errorf("main.py should not appear as a file:\n%s", out)
	}
	if !strings.Contains(out, "helper,function") {
		t.Errorf("helper definition should appear:\n%s", out)
	}
}

func TestRunSymbolAndFileFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "pkg/utils.py", "def helper():\n    pass\n")
	writeTestFile(t, dir, "other/utils.py", "def other_helper():\n    pass\n")

	var stdout, stderr bytes.Buffer
	err := run([]string{"--symbol", "helper", "--file", "pkg", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	// --file pkg filters to pkg/utils.py; --symbol helper must be in that file.
	if !strings.Contains(out, "pkg/utils.py") {
		t.Errorf("pkg/utils.py should be in output:\n%s", out)
	}
}

func TestRunSymbolFilterCacheSkipped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "main.py", "def greet():\n    pass\n")
	cachePath := filepath.Join(t.TempDir(), "cache.toon")

	// First run: no filter, write cache.
	var stdout1 bytes.Buffer
	if err := run([]string{"--cache", cachePath, dir}, &stdout1, &bytes.Buffer{}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache not written: %v", err)
	}

	// Second run: with --symbol filter. Cache should be bypassed (filter still works).
	var stdout2 bytes.Buffer
	if err := run([]string{"--symbol", "greet", "--cache", cachePath, dir}, &stdout2, &bytes.Buffer{}); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(stdout2.String(), "greet") {
		t.Errorf("filter should work even when cache exists:\n%s", stdout2.String())
	}
}

func TestRunSymbolFilterCallSites(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "utils.py", "def helper():\n    pass\n")
	writeTestFile(t, dir, "main.py", "def greet():\n    helper()\n    helper()\n")

	var stdout, stderr bytes.Buffer
	err := run([]string{"--symbol", "helper", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "callsites") {
		t.Errorf("--symbol output should include callsites table:\n%s", out)
	}
}

func TestRunFullMapNoCallSites(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "utils.py", "def helper():\n    pass\n")
	writeTestFile(t, dir, "main.py", "def greet():\n    helper()\n")

	var stdout, stderr bytes.Buffer
	err := run([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	if strings.Contains(stdout.String(), "callsites[") {
		t.Errorf("full map output should not include callsites table:\n%s", stdout.String())
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
