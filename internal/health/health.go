package health

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"codescope/internal/types"
)

func Check(project types.Project, files []types.FileInfo) *types.HealthReport {
	report := &types.HealthReport{}
	score := 100

	critical := []struct {
		name    string
		weight  int
		checkFn func() types.HealthCheck
	}{
		{"README", 20, func() types.HealthCheck { return checkCriticalFile(project.Root, types.HealthCritical, 20, "README.md", "README.txt", "README") }},
		{"LICENSE", 15, func() types.HealthCheck { return checkCriticalFile(project.Root, types.HealthCritical, 15, "LICENSE", "LICENSE.txt", "LICENSE.md") }},
		{"Tests", 20, func() types.HealthCheck { return checkTests(project, files) }},
		{"Build System", 10, func() types.HealthCheck { return checkBuildSystem(project) }},
	}

	recommended := []struct {
		name    string
		weight  int
		checkFn func() types.HealthCheck
	}{
		{"CHANGELOG", 8, func() types.HealthCheck { return checkRecommendedFile(project.Root, types.HealthRecommended, 8, "CHANGELOG.md", "CHANGELOG", "CHANGELOG.txt") }},
		{"CONTRIBUTING", 5, func() types.HealthCheck { return checkRecommendedFile(project.Root, types.HealthRecommended, 5, "CONTRIBUTING.md", "CONTRIBUTING", "CONTRIBUTING.txt") }},
		{".gitignore", 8, func() types.HealthCheck { return checkRecommendedFile(project.Root, types.HealthRecommended, 8, ".gitignore") }},
		{".editorconfig", 5, func() types.HealthCheck { return checkRecommendedFile(project.Root, types.HealthRecommended, 5, ".editorconfig") }},
		{"CI Configuration", 8, func() types.HealthCheck { return checkCI(project) }},
		{"Git Repository", 3, func() types.HealthCheck { return checkGit(project) }},
	}

	optional := []struct {
		name    string
		weight  int
		checkFn func() types.HealthCheck
	}{
		{"SECURITY.md", 3, func() types.HealthCheck { return checkOptionalFile(project.Root, types.HealthOptional, 3, "SECURITY.md", "SECURITY") }},
		{"CODE_OF_CONDUCT", 3, func() types.HealthCheck { return checkOptionalFile(project.Root, types.HealthOptional, 3, "CODE_OF_CONDUCT.md", "CODE_OF_CONDUCT") }},
		{"Documentation", 5, func() types.HealthCheck { return checkDocs(files) }},
		{"Organization", 5, func() types.HealthCheck { return checkOrganization(files) }},
	}

	report.CriticalTotal = len(critical)
	report.RecommendedTotal = len(recommended)
	report.OptionalTotal = len(optional)

	for _, c := range critical {
		check := c.checkFn()
		report.Checks = append(report.Checks, check)
		if check.Status == types.HealthPass {
			report.CriticalPass++
		} else {
			score -= c.weight
		}
	}

	for _, c := range recommended {
		check := c.checkFn()
		report.Checks = append(report.Checks, check)
		if check.Status == types.HealthPass {
			report.RecommendedPass++
		} else if check.Status == types.HealthWarn {
			score -= c.weight / 2
		} else {
			score -= c.weight
		}
	}

	for _, c := range optional {
		check := c.checkFn()
		report.Checks = append(report.Checks, check)
		if check.Status == types.HealthPass {
			report.OptionalPass++
		} else if check.Status == types.HealthWarn {
			score -= c.weight / 2
		} else {
			score -= c.weight / 4
		}
	}

	if score < 0 {
		score = 0
	}
	report.Score = score

	if score < 50 {
		report.Recommendations = append(report.Recommendations,
			"Add essential files: README, LICENSE, and test coverage")
	}
	if !hasTests(files) {
		report.Recommendations = append(report.Recommendations,
			"Add automated tests to improve code reliability")
	}
	if !project.HasGit {
		report.Recommendations = append(report.Recommendations,
			"Initialize Git repository for version control")
	}
	if !hasFile(project.Root, ".gitignore") {
		report.Recommendations = append(report.Recommendations,
			"Add .gitignore to exclude build artifacts from version control")
	}
	if !hasFile(project.Root, ".editorconfig") {
		report.Recommendations = append(report.Recommendations,
			"Add .editorconfig for consistent coding styles across editors")
	}

	sort.Slice(report.Checks, func(i, j int) bool {
		order := map[types.HealthCategory]int{
			types.HealthCritical:    0,
			types.HealthRecommended: 1,
			types.HealthOptional:    2,
		}
		oi := order[report.Checks[i].Category]
		oj := order[report.Checks[j].Category]
		if oi != oj {
			return oi < oj
		}
		return report.Checks[i].Name < report.Checks[j].Name
	})

	return report
}

func checkCriticalFile(root string, cat types.HealthCategory, weight int, names ...string) types.HealthCheck {
	for _, name := range names {
		if hasFile(root, name) {
			return types.HealthCheck{
				Name:     fmt.Sprintf("%s", names[0]),
				Status:   types.HealthPass,
				Message:  fmt.Sprintf("%s exists", name),
				Category: cat,
			}
		}
	}
	return types.HealthCheck{
		Name:     fmt.Sprintf("%s", names[0]),
		Status:   types.HealthFail,
		Message:  fmt.Sprintf("No %s file found", names[0]),
		Category: cat,
	}
}

