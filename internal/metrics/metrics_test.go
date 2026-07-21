package metrics

import (
	"testing"

	"codescope/internal/types"
)

func TestCalculateBasic(t *testing.T) {
	files := []types.FileInfo{
		{Path: "main.go", Language: types.LanguageGo, Category: types.CategorySource, Lines: 100, CodeLines: 80, CommentLines: 10, BlankLines: 10, Functions: []types.Function{{Name: "main", Lines: 20}}},
		{Path: "util.go", Language: types.LanguageGo, Category: types.CategorySource, Lines: 50, CodeLines: 40, CommentLines: 5, BlankLines: 5, Functions: []types.Function{{Name: "helper", Lines: 15}}},
		{Path: "main_test.go", Language: types.LanguageGo, Category: types.CategoryTest, Lines: 30, CodeLines: 25, CommentLines: 3, BlankLines: 2},
	}

	project := types.Project{Name: "test", Language: types.LanguageGo}

	m := Calculate(files, nil, project, types.ScanStats{})

	if m.TotalFiles != 3 {
		t.Errorf("TotalFiles = %d, want 3", m.TotalFiles)
	}
	if m.SourceFiles != 2 {
		t.Errorf("SourceFiles = %d, want 2", m.SourceFiles)
	}
	if m.TestFiles != 1 {
		t.Errorf("TestFiles = %d, want 1", m.TestFiles)
	}
	if m.TotalFunctions != 2 {
		t.Errorf("TotalFunctions = %d, want 2", m.TotalFunctions)
	}
	if m.CommentRatio != 10.0 {
		t.Errorf("CommentRatio = %f, want 10.0", m.CommentRatio)
	}
}

func TestCalculateEmpty(t *testing.T) {
	m := Calculate(nil, nil, types.Project{}, types.ScanStats{})

	if m.TotalFiles != 0 {
		t.Errorf("TotalFiles = %d, want 0", m.TotalFiles)
	}
}

func TestCalculateLargeFiles(t *testing.T) {
	files := make([]types.FileInfo, 20)
	for i := 0; i < 20; i++ {
		files[i] = types.FileInfo{
			Path:     "file.go",
			Category: types.CategorySource,
			Lines:    100 + i*10,
		}
	}

	m := Calculate(files, nil, types.Project{}, types.ScanStats{})

	if len(m.LargestFiles) != 10 {
		t.Errorf("LargestFiles should be limited to 10, got %d", len(m.LargestFiles))
	}

	for i := 1; i < len(m.LargestFiles); i++ {
		if m.LargestFiles[i-1].Lines < m.LargestFiles[i].Lines {
			t.Error("LargestFiles not sorted descending")
		}
	}
}
