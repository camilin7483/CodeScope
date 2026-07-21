package display

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"codescope/internal/types"
)

const (
	colorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

var noColor = os.Getenv("NO_COLOR") != "" || !isTerminal()

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func Colorize(text string, color string) string {
	if noColor {
		return text
	}
	return color + text + colorReset
}

func Bold(text string) string {
	if noColor {
		return text
	}
	return colorBold + text + colorReset
}

func Dim(text string) string {
	if noColor {
		return text
	}
	return colorDim + text + colorReset
}

type Table struct {
	Headers []string
	Rows    [][]string
	Widths  []int
}

func NewTable(headers []string) *Table {
	return &Table{Headers: headers, Widths: make([]int, len(headers))}
}

func (t *Table) AddRow(row []string) {
	t.Rows = append(t.Rows, row)
	for i, cell := range row {
		if i < len(t.Widths) && len(cell) > t.Widths[i] {
			t.Widths[i] = len(cell)
		}
	}
}

func (t *Table) Render() string {
	if len(t.Headers) == 0 {
		return ""
	}

	for i, h := range t.Headers {
		if len(h) > t.Widths[i] {
			t.Widths[i] = len(h)
		}
	}

	var b strings.Builder
	sepLine := t.separator()

	b.WriteString(sepLine)
	b.WriteByte('\n')
	b.WriteString(t.formatRow(t.Headers, true))
	b.WriteByte('\n')
	b.WriteString(sepLine)
	b.WriteByte('\n')

	for _, row := range t.Rows {
		b.WriteString(t.formatRow(row, false))
		b.WriteByte('\n')
	}

	b.WriteString(sepLine)
	return b.String()
}

func (t *Table) separator() string {
	var parts []string
	for _, w := range t.Widths {
		parts = append(parts, strings.Repeat("─", w+2))
	}
	return "├" + strings.Join(parts, "┼") + "┤"
}

func (t *Table) formatRow(row []string, isHeader bool) string {
	var parts []string
	for i, cell := range row {
		w := t.Widths[i]
		cell = fmt.Sprintf(" %-*s ", w, cell)
		if isHeader {
			cell = Bold(cell)
		}
		parts = append(parts, cell)
	}
	return "│" + strings.Join(parts, "│") + "│"
}

func PrintHeader(title string) {
	fmt.Println()
	fmt.Println(Colorize("  "+title, colorCyan))
	fmt.Println(Colorize("  "+strings.Repeat("─", len(title)+2), colorDim))
	fmt.Println()
}

func PrintSummary(m *types.Metrics) {
	PrintHeader("PROJECT SUMMARY")

	fmt.Printf("  %-25s %s\n", Colorize("Project:", colorBold), m.Project.Name)
	fmt.Printf("  %-25s %s\n", Colorize("Language:", colorBold), string(m.Project.Language))
	if m.Project.Framework != "" {
		fmt.Printf("  %-25s %s\n", Colorize("Framework:", colorBold), m.Project.Framework)
	}
	fmt.Printf("  %-25s %s\n", Colorize("Build System:", colorBold), m.Project.BuildSystem)
	fmt.Printf("  %-25s %s\n", Colorize("Package Manager:", colorBold), m.Project.PackageManager)
	fmt.Printf("  %-25s %s\n", Colorize("Root:", colorBold), m.Project.Root)
	fmt.Println()

	t := NewTable([]string{"Metric", "Value", ""})
	t.AddRow([]string{"Total Files", formatCount(m.TotalFiles), ""})
	t.AddRow([]string{"Source Files", formatCount(m.SourceFiles), ColorGreen + "●" + colorReset})
	t.AddRow([]string{"Test Files", formatCount(m.TestFiles), ColorGreen + "●" + colorReset})
	t.AddRow([]string{"Documentation", formatCount(m.DocFiles), colorBlue + "●" + colorReset})
	t.AddRow([]string{"Configuration", formatCount(m.ConfigFiles), ColorYellow + "●" + colorReset})
	t.AddRow([]string{"Assets", formatCount(m.AssetFiles), ""})
	t.AddRow([]string{"", "", ""})
	t.AddRow([]string{"Total Lines", formatCount(m.TotalLines), ""})
	t.AddRow([]string{"Code Lines", formatCount(m.CodeLines), ColorGreen + "●" + colorReset})
	t.AddRow([]string{"Comment Lines", formatCount(m.CommentLines), colorBlue + "●" + colorReset})
	t.AddRow([]string{"Blank Lines", formatCount(m.BlankLines), ""})
	t.AddRow([]string{"", "", ""})
	t.AddRow([]string{"Comment Ratio", formatPercent(m.CommentRatio), ""})
	t.AddRow([]string{"Avg File Size", formatFloat(m.AvgFileSize) + " lines", ""})
	t.AddRow([]string{"Functions", formatCount(m.TotalFunctions), ""})
	fmt.Println(t.Render())
	fmt.Printf("  %s %s\n", Colorize("Scan completed in:", colorDim), m.ScanTime.Round(time.Millisecond))
}

func PrintMetrics(m *types.Metrics) {
	PrintHeader("CODE METRICS")

	t := NewTable([]string{"Metric", "Value", "Rating"})

	commentRating := "Good"
	if m.CommentRatio < 5 {
		commentRating = ColorRed + "Low" + colorReset
	} else if m.CommentRatio < 15 {
		commentRating = ColorYellow + "Moderate" + colorReset
	} else if m.CommentRatio > 40 {
		commentRating = ColorYellow + "High" + colorReset
	}

	fileSizeRating := "Good"
	if m.AvgFileSize > 200 {
		fileSizeRating = ColorYellow + "Large" + colorReset
	} else if m.AvgFileSize > 500 {
		fileSizeRating = ColorRed + "Very Large" + colorReset
	}

	testRating := "No Tests"
	if m.TestFiles > 0 {
		testRatio := float64(m.TestFiles) / float64(m.SourceFiles+m.TestFiles) * 100
		if testRatio > 30 {
			testRating = ColorGreen + "Good" + colorReset
		} else if testRatio > 10 {
			testRating = ColorYellow + "Moderate" + colorReset
		} else {
			testRating = ColorYellow + "Low" + colorReset
		}
	}

	t.AddRow([]string{"Comment Ratio", formatPercent(m.CommentRatio), commentRating})
	t.AddRow([]string{"Average File Size", formatFloat(m.AvgFileSize) + " lines", fileSizeRating})
	t.AddRow([]string{"Test Coverage (files)", formatPercent(float64(m.TestFiles) / float64(m.SourceFiles+m.TestFiles) * 100), testRating})
	t.AddRow([]string{"Function Count", formatCount(m.TotalFunctions), ""})
	t.AddRow([]string{"Code/Blank Ratio", fmt.Sprintf("%.1f", float64(m.CodeLines)/float64(m.BlankLines+1)), ""})
	fmt.Println(t.Render())
}

func PrintLargestFiles(files []types.FileInfo) {
	PrintHeader("LARGEST FILES")

	t := NewTable([]string{"Lines", "File", "Language"})
	for _, f := range files {
		lang := string(f.Language)
		if lang == "Unknown" {
			lang = Dim(lang)
		}
		t.AddRow([]string{formatCount(f.Lines), f.Path, lang})
	}
	fmt.Println(t.Render())
}

func PrintLargestFunctions(funcs []types.Function) {
	if len(funcs) == 0 {
		return
	}
	PrintHeader("LARGEST FUNCTIONS")

	t := NewTable([]string{"Lines", "Function", "File", "Line"})
	for _, f := range funcs {
		shortPath := f.File
		if len(shortPath) > 60 {
			shortPath = "..." + shortPath[len(shortPath)-57:]
		}
		t.AddRow([]string{formatCount(f.Lines), f.Name, shortPath, formatCount(f.Line)})
	}
	fmt.Println(t.Render())
}

func PrintArchitecture(entryPoints []string, modules []string, folderOrg map[string]string, sharedUtils []string) {
	PrintHeader("ARCHITECTURE ANALYSIS")

	if len(entryPoints) > 0 {
		fmt.Println("  " + Bold("Entry Points:"))
		for _, ep := range entryPoints {
			fmt.Printf("    • %s\n", ep)
		}
		fmt.Println()
	}

	if len(modules) > 0 {
		fmt.Println("  " + Bold("Internal Modules:"))
		for _, m := range modules {
			fmt.Printf("    • %s\n", m)
		}
		fmt.Println()
	}

	if len(folderOrg) > 0 {
		fmt.Println("  " + Bold("Folder Organization:"))
		t := NewTable([]string{"Directory", "Responsibility"})
		var dirs []string
		for d := range folderOrg {
			dirs = append(dirs, d)
		}
		sort.Strings(dirs)
		for _, d := range dirs {
			short := d
			if len(short) > 50 {
				short = "..." + short[len(short)-47:]
			}
			t.AddRow([]string{short, folderOrg[d]})
		}
		fmt.Println(t.Render())
		fmt.Println()
	}

	if len(sharedUtils) > 0 {
		fmt.Println("  " + Bold("Shared Utilities:"))
		for _, u := range sharedUtils {
			fmt.Printf("    • %s\n", u)
		}
		fmt.Println()
	}
}

func PrintTree(node *types.DirNode, prefix string, isLast bool) {
	if node == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	if prefix == "" {
		sizeStr := fmt.Sprintf(" (%d files, %s lines)", node.Files, formatCount(node.CodeLines))
		fmt.Printf("  %s %s\n", Colorize(".", colorCyan), Dim(sizeStr))
	} else {
		sizeStr := fmt.Sprintf(" (%d files, %s lines)", node.Files, formatCount(node.CodeLines))
		fmt.Printf("  %s%s%s%s\n", prefix, connector, Bold(node.Name), Dim(sizeStr))
	}

	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}

	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Name < node.Children[j].Name
	})

	for i, child := range node.Children {
		PrintTree(child, childPrefix, i == len(node.Children)-1)
	}
}

