package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codescope/internal/analysis"
	"codescope/internal/detector"
	"codescope/internal/display"
	"codescope/internal/export"
	"codescope/internal/health"
	"codescope/internal/insights"
	"codescope/internal/metrics"
	"codescope/internal/scanner"
	"codescope/internal/types"
)

const version = "0.3.0"

func main() {
	if len(os.Args) < 2 {
		withCursor(func() { runScan(".") })
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "scan":
		withCursor(func() { runScan(getArg(2, ".")) })
	case "summary":
		withCursor(func() { runSummary(getArg(2, ".")) })
	case "metrics":
		withCursor(func() { runMetrics(getArg(2, ".")) })
	case "architecture":
		withCursor(func() { runArchitecture(getArg(2, ".")) })
	case "graph":
		withCursor(func() { runGraph(getArg(2, ".")) })
	case "deps":
		withCursor(func() { runDeps(getArg(2, ".")) })
	case "health":
		withCursor(func() { runHealth(getArg(2, ".")) })
	case "risk":
		withCursor(func() { runRisk(getArg(2, ".")) })
	case "hotspots":
		withCursor(func() { runHotspots(getArg(2, ".")) })
	case "tree":
		withCursor(func() { runTree(getArg(2, ".")) })
	case "docs":
		withCursor(func() {
			runDocs(getArg(2, "."), getArg(3, filepath.Join(getArg(2, "."), "docs")))
		})
	case "export":
		withCursor(func() {
			runExport(getArg(2, "."), getArg(3, "json"), getArg(4, ""))
		})
	case "version", "-v", "--version":
		fmt.Printf("CodeScope v%s\n", version)
	case "-h", "--help":
		printHelp()
	default:
		if strings.HasPrefix(cmd, "-") {
			printHelp()
		} else {
			withCursor(func() { runScan(cmd) })
		}
	}
}

func getArg(idx int, defaultVal string) string {
	if idx < len(os.Args) {
		return os.Args[idx]
	}
	return defaultVal
}

func printHelp() {
	fmt.Println(display.Bold("CodeScope") + " — Understand any codebase in seconds.")
	fmt.Println()
	fmt.Println(display.Bold("Usage:"))
	fmt.Println("  codescope [command] [path] [options]")
	fmt.Println()
	fmt.Println(display.Bold("Commands:"))
	fmt.Printf("  %-30s %s\n", "codescope .", "Full analysis (default)")
	fmt.Printf("  %-30s %s\n", "codescope scan [path]", "Scan and analyze a project")
	fmt.Printf("  %-30s %s\n", "codescope summary [path]", "Project summary overview")
	fmt.Printf("  %-30s %s\n", "codescope metrics [path]", "Code metrics and statistics")
	fmt.Printf("  %-30s %s\n", "codescope architecture [path]", "Architecture analysis")
	fmt.Printf("  %-30s %s\n", "codescope graph [path]", "Relationship graphs")
	fmt.Printf("  %-30s %s\n", "codescope deps [path]", "Dependency analysis")
	fmt.Printf("  %-30s %s\n", "codescope health [path]", "Repository health check")
	fmt.Printf("  %-30s %s\n", "codescope risk [path]", "Risk analysis")
	fmt.Printf("  %-30s %s\n", "codescope hotspots [path]", "Code hotspots")
	fmt.Printf("  %-30s %s\n", "codescope tree [path]", "Directory tree")
	fmt.Printf("  %-30s %s\n", "codescope docs [path] [outdir]", "Generate documentation")
	fmt.Printf("  %-30s %s\n", "codescope export [path] [format] [out]", "Export report")
	fmt.Println()
	fmt.Println(display.Bold("Examples:"))
	fmt.Println("  codescope .")
	fmt.Println("  codescope scan ./my-project")
	fmt.Println("  codescope export . json report.json")
	fmt.Println("  codescope risk .")
	fmt.Println()
}

func ensureDir(path string) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", path)
		os.Exit(1)
	}
}

func analyzeProject(root string) (types.Metrics, *types.HealthReport, *types.ArchAnalysis, *scanner.Scanner) {
	root = resolveDir(root)
	project := detector.DetectProject(root)
	s := scanner.New(root)
	start := time.Now()
	files, dirTree := s.Scan()
	stats := s.Stats()
	stats.ScanTime = time.Since(start)
	stats.TotalFiles = len(files)

	// Count source files
	for _, f := range files {
		if f.Category == types.CategorySource {
			stats.SourceFiles++
		}
	}

	m := metrics.Calculate(files, dirTree, project, stats)
	m.ScanTime = stats.ScanTime

	h := health.Check(project, files)
	arch := analysis.AnalyzeArchitecture(files, project)

	return m, h, arch, s
}

func analyzeSimple(root string) (types.Metrics, *scanner.Scanner) {
	root = resolveDir(root)
	project := detector.DetectProject(root)
	s := scanner.New(root)
	files, dirTree := s.Scan()
	stats := s.Stats()
	m := metrics.Calculate(files, dirTree, project, stats)
	return m, s
}

