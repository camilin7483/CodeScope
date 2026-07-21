package types

import "time"

type Language string

const (
	LanguageGo         Language = "Go"
	LanguageRust       Language = "Rust"
	LanguageJavaScript Language = "JavaScript"
	LanguageTypeScript Language = "TypeScript"
	LanguagePython     Language = "Python"
	LanguageJava       Language = "Java"
	LanguageKotlin     Language = "Kotlin"
	LanguageScala      Language = "Scala"
	LanguageC          Language = "C"
	LanguageCpp        Language = "C++"
	LanguageCSharp     Language = "C#"
	LanguageRuby       Language = "Ruby"
	LanguagePHP        Language = "PHP"
	LanguageSwift      Language = "Swift"
	LanguageDart       Language = "Dart"
	LanguageLua        Language = "Lua"
	LanguageZig        Language = "Zig"
	LanguageElixir     Language = "Elixir"
	LanguageHaskell    Language = "Haskell"
	LanguageErlang     Language = "Erlang"
	LanguageClojure    Language = "Clojure"
	LanguageRacket     Language = "Racket"
	LanguageJulia      Language = "Julia"
	LanguageNim        Language = "Nim"
	LanguageUnknown    Language = "Unknown"
)

type FileCategory int

const (
	CategorySource FileCategory = iota
	CategoryTest
	CategoryDocumentation
	CategoryConfiguration
	CategoryAsset
	CategoryOther
)

type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

type Project struct {
	Root           string   `json:"root"`
	Name           string   `json:"name"`
	Language       Language `json:"language"`
	Framework      string   `json:"framework,omitempty"`
	BuildSystem    string   `json:"build_system,omitempty"`
	PackageManager string   `json:"package_manager,omitempty"`
	HasGit         bool     `json:"has_git"`
	HasDocker      bool     `json:"has_docker"`
	HasCI          bool     `json:"has_ci"`
}

type Function struct {
	Name       string `json:"name"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Lines      int    `json:"lines"`
	Complexity int    `json:"complexity"`
}

type FileInfo struct {
	Path         string       `json:"path"`
	Language     Language     `json:"language"`
	Category     FileCategory `json:"category"`
	Size         int64        `json:"size"`
	Lines        int          `json:"lines"`
	CodeLines    int          `json:"code_lines"`
	CommentLines int          `json:"comment_lines"`
	BlankLines   int          `json:"blank_lines"`
	Functions    []Function   `json:"functions,omitempty"`
	Imports      []string     `json:"imports,omitempty"`
	Complexity   int          `json:"complexity"`
	Classes      int          `json:"classes"`
	Structs      int          `json:"structs"`
	Interfaces   int          `json:"interfaces"`
}

type DirNode struct {
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Size      int64      `json:"size"`
	Files     int        `json:"files"`
	CodeLines int        `json:"code_lines"`
	Children  []*DirNode `json:"children,omitempty"`
}

type DirInfo struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Files     int    `json:"files"`
	CodeLines int    `json:"code_lines"`
}

type ScanStats struct {
	TotalFiles    int      `json:"total_files"`
	SourceFiles   int      `json:"source_files"`
	IgnoredFiles  int      `json:"ignored_files"`
	IgnoredDirs   []string `json:"ignored_dirs"`
	BinaryFiles   int      `json:"binary_files"`
	EmptyFiles    int      `json:"empty_files"`
	ErrorFiles    int      `json:"error_files"`
	ScanTime      time.Duration `json:"scan_time"`
}

type Metrics struct {
	Project  Project       `json:"project"`
	ScanTime time.Duration `json:"scan_time"`
	ScanStats ScanStats    `json:"scan_stats"`

	TotalFiles  int `json:"total_files"`
	SourceFiles int `json:"source_files"`
	TestFiles   int `json:"test_files"`
	DocFiles    int `json:"doc_files"`
	ConfigFiles int `json:"config_files"`
	AssetFiles  int `json:"asset_files"`

	TotalLines      int `json:"total_lines"`
	CodeLines       int `json:"code_lines"`
	CommentLines    int `json:"comment_lines"`
	BlankLines      int `json:"blank_lines"`

	TotalFunctions    int     `json:"total_functions"`
	TotalClasses      int     `json:"total_classes"`
	TotalStructs      int     `json:"total_structs"`
	TotalInterfaces   int     `json:"total_interfaces"`

	CommentRatio    float64 `json:"comment_ratio"`
	AvgFileSize     float64 `json:"avg_file_size"`
	AvgFunctionSize float64 `json:"avg_function_size"`

	LargestFiles []FileInfo `json:"largest_files"`
	LargestFuncs []Function `json:"largest_functions"`
	LargestDirs  []DirInfo  `json:"largest_dirs"`

	Files   []FileInfo `json:"files,omitempty"`
	DirTree *DirNode   `json:"dir_tree,omitempty"`
}

type Insight struct {
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Category string   `json:"category"`
}

type RiskItem struct {
	File       string   `json:"file"`
	RiskScore  int      `json:"risk_score"`
	Reasons    []string `json:"reasons"`
	Lines      int      `json:"lines"`
	Complexity int      `json:"complexity"`
	Functions  int      `json:"functions"`
	HasTests   bool     `json:"has_tests"`
}

type Hotspot struct {
	File        string   `json:"file"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons"`
	ImportCount int      `json:"import_count"`
	Complexity  int      `json:"complexity"`
}

type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Size  int    `json:"size"`
}

type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type HealthStatus int

const (
	HealthPass HealthStatus = iota
	HealthWarn
	HealthFail
)

func (s HealthStatus) String() string {
	switch s {
	case HealthPass:
		return "PASS"
	case HealthWarn:
		return "WARN"
	case HealthFail:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

type HealthCategory string

const (
	HealthCritical    HealthCategory = "critical"
	HealthRecommended HealthCategory = "recommended"
	HealthOptional    HealthCategory = "optional"
)

type HealthCheck struct {
	Name     string         `json:"name"`
	Status   HealthStatus   `json:"status"`
	Message  string         `json:"message"`
	Category HealthCategory `json:"category"`
}

type HealthReport struct {
	Score           int           `json:"score"`
	CriticalPass    int           `json:"critical_pass"`
	CriticalTotal   int           `json:"critical_total"`
	RecommendedPass int           `json:"recommended_pass"`
	RecommendedTotal int          `json:"recommended_total"`
	OptionalPass    int           `json:"optional_pass"`
	OptionalTotal   int           `json:"optional_total"`
	Checks          []HealthCheck `json:"checks"`
	Recommendations []string      `json:"recommendations"`
}

type ArchAnalysis struct {
	EntryPoints       []string          `json:"entry_points"`
	InternalModules   []string          `json:"internal_modules"`
	LayerOrganization map[string]string `json:"layer_organization"`
	SharedUtilities   []string          `json:"shared_utilities"`
	DependencyFlow    Graph             `json:"dependency_flow"`
	FolderGraph       Graph             `json:"folder_graph"`
	CircularDeps      [][]string        `json:"circular_deps,omitempty"`
}