func PrintDirs(dirs []types.DirInfo) {
	PrintHeader("LARGEST DIRECTORIES")

	t := NewTable([]string{"Files", "Lines", "Directory"})
	for _, d := range dirs {
		short := d.Path
		if len(short) > 50 {
			short = "..." + short[len(short)-47:]
		}
		t.AddRow([]string{formatCount(d.Files), formatCount(d.CodeLines), short})
	}
	fmt.Println(t.Render())
}

func PrintDeps(project types.Project) {
	PrintHeader("DEPENDENCY ANALYSIS")

	fmt.Println("  " + Bold("Build System:") + " " + project.BuildSystem)
	fmt.Println("  " + Bold("Package Manager:") + " " + project.PackageManager)

	if project.BuildSystem == "Go (built-in)" {
		fmt.Println("\n  " + Dim("Dependency details available in go.mod"))
	} else if project.BuildSystem == "Cargo" {
		fmt.Println("\n  " + Dim("Dependency details available in Cargo.toml"))
	} else {
		fmt.Println("\n  " + Dim("Parsing dependency files..."))
	}
	fmt.Println()
}

func PrintGraph(g *types.Graph) {
	if g == nil || len(g.Nodes) == 0 {
		fmt.Println("  No relationship data available.")
		return
	}

	PrintHeader("RELATIONSHIP GRAPH")

	fmt.Println("  " + Bold("Nodes:") + fmt.Sprintf(" %d", len(g.Nodes)))
	fmt.Println("  " + Bold("Edges:") + fmt.Sprintf(" %d", len(g.Edges)))
	fmt.Println()

	if len(g.Nodes) <= 30 {
		t := NewTable([]string{"ID", "Label", "Type", "Size"})
		for _, n := range g.Nodes {
			t.AddRow([]string{n.ID, n.Label, n.Type, formatCount(n.Size)})
		}
		fmt.Println(t.Render())
	}
}