func runScan(root string) {
	root = resolveDir(root)
	fmt.Printf("\n  %s %s\n", display.Bold("Scanning:"), root)

	m, h, arch, s := analyzeProject(root)

	display.ClearProgress()
	display.PrintFullReport(&m, h, arch)

	ins := insights.Generate(&m, arch)
	if len(ins) > 0 {
		display.PrintInsights(ins)
	}

	risks := insights.CalculateRisk(&m)
	if len(risks) > 0 {
		display.PrintRisks(risks)
	}

	dupes := insights.AnalyzeDuplicates(m.Files)
	if len(dupes) > 0 {
		display.PrintHeader("DUPLICATE DETECTION")
		for _, d := range dupes {
			fmt.Printf("  • %s\n", d)
		}
		fmt.Println()
	}

	display.PrintScanFooter(m.ScanStats, s.Stats())
}

func runSummary(root string) {
	m, _ := analyzeSimple(root)
	display.PrintSummary(&m)
}

func runMetrics(root string) {
	m, _ := analyzeSimple(root)
	display.PrintMetrics(&m)
	display.PrintLargestFiles(m.LargestFiles)
	display.PrintLargestFunctions(m.LargestFuncs)
	display.PrintDirs(m.LargestDirs)
}

func runArchitecture(root string) {
	root = resolveDir(root)
	project := detector.DetectProject(root)
	s := scanner.New(root)
	files, _ := s.Scan()
	arch := analysis.AnalyzeArchitecture(files, project)
	display.PrintArchitecture(arch.EntryPoints, arch.InternalModules, arch.LayerOrganization, arch.SharedUtilities)
}

func runGraph(root string) {
	root = resolveDir(root)
	project := detector.DetectProject(root)
	s := scanner.New(root)
	files, _ := s.Scan()
	g := analysis.BuildDependencyGraph(files, project)
	fg := analysis.BuildFolderGraph(files)

	display.PrintHeader("DEPENDENCY GRAPH")
	display.PrintGraph(g)
	display.PrintMermaidDiagram(g)

	fmt.Println()
	display.PrintHeader("FOLDER GRAPH")
	display.PrintGraph(fg)
	display.PrintMermaidDiagram(fg)
}

func runDeps(root string) {
	root = resolveDir(root)
	project := detector.DetectProject(root)
	display.PrintDeps(project)
}

func runHealth(root string) {
	root = resolveDir(root)
	project := detector.DetectProject(root)
	s := scanner.New(root)
	files, _ := s.Scan()
	h := health.Check(project, files)
	display.PrintHealth(h)
}

func runRisk(root string) {
	m, _ := analyzeSimple(root)
	risks := insights.CalculateRisk(&m)
	display.PrintRisks(risks)
}

func runHotspots(root string) {
	m, _ := analyzeSimple(root)
	hotspots := insights.CalculateHotspots(&m)
	display.PrintHotspots(hotspots)
}

func runTree(root string) {
	root = resolveDir(root)
	project := detector.DetectProject(root)
	s := scanner.New(root)

	display.PrintHeader(fmt.Sprintf("DIRECTORY TREE — %s", project.Name))
	files, dirTree := s.Scan()
	if dirTree != nil {
		display.PrintTree(dirTree, "", true)
	}
	fmt.Printf("\n  %s %s %s\n", display.Dim("Total:"), display.Bold(fmt.Sprintf("%d", len(files))), display.Dim("files"))
}

func runDocs(root string, outDir string) {
	root = resolveDir(root)
	outDir = resolvePath(outDir)

	m, h, arch, _ := analyzeProject(root)

	report := export.Report{
		Project: m.Project,
		Metrics: m,
		Health:  h,
		Arch:    arch,
	}

	if err := export.WriteDocs(report, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing docs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n  %s Documentation generated in: %s\n", display.Colorize("✓", display.ColorGreen), outDir)
}

func runExport(root string, format string, outFile string) {
	root = resolveDir(root)

	m, h, arch, _ := analyzeProject(root)

	report := export.Report{
		Project: m.Project,
		Metrics: m,
		Health:  h,
		Arch:    arch,
	}

	var output string
	var err error

	switch strings.ToLower(format) {
	case "json":
		output, err = export.ToJSON(report, true)
	case "yaml", "yml":
		output, err = export.ToYAML(report)
	case "md", "markdown":
		output, err = export.ToMarkdown(report)
	case "html":
		output, err = export.ToHTML(report)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s (supported: json, yaml, md, html)\n", format)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Export error: %v\n", err)
		os.Exit(1)
	}

	if outFile != "" {
		if err := export.WriteToFile(outFile, output); err != nil {
			fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n  %s Report written to: %s\n", display.Colorize("✓", display.ColorGreen), outFile)
	} else {
		fmt.Println(output)
	}
}

func resolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func resolveDir(path string) string {
	abs := resolvePath(path)
	ensureDir(abs)
	return abs
}

func withCursor(fn func()) {
	display.HideCursor()
	defer display.ShowCursor()
	fn()
}
