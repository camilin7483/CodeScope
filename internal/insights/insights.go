package insights

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"codescope/internal/types"
)

func Generate(m *types.Metrics, arch *types.ArchAnalysis) []types.Insight {
	var result []types.Insight

	result = append(result, projectInsights(m)...)
	result = append(result, complexityInsights(m)...)
	result = append(result, fileInsights(m)...)
	result = append(result, testInsights(m)...)
	result = append(result, archInsights(m, arch)...)

	sort.Slice(result, func(i, j int) bool {
		if result[i].Severity != result[j].Severity {
			return result[i].Severity > result[j].Severity
		}
		return result[i].Category < result[j].Category
	})

	return result
}

func projectInsights(m *types.Metrics) []types.Insight {
	var ins []types.Insight

	if string(m.Project.Language) != "Unknown" {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Project uses %s", string(m.Project.Language)),
			Severity: types.SeverityLow,
			Category: "architecture",
		})
	}

	if m.Project.Framework != "" {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Framework detected: %s", m.Project.Framework),
			Severity: types.SeverityLow,
			Category: "architecture",
		})
	}

	if m.Project.HasGit {
		ins = append(ins, types.Insight{
			Message:  "Git repository detected",
			Severity: types.SeverityLow,
			Category: "health",
		})
	}

	if m.ScanStats.IgnoredFiles > 0 {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Skipped %d non-essential files", m.ScanStats.IgnoredFiles),
			Severity: types.SeverityLow,
			Category: "performance",
		})
	}

	return ins
}

func complexityInsights(m *types.Metrics) []types.Insight {
	var ins []types.Insight

	if m.CommentRatio < 5 {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Comment ratio is very low (%.1f%%). Consider adding documentation.", m.CommentRatio),
			Severity: types.SeverityMedium,
			Category: "quality",
		})
	} else if m.CommentRatio > 40 {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Comment ratio is high (%.1f%%). Verify comments add value vs noise.", m.CommentRatio),
			Severity: types.SeverityLow,
			Category: "quality",
		})
	}

	if m.AvgFileSize > 300 {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Average file size is %.0f lines. Consider splitting large files.", m.AvgFileSize),
			Severity: types.SeverityMedium,
			Category: "maintainability",
		})
	}

	if m.AvgFunctionSize > 50 {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Average function size is %.0f lines. Functions may be too long.", m.AvgFunctionSize),
			Severity: types.SeverityMedium,
			Category: "maintainability",
		})
	}

	return ins
}

func fileInsights(m *types.Metrics) []types.Insight {
	var ins []types.Insight

	if len(m.LargestFiles) > 0 {
		top := m.LargestFiles[0]
		totalLines := m.TotalLines
		if totalLines > 0 {
			pct := float64(top.Lines) / float64(totalLines) * 100
			if pct > 15 {
				ins = append(ins, types.Insight{
					Message:  fmt.Sprintf("%s contains %.0f%% of project code. Consider splitting.", filepath.Base(top.Path), pct),
					Severity: types.SeverityHigh,
					Category: "maintainability",
				})
			}
		}
	}

	if len(m.LargestFuncs) > 0 {
		topFunc := m.LargestFuncs[0]
		if topFunc.Lines > 100 {
			ins = append(ins, types.Insight{
				Message:  fmt.Sprintf("Function %s in %s has %d lines. High complexity risk.", topFunc.Name, filepath.Base(topFunc.File), topFunc.Lines),
				Severity: types.SeverityHigh,
				Category: "quality",
			})
		}
	}

	for _, f := range m.LargestFiles {
		if f.Complexity > 50 {
			ins = append(ins, types.Insight{
				Message:  fmt.Sprintf("%s has high cyclomatic complexity (%d)", filepath.Base(f.Path), f.Complexity),
				Severity: types.SeverityHigh,
				Category: "quality",
			})
			break
		}
	}

	return ins
}