func PrintMermaidDiagram(g *types.Graph) {
	if g == nil || len(g.Nodes) == 0 {
		return
	}

	fmt.Println("\n  " + Bold("Mermaid Diagram:"))
	fmt.Println()
	fmt.Println("  ```mermaid")
	fmt.Println("  graph TD;")
	for _, n := range g.Nodes {
		fmt.Printf("    %s[%s];\n", n.ID, n.Label)
	}
	for _, e := range g.Edges {
		fmt.Printf("    %s-->%s;\n", e.From, e.To)
	}
	fmt.Println("  ```")
	fmt.Println()
}

func PrintFullReport(m *types.Metrics, health *types.HealthReport, arch *types.ArchAnalysis) {
	PrintSummary(m)
	PrintMetrics(m)
	PrintDirs(m.LargestDirs)
	PrintLargestFiles(m.LargestFiles)
	PrintLargestFunctions(m.LargestFuncs)

	if arch != nil {
		PrintArchitecture(arch.EntryPoints, arch.InternalModules, arch.LayerOrganization, arch.SharedUtilities)
	}

	if health != nil {
		PrintHealth(health)
	}
}

func formatCount(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

func formatPercent(f float64) string {
	return fmt.Sprintf("%.1f%%", f)
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}

func PrintScanProgress(current int, totalEstimate int) {
	if current%100 == 0 && current > 0 {
		fmt.Fprintf(os.Stderr, "\r  Scanned %d files...", current)
	}
}

func ClearProgress() {
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 50))
}

func HideCursor() {
	if !noColor {
		fmt.Fprint(os.Stderr, "\033[?25l")
	}
}

func ShowCursor() {
	if !noColor {
		fmt.Fprint(os.Stderr, "\033[?25h")
	}
}

