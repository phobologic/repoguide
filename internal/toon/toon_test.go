package toon

import (
	"strings"
	"testing"

	"github.com/phobologic/repoguide/internal/model"
)

func TestEncodeValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", `""`},
		{"simple", "hello", "hello"},
		{"leading space", " hello", `" hello"`},
		{"trailing space", "hello ", `"hello "`},
		{"newline", "a\nb", `"a\nb"`},
		{"tab", "a\tb", `"a\tb"`},
		{"carriage return", "a\rb", `"a\rb"`},
		{"true keyword", "true", `"true"`},
		{"True keyword", "True", `"True"`},
		{"false keyword", "false", `"false"`},
		{"null keyword", "null", `"null"`},
		{"integer", "42", "42"},
		{"negative integer", "-1", "-1"},
		{"float", "3.14", "3.14"},
		{"zero", "0", "0"},
		{"leading zero invalid", "01", "01"},
		{"comma", "a,b", `"a,b"`},
		{"colon", "a:b", `"a:b"`},
		{"quote", `a"b`, `"a\"b"`},
		{"backslash", `a\b`, `"a\\b"`},
		{"bracket", "a[b", `"a[b"`},
		{"brace", "a{b", `"a{b"`},
		{"dash prefix", "-foo", `"-foo"`},
		{"path", "src/main.py", "src/main.py"},
		{"dotted name", "Foo.__init__", "Foo.__init__"},
		{"signature no special", "run(self) -> None", "run(self) -> None"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := encodeValue(tt.in)
			if got != tt.want {
				t.Errorf("encodeValue(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	t.Parallel()

	rm := &model.RepoMap{
		RepoName: "myrepo",
		Root:     "myrepo",
		Files: []model.FileInfo{
			{
				Path:     "src/main.py",
				Language: "python",
				Rank:     0.75,
				Tags: []model.Tag{
					{
						Name:       "main",
						Kind:       model.Definition,
						SymbolKind: model.Function,
						Line:       1,
						Signature:  "main()",
					},
					{
						Name:       "helper",
						Kind:       model.Reference,
						SymbolKind: model.Function,
						Line:       5,
					},
				},
			},
			{
				Path:     "src/util.py",
				Language: "python",
				Rank:     0.25,
				Tags: []model.Tag{
					{
						Name:       "helper",
						Kind:       model.Definition,
						SymbolKind: model.Function,
						Line:       1,
						Signature:  "helper(x)",
					},
				},
			},
		},
		Dependencies: []model.Dependency{
			{
				Source:  "src/main.py",
				Target:  "src/util.py",
				Symbols: []string{"helper"},
			},
		},
	}

	got := Encode(rm)

	// Verify structure
	lines := strings.Split(got, "\n")
	if lines[0] != "repo: myrepo" {
		t.Errorf("line 0: got %q", lines[0])
	}
	if lines[1] != "root: myrepo" {
		t.Errorf("line 1: got %q", lines[1])
	}
	if lines[2] != "files[2]{path,language,rank}:" {
		t.Errorf("line 2: got %q", lines[2])
	}
	if lines[3] != "  src/main.py,python,0.7500" {
		t.Errorf("line 3: got %q", lines[3])
	}
	if lines[4] != "  src/util.py,python,0.2500" {
		t.Errorf("line 4: got %q", lines[4])
	}
	// symbols should only include definitions (2 total, not the reference)
	if lines[5] != "symbols[2]{file,name,kind,line,signature}:" {
		t.Errorf("line 5: got %q", lines[5])
	}
	if lines[6] != "  src/main.py,main,function,1,main()" {
		t.Errorf("line 6: got %q", lines[6])
	}
	if lines[7] != "  src/util.py,helper,function,1,helper(x)" {
		t.Errorf("line 7: got %q", lines[7])
	}
	if lines[8] != "dependencies[1]{source,target,symbols}:" {
		t.Errorf("line 8: got %q", lines[8])
	}
	if lines[9] != "  src/main.py,src/util.py,helper" {
		t.Errorf("line 9: got %q", lines[9])
	}
}

func TestEncodeEmpty(t *testing.T) {
	t.Parallel()

	rm := &model.RepoMap{
		RepoName: "empty",
		Root:     "empty",
	}

	got := Encode(rm)
	if !strings.Contains(got, "files[0]{path,language,rank}:") {
		t.Errorf("expected empty files section, got:\n%s", got)
	}
	if !strings.Contains(got, "symbols[0]{file,name,kind,line,signature}:") {
		t.Errorf("expected empty symbols section, got:\n%s", got)
	}
}