func testInsights(m *types.Metrics) []types.Insight {
	var ins []types.Insight

	if m.TestFiles == 0 {
		ins = append(ins, types.Insight{
			Message:  "No test files found. Test coverage is missing.",
			Severity: types.SeverityCritical,
			Category: "quality",
		})
	} else {
		sourceTotal := m.SourceFiles + m.TestFiles
		if sourceTotal > 0 {
			testRatio := float64(m.TestFiles) / float64(sourceTotal) * 100
			if testRatio < 10 {
				ins = append(ins, types.Insight{
					Message:  fmt.Sprintf("Test coverage is low (%.0f%% of source files)", testRatio),
					Severity: types.SeverityMedium,
					Category: "quality",
				})
			} else {
				ins = append(ins, types.Insight{
					Message:  fmt.Sprintf("Test coverage is adequate (%.0f%% of source files)", testRatio),
					Severity: types.SeverityLow,
					Category: "quality",
				})
			}
		}
	}

	return ins
}

func archInsights(m *types.Metrics, arch *types.ArchAnalysis) []types.Insight {
	var ins []types.Insight

	if arch == nil {
		return ins
	}

	if len(arch.EntryPoints) == 0 {
		ins = append(ins, types.Insight{
			Message:  "No clear entry point detected. Project may need a main file.",
			Severity: types.SeverityMedium,
			Category: "architecture",
		})
	}

	if len(arch.CircularDeps) > 0 {
		for _, cycle := range arch.CircularDeps {
			ins = append(ins, types.Insight{
				Message:  fmt.Sprintf("Circular dependency detected: %s", cycle),
				Severity: types.SeverityCritical,
				Category: "architecture",
			})
		}
	}

	dirs := make(map[string]int)
	for _, f := range m.Files {
		dir := filepath.Dir(f.Path)
		dirs[dir]++
	}

	if len(dirs) > 20 {
		ins = append(ins, types.Insight{
			Message:  fmt.Sprintf("Project has %d directories. Consider simplifying structure.", len(dirs)),
			Severity: types.SeverityLow,
			Category: "architecture",
		})
	}

	if m.TotalFiles > 50 {
		maxFiles := 0
		maxDir := ""
		for d, count := range dirs {
			if count > maxFiles {
				maxFiles = count
				maxDir = d
			}
		}
		if float64(maxFiles)/float64(m.TotalFiles) > 0.3 {
			ins = append(ins, types.Insight{
				Message:  fmt.Sprintf("%s contains %.0f%% of files. Consider reorganizing.", maxDir, float64(maxFiles)/float64(m.TotalFiles)*100),
				Severity: types.SeverityMedium,
				Category: "architecture",
			})
		}
	}

	return ins
}

func CalculateRisk(m *types.Metrics) []types.RiskItem {
	var risks []types.RiskItem

	threshold := 0.8
	if m.TotalLines > 0 {
		for _, f := range m.LargestFiles {
			score := 0
			var reasons []string

			lineRatio := float64(f.Lines) / float64(m.TotalLines)
			if lineRatio > 0.2 {
				score += 30
				reasons = append(reasons, fmt.Sprintf("Contains %.0f%% of total code", lineRatio*100))
			} else if lineRatio > threshold*0.15/float64(m.TotalLines) {
				score += 15
			}

			if f.Lines > 500 {
				score += 25
				reasons = append(reasons, "Very large file (>500 lines)")
			} else if f.Lines > 200 {
				score += 10
				reasons = append(reasons, "Large file (>200 lines)")
			}

			if f.Complexity > 50 {
				score += 25
				reasons = append(reasons, fmt.Sprintf("High cyclomatic complexity (%d)", f.Complexity))
			} else if f.Complexity > 20 {
				score += 10
			}

			if len(f.Functions) > 10 {
				score += 15
				reasons = append(reasons, fmt.Sprintf("%d functions in one file", len(f.Functions)))
			}

			hasTests := false
			for _, other := range m.Files {
				if other.Category == types.CategoryTest {
					testBase := filepath.Base(other.Path)
					fileBase := filepath.Base(f.Path)
					if testBase == fileBase || strings.Contains(testBase, fileBase) {
						hasTests = true
						break
					}
				}
			}

			if !hasTests {
				score += 15
				reasons = append(reasons, "No corresponding test found")
			}

			if score > 0 {
				risks = append(risks, types.RiskItem{
					File:       f.Path,
					RiskScore:  score,
					Reasons:    reasons,
					Lines:      f.Lines,
					Complexity: f.Complexity,
					Functions:  len(f.Functions),
					HasTests:   hasTests,
				})
			}
		}
	}

	for _, fn := range m.LargestFuncs {
		if fn.Lines > 100 {
			found := false
			for i := range risks {
				if risks[i].File == fn.File {
					risks[i].Reasons = append(risks[i].Reasons,
						fmt.Sprintf("Function %s has %d lines", fn.Name, fn.Lines))
					risks[i].RiskScore += 10
					found = true
					break
				}
			}
			if !found {
				risks = append(risks, types.RiskItem{
					File:      fn.File,
					RiskScore: 10,
					Reasons:   []string{fmt.Sprintf("Function %s has %d lines", fn.Name, fn.Lines)},
					Lines:     fn.Lines,
				})
			}
		}
	}

	sort.Slice(risks, func(i, j int) bool {
		return risks[i].RiskScore > risks[j].RiskScore
	})

	if len(risks) > 10 {
		risks = risks[:10]
	}

	return risks
}