func checkRecommendedFile(root string, cat types.HealthCategory, weight int, names ...string) types.HealthCheck {
	for _, name := range names {
		if hasFile(root, name) {
			return types.HealthCheck{
				Name:     fmt.Sprintf("%s", names[0]),
				Status:   types.HealthPass,
				Message:  fmt.Sprintf("%s exists", name),
				Category: cat,
			}
		}
	}
	return types.HealthCheck{
		Name:     fmt.Sprintf("%s", names[0]),
		Status:   types.HealthWarn,
		Message:  fmt.Sprintf("No %s file found", names[0]),
		Category: cat,
	}
}

func checkOptionalFile(root string, cat types.HealthCategory, weight int, names ...string) types.HealthCheck {
	for _, name := range names {
		if hasFile(root, name) {
			return types.HealthCheck{
				Name:     fmt.Sprintf("%s", names[0]),
				Status:   types.HealthPass,
				Message:  fmt.Sprintf("%s exists", name),
				Category: cat,
			}
		}
	}
	return types.HealthCheck{
		Name:     fmt.Sprintf("%s", names[0]),
		Status:   types.HealthFail,
		Message:  fmt.Sprintf("No %s file found (optional)", names[0]),
		Category: cat,
	}
}

func hasFile(root, name string) bool {
	path := filepath.Join(root, name)
	_, err := os.Stat(path)
	return err == nil
}

func checkBuildSystem(project types.Project) types.HealthCheck {
	if project.BuildSystem != "" && project.BuildSystem != "Unknown" {
		return types.HealthCheck{
			Name:     "Build System",
			Status:   types.HealthPass,
			Message:  fmt.Sprintf("Detected: %s", project.BuildSystem),
			Category: types.HealthCritical,
		}
	}
	return types.HealthCheck{
		Name:     "Build System",
		Status:   types.HealthWarn,
		Message:  "No build system detected",
		Category: types.HealthCritical,
	}
}

func checkCI(project types.Project) types.HealthCheck {
	if project.HasCI {
		return types.HealthCheck{
			Name:     "CI Configuration",
			Status:   types.HealthPass,
			Message:  "CI configuration detected",
			Category: types.HealthRecommended,
		}
	}
	ciFiles := []string{".github/workflows", ".gitlab-ci.yml", ".cirrus.yml", ".travis.yml", "Jenkinsfile"}
	for _, f := range ciFiles {
		if hasFile(project.Root, f) {
			return types.HealthCheck{
				Name:     "CI Configuration",
				Status:   types.HealthPass,
				Message:  fmt.Sprintf("Found %s", f),
				Category: types.HealthRecommended,
			}
		}
	}
	return types.HealthCheck{
		Name:     "CI Configuration",
		Status:   types.HealthWarn,
		Message:  "No CI configuration detected",
		Category: types.HealthRecommended,
	}
}

func checkGit(project types.Project) types.HealthCheck {
	if project.HasGit {
		return types.HealthCheck{
			Name:     "Git Repository",
			Status:   types.HealthPass,
			Message:  "Git repository initialized",
			Category: types.HealthRecommended,
		}
	}
	return types.HealthCheck{
		Name:     "Git Repository",
		Status:   types.HealthWarn,
		Message:  "Not a Git repository",
		Category: types.HealthRecommended,
	}
}

func checkTests(project types.Project, files []types.FileInfo) types.HealthCheck {
	var testCount int
	for _, f := range files {
		if f.Category == types.CategoryTest {
			testCount++
		}
	}
	if testCount == 0 {
		return types.HealthCheck{
			Name:     "Tests",
			Status:   types.HealthFail,
			Message:  "No test files found",
			Category: types.HealthCritical,
		}
	}
	total := len(files)
	coverage := float64(testCount) / float64(total) * 100
	status := types.HealthPass
	if coverage < 5 {
		status = types.HealthWarn
	}
	return types.HealthCheck{
		Name:     "Tests",
		Status:   status,
		Message:  fmt.Sprintf("%d test file(s) (%.0f%% of all files)", testCount, coverage),
		Category: types.HealthCritical,
	}
}

func checkDocs(files []types.FileInfo) types.HealthCheck {
	var docCount int
	for _, f := range files {
		if f.Category == types.CategoryDocumentation {
			docCount++
		}
	}
	if docCount == 0 {
		return types.HealthCheck{
			Name:     "Documentation",
			Status:   types.HealthWarn,
			Message:  "No documentation files found",
			Category: types.HealthOptional,
		}
	}
	return types.HealthCheck{
		Name:     "Documentation",
		Status:   types.HealthPass,
		Message:  fmt.Sprintf("Found %d documentation file(s)", docCount),
		Category: types.HealthOptional,
	}
}

func checkOrganization(files []types.FileInfo) types.HealthCheck {
	dirs := make(map[string]bool)
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		dirs[dir] = true
	}
	var srcDirs []string
	for d := range dirs {
		base := filepath.Base(d)
		switch base {
		case "src", "lib", "cmd", "internal", "pkg", "app":
			srcDirs = append(srcDirs, d)
		}
	}
	message := "Project has a standard layout"
	if len(srcDirs) > 0 {
		message = fmt.Sprintf("Well-organized with %d source directories", len(srcDirs))
	}
	return types.HealthCheck{
		Name:     "Organization",
		Status:   types.HealthPass,
		Message:  message,
		Category: types.HealthOptional,
	}
}

func hasTests(files []types.FileInfo) bool {
	for _, f := range files {
		if f.Category == types.CategoryTest {
			return true
		}
	}
	return false
}
