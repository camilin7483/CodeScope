package detector

import (
	"os"
	"path/filepath"
	"strings"

	"codescope/internal/types"
)

func DetectProject(root string) types.Project {
	p := types.Project{Root: root}
	p.Name = filepath.Base(root)

	entries := listDir(root)

	p.Language = detectLanguage(root, entries)
	p.Framework = detectFramework(entries, p.Language)
	p.BuildSystem = detectBuildSystem(entries, p.Language)
	p.PackageManager = detectPackageManager(entries, p.Language)

	_, p.HasGit = findFile(root, ".git", true)
	_, dockerFound := findFile(root, "Dockerfile", false)
	p.HasDocker = dockerFound || hasFileWithPrefix(entries, "docker-compose")
	p.HasCI = hasCIFile(entries)

	return p
}

func detectLanguage(root string, entries []string) types.Language {
	if hasEntry(entries, "go.mod") {
		return types.LanguageGo
	}
	if hasEntry(entries, "Cargo.toml") {
		return types.LanguageRust
	}
	if hasEntry(entries, "package.json") {
		if hasEntry(entries, "tsconfig.json") {
			return types.LanguageTypeScript
		}
		return types.LanguageJavaScript
	}
	if hasEntry(entries, "deno.json") || hasEntry(entries, "deno.jsonc") {
		return types.LanguageTypeScript
	}
	if hasEntry(entries, "setup.py") || hasEntry(entries, "pyproject.toml") ||
		hasEntry(entries, "requirements.txt") || hasEntry(entries, "Pipfile") {
		return types.LanguagePython
	}
	if hasEntry(entries, "pom.xml") || hasEntry(entries, "build.gradle") ||
		hasEntry(entries, "build.gradle.kts") {
		if hasEntry(entries, "build.gradle.kts") || hasFileWithExtension(entries, ".kt") {
			return types.LanguageKotlin
		}
		return types.LanguageJava
	}
	if hasEntry(entries, "build.sbt") {
		return types.LanguageScala
	}
	if hasEntry(entries, "CMakeLists.txt") || hasEntry(entries, "Makefile") {
		for _, e := range entries {
			if strings.HasSuffix(e, ".c") || strings.HasSuffix(e, ".cpp") || strings.HasSuffix(e, ".cc") || strings.HasSuffix(e, ".h") || strings.HasSuffix(e, ".hpp") {
				return types.LanguageCpp
			}
		}
	}
	if hasEntry(entries, "Gemfile") || hasEntry(entries, "Rakefile") {
		return types.LanguageRuby
	}
	if hasEntry(entries, "composer.json") {
		return types.LanguagePHP
	}
	if hasEntry(entries, "Package.swift") || hasFileWithExtDeep(root, ".swift") {
		return types.LanguageSwift
	}
	if hasEntry(entries, "pubspec.yaml") {
		return types.LanguageDart
	}
	if hasEntry(entries, "mix.exs") {
		return types.LanguageElixir
	}
	if hasEntry(entries, "rebar.config") || hasEntry(entries, "erlang.mk") {
		return types.LanguageErlang
	}
	if hasEntry(entries, "project.clj") || hasEntry(entries, "deps.edn") || hasEntry(entries, "shadow-cljs.edn") {
		return types.LanguageClojure
	}
	if hasEntry(entries, "stack.yaml") || hasFileWithExtDeep(root, ".cabal") {
		return types.LanguageHaskell
	}
	if hasEntry(entries, "Project.toml") {
		return types.LanguageJulia
	}
	if hasEntry(entries, "*.nim.cfg") || hasFileWithExtDeep(root, ".nimble") {
		return types.LanguageNim
	}
	if hasEntry(entries, "build.zig") || hasFileWithExtDeep(root, ".zon") {
		return types.LanguageZig
	}
	if hasEntry(entries, "*.csproj") || hasEntry(entries, "*.sln") {
		return types.LanguageCSharp
	}

	if detectLanguageInSubdir(root, "package.json") {
		if detectLanguageInSubdir(root, "tsconfig.json") {
			return types.LanguageTypeScript
		}
		return types.LanguageJavaScript
	}

	return types.LanguageUnknown
}

func detectLanguageInSubdir(root, name string) bool {
	if hasEntry(listDir(root), name) {
		return true
	}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && e.Name() != "node_modules" {
			subRoot := filepath.Join(root, e.Name())
			if detectLanguageInSubdir(subRoot, name) {
				return true
			}
		}
	}
	return false
}