func CalculateHotspots(m *types.Metrics) []types.Hotspot {
	var hotspots []types.Hotspot

	importCount := make(map[string]int)
	complexityMap := make(map[string]int)

	for _, f := range m.Files {
		base := filepath.Base(f.Path)
		complexityMap[base] = f.Complexity
		for _, imp := range f.Imports {
			importCount[imp]++
		}
	}

	for _, f := range m.Files {
		score := 0
		var reasons []string

		base := filepath.Base(f.Path)
		impCount := importCount[base]

		if impCount > 5 {
			score += 20
			reasons = append(reasons, fmt.Sprintf("Imported by %d modules", impCount))
		} else if impCount > 2 {
			score += 10
			reasons = append(reasons, fmt.Sprintf("Imported by %d modules", impCount))
		}

		if f.Complexity > 50 {
			score += 25
			reasons = append(reasons, fmt.Sprintf("High complexity (%d)", f.Complexity))
		} else if f.Complexity > 20 {
			score += 10
		}

		if f.Lines > 500 {
			score += 15
			reasons = append(reasons, "Very large file")
		}

		if len(f.Functions) > 15 {
			score += 10
			reasons = append(reasons, fmt.Sprintf("%d functions", len(f.Functions)))
		}

		if score > 0 && f.Category == types.CategorySource {
			hotspots = append(hotspots, types.Hotspot{
				File:        f.Path,
				Score:       score,
				Reasons:     reasons,
				ImportCount: impCount,
				Complexity:  f.Complexity,
			})
		}
	}

	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Score > hotspots[j].Score
	})

	if len(hotspots) > 10 {
		hotspots = hotspots[:10]
	}

	return hotspots
}

func AnalyzeDuplicates(files []types.FileInfo) []string {
	var results []string

	type block struct {
		content string
		file    string
		line    int
	}

	var blocks []block
	for _, f := range files {
		if f.Category != types.CategorySource {
			continue
		}
		for _, fn := range f.Functions {
			blocks = append(blocks, block{
				content: fn.Name,
				file:    fn.File,
				line:    fn.Line,
			})
		}
	}

	for i := 0; i < len(blocks); i++ {
		for j := i + 1; j < len(blocks); j++ {
			if blocks[i].content == blocks[j].content &&
				blocks[i].file != blocks[j].file {
				results = append(results, fmt.Sprintf(
					"Function '%s' appears in both %s:%d and %s:%d",
					blocks[i].content, blocks[i].file, blocks[i].line,
					blocks[j].file, blocks[j].line))
			}
		}
	}

	if len(results) > 10 {
		results = results[:10]
	}

	return results
}
