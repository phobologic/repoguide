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
		{".go", ""},
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

	py, ok := Languages["python"]
	if !ok {
		t.Fatal("python language not registered")
	}
	if py.GetLanguage() == nil {
		t.Error("python language is nil")
	}
}

func TestNewParser(t *testing.T) {
	t.Parallel()

	py := Languages["python"]
	p := py.NewParser()
	if p == nil {
		t.Fatal("NewParser returned nil")
	}
}

func TestGetTagQuery(t *testing.T) {
	t.Parallel()

	py := Languages["python"]
	q, err := py.GetTagQuery()
	if err != nil {
		t.Fatalf("GetTagQuery: %v", err)
	}
	if q == nil {
		t.Fatal("query is nil")
	}
}