func detectFramework(entries []string, lang types.Language) string {
	switch lang {
	case types.LanguageJavaScript, types.LanguageTypeScript:
		if hasEntry(entries, "next.config.js") || hasEntry(entries, "next.config.ts") || hasDirWithPrefix(entries, "pages") {
			return "Next.js"
		}
		if hasEntry(entries, "remix.config.js") || hasEntry(entries, "remix.config.ts") {
			return "Remix"
		}
		if hasEntry(entries, "astro.config.mjs") || hasEntry(entries, "astro.config.ts") {
			return "Astro"
		}
		if hasEntry(entries, "nuxt.config.ts") || hasEntry(entries, "nuxt.config.js") {
			return "Nuxt.js"
		}
		if hasEntry(entries, "angular.json") {
			return "Angular"
		}
		if hasEntry(entries, "gatsby-config.js") || hasEntry(entries, "gatsby-config.ts") {
			return "Gatsby"
		}
		if hasEntry(entries, "vue.config.js") {
			return "Vue"
		}
		if hasEntry(entries, "svelte.config.js") {
			return "Svelte"
		}
		if hasEntry(entries, "svelte.config.js") || hasEntry(entries, "sveltekit.config.js") {
			return "SvelteKit"
		}
		if hasEntry(entries, "vite.config.ts") || hasEntry(entries, "vite.config.js") {
			return "Vite"
		}
		if hasEntry(entries, "webpack.config.js") {
			return "Webpack"
		}
		return "Node.js"
	case types.LanguagePython:
		if hasEntry(entries, "manage.py") {
			return "Django"
		}
		if hasEntry(entries, "app.py") || hasEntry(entries, "application.py") {
			return "Flask"
		}
		if hasEntry(entries, "fastapi") || hasFileWithExtDeep("", "fastapi") {
			return "FastAPI"
		}
		return "Python"
	case types.LanguageGo:
		if hasEntry(entries, "main.go") {
			return "Go CLI"
		}
		return "Go"
	case types.LanguageRust:
		return "Rust"
	case types.LanguageJava:
		if hasEntry(entries, "pom.xml") {
			return "Spring Boot"
		}
		if hasEntry(entries, "build.gradle") || hasEntry(entries, "build.gradle.kts") {
			return "Android/Gradle"
		}
		return "Java"
	case types.LanguageKotlin:
		if hasEntry(entries, "build.gradle.kts") {
			return "Android/Kotlin"
		}
		return "Kotlin"
	case types.LanguageScala:
		return "Scala"
	case types.LanguageC:
		return "C"
	case types.LanguageCpp:
		if hasEntry(entries, "Makefile") {
			return "Make-based"
		}
		if hasEntry(entries, "CMakeLists.txt") {
			return "CMake"
		}
		return "C++"
	case types.LanguageCSharp:
		if hasEntry(entries, "*.csproj") {
			return ".NET"
		}
		if hasEntry(entries, "*.sln") {
			return ".NET"
		}
		return "C#"
	case types.LanguageRuby:
		if hasEntry(entries, "config/routes.rb") || hasEntry(entries, "app/controllers") {
			return "Rails"
		}
		if hasEntry(entries, "Gemfile") {
			return "Ruby/RubyGems"
		}
		return "Ruby"
	case types.LanguagePHP:
		if hasEntry(entries, "artisan") {
			return "Laravel"
		}
		if hasEntry(entries, "symfony.lock") {
			return "Symfony"
		}
		return "PHP"
	case types.LanguageSwift:
		if hasEntry(entries, "Package.swift") {
			return "Swift/Swift PM"
		}
		return "Swift"
	case types.LanguageDart:
		return "Dart"
	case types.LanguageLua:
		return "Lua"
	case types.LanguageZig:
		return "Zig"
	case types.LanguageElixir:
		if hasEntry(entries, "mix.exs") {
			return "Phoenix"
		}
		return "Elixir"
	case types.LanguageHaskell:
		return "Haskell"
	case types.LanguageErlang:
		return "Erlang"
	case types.LanguageClojure:
		return "Clojure"
	case types.LanguageRacket:
		return "Racket"
	case types.LanguageJulia:
		return "Julia"
	case types.LanguageNim:
		return "Nim"
	}
	return ""
}

