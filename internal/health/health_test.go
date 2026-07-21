package health

import (
	"os"
	"path/filepath"
	"testing"

	"codescope/internal/types"
)

func TestCheckComplete(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(dir, "LICENSE"), []byte("MIT"), 0644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	project := types.Project{
		Root:        dir,
		Name:        "test",
		Language:    types.LanguageGo,
		BuildSystem: "Go (built-in)",
		HasGit:      true,
	}

	files := []types.FileInfo{
		{Path: "main.go", Category: types.CategorySource, Lines: 50},
		{Path: "main_test.go", Category: types.CategoryTest, Lines: 20},
	}

	report := Check(project, files)

	if report.Score < 50 {
		t.Errorf("Health score too low: %d", report.Score)
	}

	if report.CriticalTotal == 0 {
		t.Error("Expected critical checks")
	}

	foundReadme := false
	for _, c := range report.Checks {
		if c.Status == types.HealthPass && c.Name == "README.md" {
			foundReadme = true
			break
		}
	}
	if !foundReadme {
		t.Error("README check should pass")
	}
}

func TestCheckMissingFiles(t *testing.T) {
	dir := t.TempDir()
	project := types.Project{
		Root: dir,
		Name: "empty",
	}

	report := Check(project, nil)

	if report.Score > 80 {
		t.Errorf("Score should be low for empty project, got %d", report.Score)
	}

	hasCriticalFail := false
	for _, c := range report.Checks {
		if c.Status == types.HealthFail && c.Category == types.HealthCritical {
			hasCriticalFail = true
			break
		}
	}
	if !hasCriticalFail {
		t.Error("Expected at least one critical failure")
	}
}

func TestScoreRange(t *testing.T) {
	dir := t.TempDir()
	project := types.Project{Root: dir, Name: "test"}

	report := Check(project, nil)
	if report.Score < 0 || report.Score > 100 {
		t.Errorf("Score out of range: %d", report.Score)
	}
}
