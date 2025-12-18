package scope

import (
	"os"
	"regexp"
	"strings"
)

// ScopeStatus represents the clarity status of a task
type ScopeStatus string

const (
	StatusClear   ScopeStatus = "clear"
	StatusUnclear ScopeStatus = "unclear"
	StatusBlocked ScopeStatus = "blocked"
	StatusDeferred ScopeStatus = "deferred"
)

// ScopeResult is the output of scope analysis
type ScopeResult struct {
	Status        ScopeStatus `json:"status"`
	Confidence    float64     `json:"confidence"`
	Task          string      `json:"task"`
	Missing       []string    `json:"missing,omitempty"`
	Questions     []string    `json:"questions,omitempty"`
	SuggestedScope string     `json:"suggested_scope,omitempty"`
	Reason        string      `json:"reason,omitempty"`
	WaitingOn     string      `json:"waiting_on,omitempty"`
	Signals       []string    `json:"signals,omitempty"` // What triggered the analysis
}

// Analyzer performs static analysis on task descriptions
type Analyzer struct {
	projectContext string
}

// NewAnalyzer creates a new static analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		projectContext: loadProjectContext(),
	}
}

// Analyze performs static analysis on a task description
func (a *Analyzer) Analyze(task string) *ScopeResult {
	result := &ScopeResult{
		Task:       task,
		Status:     StatusClear,
		Confidence: 1.0,
		Missing:    []string{},
		Questions:  []string{},
		Signals:    []string{},
	}

	// Check for ambiguous verbs
	a.checkAmbiguousVerbs(task, result)

	// Check for missing specifics
	a.checkMissingSpecifics(task, result)

	// Check for unresolved decisions
	a.checkUnresolvedDecisions(task, result)

	// Check for file/function references
	a.checkReferences(task, result)

	// Calculate final status based on findings
	a.calculateStatus(result)

	return result
}

// checkAmbiguousVerbs detects vague action words without specifics
func (a *Analyzer) checkAmbiguousVerbs(task string, result *ScopeResult) {
	ambiguousPatterns := []struct {
		pattern  string
		question string
		missing  string
	}{
		{`(?i)^(implement|add|create)\s+\w+$`, "What specific functionality should be implemented?", "implementation details"},
		{`(?i)^(fix|update|change)\s+\w+$`, "What exactly needs to be fixed or changed?", "specific changes"},
		{`(?i)^(improve|optimize|enhance)\s+`, "What specific improvements are needed?", "improvement criteria"},
		{`(?i)^(refactor)\s+`, "What is the target architecture?", "refactoring goals"},
		{`(?i)\b(make it better|make it work)\b`, "What does 'better' or 'working' mean here?", "success criteria"},
		{`(?i)^(do|handle|manage)\s+\w+$`, "What specific action is required?", "action specifics"},
	}

	taskLower := strings.ToLower(task)
	for _, p := range ambiguousPatterns {
		matched, _ := regexp.MatchString(p.pattern, task)
		if matched {
			result.Questions = append(result.Questions, p.question)
			result.Missing = append(result.Missing, p.missing)
			result.Signals = append(result.Signals, "ambiguous_verb: "+taskLower)
			result.Confidence -= 0.2
		}
	}
}

// checkMissingSpecifics looks for tasks that lack concrete details
func (a *Analyzer) checkMissingSpecifics(task string, result *ScopeResult) {
	// Check if task mentions files/paths
	hasFilePath := regexp.MustCompile(`[./\\]\w+|\.\w{2,4}$|internal/|cmd/|pkg/`).MatchString(task)

	// Check if task mentions functions/methods
	hasFunction := regexp.MustCompile(`\w+\(\)|func\s+\w+|method\s+\w+`).MatchString(task)

	// Check if task mentions specific packages/modules
	hasPackage := regexp.MustCompile(`package\s+\w+|import\s+|module\s+`).MatchString(task)

	// Check word count - very short tasks are often unclear
	wordCount := len(strings.Fields(task))

	if wordCount < 4 && !hasFilePath && !hasFunction {
		result.Missing = append(result.Missing, "specific target (file, function, or component)")
		result.Questions = append(result.Questions, "Which file or component should this change?")
		result.Signals = append(result.Signals, "short_task_no_target")
		result.Confidence -= 0.15
	}

	// If task mentions "user auth" type things, check for method specifics
	authPatterns := regexp.MustCompile(`(?i)(auth|login|session|jwt|oauth|token)`)
	if authPatterns.MatchString(task) && !regexp.MustCompile(`(?i)(jwt|oauth|session|cookie|token|basic)`).MatchString(task) {
		result.Missing = append(result.Missing, "authentication method")
		result.Questions = append(result.Questions, "Which authentication method? (JWT, OAuth, session-based, etc.)")
		result.Signals = append(result.Signals, "auth_no_method")
		result.Confidence -= 0.15
	}

	// Database tasks without specifics
	dbPatterns := regexp.MustCompile(`(?i)(database|db|sql|query|table)`)
	if dbPatterns.MatchString(task) && !regexp.MustCompile(`(?i)(postgres|mysql|sqlite|mongo|redis|table\s+\w+)`).MatchString(task) {
		result.Missing = append(result.Missing, "database details")
		result.Questions = append(result.Questions, "Which database/table is involved?")
		result.Signals = append(result.Signals, "db_no_specifics")
		result.Confidence -= 0.1
	}

	_ = hasPackage // May use later
}