func detectBuildSystem(entries []string, lang types.Language) string {
	switch {
	case hasEntry(entries, "Makefile"):
		return "Make"
	case hasEntry(entries, "CMakeLists.txt"):
		return "CMake"
	case hasEntry(entries, "justfile"):
		return "Just"
	case hasEntry(entries, "taskfile.yml") || hasEntry(entries, "Taskfile.yml"):
		return "Task"
	case lang == types.LanguageGo:
		return "Go (built-in)"
	case lang == types.LanguageRust:
		return "Cargo"
	case lang == types.LanguageJava:
		if hasEntry(entries, "pom.xml") {
			return "Maven"
		}
		if hasEntry(entries, "build.gradle.kts") || hasEntry(entries, "build.gradle") {
			return "Gradle"
		}
		return "Java JDK"
	case lang == types.LanguageKotlin:
		return "Gradle (Kotlin DSL)"
	case lang == types.LanguageScala:
		return "sbt"
	case lang == types.LanguageCpp:
		return "CMake"
	case lang == types.LanguageC:
		return "Make"
	case lang == types.LanguageCSharp:
		if hasEntry(entries, "*.sln") || hasEntry(entries, "*.csproj") {
			return "MSBuild/.NET"
		}
		return "msbuild"
	case lang == types.LanguageRuby:
		return "Bundler/Rake"
	case lang == types.LanguagePHP:
		return "Composer"
	case lang == types.LanguageSwift:
		return "Swift PM"
	case lang == types.LanguageDart:
		return "Dart SDK"
	case lang == types.LanguageElixir:
		return "Mix"
	case lang == types.LanguageErlang:
		return "Rebar3"
	case lang == types.LanguageHaskell:
		return "Stack/Cabal"
	case lang == types.LanguageClojure:
		return "Leiningen/deps.edn"
	case lang == types.LanguageJulia:
		return "Julia Pkg"
	case lang == types.LanguageNim:
		return "Nimble"
	case lang == types.LanguageZig:
		return "Zig (built-in)"
	case lang == types.LanguageLua:
		return "LuaRocks"
	case lang == types.LanguageRacket:
		return "Racket"
	case hasEntry(entries, "pnpm-lock.yaml") || hasEntry(entries, "pnpm-workspace.yaml"):
		return "pnpm"
	case hasEntry(entries, "yarn.lock"):
		return "Yarn"
	case hasEntry(entries, "bun.lock") || hasEntry(entries, "bun.lockb"):
		return "Bun"
	case hasEntry(entries, "package-lock.json"):
		return "npm"
	case hasEntry(entries, "deno.json") || hasEntry(entries, "deno.jsonc"):
		return "Deno"
	}
	return "Unknown"
}

func detectPackageManager(entries []string, lang types.Language) string {
	switch {
	case hasEntry(entries, "pnpm-workspace.yaml") || hasEntry(entries, "pnpm-lock.yaml"):
		return "pnpm"
	case hasEntry(entries, "yarn.lock"):
		return "Yarn"
	case hasEntry(entries, "bun.lock") || hasEntry(entries, "bun.lockb"):
		return "Bun"
	case hasEntry(entries, "package-lock.json"):
		return "npm"
	case lang == types.LanguageGo:
		return "Go modules"
	case lang == types.LanguageRust:
		return "Cargo"
	case lang == types.LanguageJava:
		if hasEntry(entries, "pom.xml") {
			return "Maven Central"
		}
		return "Gradle/Maven"
	case lang == types.LanguagePython:
		if hasEntry(entries, "Pipfile") {
			return "Pipenv"
		}
		return "pip"
	case lang == types.LanguageRuby:
		return "RubyGems"
	case lang == types.LanguagePHP:
		return "Composer"
	case lang == types.LanguageDart:
		return "Dart Pub"
	case lang == types.LanguageElixir:
		return "Hex"
	case lang == types.LanguageHaskell:
		return "Hackage"
	case lang == types.LanguageJulia:
		return "Julia Pkg"
	case lang == types.LanguageNim:
		return "Nimble"
	case lang == types.LanguageLua:
		return "LuaRocks"
	case lang == types.LanguageSwift:
		return "Swift PM"
	case lang == types.LanguageErlang:
		return "Hex"
	case lang == types.LanguageClojure:
		return "Clojars"
	case hasEntry(entries, "deno.json") || hasEntry(entries, "deno.jsonc"):
		return "Deno"
	case hasEntry(entries, "Cargo.toml"):
		return "Cargo"
	}
	return ""
}

