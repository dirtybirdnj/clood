package analyze

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CodebaseAnalysis holds all static analysis results
type CodebaseAnalysis struct {
	ProjectRoot   string
	Timestamp     time.Time
	LineCount     LineCount
	GoVet         VetResult
	BuildStatus   BuildResult
	TestStatus    TestResult
	TodoItems     []TodoItem
	RecentCommits []string
	ChangedFiles  []string
	Symbols       SymbolSummary
}

// LineCount tracks lines of code by type
type LineCount struct {
	GoLines    int
	GoFiles    int
	TestLines  int
	TestFiles  int
	TotalLines int
	TotalFiles int
}

// VetResult holds go vet output
type VetResult struct {
	Clean    bool
	Issues   []string
	Duration time.Duration
}

// BuildResult holds go build status
type BuildResult struct {
	Success  bool
	Errors   []string
	Duration time.Duration
}

// TestResult holds go test output
type TestResult struct {
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Output   string
}

// TodoItem represents a TODO/FIXME in code
type TodoItem struct {
	File    string
	Line    int
	Type    string // TODO, FIXME, HACK, XXX
	Content string
}

// SymbolSummary counts exported symbols
type SymbolSummary struct {
	Functions int
	Types     int
	Methods   int
	Constants int
}

// RunAnalysis performs full static analysis on a Go project
func RunAnalysis(projectRoot string, runTests bool) (*CodebaseAnalysis, error) {
	analysis := &CodebaseAnalysis{
		ProjectRoot: projectRoot,
		Timestamp:   time.Now(),
	}

	// All analysis runs in parallel where possible
	type result struct {
		name string
		err  error
	}
	results := make(chan result, 6)

	// Line count
	go func() {
		analysis.LineCount = countLines(projectRoot)
		results <- result{"linecount", nil}
	}()

	// Go vet
	go func() {
		analysis.GoVet = runGoVet(projectRoot)
		results <- result{"govet", nil}
	}()

	// Build check
	go func() {
		analysis.BuildStatus = runGoBuild(projectRoot)
		results <- result{"build", nil}
	}()

	// TODO/FIXME scan
	go func() {
		analysis.TodoItems = findTodos(projectRoot)
		results <- result{"todos", nil}
	}()

	// Git history
	go func() {
		analysis.RecentCommits = getRecentCommits(projectRoot, 10)
		analysis.ChangedFiles = getChangedFiles(projectRoot, 5)
		results <- result{"git", nil}
	}()

	// Symbol summary
	go func() {
		analysis.Symbols = countSymbols(projectRoot)
		results <- result{"symbols", nil}
	}()

	// Wait for parallel tasks
	for i := 0; i < 6; i++ {
		<-results
	}

	// Tests run separately (slow)
	if runTests {
		analysis.TestStatus = runGoTest(projectRoot)
	}

	return analysis, nil
}

// FormatForClaude produces a context string optimized for Claude ingestion
func (a *CodebaseAnalysis) FormatForClaude() string {
	var sb strings.Builder

	sb.WriteString("# STATIC ANALYSIS CONTEXT (pre-computed for efficiency)\n\n")
	sb.WriteString(fmt.Sprintf("Project: %s\n", a.ProjectRoot))
	sb.WriteString(fmt.Sprintf("Analyzed: %s\n\n", a.Timestamp.Format("2006-01-02 15:04:05")))

	// Scope
	sb.WriteString("## PROJECT SCOPE\n")
	sb.WriteString(fmt.Sprintf("- Go files: %d (%d lines)\n", a.LineCount.GoFiles, a.LineCount.GoLines))
	sb.WriteString(fmt.Sprintf("- Test files: %d (%d lines)\n", a.LineCount.TestFiles, a.LineCount.TestLines))
	sb.WriteString(fmt.Sprintf("- Symbols: %d funcs, %d types, %d methods\n\n",
		a.Symbols.Functions, a.Symbols.Types, a.Symbols.Methods))

	// Build status
	sb.WriteString("## BUILD STATUS\n")
	if a.BuildStatus.Success {
		sb.WriteString("- BUILD: PASS\n")
	} else {
		sb.WriteString("- BUILD: FAIL\n")
		for _, e := range a.BuildStatus.Errors {
			sb.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}
	sb.WriteString("\n")

	// Go vet
	sb.WriteString("## GO VET\n")
	if a.GoVet.Clean {
		sb.WriteString("- VET: CLEAN\n")
	} else {
		sb.WriteString(fmt.Sprintf("- VET: %d issues\n", len(a.GoVet.Issues)))
		for _, issue := range a.GoVet.Issues {
			sb.WriteString(fmt.Sprintf("  - %s\n", issue))
		}
	}
	sb.WriteString("\n")

	// Test status (if run)
	if a.TestStatus.Duration > 0 {
		sb.WriteString("## TESTS\n")
		sb.WriteString(fmt.Sprintf("- Passed: %d, Failed: %d, Skipped: %d\n",
			a.TestStatus.Passed, a.TestStatus.Failed, a.TestStatus.Skipped))
		if a.TestStatus.Failed > 0 {
			sb.WriteString("- FAILURES:\n")
			sb.WriteString(a.TestStatus.Output)
		}
		sb.WriteString("\n")
	}

	// TODOs/FIXMEs
	if len(a.TodoItems) > 0 {
		sb.WriteString("## ACTION ITEMS (TODO/FIXME)\n")
		for _, item := range a.TodoItems {
			sb.WriteString(fmt.Sprintf("- [%s] %s:%d - %s\n",
				item.Type, item.File, item.Line, item.Content))
		}
		sb.WriteString("\n")
	}

	// Recent activity
	if len(a.RecentCommits) > 0 {
		sb.WriteString("## RECENT COMMITS (last 10)\n")
		for _, commit := range a.RecentCommits {
			sb.WriteString(fmt.Sprintf("- %s\n", commit))
		}
		sb.WriteString("\n")
	}

	if len(a.ChangedFiles) > 0 {
		sb.WriteString("## HOT FILES (recently changed)\n")
		for _, f := range a.ChangedFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatSummary returns a brief one-liner
func (a *CodebaseAnalysis) FormatSummary() string {
	status := "READY"
	if !a.BuildStatus.Success {
		status = "BUILD FAIL"
	} else if !a.GoVet.Clean {
		status = fmt.Sprintf("%d vet issues", len(a.GoVet.Issues))
	}
	return fmt.Sprintf("%s | %d files, %d lines | %d TODOs | %s",
		status, a.LineCount.GoFiles, a.LineCount.GoLines, len(a.TodoItems), a.Timestamp.Format("15:04:05"))
}

// --- Internal analysis functions ---

func countLines(root string) LineCount {
	var lc LineCount
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip vendor
		if strings.Contains(path, "/vendor/") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := bytes.Count(data, []byte("\n"))

		if strings.HasSuffix(path, "_test.go") {
			lc.TestFiles++
			lc.TestLines += lines
		} else {
			lc.GoFiles++
			lc.GoLines += lines
		}
		lc.TotalFiles++
		lc.TotalLines += lines
		return nil
	})
	return lc
}

func runGoVet(root string) VetResult {
	start := time.Now()
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := VetResult{Duration: duration}
	if err == nil && len(output) == 0 {
		result.Clean = true
		return result
	}

	// Parse vet output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			result.Issues = append(result.Issues, line)
		}
	}
	return result
}

