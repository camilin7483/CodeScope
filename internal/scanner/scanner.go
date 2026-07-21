package scanner

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"codescope/internal/detector"
	"codescope/internal/types"
)

type result struct {
	info types.FileInfo
	err  error
}

type Scanner struct {
	root       string
	ignores    []string
	skippedDirs []string
	ignoredCount int
	binaryCount  int
	emptyCount   int
	errorCount   int
	mu          sync.Mutex
}

func New(root string) *Scanner {
	s := &Scanner{root: root}
	s.loadIgnores()
	return s
}

func (s *Scanner) Scan() ([]types.FileInfo, *types.DirNode) {
	type job struct {
		path string
		d    os.DirEntry
	}

	jobs := make(chan job, 10000)
	results := make(chan result, 10000)

	var walkWg sync.WaitGroup
	walkWg.Add(1)
	go func() {
		defer walkWg.Done()
		defer close(jobs)
		filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				s.mu.Lock()
				s.errorCount++
				s.mu.Unlock()
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				info, err := d.Info()
				if err == nil && info.Mode()&os.ModeSymlink != 0 {
					return filepath.SkipDir
				}
				if shouldSkipDir(name) {
					s.mu.Lock()
					s.skippedDirs = append(s.skippedDirs, name)
					s.mu.Unlock()
					return filepath.SkipDir
				}
				if strings.HasPrefix(name, ".") && name != "." {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				s.mu.Lock()
				s.ignoredCount++
				s.mu.Unlock()
				return nil
			}
			if s.shouldIgnore(path) {
				s.mu.Lock()
				s.ignoredCount++
				s.mu.Unlock()
				return nil
			}
			jobs <- job{path: path, d: d}
			return nil
		})
	}()

	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	var procWg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		procWg.Add(1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.mu.Lock()
					s.errorCount++
					s.mu.Unlock()
				}
				procWg.Done()
			}()
			for j := range jobs {
				fi, err := s.analyzeFile(j.path)
				if err != nil {
					s.mu.Lock()
					s.errorCount++
					s.mu.Unlock()
					continue
				}
				if fi != nil {
					results <- result{info: *fi}
				}
			}
		}()
	}

	go func() {
		procWg.Wait()
		close(results)
	}()

	var files []types.FileInfo
	for r := range results {
		r.info.Path = relativize(r.info.Path, s.root)
		for j := range r.info.Functions {
			r.info.Functions[j].File = r.info.Path
		}
		files = append(files, r.info)
	}

	tree := buildDirTree(files)
	return files, tree
}

func (s *Scanner) Stats() types.ScanStats {
	return types.ScanStats{
		IgnoredFiles: s.ignoredCount,
		IgnoredDirs:  s.skippedDirs,
		BinaryFiles:  s.binaryCount,
		EmptyFiles:   s.emptyCount,
		ErrorFiles:   s.errorCount,
	}
}

func (s *Scanner) loadIgnores() {
	gitignorePath := filepath.Join(s.root, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimRight(line, "/")
		s.ignores = append(s.ignores, line)
	}
}

func (s *Scanner) shouldIgnore(path string) bool {
	rel, err := filepath.Rel(s.root, path)
	if err != nil {
		return false
	}
	for _, pattern := range s.ignores {
		if matchGitignore(pattern, rel) {
			return true
		}
	}
	return false
}

func matchGitignore(pattern, path string) bool {
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}
	if strings.HasPrefix(pattern, "/") {
		if strings.HasPrefix(path, pattern[1:]) {
			return true
		}
		return false
	}
	if strings.Contains(path, "/"+pattern) || strings.HasPrefix(path, pattern) {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		base := strings.TrimSuffix(pattern, "/**")
		if strings.HasPrefix(path, base) {
			return true
		}
	}
	return false
}

