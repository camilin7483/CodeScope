package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string)
		want  string
	}{
		{
			name: "Go project",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
			},
			want: "Go",
		},
		{
			name: "JavaScript project",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
			},
			want: "JavaScript",
		},
		{
			name: "Rust project",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)
			},
			want: "Rust",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			p := DetectProject(dir)
			if string(p.Language) != tt.want {
				t.Errorf("DetectProject().Language = %q, want %q", p.Language, tt.want)
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"foo_test.go", true},
		{"foo.go", false},
		{"bar.test.js", true},
		{"bar.js", false},
		{"test_bar.py", true},
		{"bar.py", false},
	}

	for _, tt := range tests {
		got := IsTestFile(tt.path)
		if got != tt.want {
			t.Errorf("IsTestFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsDocFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"README.md", true},
		{"docs/guide.md", true},
		{"main.go", false},
		{"LICENSE", true},
		{"CHANGELOG.md", true},
	}

	for _, tt := range tests {
		got := IsDocFile(tt.path)
		if got != tt.want {
			t.Errorf("IsDocFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsConfigFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"go.mod", true},
		{"package.json", true},
		{"tsconfig.json", true},
		{"main.go", false},
		{".gitignore", true},
	}

	for _, tt := range tests {
		got := IsConfigFile(tt.path)
		if got != tt.want {
			t.Errorf("IsConfigFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestDetectFramework(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		lang  string
		want  string
	}{
		{"Go CLI", []string{"go.mod", "main.go"}, "Go", "Go CLI"},
	}

	dir := t.TempDir()
	for _, f := range tests[0].files {
		os.WriteFile(filepath.Join(dir, f), []byte(""), 0644)
	}
	p := DetectProject(dir)
	if p.Framework != tests[0].want {
		t.Errorf("Framework = %q, want %q", p.Framework, tests[0].want)
	}
}