func runGoBuild(root string) BuildResult {
	start := time.Now()
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := BuildResult{Duration: duration}
	if err == nil {
		result.Success = true
		return result
	}

	// Parse build errors
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result.Errors = append(result.Errors, line)
		}
	}
	return result
}

func runGoTest(root string) TestResult {
	start := time.Now()
	cmd := exec.Command("go", "test", "-v", "-count=1", "./...")
	cmd.Dir = root
	output, _ := cmd.CombinedOutput()
	duration := time.Since(start)

	result := TestResult{Duration: duration, Output: string(output)}

	// Count pass/fail from output
	passRe := regexp.MustCompile(`--- PASS:`)
	failRe := regexp.MustCompile(`--- FAIL:`)
	skipRe := regexp.MustCompile(`--- SKIP:`)

	result.Passed = len(passRe.FindAllString(string(output), -1))
	result.Failed = len(failRe.FindAllString(string(output), -1))
	result.Skipped = len(skipRe.FindAllString(string(output), -1))

	return result
}

func findTodos(root string) []TodoItem {
	var items []TodoItem
	// Match TODO/FIXME/HACK/XXX in comments only (// or /* style)
	todoRe := regexp.MustCompile(`(?://|/\*)\s*(TODO|FIXME|HACK|XXX)[\s:]+(.+)`)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.Contains(path, "/vendor/") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		relPath, _ := filepath.Rel(root, path)

		for i, line := range lines {
			if match := todoRe.FindStringSubmatch(line); match != nil {
				content := strings.TrimSpace(match[2])
				// Skip if content is too short or looks like code
				if len(content) < 5 || strings.HasPrefix(content, "{") {
					continue
				}
				items = append(items, TodoItem{
					File:    relPath,
					Line:    i + 1,
					Type:    strings.ToUpper(match[1]),
					Content: content,
				})
			}
		}
		return nil
	})

	// Limit to first 20
	if len(items) > 20 {
		items = items[:20]
	}
	return items
}

func getRecentCommits(root string, n int) []string {
	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("-%d", n))
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return lines
}

func getChangedFiles(root string, commits int) []string {
	cmd := exec.Command("git", "diff", "--name-only", fmt.Sprintf("HEAD~%d", commits))
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Filter to Go files
	var goFiles []string
	for _, f := range lines {
		if strings.HasSuffix(f, ".go") {
			goFiles = append(goFiles, f)
		}
	}
	return goFiles
}

func countSymbols(root string) SymbolSummary {
	var ss SymbolSummary

	funcRe := regexp.MustCompile(`^func\s+([A-Z]\w*)`)
	methodRe := regexp.MustCompile(`^func\s+\([^)]+\)\s+([A-Z]\w*)`)
	typeRe := regexp.MustCompile(`^type\s+([A-Z]\w*)`)
	constRe := regexp.MustCompile(`^const\s+([A-Z]\w*)`)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.Contains(path, "/vendor/") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if methodRe.MatchString(line) {
				ss.Methods++
			} else if funcRe.MatchString(line) {
				ss.Functions++
			} else if typeRe.MatchString(line) {
				ss.Types++
			} else if constRe.MatchString(line) {
				ss.Constants++
			}
		}
		return nil
	})

	return ss
}
