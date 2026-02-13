package parse

import (
	"testing"

	"github.com/phobologic/repoguide/internal/lang"
	"github.com/phobologic/repoguide/internal/model"
)

func setup(t *testing.T) (*lang.Language, func(source string) []model.Tag) {
	t.Helper()
	py := lang.Languages["python"]
	q, err := py.GetTagQuery()
	if err != nil {
		t.Fatalf("GetTagQuery: %v", err)
	}
	return py, func(source string) []model.Tag {
		p := py.NewParser()
		return ExtractTags(p, q, []byte(source), "test.py")
	}
}

func TestExtractFunction(t *testing.T) {
	t.Parallel()
	_, extract := setup(t)

	tags := extract("def hello(name: str) -> None:\n    pass\n")
	defs := filterDefs(tags)
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}
	d := defs[0]
	if d.Name != "hello" {
		t.Errorf("name = %q, want hello", d.Name)
	}
	if d.SymbolKind != model.Function {
		t.Errorf("kind = %q, want function", d.SymbolKind)
	}
	if d.Line != 1 {
		t.Errorf("line = %d, want 1", d.Line)
	}
	if d.Signature != "hello(name: str) -> None" {
		t.Errorf("sig = %q", d.Signature)
	}
}

func TestExtractClass(t *testing.T) {
	t.Parallel()
	_, extract := setup(t)

	tags := extract("class Foo(Base):\n    pass\n")
	defs := filterDefs(tags)
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}
	d := defs[0]
	if d.Name != "Foo" {
		t.Errorf("name = %q, want Foo", d.Name)
	}
	if d.SymbolKind != model.Class {
		t.Errorf("kind = %q, want class", d.SymbolKind)
	}
	if d.Signature != "Foo(Base)" {
		t.Errorf("sig = %q", d.Signature)
	}
}

func TestExtractMethod(t *testing.T) {
	t.Parallel()
	_, extract := setup(t)

	source := `class MyClass:
    def my_method(self, x: int) -> str:
        return str(x)
`
	tags := extract(source)
	defs := filterDefs(tags)

	// Should have class def and method def
	if len(defs) < 2 {
		t.Fatalf("expected >= 2 defs, got %d: %+v", len(defs), defs)
	}

	var method *model.Tag
	for i := range defs {
		if defs[i].SymbolKind == model.Method {
			method = &defs[i]
			break
		}
	}
	if method == nil {
		t.Fatal("no method found")
	}
	if method.Name != "MyClass.my_method" {
		t.Errorf("name = %q, want MyClass.my_method", method.Name)
	}
	if method.Signature != "my_method(self, x: int) -> str" {
		t.Errorf("sig = %q", method.Signature)
	}
}

func TestExtractImport(t *testing.T) {
	t.Parallel()
	_, extract := setup(t)

	tags := extract("import os\nfrom pathlib import Path\n")
	refs := filterRefs(tags)
	if len(refs) < 2 {
		t.Fatalf("expected >= 2 refs, got %d", len(refs))
	}

	names := make(map[string]bool)
	for _, r := range refs {
		names[r.Name] = true
	}
	if !names["os"] {
		t.Error("missing import os")
	}
	if !names["Path"] {
		t.Error("missing import Path")
	}
}

func TestExtractCall(t *testing.T) {
	t.Parallel()
	_, extract := setup(t)

	tags := extract("x = foo()\ny = bar.baz()\n")
	refs := filterRefs(tags)

	names := make(map[string]bool)
	for _, r := range refs {
		names[r.Name] = true
	}
	if !names["foo"] {
		t.Error("missing call foo")
	}
	if !names["baz"] {
		t.Error("missing attribute call baz")
	}
}

func TestExtractEmpty(t *testing.T) {
	t.Parallel()
	_, extract := setup(t)

	tags := extract("")
	if len(tags) != 0 {
		t.Errorf("expected 0 tags for empty source, got %d", len(tags))
	}
}

func filterDefs(tags []model.Tag) []model.Tag {
	var out []model.Tag
	for _, t := range tags {
		if t.Kind == model.Definition {
			out = append(out, t)
		}
	}
	return out
}

func filterRefs(tags []model.Tag) []model.Tag {
	var out []model.Tag
	for _, t := range tags {
		if t.Kind == model.Reference {
			out = append(out, t)
		}
	}
	return out
}