func hasCIFile(entries []string) bool {
	ciDirs := []string{".github", ".gitlab", ".circleci"}
	for _, e := range entries {
		for _, d := range ciDirs {
			if e == d {
				return true
			}
		}
	}
	if hasEntry(entries, ".travis.yml") || hasEntry(entries, "Jenkinsfile") {
		return true
	}
	return false
}

func listDir(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	entries, err := f.Readdirnames(-1)
	if err != nil {
		return nil
	}
	return entries
}

func hasFileWithExtension(entries []string, ext string) bool {
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e), ext) {
			return true
		}
	}
	return false
}

func hasFileWithExtDeep(root, ext string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name()), ext) {
			return true
		}
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			if hasFileWithExtDeep(filepath.Join(root, e.Name()), ext) {
				return true
			}
		}
	}
	return false
}

func hasEntry(entries []string, name string) bool {
	for _, e := range entries {
		if e == name {
			return true
		}
	}
	return false
}

func hasFileWithPrefix(entries []string, prefix string) bool {
	for _, e := range entries {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

func hasDirWithPrefix(entries []string, prefix string) bool {
	for _, e := range entries {
		if strings.HasPrefix(e, prefix) {
			info, err := os.Stat(e)
			if err == nil && info.IsDir() {
				return true
			}
		}
	}
	return false
}

func findFile(root, name string, isDir bool) (string, bool) {
	path := filepath.Join(root, name)
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	if isDir && !info.IsDir() {
		return "", false
	}
	if !isDir && info.IsDir() {
		return "", false
	}
	return path, true
}

func IsTestFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(name, ext)

	switch ext {
	case ".go":
		return strings.HasSuffix(name, "_test.go")
	case ".rs":
		return strings.Contains(base, "_test") || strings.Contains(base, "_spec") || strings.Contains(base, "test_")
	case ".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs", ".mts", ".cts":
		return strings.Contains(base, ".test") ||
			strings.Contains(base, ".spec") ||
			strings.Contains(base, "_test") ||
			strings.Contains(base, "_spec") ||
			strings.Contains(base, ".test.") ||
			strings.Contains(base, ".spec.") ||
			strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, ".test.ts") ||
			strings.HasSuffix(name, ".spec.js") || strings.HasSuffix(name, ".spec.ts") ||
			strings.Contains(filepath.Dir(path), "__tests__")
	case ".py":
		return strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test.py") || strings.Contains(filepath.Dir(path), "tests")
	case ".java":
		return strings.HasSuffix(name, "Test.java") || strings.HasSuffix(name, "Tests.java") || strings.Contains(filepath.Dir(path), "test")
	case ".kt":
		return strings.HasSuffix(name, "Test.kt") || strings.HasSuffix(name, "Tests.kt") || strings.Contains(filepath.Dir(path), "test")
	case ".scala":
		return strings.HasSuffix(name, "Test.scala") || strings.HasSuffix(name, "Spec.scala")
	case ".rb":
		return strings.HasSuffix(name, "_test.rb") || strings.HasSuffix(name, "_spec.rb") || strings.Contains(filepath.Dir(path), "spec") || strings.Contains(filepath.Dir(path), "test")
	case ".php":
		return strings.HasSuffix(name, "Test.php") || strings.HasSuffix(name, "TestCase.php") || strings.Contains(filepath.Dir(path), "test") || strings.HasSuffix(name, "_test.php")
	case ".swift":
		return strings.HasSuffix(name, "Tests.swift") || strings.Contains(filepath.Dir(path), "Tests")
	case ".dart":
		return strings.HasSuffix(name, "_test.dart") || strings.Contains(filepath.Dir(path), "test")
	case ".lua":
		return strings.HasSuffix(name, "_test.lua") || strings.HasSuffix(name, "_spec.lua")
	case ".cs":
		return strings.HasSuffix(name, "Tests.cs") || strings.HasSuffix(name, "Test.cs") || strings.Contains(filepath.Dir(path), "Test")
	case ".erl":
		return strings.HasSuffix(name, "_tests.erl") || strings.HasSuffix(name, "_test.erl") || strings.Contains(filepath.Dir(path), "test")
	case ".ex":
		return strings.HasSuffix(name, "_test.exs") || strings.Contains(filepath.Dir(path), "test")
	case ".hs":
		return strings.HasSuffix(name, "Spec.hs") || strings.HasSuffix(name, "Test.hs") || strings.Contains(filepath.Dir(path), "test") || strings.Contains(filepath.Dir(path), "spec")
	case ".clj", ".cljs":
		return strings.Contains(base, "_test") || strings.Contains(base, "test_")
	case ".jl":
		return strings.Contains(base, "test_") || strings.Contains(filepath.Dir(path), "test")
	case ".nim":
		return strings.Contains(base, "test_") || strings.Contains(base, "_test")
	case ".c", ".cpp", ".cc", ".cxx":
		return strings.HasSuffix(name, "_test.c") || strings.HasSuffix(name, "_test.cpp") || strings.Contains(name, "test_")
	case ".zig":
		return strings.HasSuffix(name, "_test.zig") || strings.Contains(filepath.Dir(path), "test")
	case ".exs":
		return true
	}

	if strings.Contains(base, "_test") || strings.Contains(base, "_spec") || strings.HasPrefix(base, "test_") {
		return true
	}
	if strings.Contains(filepath.Dir(path), "test") || strings.Contains(filepath.Dir(path), "spec") || strings.Contains(filepath.Dir(path), "__tests__") {
		return true
	}
	return false
}

func IsDocFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	name := strings.ToLower(filepath.Base(path))

	docExts := map[string]bool{
		".md":       true,
		".rst":      true,
		".txt":      true,
		".adoc":     true,
		".asciidoc": true,
		".org":      true,
		".wiki":     true,
		".mdown":    true,
		".markdown": true,
	}

	if docExts[ext] {
		return true
	}

	docNames := map[string]bool{
		"readme": true, "license": true, "changelog": true,
		"contributing": true, "code_of_conduct": true, "security": true,
		"authors": true, "acknowledgments": true, "history": true,
		"upgrade": true, "migration": true, "todo": true,
	}
	return docNames[strings.TrimSuffix(name, ext)]
}

func IsConfigFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(path)

	configNames := map[string]bool{
		"package.json": true, "tsconfig.json": true, ".eslintrc.js": true,
		".eslintrc.json": true, ".eslintrc": true, ".prettierrc": true,
		".prettierrc.json": true, ".prettierrc.js": true, ".gitignore": true,
		".editorconfig": true, ".env": true, ".env.example": true,
		"go.mod": true, "go.sum": true, "Cargo.toml": true, "Cargo.lock": true,
		"Makefile": true, "Dockerfile": true, "docker-compose.yml": true,
		"docker-compose.yaml": true, ".dockerignore": true,
		".github": true, ".gitlab-ci.yml": true, ".cirrus.yml": true,
		"babel.config.js": true, "babel.config.json": true,
		"webpack.config.js": true, "vite.config.ts": true, "vite.config.js": true,
		"next.config.js": true, "next.config.ts": true,
		"jest.config.js": true, "jest.config.ts": true,
		".stylelintrc": true, ".stylelintrc.json": true,
		"pom.xml": true, "build.gradle": true, "build.gradle.kts": true,
		"settings.gradle": true, "settings.gradle.kts": true,
		"gradle.properties": true, "mvnw": true, "gradlew": true,
		"deno.json": true, "deno.jsonc": true,
		"pnpm-workspace.yaml": true, "pnpm-lock.yaml": true,
		"yarn.lock": true, "bun.lock": true, "bun.lockb": true,
	}

	if configNames[name] || configNames[path] {
		return true
	}

	if ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".cfg" || ext == ".conf" {
		return !IsDocFile(path)
	}

	if strings.HasPrefix(name, ".") {
		return true
	}

	return false
}

func IsAssetFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	assetExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true,
		".ico": true, ".webp": true, ".bmp": true, ".tiff": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
		".mp4": true, ".webm": true, ".avi": true, ".mov": true, ".mkv": true,
		".mp3": true, ".wav": true, ".ogg": true, ".flac": true, ".aac": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true, ".bz2": true,
		".7z": true, ".rar": true, ".xz": true,
	}
	return assetExts[ext]
}
