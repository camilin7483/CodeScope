package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"codescope/internal/types"
)

type Report struct {
	Project types.Project       `json:"project"`
	Metrics types.Metrics       `json:"metrics"`
	Health  *types.HealthReport `json:"health,omitempty"`
	Arch    *types.ArchAnalysis `json:"architecture,omitempty"`
	Graph   *types.Graph        `json:"graph,omitempty"`
}

func ToJSON(report Report, pretty bool) (string, error) {
	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(report, "", "  ")
	} else {
		data, err = json.Marshal(report)
	}
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(data), nil
}

func ToYAML(report Report) (string, error) {
	var b strings.Builder

	b.WriteString("# CodeScope Report\n")
	b.WriteString(fmt.Sprintf("# Generated: %s\n\n", "auto"))

	writeYAMLProject(&b, report.Project)
	writeYALMMetrics(&b, report.Metrics)

	if report.Health != nil {
		writeYAMLHealth(&b, report.Health)
	}

	if report.Arch != nil {
		writeYAMLArch(&b, report.Arch)
	}

	if report.Graph != nil {
		writeYAMLGraph(&b, report.Graph)
	}

	return b.String(), nil
}

func writeYAMLProject(b *strings.Builder, p types.Project) {
	b.WriteString("project:\n")
	b.WriteString(fmt.Sprintf("  root: %s\n", p.Root))
	b.WriteString(fmt.Sprintf("  name: %s\n", p.Name))
	b.WriteString(fmt.Sprintf("  language: %s\n", string(p.Language)))
	if p.Framework != "" {
		b.WriteString(fmt.Sprintf("  framework: %s\n", p.Framework))
	}
	if p.BuildSystem != "" {
		b.WriteString(fmt.Sprintf("  build_system: %s\n", p.BuildSystem))
	}
	if p.PackageManager != "" {
		b.WriteString(fmt.Sprintf("  package_manager: %s\n", p.PackageManager))
	}
	b.WriteString(fmt.Sprintf("  has_git: %t\n", p.HasGit))
	b.WriteString(fmt.Sprintf("  has_docker: %t\n", p.HasDocker))
	b.WriteString(fmt.Sprintf("  has_ci: %t\n", p.HasCI))
}

func writeYALMMetrics(b *strings.Builder, m types.Metrics) {
	b.WriteString("metrics:\n")
	b.WriteString(fmt.Sprintf("  total_files: %d\n", m.TotalFiles))
	b.WriteString(fmt.Sprintf("  source_files: %d\n", m.SourceFiles))
	b.WriteString(fmt.Sprintf("  test_files: %d\n", m.TestFiles))
	b.WriteString(fmt.Sprintf("  doc_files: %d\n", m.DocFiles))
	b.WriteString(fmt.Sprintf("  config_files: %d\n", m.ConfigFiles))
	b.WriteString(fmt.Sprintf("  asset_files: %d\n", m.AssetFiles))
	b.WriteString(fmt.Sprintf("  total_lines: %d\n", m.TotalLines))
	b.WriteString(fmt.Sprintf("  code_lines: %d\n", m.CodeLines))
	b.WriteString(fmt.Sprintf("  comment_lines: %d\n", m.CommentLines))
	b.WriteString(fmt.Sprintf("  blank_lines: %d\n", m.BlankLines))
	b.WriteString(fmt.Sprintf("  total_functions: %d\n", m.TotalFunctions))
	b.WriteString(fmt.Sprintf("  comment_ratio: %.1f\n", m.CommentRatio))
	b.WriteString(fmt.Sprintf("  avg_file_size: %.1f\n", m.AvgFileSize))
	b.WriteString(fmt.Sprintf("  scan_time: %s\n", m.ScanTime.String()))

	if len(m.LargestFiles) > 0 {
		b.WriteString("  largest_files:\n")
		for i, f := range m.LargestFiles {
			if i >= 10 {
				break
			}
			b.WriteString(fmt.Sprintf("    - lines: %d\n", f.Lines))
			b.WriteString(fmt.Sprintf("      path: %s\n", f.Path))
		}
	}

	if len(m.LargestDirs) > 0 {
		b.WriteString("  largest_directories:\n")
		for i, d := range m.LargestDirs {
			if i >= 10 {
				break
			}
			b.WriteString(fmt.Sprintf("    - files: %d\n", d.Files))
			b.WriteString(fmt.Sprintf("      lines: %d\n", d.CodeLines))
			b.WriteString(fmt.Sprintf("      path: %s\n", d.Path))
		}
	}
}