func (s *Scanner) analyzeFile(path string) (*types.FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, nil
	}

	if fi.Size() == 0 {
		s.mu.Lock()
		s.emptyCount++
		s.mu.Unlock()
		return &types.FileInfo{
			Path:     path,
			Language: types.LanguageUnknown,
			Category: types.CategoryOther,
			Size:     0,
		}, nil
	}

	if isBinary(path) {
		s.mu.Lock()
		s.binaryCount++
		s.mu.Unlock()
		return nil, nil
	}

	lang := detectLanguageByExt(path)
	cat := categorizeFile(path, lang)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return analyzeContent(f, path, fi.Size(), lang, cat), nil
}

func analyzeContent(r io.Reader, path string, size int64, lang types.Language, cat types.FileCategory) *types.FileInfo {
	var (
		codeLines      int
		commentLines   int
		blankLines     int
		totalLines     int
		inBlock        bool
		functions      []types.Function
		totalComplexity int
		classes        int
		structs        int
		interfaces     int
		imports        []string
		braceDepth     int
		inFunc         bool
		inFuncName     string
		inFuncLine     int
		inFuncBrace    int
		funcComplexity int
	)

	commentDef := getCommentDefs(lang)
	funcPatterns := getFuncPatterns(lang)

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)

	lineNum := 0
	for sc.Scan() {
		lineNum++
		totalLines++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			blankLines++
			continue
		}

		if inBlock {
			commentLines++
			if strings.Contains(trimmed, commentDef.blockEnd) {
				inBlock = false
			}
			continue
		}

		if commentDef.blockStart != "" && strings.Contains(trimmed, commentDef.blockStart) {
			commentLines++
			if !strings.Contains(trimmed, commentDef.blockEnd) {
				inBlock = true
			}
			continue
		}

		if commentDef.line != "" {
			hasLineComment := false
			for _, marker := range strings.Split(commentDef.line, ",") {
				marker = strings.TrimSpace(marker)
				if marker != "" && strings.HasPrefix(strings.TrimLeft(line, " \t"), marker) {
					hasLineComment = true
					break
				}
			}
			if hasLineComment {
				commentLines++
				continue
			}
		}

		codeLines++
		lineComplexity := countComplexity(trimmed)
		totalComplexity += lineComplexity

		if inFunc {
			funcComplexity += lineComplexity
		}

		classes += countStructClassIfc(trimmed, lang, &structs, &interfaces)

		if len(funcPatterns) > 0 {
			funcName := matchFuncPattern(trimmed, funcPatterns, lang)
			if funcName != "" {
				inFunc = true
				inFuncName = funcName
				inFuncLine = lineNum
				inFuncBrace = braceDepth
				funcComplexity = lineComplexity
			}
		}

		countBraces(line, &braceDepth)

		if inFunc && braceDepth == inFuncBrace {
			inFunc = false
			if inFuncName != "" {
				functions = append(functions, types.Function{
					Name:       inFuncName,
					File:       path,
					Line:       inFuncLine,
					Lines:      lineNum - inFuncLine + 1,
					Complexity: funcComplexity,
				})
			}
			funcComplexity = 0
		}

		imp := matchImport(line, lang)
		if imp != "" {
			imports = append(imports, imp)
		}
	}

	return &types.FileInfo{
		Path:         path,
		Language:     lang,
		Category:     cat,
		Size:         size,
		Lines:        totalLines,
		CodeLines:    codeLines,
		CommentLines: commentLines,
		BlankLines:   blankLines,
		Functions:    functions,
		Imports:      imports,
		Complexity:   totalComplexity,
		Classes:      classes,
		Structs:      structs,
		Interfaces:   interfaces,
	}
}

type commentDef struct {
	line      string
	blockStart string
	blockEnd  string
}

