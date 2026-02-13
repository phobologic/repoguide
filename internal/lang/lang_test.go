package lang

import (
	"testing"
)

func TestForExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ext  string
		want string
	}{
		{".py", "python"},
		{".go", "go"},
		{".rb", "ruby"},
		{".js", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			t.Parallel()
			got := ForExtension(tt.ext)
			if got != tt.want {
				t.Errorf("ForExtension(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}

func TestLanguagesRegistered(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"python", "go", "ruby"} {
		l, ok := Languages[name]
		if !ok {
			t.Errorf("%s language not registered", name)
			continue
		}
		if l.GetLanguage() == nil {
			t.Errorf("%s language is nil", name)
		}
	}
}

func TestNewParser(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"python", "go", "ruby"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			l := Languages[name]
			p := l.NewParser()
			if p == nil {
				t.Fatalf("NewParser returned nil for %s", name)
			}
		})
	}
}

func TestGetTagQuery(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"python", "go", "ruby"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			l := Languages[name]
			q, err := l.GetTagQuery()
			if err != nil {
				t.Fatalf("GetTagQuery for %s: %v", name, err)
			}
			if q == nil {
				t.Fatalf("query is nil for %s", name)
			}
		})
	}
}
