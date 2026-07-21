# CodeScope

Understand any codebase in seconds — from your terminal, no internet required.

CodeScope is a static analysis CLI that scans a project and produces a complete report: language detection, code metrics, architecture, health, risk, dependency graphs, and more.

## Install

```bash
go install github.com/camilin7483/CodeScope@latest
```

Or build from source:

```bash
git clone https://github.com/camilin7483/CodeScope.git
cd CodeScope
go build -o codescope .
```

Requires Go 1.22+.

## Usage

```bash
codescope                    # Full analysis of current directory
codescope scan ./project     # Scan a specific directory
codescope summary .          # Project overview
codescope metrics .          # LOC, functions, complexity, comments
codescope architecture .     # Entry points, modules, layers
codescope graph .            # Dependency graphs (Mermaid output)
codescope deps .             # Import/module dependencies
codescope health .           # README, license, CI, tests quality
codescope risk .             # Risk analysis
codescope hotspots .         # Most complex/changed files
codescope tree .             # Visual directory tree
codescope docs . ./docs      # Generate documentation site
codescope export . json      # Export as JSON / YAML / MD / HTML
codescope --help             # Show help
codescope --version          # Show version
```

## What it detects

CodeScope identifies the language, framework, build system, and package manager of any project. It then produces:

- **Code metrics** — Total and source lines, comment ratio, function count, cyclomatic complexity, largest files and functions
- **Architecture** — Entry points, internal modules, layer organization, shared utilities, dependency flow
- **Health checks** — Presence and quality of README, license, CI configuration, tests, documentation
- **Risk analysis** — High-complexity files, low-test-coverage areas, code duplication
- **Hotspots** — Files with highest complexity and change frequency
- **Dependency graph** — Import relationships rendered as text or Mermaid diagrams
- **Directory tree** — Full file tree with line counts

## Supported languages

| Language    | Status |
|-------------|--------|
| Go          | Done   |
| JavaScript  | Done   |
| TypeScript  | Done   |
| Rust        | Done   |
| Python      | Planned |
| Java        | Planned |
| C/C++       | Planned |

## Output formats

CodeScope can export its full report as JSON, YAML, Markdown, or HTML. The HTML output includes a standalone page with all findings. The docs command generates a multi-page documentation site with Mermaid diagrams.

## How it works

```
Filesystem → Scanner (parallel) → Detector (lang/framework)
    ↓
Metrics (LOC, complexity, functions)
Architecture (modules, layers, entry points)
Health (README, license, CI, tests)
Risk / Hotspots / Duplication
    ↓
Display (terminal tables, colored output)
Export (JSON, YAML, MD, HTML)
Docs (static site generator)
```

## Philosophy

- No AI — Every result comes from deterministic analysis
- No cloud — 100% local, works offline
- No telemetry — Zero data collection
- Parallel — Filesystem traversal uses concurrent workers
- Extensible — Add new languages by implementing the analyzer interface

## License

MIT