func getCommentDefs(lang types.Language) commentDef {
	switch lang {
	case types.LanguageGo, types.LanguageRust, types.LanguageJavaScript, types.LanguageTypeScript:
		return commentDef{line: "//", blockStart: "/*", blockEnd: "*/"}
	case types.LanguageJava, types.LanguageKotlin, types.LanguageScala, types.LanguageC, types.LanguageCpp, types.LanguageCSharp, types.LanguageDart, types.LanguageSwift, types.LanguageZig:
		return commentDef{line: "//", blockStart: "/*", blockEnd: "*/"}
	case types.LanguagePython, types.LanguageRuby, types.LanguageJulia, types.LanguageNim:
		return commentDef{line: "#", blockStart: `"""`, blockEnd: `"""`}
	case types.LanguagePHP:
		return commentDef{line: "//,#", blockStart: "/*", blockEnd: "*/"}
	case types.LanguageLua:
		return commentDef{line: "--", blockStart: "--[[", blockEnd: "]]"}
	case types.LanguageElixir:
		return commentDef{line: "#", blockStart: "", blockEnd: ""}
	case types.LanguageHaskell:
		return commentDef{line: "--", blockStart: "{-", blockEnd: "-}"}
	case types.LanguageErlang:
		return commentDef{line: "%", blockStart: "", blockEnd: ""}
	case types.LanguageClojure, types.LanguageRacket:
		return commentDef{line: ";", blockStart: "", blockEnd: ""}
	default:
		return commentDef{line: "//,#,--", blockStart: "/*", blockEnd: "*/"}
	}
}

var funcPatterns = map[types.Language][]string{
	types.LanguageGo:         {`^\s*func\s+(\w+)`},
	types.LanguageRust:       {`^\s*(?:pub\s+)?fn\s+(\w+)`},
	types.LanguageJavaScript: {`^\s*function\s+(\w+)`, `^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)`},
	types.LanguageTypeScript: {`^\s*function\s+(\w+)`, `^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)`},
	types.LanguagePython:     {`^\s*def\s+(\w+)`, `^\s*async\s+def\s+(\w+)`},
	types.LanguageJava:       {`^\s*(?:public|private|protected|static|\s)*\s+[\w<>\[\]]+\s+(\w+)\s*\(`},
	types.LanguageKotlin:     {`^\s*(?:suspend\s+)?(?:fun|fun)\s+(\w+)`},
	types.LanguageScala:      {`^\s*def\s+(\w+)`},
	types.LanguageC:          {`^\s*[\w*]+\s+(\w+)\s*\(`},
	types.LanguageCpp:        {`^\s*(?:[\w:]+[\s*&]+)+(\w+)\s*\(`},
	types.LanguageCSharp:     {`^\s*(?:public|private|protected|internal|static|\s)*\s+[\w<>\[\]]+\s+(\w+)\s*\(`},
	types.LanguageRuby:       {`^\s*def\s+(\w+)`},
	types.LanguagePHP:        {`^\s*(?:public\s+)?function\s+(\w+)`},
	types.LanguageSwift:      {`^\s*func\s+(\w+)`},
	types.LanguageDart:       {`^\s*(?:Future\s+)?[\w<>]+\s+(\w+)\s*\(`},
	types.LanguageLua:        {`^\s*(?:local\s+)?function\s+(\w+)`},
	types.LanguageZig:        {`^\s*(?:pub\s+)?fn\s+(\w+)`},
	types.LanguageElixir:     {`^\s*def\s+(\w+)`},
	types.LanguageHaskell:    {`^\s*(\w+)\s*::`},
	types.LanguageErlang:     {`^\s*(\w+)\s*\(`},
	types.LanguageClojure:    {`^\s*\(defn\s+(\w+)`},
	types.LanguageRacket:     {`^\s*\(define\s+\((\w+)`},
	types.LanguageJulia:      {`^\s*function\s+(\w+)`},
	types.LanguageNim:        {`^\s*proc\s+(\w+)`},
}

func getFuncPatterns(lang types.Language) []string {
	return funcPatterns[lang]
}

var complexityKeywords = map[string]int{
	"if ": 1, "else if": 1, "for ": 1, "while ": 1,
	"case ": 1, "&&": 1, "||": 1, "catch ": 1,
}