func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func PrintInsights(insights []types.Insight) {
	if len(insights) == 0 {
		return
	}
	PrintHeader("INSIGHTS")

	for _, in := range insights {
		icon := "ℹ"
		sevColor := ColorGreen
		switch in.Severity {
		case types.SeverityCritical:
			icon = "🔴"
			sevColor = ColorRed
		case types.SeverityHigh:
			icon = "⚠"
			sevColor = ColorRed
		case types.SeverityMedium:
			icon = "⚡"
			sevColor = ColorYellow
		default:
			icon = "✓"
			sevColor = ColorGreen
		}
		fmt.Printf("  %s %s\n", Colorize(icon, sevColor), in.Message)
	}
	fmt.Println()
}

func PrintRisks(risks []types.RiskItem) {
	if len(risks) == 0 {
		return
	}
	PrintHeader("RISK ANALYSIS")

	for _, r := range risks {
		color := ColorGreen
		if r.RiskScore > 50 {
			color = ColorRed
		} else if r.RiskScore > 20 {
			color = ColorYellow
		}
		fmt.Printf("  %s %s\n", Bold(r.File), Colorize(fmt.Sprintf("(Risk: %d/100)", r.RiskScore), color))
		for _, reason := range r.Reasons {
			fmt.Printf("    • %s\n", reason)
		}
		fmt.Println()
	}
}

func PrintHotspots(hotspots []types.Hotspot) {
	if len(hotspots) == 0 {
		return
	}
	PrintHeader("CODE HOTSPOTS")

	for _, h := range hotspots {
		color := ColorGreen
		if h.Score > 40 {
			color = ColorRed
		} else if h.Score > 20 {
			color = ColorYellow
		}
		fmt.Printf("  %s %s\n", Bold(h.File), Colorize(fmt.Sprintf("(Score: %d)", h.Score), color))
		for _, reason := range h.Reasons {
			fmt.Printf("    • %s\n", reason)
		}
	}
	fmt.Println()
}

func PrintScanFooter(stats types.ScanStats, _ types.ScanStats) {
	fmt.Println()
	fmt.Println("  " + Bold("Summary:"))
	fmt.Printf("  ✓ Fast scan completed (%s)\n", FormatDuration(stats.ScanTime))
	fmt.Printf("  ✓ %d files analyzed\n", stats.TotalFiles)
	if stats.IgnoredFiles > 0 {
		fmt.Printf("  ✓ %d non-essential files skipped\n", stats.IgnoredFiles)
	}
	if len(stats.IgnoredDirs) > 0 {
		fmt.Printf("  ✓ Skipped directories: %s\n", strings.Join(stats.IgnoredDirs, ", "))
	}
	if stats.ErrorFiles > 0 {
		fmt.Printf("  ⚠ %d files had errors during scanning\n", stats.ErrorFiles)
	}
	fmt.Println()
}

func PrintHealth(report *types.HealthReport) {
	PrintHeader("REPOSITORY HEALTH")

	scoreColor := ColorGreen
	if report.Score < 50 {
		scoreColor = ColorRed
	} else if report.Score < 75 {
		scoreColor = ColorYellow
	}
	fmt.Printf("  %s %s/100\n\n", Bold("Health Score:"), Colorize(strconv.Itoa(report.Score), scoreColor))

	fmt.Printf("  %s %d/%d passed\n", Bold("Critical:"), report.CriticalPass, report.CriticalTotal)
	fmt.Printf("  %s %d/%d passed\n", Bold("Recommended:"), report.RecommendedPass, report.RecommendedTotal)
	fmt.Printf("  %s %d/%d passed\n\n", Bold("Optional:"), report.OptionalPass, report.OptionalTotal)

	t := NewTable([]string{"Category", "Check", "Status", "Message"})
	for _, c := range report.Checks {
		catStr := string(c.Category)
		switch c.Category {
		case types.HealthCritical:
			catStr = ColorRed + "critical" + colorReset
		case types.HealthRecommended:
			catStr = ColorYellow + "recommended" + colorReset
		case types.HealthOptional:
			catStr = Dim("optional")
		}

		statusStr := "● " + c.Status.String()
		switch c.Status {
		case types.HealthPass:
			statusStr = ColorGreen + statusStr + colorReset
		case types.HealthWarn:
			statusStr = ColorYellow + statusStr + colorReset
		case types.HealthFail:
			statusStr = ColorRed + statusStr + colorReset
		}
		t.AddRow([]string{catStr, c.Name, statusStr, c.Message})
	}
	fmt.Println(t.Render())

	if len(report.Recommendations) > 0 {
		fmt.Println("\n  " + Bold("Recommendations:"))
		for _, r := range report.Recommendations {
			fmt.Printf("    • %s\n", r)
		}
	}
	fmt.Println()
}
