package metrics

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"codescope/internal/types"
)

func Calculate(files []types.FileInfo, dirTree *types.DirNode, project types.Project, stats types.ScanStats) types.Metrics {
	start := time.Now()

	m := types.Metrics{
		Project:   project,
		Files:     files,
		DirTree:   dirTree,
		ScanStats: stats,
	}

	for _, f := range files {
		m.TotalFiles++

		switch f.Category {
		case types.CategorySource:
			m.SourceFiles++
		case types.CategoryTest:
			m.TestFiles++
		case types.CategoryDocumentation:
			m.DocFiles++
		case types.CategoryConfiguration:
			m.ConfigFiles++
		case types.CategoryAsset:
			m.AssetFiles++
		}

		m.TotalLines += f.Lines
		m.CodeLines += f.CodeLines
		m.CommentLines += f.CommentLines
		m.BlankLines += f.BlankLines

		m.TotalFunctions += len(f.Functions)
		m.TotalClasses += f.Classes
		m.TotalStructs += f.Structs
		m.TotalInterfaces += f.Interfaces
	}

	if m.TotalLines > 0 {
		m.CommentRatio = float64(m.CommentLines) / float64(m.TotalLines) * 100
	}
	if m.TotalFiles > 0 {
		m.AvgFileSize = float64(m.TotalLines) / float64(m.TotalFiles)
	}

	var totalFuncLines int
	for _, f := range files {
		for _, fn := range f.Functions {
			totalFuncLines += fn.Lines
		}
	}
	if m.TotalFunctions > 0 {
		m.AvgFunctionSize = float64(totalFuncLines) / float64(m.TotalFunctions)
	}

	sortedFiles := make([]types.FileInfo, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Lines > sortedFiles[j].Lines
	})
	limit := 10
	if len(sortedFiles) < limit {
		limit = len(sortedFiles)
	}
	m.LargestFiles = sortedFiles[:limit]

	var allFuncs []types.Function
	for _, f := range files {
		for _, fn := range f.Functions {
			fn.File = f.Path
			allFuncs = append(allFuncs, fn)
		}
	}
	sort.Slice(allFuncs, func(i, j int) bool {
		return allFuncs[i].Lines > allFuncs[j].Lines
	})
	limit = 10
	if len(allFuncs) < limit {
		limit = len(allFuncs)
	}
	m.LargestFuncs = allFuncs[:limit]

	dirStats := aggregateDirStats(files)
	m.LargestDirs = dirStats

	m.ScanTime = time.Since(start)
	return m
}

func aggregateDirStats(files []types.FileInfo) []types.DirInfo {
	dirMap := make(map[string]*types.DirInfo)
	var dirOrder []string

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if _, ok := dirMap[dir]; !ok {
			dirMap[dir] = &types.DirInfo{Path: dir}
			dirOrder = append(dirOrder, dir)
		}
		dirMap[dir].Files++
		dirMap[dir].Size += f.Size
		dirMap[dir].CodeLines += f.CodeLines
	}

	result := make([]types.DirInfo, 0, len(dirMap))
	for _, d := range dirOrder {
		result = append(result, *dirMap[d])
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CodeLines > result[j].CodeLines
	})

	limit := 15
	if len(result) < limit {
		limit = len(result)
	}
	return result[:limit]
}

func DetectEntryPoints(files []types.FileInfo, project types.Project) []string {
	var entries []string

	entryNames := map[string]bool{
		"main.go": true, "main.rs": true, "main.ts": true,
		"index.js": true, "index.ts": true, "index.jsx": true, "index.tsx": true,
		"app.js": true, "app.ts": true, "app.py": true, "main.py": true,
		"cli.go": true, "cmd.go": true, "lib.rs": true, "mod.rs": true,
	}

	for _, f := range files {
		name := filepath.Base(f.Path)
		if entryNames[name] {
			entries = append(entries, f.Path)
		}
	}

	if len(entries) == 0 && len(files) > 0 {
		dirFiles := make(map[string]int)
		for _, f := range files {
			dir := filepath.Dir(f.Path)
			dirFiles[dir]++
		}

		root := project.Root
		for _, f := range files {
			if filepath.Dir(f.Path) == root && isLikelyEntry(f.Path) {
				entries = append(entries, f.Path)
			}
		}
	}

	return entries
}