func countComplexity(line string) int {
	count := 0
	lower := strings.ToLower(line)
	for kw, c := range complexityKeywords {
		if strings.Contains(lower, kw) {
			count += c
		}
	}
	return count
}

func countBraces(line string, depth *int) {
	for _, ch := range line {
		switch ch {
		case '{':
			*depth++
		case '}':
			*depth--
		}
	}
}

func matchFuncPattern(line string, patterns []string, lang types.Language) string {
	trimmed := strings.TrimSpace(line)

	switch lang {
	case types.LanguageGo:
		if strings.HasPrefix(trimmed, "func ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := strings.SplitN(parts[1], "(", 2)[0]
				return strings.TrimRight(name, "({")
			}
		}
	case types.LanguageRust, types.LanguageZig:
		clean := strings.TrimPrefix(trimmed, "pub ")
		if strings.HasPrefix(clean, "fn ") {
			parts := strings.Fields(clean)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({<")
			}
		}
	case types.LanguageJavaScript, types.LanguageTypeScript:
		if strings.HasPrefix(trimmed, "function ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({")
			}
		}
		if strings.HasPrefix(trimmed, "export function ") || strings.HasPrefix(trimmed, "export async function ") {
			parts := strings.Fields(trimmed)
			for i, p := range parts {
				if p == "function" && i+1 < len(parts) {
					return strings.TrimRight(parts[i+1], "({")
				}
			}
		}
		if strings.Contains(trimmed, "= function(") || strings.Contains(trimmed, "= (") || strings.Contains(trimmed, "=> {") {
			before := strings.SplitN(trimmed, "=", 2)[0]
			before = strings.TrimSpace(before)
			if before != "" && !strings.HasPrefix(before, "export") {
				return strings.TrimRight(before, " \t")
			}
		}
	case types.LanguagePython:
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") {
			parts := strings.Fields(strings.TrimPrefix(trimmed, "async "))
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({:")
			}
		}
	case types.LanguageJava:
		if strings.Contains(trimmed, "(") && !strings.HasSuffix(trimmed, ";") {
			parts := strings.Fields(trimmed)
			for i, p := range parts {
				if strings.Contains(p, "(") {
					name := strings.SplitN(p, "(", 2)[0]
					if name != "" && !strings.HasPrefix(name, "@") && name != "class" && name != "interface" && name != "enum" && name != "if" && name != "for" && name != "while" && name != "switch" {
						return name
					}
					if i > 0 && parts[i-1] != "class" {
						return parts[i-1]
					}
				}
			}
		}
	case types.LanguageKotlin:
		if strings.HasPrefix(trimmed, "fun ") || strings.HasPrefix(trimmed, "suspend fun ") {
			clean := trimmed
			clean = strings.TrimPrefix(clean, "suspend ")
			parts := strings.Fields(clean)
			if len(parts) >= 2 {
				name := strings.SplitN(parts[1], "(", 2)[0]
				return strings.TrimRight(name, "({")
			}
		}
	case types.LanguageRuby:
		if strings.HasPrefix(trimmed, "def ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				if parts[1] == "self." && len(parts) >= 3 {
					return "self." + strings.TrimRight(parts[2], "({")
				}
				return strings.TrimRight(parts[1], "({")
			}
		}
	case types.LanguagePHP:
		if strings.Contains(trimmed, "function ") {
			idx := strings.Index(trimmed, "function ")
			parts := strings.Fields(trimmed[idx:])
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({")
			}
		}
	case types.LanguageSwift:
		if strings.HasPrefix(trimmed, "func ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({")
			}
		}
	case types.LanguageLua:
		clean := strings.TrimPrefix(trimmed, "local ")
		if strings.HasPrefix(clean, "function ") {
			parts := strings.Fields(clean)
			if len(parts) >= 2 {
				name := parts[1]
				if idx := strings.Index(name, "."); idx > 0 {
					return name
				}
				return strings.TrimRight(name, "({")
			}
		}
	case types.LanguageElixir:
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "defp ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({")
			}
		}
	case types.LanguageC, types.LanguageCpp:
		if strings.Contains(trimmed, "(") && !strings.HasPrefix(trimmed, "#") && !strings.HasSuffix(trimmed, ";") {
			parts := strings.Fields(trimmed)
			for i, p := range parts {
				if strings.Contains(p, "(") && i > 0 {
					name := strings.SplitN(p, "(", 2)[0]
					if name != "" && !isKeyword(name) {
						return name
					}
				}
			}
		}
	case types.LanguageCSharp:
		if strings.Contains(trimmed, "(") && !strings.HasSuffix(trimmed, ";") {
			parts := strings.Fields(trimmed)
			for i, p := range parts {
				if strings.Contains(p, "(") {
					name := strings.SplitN(p, "(", 2)[0]
					if name != "" && !isKeyword(name) && !hasAngleBracket(name) {
						return name
					}
					if i > 0 && !isKeyword(parts[i-1]) {
						return parts[i-1]
					}
				}
			}
		}
	case types.LanguageHaskell:
		if strings.Contains(trimmed, " :: ") {
			parts := strings.SplitN(trimmed, " :: ", 2)
			return strings.TrimSpace(parts[0])
		}
	case types.LanguageJulia:
		if strings.HasPrefix(trimmed, "function ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({")
			}
		}
	case types.LanguageNim:
		if strings.HasPrefix(trimmed, "proc ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({")
			}
		}
	case types.LanguageClojure:
		if strings.HasPrefix(trimmed, "(defn ") {
			parts := strings.Fields(trimmed[1:])
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	case types.LanguageRacket:
		if strings.HasPrefix(trimmed, "(define (") {
			idx := strings.Index(trimmed, "(")
			if idx >= 0 {
				after := trimmed[idx+1:]
				parts := strings.Fields(after)
				if len(parts) >= 2 {
					return parts[1]
				}
			}
		}
	case types.LanguageDart:
		if strings.Contains(trimmed, "(") && (strings.Contains(trimmed, "void ") || strings.Contains(trimmed, " int ") || strings.Contains(trimmed, " String ") || strings.Contains(trimmed, " dynamic ") || strings.Contains(trimmed, "Widget ") || strings.Contains(trimmed, "Future<")) {
			parts := strings.Fields(trimmed)
			for i, p := range parts {
				if strings.Contains(p, "(") {
					name := strings.SplitN(p, "(", 2)[0]
					if name != "" {
						return name
					}
					if i > 0 {
						return parts[i-1]
					}
				}
			}
		}
	case types.LanguageScala:
		if strings.HasPrefix(trimmed, "def ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], "({")
			}
		}
	}
	return ""
}

