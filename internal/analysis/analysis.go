package analysis

import (
	"path/filepath"
	"strings"

	"codescope/internal/types"
)

func BuildDependencyGraph(files []types.FileInfo, project types.Project) *types.Graph {
	g := &types.Graph{}

	nodeMap := make(map[string]bool)

	for _, f := range files {
		if f.Language == types.LanguageUnknown {
			continue
		}

		dir := filepath.Dir(f.Path)
		dirID := sanitizeID(dir)

		if !nodeMap[dirID] {
			g.Nodes = append(g.Nodes, types.GraphNode{
				ID:    dirID,
				Label: filepath.Base(dir),
				Type:  "directory",
				Size:  f.CodeLines,
			})
			nodeMap[dirID] = true
		}

		for _, imp := range f.Imports {
			impID := sanitizeID(imp)
			if !nodeMap[impID] {
				g.Nodes = append(g.Nodes, types.GraphNode{
					ID:    impID,
					Label: imp,
					Type:  "dependency",
					Size:  1,
				})
				nodeMap[impID] = true
			}
			g.Edges = append(g.Edges, types.GraphEdge{
				From: dirID,
				To:   impID,
				Type: "imports",
			})
		}
	}

	return g
}

func BuildFolderGraph(files []types.FileInfo) *types.Graph {
	g := &types.Graph{}

	dirFiles := make(map[string][]string)
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		dirFiles[dir] = append(dirFiles[dir], f.Path)
	}

	nodeMap := make(map[string]bool)

	var dirs []string
	for d := range dirFiles {
		dirs = append(dirs, d)
	}

	for _, d := range dirs {
		id := sanitizeID(d)
		if !nodeMap[id] {
			g.Nodes = append(g.Nodes, types.GraphNode{
				ID:    id,
				Label: filepath.Base(d),
				Type:  "folder",
				Size:  len(dirFiles[d]),
			})
			nodeMap[id] = true
		}

		parent := filepath.Dir(d)
		if parent != "." && parent != d {
			parentID := sanitizeID(parent)
			if !nodeMap[parentID] {
				g.Nodes = append(g.Nodes, types.GraphNode{
					ID:    parentID,
					Label: filepath.Base(parent),
					Type:  "folder",
					Size:  0,
				})
				nodeMap[parentID] = true
			}
			g.Edges = append(g.Edges, types.GraphEdge{
				From: parentID,
				To:   id,
				Type: "contains",
			})
		}
	}

	return g
}

func AnalyzeArchitecture(files []types.FileInfo, project types.Project) *types.ArchAnalysis {
	arch := &types.ArchAnalysis{
		LayerOrganization: make(map[string]string),
	}

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		base := filepath.Base(dir)

		desc := describeDir(base)
		if desc != "" {
			arch.LayerOrganization[dir] = desc
		}
	}

	entryNames := map[string]bool{
		"main.go": true, "main.rs": true, "main.ts": true,
		"index.js": true, "index.ts": true, "index.jsx": true, "index.tsx": true,
		"app.js": true, "app.ts": true, "main.py": true, "lib.rs": true,
		"main.java": true, "Main.java": true, "Program.cs": true,
		"main.c": true, "main.cpp": true, "main.cc": true,
		"main.swift": true, "main.kt": true, "Main.kt": true,
	}
	for _, f := range files {
		name := filepath.Base(f.Path)
		if entryNames[name] {
			arch.EntryPoints = append(arch.EntryPoints, f.Path)
		}
	}

	moduleSet := make(map[string]bool)
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		parts := strings.Split(dir, string(filepath.Separator))
		for _, part := range parts {
			if part != "" && part != "." && part != ".." && part != project.Root {
				if !isCommonDir(part) {
					moduleSet[part] = true
				}
			}
		}
	}
	for m := range moduleSet {
		arch.InternalModules = append(arch.InternalModules, m)
	}

	utilDirs := map[string]bool{"utils": true, "util": true, "helpers": true, "shared": true, "common": true}
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		base := filepath.Base(dir)
		if utilDirs[base] {
			arch.SharedUtilities = append(arch.SharedUtilities, dir)
		}
	}

	arch.DependencyFlow = *BuildDependencyGraph(files, project)
	arch.FolderGraph = *BuildFolderGraph(files)
	arch.CircularDeps = detectCircularDeps(files)

	return arch
}

func detectCircularDeps(files []types.FileInfo) [][]string {
	depMap := make(map[string]map[string]bool)
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if depMap[dir] == nil {
			depMap[dir] = make(map[string]bool)
		}
		for _, imp := range f.Imports {
			impDir := filepath.Dir(imp)
			if impDir != dir && impDir != "." {
				depMap[dir][impDir] = true
			}
		}
	}

	visited := make(map[string]int)

	var cycles [][]string
	for dir := range depMap {
		if visited[dir] == 0 {
			path := []string{dir}
			findCycles(dir, depMap, visited, path, &cycles)
		}
	}

	if len(cycles) > 5 {
		cycles = cycles[:5]
	}
	return cycles
}

func findCycles(node string, depMap map[string]map[string]bool, visited map[string]int, path []string, cycles *[][]string) {
	visited[node] = 1

	for dep := range depMap[node] {
		if visited[dep] == 1 {
			for i, p := range path {
				if p == dep {
					cycle := append([]string{}, path[i:]...)
					cycle = append(cycle, dep)
					*cycles = append(*cycles, cycle)
					return
				}
			}
		} else if visited[dep] == 0 {
			findCycles(dep, depMap, visited, append(path, dep), cycles)
		}
	}

	visited[node] = 2
}

func describeDir(name string) string {
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
	case "terraform":
		return "Infrastructure as Code"
	case "k8s", "kubernetes":
		return "Kubernetes manifests"
	case "helm":
		return "Helm chart configuration"
	}
	return ""
}

func isCommonDir(name string) bool {
	common := map[string]bool{
		"node_modules": true, ".git": true, "dist": true, "build": true,
		"target": true, "vendor": true, ".venv": true, "venv": true,
		"__pycache__": true, ".cache": true, "coverage": true,
	}
	return common[name]
}

func sanitizeID(s string) string {
	id := strings.ReplaceAll(s, "/", "_")
	id = strings.ReplaceAll(id, "\\", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, " ", "_")
	return id
}
