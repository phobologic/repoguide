package parse

import (
	"testing"

	"github.com/phobologic/repoguide/internal/lang"
	"github.com/phobologic/repoguide/internal/model"
)

func setup(t *testing.T, langName string) (*lang.Language, func(source string) []model.Tag) {
	t.Helper()
	l := lang.Languages[langName]
	if l == nil {
		t.Fatalf("language %q not registered", langName)
	}
	q, err := l.GetTagQuery()
	if err != nil {
		t.Fatalf("GetTagQuery: %v", err)
	}
	ext := l.Extensions[0]
	return l, func(source string) []model.Tag {
		p := l.NewParser()
		return ExtractTags(l, p, q, []byte(source), "test"+ext)
	}
}

// --- Python tests ---

func TestPythonExtractFunction(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

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

func TestPythonExtractClass(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

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

func TestPythonExtractMethod(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

	source := `class MyClass:
    def my_method(self, x: int) -> str:
        return str(x)
`
	tags := extract(source)
	defs := filterDefs(tags)

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

func TestPythonExtractImport(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

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

func TestPythonExtractCall(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

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

func TestPythonExtractEmpty(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

	tags := extract("")
	if len(tags) != 0 {
		t.Errorf("expected 0 tags for empty source, got %d", len(tags))
	}
}

// --- Go tests ---

func TestGoExtractFunction(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	tags := extract("package main\n\nfunc Hello(name string) error { return nil }\n")
	defs := filterDefs(tags)
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d: %+v", len(defs), defs)
	}
	d := defs[0]
	if d.Name != "Hello" {
		t.Errorf("name = %q, want Hello", d.Name)
	}
	if d.SymbolKind != model.Function {
		t.Errorf("kind = %q, want function", d.SymbolKind)
	}
	if d.Signature != "Hello(name string) error" {
		t.Errorf("sig = %q", d.Signature)
	}
}

func TestGoExtractMethod(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	source := `package main

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
}
`
	tags := extract(source)
	defs := filterDefs(tags)

	var method *model.Tag
	for i := range defs {
		if defs[i].SymbolKind == model.Method {
			method = &defs[i]
			break
		}
	}
	if method == nil {
		t.Fatalf("no method found in defs: %+v", defs)
	}
	if method.Name != "Server.Handle" {
		t.Errorf("name = %q, want Server.Handle", method.Name)
	}
	if method.Signature != "Handle(w http.ResponseWriter, r *http.Request)" {
		t.Errorf("sig = %q", method.Signature)
	}
}

func TestGoExtractType(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	source := `package main

type Server struct {
	Port int
}
`
	tags := extract(source)
	defs := filterDefs(tags)

	var classDef *model.Tag
	for i := range defs {
		if defs[i].SymbolKind == model.Class {
			classDef = &defs[i]
			break
		}
	}
	if classDef == nil {
		t.Fatalf("no class/type def found: %+v", defs)
	}
	if classDef.Name != "Server" {
		t.Errorf("name = %q, want Server", classDef.Name)
	}
	if classDef.Signature != "Server" {
		t.Errorf("sig = %q", classDef.Signature)
	}
}

func TestGoExtractCall(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	source := `package main

func main() {
	fmt.Println("hello")
	doStuff()
}
`
	tags := extract(source)
	refs := filterRefs(tags)

	names := make(map[string]bool)
	for _, r := range refs {
		names[r.Name] = true
	}
	if !names["Println"] {
		t.Error("missing call Println")
	}
	if !names["doStuff"] {
		t.Error("missing call doStuff")
	}
}

func TestGoExtractImport(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	source := `package main

import (
	"fmt"
	"os"
)
`
	tags := extract(source)
	refs := filterRefs(tags)

	names := make(map[string]bool)
	for _, r := range refs {
		names[r.Name] = true
	}
	if !names[`"fmt"`] {
		t.Errorf("missing import fmt, got names: %v", names)
	}
	if !names[`"os"`] {
		t.Errorf("missing import os, got names: %v", names)
	}
}

// --- Ruby tests ---

func TestRubyExtractClass(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	tags := extract("class Foo < Bar\nend\n")
	defs := filterDefs(tags)
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d: %+v", len(defs), defs)
	}
	d := defs[0]
	if d.Name != "Foo" {
		t.Errorf("name = %q, want Foo", d.Name)
	}
	if d.SymbolKind != model.Class {
		t.Errorf("kind = %q, want class", d.SymbolKind)
	}
	if d.Signature != "Foo < Bar" {
		t.Errorf("sig = %q", d.Signature)
	}
}

func TestRubyExtractModule(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	tags := extract("module Utils\nend\n")
	defs := filterDefs(tags)
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d: %+v", len(defs), defs)
	}
	d := defs[0]
	if d.Name != "Utils" {
		t.Errorf("name = %q, want Utils", d.Name)
	}
	if d.SymbolKind != model.Class {
		t.Errorf("kind = %q, want class", d.SymbolKind)
	}
}

func TestRubyExtractMethod(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	source := `def greet(name)
  puts "Hello, #{name}"
end
`
	tags := extract(source)
	defs := filterDefs(tags)
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d: %+v", len(defs), defs)
	}
	d := defs[0]
	if d.Name != "greet" {
		t.Errorf("name = %q, want greet", d.Name)
	}
	if d.SymbolKind != model.Function {
		t.Errorf("kind = %q, want function", d.SymbolKind)
	}
	if d.Signature != "greet(name)" {
		t.Errorf("sig = %q", d.Signature)
	}
}

func TestRubyExtractMethodInClass(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	source := `class MyClass
  def my_method(x)
    x + 1
  end
end
`
	tags := extract(source)
	defs := filterDefs(tags)

	var method *model.Tag
	for i := range defs {
		if defs[i].SymbolKind == model.Method {
			method = &defs[i]
			break
		}
	}
	if method == nil {
		t.Fatalf("no method found in defs: %+v", defs)
	}
	if method.Name != "MyClass.my_method" {
		t.Errorf("name = %q, want MyClass.my_method", method.Name)
	}
}

func TestRubyExtractSingletonMethod(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	source := `class Config
  def self.load(path)
    new(path)
  end
end
`
	tags := extract(source)
	defs := filterDefs(tags)

	var method *model.Tag
	for i := range defs {
		if defs[i].SymbolKind == model.Method {
			method = &defs[i]
			break
		}
	}
	if method == nil {
		t.Fatalf("no method found in defs: %+v", defs)
	}
	if method.Name != "Config.load" {
		t.Errorf("name = %q, want Config.load", method.Name)
	}
}

func TestRubyExtractCall(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	source := `puts "hello"
foo.bar(1)
`
	tags := extract(source)
	refs := filterRefs(tags)

	names := make(map[string]bool)
	for _, r := range refs {
		names[r.Name] = true
	}
	if !names["puts"] {
		t.Error("missing call puts")
	}
	if !names["bar"] {
		t.Error("missing call bar")
	}
}

// --- helpers ---

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