func isKeyword(s string) bool {
	keywords := map[string]bool{
		"if": true, "else": true, "for": true, "while": true, "do": true,
		"switch": true, "case": true, "return": true, "break": true, "continue": true,
		"try": true, "catch": true, "throw": true, "new": true, "delete": true,
		"class": true, "struct": true, "interface": true, "enum": true,
		"public": true, "private": true, "protected": true, "static": true,
		"virtual": true, "override": true, "abstract": true, "const": true,
		"void": true, "int": true, "char": true, "bool": true, "double": true,
		"float": true, "long": true, "short": true, "unsigned": true,
		"String": true, "final": true, "volatile": true, "typedef": true,
	}
	return keywords[s]
}

func hasAngleBracket(s string) bool {
	for _, c := range s {
		if c == '<' || c == '>' {
			return true
		}
	}
	return false
}

func countStructClassIfc(trimmed string, lang types.Language, structs, interfaces *int) int {
	classes := 0
	lower := strings.ToLower(trimmed)

	switch lang {
	case types.LanguageGo:
		if strings.Contains(trimmed, "type ") && strings.Contains(trimmed, " struct") {
			*structs++
		} else if strings.Contains(trimmed, "type ") && strings.Contains(trimmed, " interface") {
			*interfaces++
		}
	case types.LanguageRust:
		if strings.Contains(lower, "struct ") {
			*structs++
		} else if strings.Contains(lower, "enum ") {
			classes++
		} else if strings.Contains(lower, "trait ") {
			*interfaces++
		}
	case types.LanguageJava, types.LanguageKotlin, types.LanguageScala, types.LanguageDart:
		if strings.Contains(lower, "class ") {
			classes++
		} else if strings.Contains(lower, "interface ") {
			*interfaces++
		} else if strings.Contains(lower, "object ") {
			classes++
		}
	case types.LanguageCpp:
		if strings.Contains(lower, "class ") {
			classes++
		} else if strings.Contains(lower, "struct ") {
			*structs++
		}
	case types.LanguageCSharp:
		if strings.Contains(lower, "class ") {
			classes++
		} else if strings.Contains(lower, "struct ") {
			*structs++
		} else if strings.Contains(lower, "interface ") {
			*interfaces++
		}
	case types.LanguagePython, types.LanguageRuby:
		if strings.Contains(lower, "class ") || strings.Contains(lower, "module ") {
			classes++
		}
	case types.LanguagePHP:
		if strings.Contains(lower, "class ") {
			classes++
		} else if strings.Contains(lower, "interface ") {
			*interfaces++
		} else if strings.Contains(lower, "trait ") {
			classes++
		}
	case types.LanguageSwift:
		if strings.Contains(lower, "class ") {
			classes++
		} else if strings.Contains(lower, "struct ") {
			*structs++
		} else if strings.Contains(lower, "protocol ") {
			*interfaces++
		} else if strings.Contains(lower, "enum ") {
			classes++
		}
	case types.LanguageJavaScript, types.LanguageTypeScript:
		if strings.Contains(lower, "class ") {
			classes++
		} else if strings.Contains(lower, "interface ") {
			*interfaces++
		}
	case types.LanguageNim:
		if strings.Contains(lower, "type ") && strings.Contains(lower, " = object") {
			*structs++
		}
	case types.LanguageZig:
		if strings.Contains(lower, "= struct") {
			*structs++
		}
	case types.LanguageHaskell:
		if strings.Contains(lower, "data ") && strings.Contains(lower, " =") {
			*structs++
		} else if strings.Contains(lower, "class ") && strings.Contains(lower, " where") {
			*interfaces++
		}
	case types.LanguageElixir:
		if strings.Contains(lower, "defmodule ") {
			classes++
		} else if strings.Contains(lower, "defstruct ") {
			*structs++
		}
	}
	return classes
}