func writeYAMLHealth(b *strings.Builder, h *types.HealthReport) {
	b.WriteString("health:\n")
	b.WriteString(fmt.Sprintf("  score: %d\n", h.Score))
	b.WriteString("  checks:\n")
	for _, c := range h.Checks {
		b.WriteString(fmt.Sprintf("    - name: %s\n", c.Name))
		b.WriteString(fmt.Sprintf("      status: %s\n", c.Status.String()))
		b.WriteString(fmt.Sprintf("      message: %s\n", c.Message))
	}
	if len(h.Recommendations) > 0 {
		b.WriteString("  recommendations:\n")
		for _, r := range h.Recommendations {
			b.WriteString(fmt.Sprintf("    - \"%s\"\n", r))
		}
	}
}

func writeYAMLArch(b *strings.Builder, a *types.ArchAnalysis) {
	b.WriteString("architecture:\n")
	if len(a.EntryPoints) > 0 {
		b.WriteString("  entry_points:\n")
		for _, ep := range a.EntryPoints {
			b.WriteString(fmt.Sprintf("    - %s\n", ep))
		}
	}
	if len(a.InternalModules) > 0 {
		b.WriteString("  internal_modules:\n")
		for _, m := range a.InternalModules {
			b.WriteString(fmt.Sprintf("    - %s\n", m))
		}
	}
}

func writeYAMLGraph(b *strings.Builder, g *types.Graph) {
	b.WriteString("graph:\n")
	b.WriteString(fmt.Sprintf("  nodes: %d\n", len(g.Nodes)))
	b.WriteString(fmt.Sprintf("  edges: %d\n", len(g.Edges)))
}

func ToMarkdown(report Report) (string, error) {
	var b strings.Builder

	b.WriteString("# CodeScope Report\n\n")
	b.WriteString(fmt.Sprintf("**Project:** %s  \n", report.Project.Name))
	b.WriteString(fmt.Sprintf("**Language:** %s  \n", string(report.Project.Language)))
	if report.Project.Framework != "" {
		b.WriteString(fmt.Sprintf("**Framework:** %s  \n", report.Project.Framework))
	}
	b.WriteString(fmt.Sprintf("**Root:** `%s`  \n", report.Project.Root))
	b.WriteString("\n")

	writeMarkdownMetrics(&b, report.Metrics)

	if report.Health != nil {
		writeMarkdownHealth(&b, report.Health)
	}

	return b.String(), nil
}

func writeMarkdownMetrics(b *strings.Builder, m types.Metrics) {
	b.WriteString("## Metrics\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Total Files | %d |\n", m.TotalFiles))
	b.WriteString(fmt.Sprintf("| Source Files | %d |\n", m.SourceFiles))
	b.WriteString(fmt.Sprintf("| Test Files | %d |\n", m.TestFiles))
	b.WriteString(fmt.Sprintf("| Documentation | %d |\n", m.DocFiles))
	b.WriteString(fmt.Sprintf("| Configuration | %d |\n", m.ConfigFiles))
	b.WriteString(fmt.Sprintf("| Assets | %d |\n", m.AssetFiles))
	b.WriteString(fmt.Sprintf("| Total Lines | %d |\n", m.TotalLines))
	b.WriteString(fmt.Sprintf("| Code Lines | %d |\n", m.CodeLines))
	b.WriteString(fmt.Sprintf("| Comment Lines | %d |\n", m.CommentLines))
	b.WriteString(fmt.Sprintf("| Blank Lines | %d |\n", m.BlankLines))
	b.WriteString(fmt.Sprintf("| Comment Ratio | %.1f%% |\n", m.CommentRatio))
	b.WriteString(fmt.Sprintf("| Functions | %d |\n", m.TotalFunctions))
	b.WriteString("\n")
}

