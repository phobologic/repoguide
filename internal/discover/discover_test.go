package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverPythonFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create Python files
	writeFile(t, dir, "main.py", "print('hello')")
	writeFile(t, dir, "lib/util.py", "def helper(): pass")
	// Non-Python file should be ignored
	writeFile(t, dir, "readme.txt", "hello")
	// Hidden file should be ignored
	writeFile(t, dir, ".hidden.py", "secret")

	entries, err := Files(dir, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), paths)
	}

	// Should be sorted
	if entries[0].Path != filepath.Join("lib", "util.py") {
		t.Errorf("entry 0: got %q", entries[0].Path)
	}
	if entries[1].Path != "main.py" {
		t.Errorf("entry 1: got %q", entries[1].Path)
	}

	for _, e := range entries {
		if e.Language != "python" {
			t.Errorf("entry %q: language = %q, want python", e.Path, e.Language)
		}
	}
}

func TestDiscoverSkipDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "main.py", "pass")
	writeFile(t, dir, "node_modules/pkg.py", "pass")
	writeFile(t, dir, "__pycache__/cached.py", "pass")
	writeFile(t, dir, ".hidden/secret.py", "pass")

	entries, err := Files(dir, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Path != "main.py" {
		t.Errorf("expected main.py, got %q", entries[0].Path)
	}
}

func TestDiscoverLanguageFilter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "main.py", "pass")
	writeFile(t, dir, "lib.py", "pass")

	entries, err := Files(dir, []string{"python"})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for python filter, got %d", len(entries))
	}

	entries, err = Files(dir, []string{"javascript"})
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for javascript filter, got %d", len(entries))
	}
}

func TestDiscoverSymlinksSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "real.py", "pass")

	// Create symlink
	err := os.Symlink(filepath.Join(dir, "real.py"), filepath.Join(dir, "link.py"))
	if err != nil {
		t.Skip("symlinks not supported")
	}

	entries, err := Files(dir, nil)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (no symlink), got %d", len(entries))
	}
	if entries[0].Path != "real.py" {
		t.Errorf("expected real.py, got %q", entries[0].Path)
	}
}

func TestIsTestFile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want bool
	}{
		// Test directory components
		{"tests/test_scenes.py", true},
		{"tests/conftest.py", true},
		{"tests/__init__.py", true},
		{"spec/models/user_spec.rb", true},
		{"src/__tests__/foo.js", true},
		{"src/test/java/FooTest.java", true},
		{"test/foo_test.exs", true},
		// Filename patterns
		{"internal/graph/graph_test.go", true},
		{"test_helpers.py", true},
		{"user_spec.rb", true},
		{"foo.test.js", true},
		{"foo.spec.ts", true},
		// Production files
		{"loom/models.py", false},
		{"loom/routers/scenes.py", false},
		{"internal/graph/graph.go", false},
		{"conftest.py", false},      // top-level conftest, not in tests/
		{"testing_utils.go", false}, // contains "testing" but not a test pattern
		{"loom/database.py", false},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			got := IsTestFile(tc.path)
			if got != tc.want {
				t.Errorf("IsTestFile(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func writeFile(t *testing.T, root, rel, content string) {
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