func matchImport(line string, lang types.Language) string {
	trimmed := strings.TrimSpace(line)
	switch lang {
	case types.LanguageGo:
		if strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
			return strings.Trim(trimmed, `"`)
		}
	case types.LanguageJavaScript, types.LanguageTypeScript:
		if strings.HasPrefix(trimmed, "import ") {
			parts := strings.Fields(trimmed)
			for _, p := range parts {
				if strings.HasPrefix(p, `"`) || strings.HasPrefix(p, "'") || strings.HasPrefix(p, "`") {
					return strings.Trim(p, `"'`+"`")
				}
				if strings.HasPrefix(p, `from"`) || strings.HasPrefix(p, "from'") {
					return strings.Trim(p[4:], `"'`+"`")
				}
			}
		}
		if strings.Contains(trimmed, "require(") {
			start := strings.Index(trimmed, "require(") + 8
			end := strings.Index(trimmed[start:], ")")
			if end > 0 {
				imported := trimmed[start : start+end]
				return strings.Trim(imported, `"'`+"`")
			}
		}
	case types.LanguagePython:
		if strings.HasPrefix(trimmed, "import ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], ",")
			}
		}
		if strings.HasPrefix(trimmed, "from ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], ",")
			}
		}
	case types.LanguageJava:
		if strings.HasPrefix(trimmed, "import ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], ";")
			}
		}
	case types.LanguageRust:
		if strings.HasPrefix(trimmed, "use ") || strings.HasPrefix(trimmed, "extern crate ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				pkg := strings.TrimRight(parts[1], ";")
				pkg = strings.TrimLeft(pkg, "::")
				return pkg
			}
		}
	case types.LanguageRuby:
		if strings.HasPrefix(trimmed, "require ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.Trim(parts[1], `"'`)
			}
		}
	case types.LanguagePHP:
		if strings.HasPrefix(trimmed, "use ") && !strings.Contains(trimmed, "function") {
			parts := strings.Fields(trimmed)
			for i, p := range parts {
				if p == "use" && i+1 < len(parts) {
					return strings.TrimRight(parts[i+1], ";")
				}
			}
		}
	case types.LanguageLua:
		if strings.HasPrefix(trimmed, "require(") {
			start := strings.Index(trimmed, `"`) + 1
			end := strings.LastIndex(trimmed, `"`)
			if start > 0 && end > start {
				return trimmed[start:end]
			}
		}
	case types.LanguageHaskell:
		if strings.HasPrefix(trimmed, "import ") && !strings.Contains(trimmed, "qualified") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], ";,")
			}
		}
	case types.LanguageElixir:
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "alias ") || strings.HasPrefix(trimmed, "require ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	case types.LanguageDart:
		if strings.HasPrefix(trimmed, "import ") {
			start := strings.Index(trimmed, `"`) 
			end := strings.LastIndex(trimmed, `"`)
			if start < 0 {
				start = strings.Index(trimmed, `'`)
				end = strings.LastIndex(trimmed, `'`)
			}
			if start >= 0 && end > start {
				return trimmed[start+1 : end]
			}
		}
	case types.LanguageZig:
		if strings.HasPrefix(trimmed, "const ") && strings.Contains(trimmed, " = @import") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.Trim(parts[1], `"'();`)
			}
		}
	case types.LanguageCSharp:
		if strings.HasPrefix(trimmed, "using ") && !strings.Contains(trimmed, "(") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], ";")
			}
		}
	case types.LanguageSwift:
		if strings.HasPrefix(trimmed, "import ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	case types.LanguageJulia:
		if strings.HasPrefix(trimmed, "using ") || strings.HasPrefix(trimmed, "import ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], ",")
			}
		}
	case types.LanguageNim:
		if strings.HasPrefix(trimmed, "import ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	case types.LanguageScala:
		if strings.HasPrefix(trimmed, "import ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimRight(parts[1], ";")
			}
		}
	}
	return ""
}

