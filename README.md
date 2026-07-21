# CodeScope

Understand any codebase in seconds.

CodeScope is a modern, high-performance static analysis CLI that helps developers understand the architecture, structure, quality, and health of any software project.

**No AI. No cloud. No telemetry. No internet required.**

## Features

- **Project Detection** — Automatically detect language, framework, build system, and package manager
- **Code Metrics** — Lines of code, comment ratio, function count, cyclomatic complexity
- **Architecture Analysis** — Entry points, modules, layer organization, dependency flow
- **Health Checks** — README, LICENSE, CI, tests, documentation quality
- **Dependency Graph** — Import relationships and module dependencies
- **Directory Tree** — Visual tree with file and line counts
- **Export** — JSON, YAML, Markdown, HTML
- **Documentation Generation** — Auto-generated docs with Mermaid diagrams
- **Plugin Architecture** — Easy to extend with new language support

## Installation

```bash
go install github.com/codescope/codescope@latest
```

Or build from source:

```bash
git clone https://github.com/codescope/codescope.git
cd codescope
go build -o codescope .
```

## Usage

```bash
# Full analysis of current directory
codescope .

# Scan a specific project
codescope scan ./my-project

# Project summary
codescope summary .

# Code metrics
codescope metrics .

# Architecture analysis
codescope architecture .

# Repository health check
codescope health .

# Directory tree
codescope tree .

# Dependency analysis
codescope deps .

# Relationship graphs (with Mermaid diagrams)
codescope graph .

# Export report
codescope export . json report.json
codescope export . yaml report.yaml
codescope export . md report.md
codescope export . html report.html

# Generate documentation
codescope docs . ./docs
```

## Supported Languages

| Language | Status |
|----------|--------|
| Go       | ✅      |
| JavaScript | ✅    |
| TypeScript | ✅    |
| Rust     | ✅      |
| Python   | 🚧 Planned |
| Java     | 🚧 Planned |
| C        | 🚧 Planned |
| C++      | 🚧 Planned |

## Architecture

```
cmd/           — CLI commands
internal/
  types/       — Shared data types
  detector/    — Language/framework detection
  scanner/     — Parallel filesystem traversal
  metrics/     — Code metrics and analysis
  display/     — Terminal output (tables, tree, colors)
  health/      — Repository health checks
  analysis/    — Architecture and dependency analysis
  export/      — Output formats (JSON, YAML, MD, HTML)
```

## Philosophy

- **No AI** — Every result comes from deterministic static analysis
- **No cloud** — Everything runs locally, offline
- **No telemetry** — Zero data collection
- **Fast** — Parallel filesystem traversal with efficient algorithms
- **Deterministic** — Same input always produces the same output
- **Extensible** — Add new languages by implementing the analyzer interface

## License

MIT