func isLikelyEntry(path string) bool {
	name := filepath.Base(path)
	return name == "main.go" || name == "main.rs" || name == "main.ts" ||
		name == "index.js" || name == "index.ts" || name == "app.js" ||
		name == "app.ts" || name == "cli.go" || name == "lib.rs"
}

func DetectModules(files []types.FileInfo) []string {
	moduleMap := make(map[string]bool)

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		parts := strings.Split(dir, string(filepath.Separator))

		for i := 0; i < len(parts); i++ {
			moduleMap[parts[i]] = true
		}
	}

	var modules []string
	for m := range moduleMap {
		if isMeaningfulModule(m) {
			modules = append(modules, m)
		}
	}
	sort.Strings(modules)
	return modules
}

func isMeaningfulModule(name string) bool {
	skip := map[string]bool{
		"": true, ".": true, "..": true, "node_modules": true,
		".git": true, "src": true, "lib": true, "dist": true, "build": true,
		"cmd": true, "internal": true, "pkg": true, "vendor": true,
	}
	return !skip[name] && !strings.HasPrefix(name, ".")
}

func DetectFolderOrganization(files []types.FileInfo) map[string]string {
	org := make(map[string]string)
	seen := make(map[string]bool)

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if seen[dir] {
			continue
		}
		seen[dir] = true

		base := filepath.Base(dir)
		desc := describeFolder(base)
		if desc != "" {
			org[dir] = desc
		}
	}

	return org
}

func describeFolder(name string) string {
	switch name {
	case "cmd":
		return "CLI entry points and command definitions"
	case "internal":
		return "Private implementation not importable by external packages"
	case "pkg":
		return "Reusable public packages"
	case "src":
		return "Main application source code"
	case "lib":
		return "Library code"
	case "api":
		return "API definitions, handlers, and routes"
	case "handlers":
		return "HTTP request handlers"
	case "models":
		return "Data models and type definitions"
	case "db", "database":
		return "Database layer, migrations, queries"
	case "migrations":
		return "Database migration files"
	case "config":
		return "Configuration handling"
	case "middleware":
		return "HTTP middleware components"
	case "routes":
		return "Route definitions"
	case "services":
		return "Business logic services"
	case "utils", "util":
		return "Shared utility functions"
	case "helpers":
		return "Helper functions"
	case "components":
		return "Reusable UI components"
	case "pages":
		return "Page-level components"
	case "layouts":
		return "Layout components"
	case "hooks":
		return "Custom React hooks"
	case "store":
		return "State management"
	case "styles", "css":
		return "Stylesheets and styling"
	case "assets", "public":
		return "Static assets (images, fonts, etc.)"
	case "tests", "test", "__tests__":
		return "Test files"
	case "scripts":
		return "Build and utility scripts"
	case "docs":
		return "Documentation"
	case "examples":
		return "Example code"
	case "benchmarks":
		return "Benchmark tests"
	case "proto":
		return "Protocol buffer definitions"
	case "graphql":
		return "GraphQL schemas and resolvers"
	case "cli":
		return "Command-line interface logic"
	case "web":
		return "Web-specific code"
	case "server":
		return "Server initialization and configuration"
	case "client":
		return "Client-side code"
	case "lang", "i18n":
		return "Internationalization and localization"
	case "docker":
		return "Docker configuration"
	case "deploy":
		return "Deployment configuration"
	}
	return ""
}

func DetectSharedUtilities(files []types.FileInfo) []string {
	utils := make(map[string]bool)

	utilDirs := map[string]bool{"utils": true, "util": true, "helpers": true, "shared": true, "common": true}
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		base := filepath.Base(dir)
		if utilDirs[base] || strings.Contains(f.Path, "/utils/") || strings.Contains(f.Path, "/helpers/") || strings.Contains(f.Path, "/shared/") {
			utils[dir] = true
		}
	}

	var result []string
	for u := range utils {
		result = append(result, u)
	}
	sort.Strings(result)
	return result
}