func writeMarkdownHealth(b *strings.Builder, h *types.HealthReport) {
	b.WriteString("## Health\n\n")
	b.WriteString(fmt.Sprintf("**Score:** %d/100\n\n", h.Score))
	b.WriteString("| Check | Status |\n")
	b.WriteString("|-------|--------|\n")
	for _, c := range h.Checks {
		statusIcon := "✅"
		if c.Status == types.HealthWarn {
			statusIcon = "⚠️"
		} else if c.Status == types.HealthFail {
			statusIcon = "❌"
		}
		b.WriteString(fmt.Sprintf("| %s | %s %s |\n", c.Name, statusIcon, c.Message))
	}
	b.WriteString("\n")

	if len(h.Recommendations) > 0 {
		b.WriteString("### Recommendations\n\n")
		for _, r := range h.Recommendations {
			b.WriteString(fmt.Sprintf("- %s\n", r))
		}
		b.WriteString("\n")
	}
}

func ToHTML(report Report) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>CodeScope Report - %s</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
           max-width: 960px; margin: 0 auto; padding: 24px; line-height: 1.7;
           color: #1a1a2e; background: #fafafa; }
    h1 { color: #16213e; border-bottom: 3px solid #0f3460; padding-bottom: 12px; }
    h2 { color: #0f3460; margin-top: 36px; border-bottom: 1px solid #ddd; padding-bottom: 8px; }
    table { border-collapse: collapse; width: 100%%; margin: 16px 0;
            background: #fff; border-radius: 4px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
    th, td { text-align: left; padding: 10px 14px; border-bottom: 1px solid #eee; }
    th { background: #0f3460; color: #fff; font-weight: 600; font-size: 0.9em; text-transform: uppercase; }
    tr:hover { background: #f0f4ff; }
    .metric-label { font-weight: 500; color: #555; }
    .pass { color: #15803d; } .warn { color: #ca8a04; } .fail { color: #dc2626; }
    .badge { display: inline-block; padding: 2px 8px; border-radius: 3px; font-size: 0.85em; font-weight: 600; }
    .badge-good { background: #dcfce7; color: #166534; }
    .badge-warn { background: #fef9c3; color: #854d0e; }
    .badge-bad { background: #fee2e2; color: #991b1b; }
    ul { padding-left: 20px; }
    .footer { margin-top: 40px; color: #999; font-size: 0.85em; text-align: center; }
  </style>
</head>
<body>
  <h1>CodeScope Report: %s</h1>
`, report.Project.Name, report.Project.Name))

	b.WriteString(fmt.Sprintf(`<h2>Project</h2>
<table>
  <tr><td class="metric-label">Name</td><td>%s</td></tr>
  <tr><td class="metric-label">Language</td><td>%s</td></tr>
`, htmlEscape(report.Project.Name), htmlEscape(string(report.Project.Language))))

	if report.Project.Framework != "" {
		b.WriteString(fmt.Sprintf("  <tr><td class=\"metric-label\">Framework</td><td>%s</td></tr>\n", htmlEscape(report.Project.Framework)))
	}
	b.WriteString(fmt.Sprintf(`  <tr><td class="metric-label">Build System</td><td>%s</td></tr>
  <tr><td class="metric-label">Package Manager</td><td>%s</td></tr>
  <tr><td class="metric-label">Root</td><td><code>%s</code></td></tr>
</table>
`, htmlEscape(report.Project.BuildSystem), htmlEscape(report.Project.PackageManager), htmlEscape(report.Project.Root)))

	b.WriteString(fmt.Sprintf(`<h2>Overview</h2>
<table>
  <tr><th>Metric</th><th>Value</th></tr>
  <tr><td>Total Files</td><td>%d</td></tr>
  <tr><td>Source Files</td><td>%d</td></tr>
  <tr><td>Test Files</td><td>%d</td></tr>
  <tr><td>Documentation Files</td><td>%d</td></tr>
  <tr><td>Configuration Files</td><td>%d</td></tr>
  <tr><td>Asset Files</td><td>%d</td></tr>
  <tr><td>Total Lines</td><td>%d</td></tr>
  <tr><td>Code Lines</td><td>%d</td></tr>
  <tr><td>Comment Lines</td><td>%d</td></tr>
  <tr><td>Blank Lines</td><td>%d</td></tr>
  <tr><td>Comment Ratio</td><td>%.1f%%</td></tr>
  <tr><td>Functions</td><td>%d</td></tr>
  <tr><td>Classes</td><td>%d</td></tr>
  <tr><td>Structs</td><td>%d</td></tr>
  <tr><td>Interfaces</td><td>%d</td></tr>
  <tr><td>Avg File Size</td><td>%.1f lines</td></tr>
  <tr><td>Avg Function Size</td><td>%.1f lines</td></tr>
</table>
`,
		report.Metrics.TotalFiles, report.Metrics.SourceFiles, report.Metrics.TestFiles,
		report.Metrics.DocFiles, report.Metrics.ConfigFiles, report.Metrics.AssetFiles,
		report.Metrics.TotalLines, report.Metrics.CodeLines, report.Metrics.CommentLines,
		report.Metrics.BlankLines, report.Metrics.CommentRatio, report.Metrics.TotalFunctions,
		report.Metrics.TotalClasses, report.Metrics.TotalStructs, report.Metrics.TotalInterfaces,
		report.Metrics.AvgFileSize, report.Metrics.AvgFunctionSize))

	if report.Health != nil {
		scoreClass := "good"
		if report.Health.Score < 50 {
			scoreClass = "bad"
		} else if report.Health.Score < 75 {
			scoreClass = "warn"
		}
		b.WriteString(fmt.Sprintf(`<h2>Health</h2>
<p><strong>Score:</strong> <span class="badge badge-%s" style="font-size:1.2em">%d/100</span></p>
`, scoreClass, report.Health.Score))

		b.WriteString("<table>\n  <tr><th>Check</th><th>Status</th><th>Message</th></tr>\n")
		for _, c := range report.Health.Checks {
			statusClass := "badge-good"
			statusLabel := "PASS"
			if c.Status == types.HealthWarn {
				statusClass = "badge-warn"
				statusLabel = "WARN"
			} else if c.Status == types.HealthFail {
				statusClass = "badge-bad"
				statusLabel = "FAIL"
			}
			b.WriteString(fmt.Sprintf("  <tr><td>%s</td><td><span class=\"badge %s\">%s</span></td><td>%s</td></tr>\n",
				htmlEscape(c.Name), statusClass, statusLabel, htmlEscape(c.Message)))
		}
		b.WriteString("</table>\n")

		if len(report.Health.Recommendations) > 0 {
			b.WriteString("<h3>Recommendations</h3>\n<ul>\n")
			for _, r := range report.Health.Recommendations {
				b.WriteString(fmt.Sprintf("  <li>%s</li>\n", htmlEscape(r)))
			}
			b.WriteString("</ul>\n")
		}
	}

	if report.Arch != nil {
		b.WriteString("<h2>Architecture</h2>\n")
		if len(report.Arch.EntryPoints) > 0 {
			b.WriteString("<h3>Entry Points</h3>\n<ul>\n")
			for _, ep := range report.Arch.EntryPoints {
				b.WriteString(fmt.Sprintf("  <li><code>%s</code></li>\n", htmlEscape(ep)))
			}
			b.WriteString("</ul>\n")
		}
		if len(report.Arch.InternalModules) > 0 {
			b.WriteString("<h3>Internal Modules</h3>\n<ul>\n")
			for _, m := range report.Arch.InternalModules {
				b.WriteString(fmt.Sprintf("  <li>%s</li>\n", htmlEscape(m)))
			}
			b.WriteString("</ul>\n")
		}
	}

	if len(report.Metrics.LargestFiles) > 0 {
		b.WriteString("<h2>Largest Files</h2>\n<table>\n  <tr><th>Lines</th><th>File</th></tr>\n")
		for _, f := range report.Metrics.LargestFiles {
			b.WriteString(fmt.Sprintf("  <tr><td>%d</td><td><code>%s</code></td></tr>\n", f.Lines, htmlEscape(f.Path)))
		}
		b.WriteString("</table>\n")
	}

	b.WriteString(fmt.Sprintf(`<div class="footer">
  Generated by CodeScope — %s
</div>
</body>
</html>`, "v0.3.0"))

	return b.String(), nil
}

func WriteToFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func WriteDocs(report Report, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	docs := map[string]string{
		"summary.md": fmt.Sprintf(`# %s - Summary

**Language:** %s
**Root:** %s

## Overview

- Total Files: %d
- Source Files: %d
- Test Files: %d
- Documentation Files: %d
- Configuration Files: %d
- Assets: %d

## Lines

- Total Lines: %d
- Code Lines: %d
- Comment Lines: %d
- Blank Lines: %d
- Comment Ratio: %.1f%%
`,
			report.Project.Name, string(report.Project.Language), report.Project.Root,
			report.Metrics.TotalFiles, report.Metrics.SourceFiles, report.Metrics.TestFiles,
			report.Metrics.DocFiles, report.Metrics.ConfigFiles, report.Metrics.AssetFiles,
			report.Metrics.TotalLines, report.Metrics.CodeLines, report.Metrics.CommentLines,
			report.Metrics.BlankLines, report.Metrics.CommentRatio),

		"architecture.md": func() string {
			var b strings.Builder
			b.WriteString(fmt.Sprintf("# %s - Architecture\n\n", report.Project.Name))
			if report.Arch != nil {
				if len(report.Arch.EntryPoints) > 0 {
					b.WriteString("## Entry Points\n\n")
					for _, ep := range report.Arch.EntryPoints {
						b.WriteString(fmt.Sprintf("- `%s`\n", ep))
					}
					b.WriteString("\n")
				}
				if len(report.Arch.InternalModules) > 0 {
					b.WriteString("## Modules\n\n")
					for _, m := range report.Arch.InternalModules {
						b.WriteString(fmt.Sprintf("- %s\n", m))
					}
					b.WriteString("\n")
				}
			}
			return b.String()
		}(),

		"metrics.md": fmt.Sprintf(`# %s - Metrics

- Total Functions: %d
- Average File Size: %.1f lines
- Average Function Size: %.1f lines
- Comment Ratio: %.1f%%

## Largest Files

%s

## Largest Directories

%s
`,
			report.Project.Name,
			report.Metrics.TotalFunctions, report.Metrics.AvgFileSize,
			report.Metrics.AvgFunctionSize, report.Metrics.CommentRatio,
			formatFileList(report.Metrics.LargestFiles),
			formatDirList(report.Metrics.LargestDirs)),

		"health.md": func() string {
			var b strings.Builder
			b.WriteString(fmt.Sprintf("# %s - Health\n\n", report.Project.Name))
			b.WriteString(fmt.Sprintf("**Score:** %d/100\n\n", report.Health.Score))
			for _, c := range report.Health.Checks {
				icon := "✅"
				if c.Status == types.HealthWarn {
					icon = "⚠️"
				} else if c.Status == types.HealthFail {
					icon = "❌"
				}
				b.WriteString(fmt.Sprintf("- %s %s: %s\n", icon, c.Name, c.Message))
			}
			if len(report.Health.Recommendations) > 0 {
				b.WriteString("\n## Recommendations\n\n")
				for _, r := range report.Health.Recommendations {
					b.WriteString(fmt.Sprintf("- %s\n", r))
				}
			}
			return b.String()
		}(),
	}

	for name, content := range docs {
		path := filepath.Join(outDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func formatFileList(files []types.FileInfo) string {
	if len(files) == 0 {
		return "None"
	}
	var b strings.Builder
	for _, f := range files {
		b.WriteString(fmt.Sprintf("- %d lines: %s\n", f.Lines, f.Path))
	}
	return b.String()
}

func formatDirList(dirs []types.DirInfo) string {
	if len(dirs) == 0 {
		return "None"
	}
	var b strings.Builder
	for _, d := range dirs {
		b.WriteString(fmt.Sprintf("- %d files, %d lines: %s\n", d.Files, d.CodeLines, d.Path))
	}
	return b.String()
}

func BuildSortedDirList(dirTree *types.DirNode) []types.DirInfo {
	var result []types.DirInfo
	collectDirs(dirTree, &result)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CodeLines > result[j].CodeLines
	})
	return result
}

func collectDirs(node *types.DirNode, result *[]types.DirInfo) {
	if node == nil {
		return
	}
	if node.Path != "" {
		*result = append(*result, types.DirInfo{
			Path:      node.Path,
			Size:      node.Size,
			Files:     node.Files,
			CodeLines: node.CodeLines,
		})
	}
	for _, child := range node.Children {
		collectDirs(child, result)
	}
}