func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return false
	}
	buf = buf[:n]

	for _, b := range buf {
		if b == 0 {
			return true
		}
	}
	return false
}

func detectLanguageByExt(path string) types.Language {
	ext := strings.ToLower(filepath.Ext(path))
	name := strings.ToLower(filepath.Base(path))

	extMap := map[string]types.Language{
		".go":   types.LanguageGo,
		".rs":   types.LanguageRust,
		".js":   types.LanguageJavaScript,
		".jsx":  types.LanguageJavaScript,
		".ts":   types.LanguageTypeScript,
		".tsx":  types.LanguageTypeScript,
		".mjs":  types.LanguageJavaScript,
		".cjs":  types.LanguageJavaScript,
		".mts":  types.LanguageTypeScript,
		".cts":  types.LanguageTypeScript,
		".py":   types.LanguagePython,
		".pyw":  types.LanguagePython,
		".pyi":  types.LanguagePython,
		".java": types.LanguageJava,
		".kt":   types.LanguageKotlin,
		".kts":  types.LanguageKotlin,
		".scala":types.LanguageScala,
		".sc":   types.LanguageScala,
		".c":    types.LanguageC,
		".h":    types.LanguageC,
		".cc":   types.LanguageCpp,
		".cpp":  types.LanguageCpp,
		".cxx":  types.LanguageCpp,
		".hpp":  types.LanguageCpp,
		".hh":   types.LanguageCpp,
		".hxx":  types.LanguageCpp,
		".cs":   types.LanguageCSharp,
		".rb":   types.LanguageRuby,
		".rake": types.LanguageRuby,
		".gemspec": types.LanguageRuby,
		".php":  types.LanguagePHP,
		".phtml":types.LanguagePHP,
		".swift":types.LanguageSwift,
		".dart": types.LanguageDart,
		".lua":  types.LanguageLua,
		".zig":  types.LanguageZig,
		".ex":   types.LanguageElixir,
		".exs":  types.LanguageElixir,
		".hs":   types.LanguageHaskell,
		".lhs":  types.LanguageHaskell,
		".erl":  types.LanguageErlang,
		".hrl":  types.LanguageErlang,
		".clj":  types.LanguageClojure,
		".cljs": types.LanguageClojure,
		".cljc": types.LanguageClojure,
		".edn":  types.LanguageClojure,
		".rkt":  types.LanguageRacket,
		".jl":   types.LanguageJulia,
		".nim":  types.LanguageNim,
	}

	if lang, ok := extMap[ext]; ok {
		return lang
	}

	if name == "go.mod" || name == "go.sum" {
		return types.LanguageGo
	}
	if name == "cargo.lock" || name == "cargo.toml" {
		return types.LanguageRust
	}

	return types.LanguageUnknown
}

