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

// --- Enclosing field tests ---

func TestGoCallEnclosing(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	tags := extract(`package main
func outer() {
	doStuff()
}
`)
	refs := filterRefs(tags)
	for _, r := range refs {
		if r.Name == "doStuff" {
			if r.Enclosing != "outer" {
				t.Errorf("Enclosing = %q, want outer", r.Enclosing)
			}
			return
		}
	}
	t.Error("doStuff call not found")
}

func TestGoMethodCallEnclosing(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	tags := extract(`package main
func (s *Server) Handle() {
	s.parse()
}
`)
	refs := filterRefs(tags)
	for _, r := range refs {
		if r.Name == "parse" {
			if r.Enclosing != "Server.Handle" {
				t.Errorf("Enclosing = %q, want Server.Handle", r.Enclosing)
			}
			return
		}
	}
	t.Error("parse call not found")
}

func TestGoTopLevelCallNoEnclosing(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	// Package-level variable initializer — call is outside any function.
	tags := extract(`package main

var x = foo()
`)
	refs := filterRefs(tags)
	for _, r := range refs {
		if r.Name == "foo" && r.Enclosing != "" {
			t.Errorf("top-level call should have empty Enclosing, got %q", r.Enclosing)
		}
	}
}

func TestGoClosureCallNoEnclosing(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	// Call inside a closure should not be attributed to outer().
	tags := extract(`package main
func outer() {
	f := func() {
		inner()
	}
	_ = f
}
`)
	refs := filterRefs(tags)
	for _, r := range refs {
		if r.Name == "inner" {
			if r.Enclosing != "" {
				t.Errorf("closure call Enclosing = %q, want empty (not attributed to outer)", r.Enclosing)
			}
			return
		}
	}
	t.Error("inner call not found")
}

func TestPythonMethodCallEnclosing(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

	tags := extract(`class MyClass:
    def method(self):
        helper()
`)
	refs := filterRefs(tags)
	for _, r := range refs {
		if r.Name == "helper" {
			if r.Enclosing != "MyClass.method" {
				t.Errorf("Enclosing = %q, want MyClass.method", r.Enclosing)
			}
			return
		}
	}
	t.Error("helper call not found")
}

func TestPythonTopLevelCallNoEnclosing(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

	tags := extract("foo()\n")
	refs := filterRefs(tags)
	for _, r := range refs {
		if r.Name == "foo" && r.Enclosing != "" {
			t.Errorf("top-level call should have empty Enclosing, got %q", r.Enclosing)
		}
	}
}

func TestRubyMethodCallEnclosing(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	tags := extract(`class MyClass
  def my_method
    helper()
  end
end
`)
	refs := filterRefs(tags)
	for _, r := range refs {
		if r.Name == "helper" {
			if r.Enclosing != "MyClass.my_method" {
				t.Errorf("Enclosing = %q, want MyClass.my_method", r.Enclosing)
			}
			return
		}
	}
	t.Error("helper call not found")
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

func filterFields(tags []model.Tag) []model.Tag {
	var out []model.Tag
	for _, t := range tags {
		if t.Kind == model.Definition && t.SymbolKind == model.Field {
			out = append(out, t)
		}
	}
	return out
}

// --- Field extraction tests ---

func TestGoStructFields(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	src := `package p

type Beat struct {
	ID       int
	SceneID  int
	Status   string
}
`
	tags := filterFields(extract(src))
	if len(tags) != 3 {
		t.Fatalf("expected 3 field tags, got %d: %+v", len(tags), tags)
	}
	byName := map[string]model.Tag{}
	for _, tag := range tags {
		byName[tag.Name] = tag
	}
	for _, tc := range []struct {
		name string
		sig  string
	}{
		{"Beat.ID", "ID int"},
		{"Beat.SceneID", "SceneID int"},
		{"Beat.Status", "Status string"},
	} {
		tag, ok := byName[tc.name]
		if !ok {
			t.Errorf("missing field %q", tc.name)
			continue
		}
		if tag.SymbolKind != model.Field {
			t.Errorf("%s: kind = %q, want field", tc.name, tag.SymbolKind)
		}
		if tag.Signature != tc.sig {
			t.Errorf("%s: sig = %q, want %q", tc.name, tag.Signature, tc.sig)
		}
	}
}

func TestGoInterfaceMethods(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	src := `package p

type Writer interface {
	Write(p []byte) (n int, err error)
	Close() error
}
`
	tags := filterFields(extract(src))
	if len(tags) != 2 {
		t.Fatalf("expected 2 interface method tags, got %d: %+v", len(tags), tags)
	}
	names := map[string]bool{}
	for _, tag := range tags {
		names[tag.Name] = true
	}
	if !names["Writer.Write"] || !names["Writer.Close"] {
		t.Errorf("unexpected field names: %v", names)
	}
}

func TestGoFieldsNotCapturedOutsideStruct(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "go")

	// Standalone function — no fields should be captured.
	src := `package p

func Foo(x int) string {
	return ""
}
`
	tags := filterFields(extract(src))
	if len(tags) != 0 {
		t.Errorf("expected no field tags outside struct, got %d: %+v", len(tags), tags)
	}
}

func TestPythonClassFields(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

	src := `class Beat:
    id: int = 0
    name = "default"
`
	tags := filterFields(extract(src))
	if len(tags) != 2 {
		t.Fatalf("expected 2 field tags, got %d: %+v", len(tags), tags)
	}
	byName := map[string]model.Tag{}
	for _, tag := range tags {
		byName[tag.Name] = tag
	}
	if tag, ok := byName["Beat.id"]; !ok {
		t.Error("missing Beat.id")
	} else if tag.Signature != "id: int" {
		t.Errorf("Beat.id sig = %q, want %q", tag.Signature, "id: int")
	}
	if _, ok := byName["Beat.name"]; !ok {
		t.Error("missing Beat.name")
	}
}

func TestPythonFieldsNotCapturedInMethod(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "python")

	// Assignments inside a method should NOT be captured as fields.
	src := `class Foo:
    def bar(self):
        x = 1
        return x
`
	tags := filterFields(extract(src))
	if len(tags) != 0 {
		t.Errorf("expected no field tags for method-local assignments, got %d: %+v", len(tags), tags)
	}
}

func TestRubyAttrFields(t *testing.T) {
	t.Parallel()
	_, extract := setup(t, "ruby")

	src := `class Beat
  attr_accessor :id, :name
  attr_reader :status
end
`
	tags := filterFields(extract(src))
	if len(tags) != 3 {
		t.Fatalf("expected 3 field tags, got %d: %+v", len(tags), tags)
	}
	names := map[string]bool{}
	for _, tag := range tags {
		names[tag.Name] = true
	}
	for _, want := range []string{"Beat.id", "Beat.name", "Beat.status"} {
		if !names[want] {
			t.Errorf("missing field %q; got %v", want, names)
		}
	}
}