// checkUnresolvedDecisions looks for "or" patterns suggesting undecided choices
func (a *Analyzer) checkUnresolvedDecisions(task string, result *ScopeResult) {
	// "X or Y" patterns
	orPattern := regexp.MustCompile(`(?i)\b(\w+)\s+or\s+(\w+)\b`)
	matches := orPattern.FindAllStringSubmatch(task, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			// Skip common false positives
			skipPairs := map[string]bool{
				"true_false": true, "yes_no": true, "on_off": true,
				"success_failure": true, "read_write": true,
			}
			pair := strings.ToLower(match[1] + "_" + match[2])
			if !skipPairs[pair] {
				result.Questions = append(result.Questions,
					"Should it be "+match[1]+" or "+match[2]+"?")
				result.Missing = append(result.Missing, "decision: "+match[1]+" vs "+match[2])
				result.Signals = append(result.Signals, "unresolved_or: "+match[0])
				result.Confidence -= 0.2
			}
		}
	}

	// "should we/I" patterns
	shouldPattern := regexp.MustCompile(`(?i)\bshould\s+(we|i|it)\b`)
	if shouldPattern.MatchString(task) {
		result.Questions = append(result.Questions, "This sounds like a question, not a task. What's the decision?")
		result.Missing = append(result.Missing, "decision required")
		result.Signals = append(result.Signals, "question_not_task")
		result.Confidence -= 0.25
	}

	// "maybe" or "possibly" patterns
	uncertainPattern := regexp.MustCompile(`(?i)\b(maybe|possibly|perhaps|might|could)\b`)
	if uncertainPattern.MatchString(task) {
		result.Signals = append(result.Signals, "uncertainty_words")
		result.Confidence -= 0.1
	}
}

// checkReferences looks for clear file/function references (positive signal)
func (a *Analyzer) checkReferences(task string, result *ScopeResult) {
	// File path patterns
	filePattern := regexp.MustCompile(`\b[\w/]+\.(go|js|ts|py|rs|java|c|cpp|h|yaml|json|md)\b`)
	if filePattern.MatchString(task) {
		result.Signals = append(result.Signals, "has_file_reference")
		result.Confidence += 0.1
	}

	// Function/method patterns
	funcPattern := regexp.MustCompile(`\b\w+\(\)|\bfunc\s+\w+|\bfunction\s+\w+`)
	if funcPattern.MatchString(task) {
		result.Signals = append(result.Signals, "has_function_reference")
		result.Confidence += 0.1
	}

	// Line number references
	linePattern := regexp.MustCompile(`(?i)(line|L)\s*\d+|:\d+`)
	if linePattern.MatchString(task) {
		result.Signals = append(result.Signals, "has_line_reference")
		result.Confidence += 0.1
	}

	// Clamp confidence
	if result.Confidence > 1.0 {
		result.Confidence = 1.0
	}
	if result.Confidence < 0.0 {
		result.Confidence = 0.0
	}
}

// calculateStatus determines final status based on confidence
func (a *Analyzer) calculateStatus(result *ScopeResult) {
	// Deduplicate missing and questions
	result.Missing = dedupe(result.Missing)
	result.Questions = dedupe(result.Questions)

	// Determine status based on confidence
	switch {
	case result.Confidence >= 0.7:
		result.Status = StatusClear
	case result.Confidence >= 0.4:
		result.Status = StatusUnclear
		if result.SuggestedScope == "" && len(result.Missing) > 0 {
			result.SuggestedScope = "Clarify: " + strings.Join(result.Missing, ", ")
		}
	default:
		result.Status = StatusUnclear
	}
}

// loadProjectContext reads CLAUDE.md for pattern matching
func loadProjectContext() string {
	contextFiles := []string{"CLAUDE.md", "AGENTS.md"}
	for _, file := range contextFiles {
		if content, err := os.ReadFile(file); err == nil {
			return string(content)
		}
	}
	return ""
}

// dedupe removes duplicate strings
func dedupe(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// CreateDeferredResult creates a result for a deferred task
func CreateDeferredResult(task, reason string) *ScopeResult {
	return &ScopeResult{
		Task:       task,
		Status:     StatusDeferred,
		Confidence: 0.0,
		Reason:     reason,
	}
}

// CreateBlockedResult creates a result for a blocked task
func CreateBlockedResult(task, waitingOn string) *ScopeResult {
	return &ScopeResult{
		Task:       task,
		Status:     StatusBlocked,
		Confidence: 0.0,
		WaitingOn:  waitingOn,
	}
}