func categorizeFile(path string, lang types.Language) types.FileCategory {
	if detector.IsTestFile(path) {
		return types.CategoryTest
	}
	if detector.IsDocFile(path) {
		return types.CategoryDocumentation
	}
	if detector.IsConfigFile(path) {
		return types.CategoryConfiguration
	}
	if detector.IsAssetFile(path) {
		return types.CategoryAsset
	}
	if lang != types.LanguageUnknown {
		return types.CategorySource
	}
	return types.CategoryOther
}

func shouldSkipDir(name string) bool {
	skipDirs := map[string]bool{
		"node_modules":    true, ".git": true, "__pycache__": true,
		".venv":           true, "venv": true, "vendor": true,
		".next":           true, ".nuxt": true, "dist": true,
		"build":           true, "target": true, "bin": true,
		"obj":             true, ".cache": true, ".yarn": true,
		".pnp":            true, ".svelte-kit": true, ".turbo": true,
		"coverage":        true, ".nyc_output": true, ".swc": true,
		".dart_tool":      true, "third_party": true,
	}
	return skipDirs[name]
}

func relativize(path, root string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func computeCumulative(node *types.DirNode) (files int, codeLines int) {
	if node == nil {
		return 0, 0
	}
	totalFiles := node.Files
	totalLines := node.CodeLines
	for _, child := range node.Children {
		cf, cl := computeCumulative(child)
		totalFiles += cf
		totalLines += cl
	}
	node.Files = totalFiles
	node.CodeLines = totalLines
	return totalFiles, totalLines
}

func buildDirTree(files []types.FileInfo) *types.DirNode {
	root := &types.DirNode{
		Name: ".",
		Path: ".",
	}
	dirs := map[string]*types.DirNode{".": root}

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if dir == "." {
			root.Files++
			root.Size += f.Size
			root.CodeLines += f.CodeLines
			continue
		}

		parts := strings.Split(dir, string(filepath.Separator))

		path := ""
		for _, part := range parts {
			if part == "" {
				continue
			}
			parentPath := path
			if path == "" {
				path = part
			} else {
				path = path + string(filepath.Separator) + part
			}
			if _, ok := dirs[path]; !ok {
				node := &types.DirNode{
					Name: part,
					Path: path,
				}
				dirs[path] = node
				if parent, ok := dirs[parentPath]; ok {
					parent.Children = append(parent.Children, node)
				} else {
					root.Children = append(root.Children, node)
				}
			}
		}

		if dn, ok := dirs[dir]; ok {
			dn.Files++
			dn.Size += f.Size
			dn.CodeLines += f.CodeLines
		}
	}

	computeCumulative(root)
	return root
}
