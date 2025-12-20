package output

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/tui"
)

// Severity levels for issues
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
	SeverityInfo   Severity = "info"
)

// Issue represents a single finding from an agent
type Issue struct {
	Severity    Severity `json:"severity"`
	Type        string   `json:"type,omitempty"`
	Line        int      `json:"line,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
	CodeSnippet string   `json:"code_snippet,omitempty"`
}

// ActionItem is a concrete next step
type ActionItem struct {
	Priority    int    `json:"priority"`
	Description string `json:"description"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
}

// AgentResult is the structured output from any agent task
type AgentResult struct {
	// Metadata
	TaskType   string        `json:"task_type"`
	File       string        `json:"file,omitempty"`
	Files      []string      `json:"files,omitempty"`
	Agent      string        `json:"agent"`
	Model      string        `json:"model"`
	Host       string        `json:"host"`
	DurationMs int64         `json:"duration_ms"`
	Tokens     int           `json:"tokens,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`

	// Summary
	Summary     string `json:"summary"`
	IssueCount  int    `json:"issue_count,omitempty"`
	HighCount   int    `json:"high_count,omitempty"`
	MediumCount int    `json:"medium_count,omitempty"`
	LowCount    int    `json:"low_count,omitempty"`

	// Content (task-specific)
	Issues      []Issue      `json:"issues,omitempty"`
	ActionItems []ActionItem `json:"action_items,omitempty"`
	RawResponse string       `json:"raw_response,omitempty"`

	// For code generation tasks
	GeneratedCode string `json:"generated_code,omitempty"`
	Explanation   string `json:"explanation,omitempty"`

	// For documentation tasks
	Sections []DocSection `json:"sections,omitempty"`

	// Error handling
	Error string `json:"error,omitempty"`
}

// DocSection for documentation output
type DocSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// OutputMap is the Strata-compatible target output structure
type OutputMap struct {
	Version   string                 `json:"version"`
	Generated time.Time              `json:"generated"`
	Project   string                 `json:"project,omitempty"`
	Results   []AgentResult          `json:"results"`
	Summary   map[string]interface{} `json:"summary,omitempty"`
}

// NewOutputMap creates a new output map
func NewOutputMap(project string) *OutputMap {
	return &OutputMap{
		Version:   "1.0",
		Generated: time.Now(),
		Project:   project,
		Results:   []AgentResult{},
		Summary:   make(map[string]interface{}),
	}
}

// AddResult appends an agent result to the output map
func (om *OutputMap) AddResult(result AgentResult) {
	om.Results = append(om.Results, result)
}

// ToJSON returns the output map as formatted JSON
func (om *OutputMap) ToJSON() (string, error) {
	data, err := json.MarshalIndent(om, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatAgentResult renders an agent result for terminal display
func FormatAgentResult(result *AgentResult) string {
	var sb strings.Builder

	// Header box
	header := result.TaskType
	if result.File != "" {
		header = fmt.Sprintf("%s: %s", result.TaskType, result.File)
	}
	sb.WriteString(renderBox(header, fmt.Sprintf("Agent: %s | Model: %s | Time: %.1fs",
		result.Agent, result.Model, float64(result.DurationMs)/1000)))
	sb.WriteString("\n")

	// Summary
	if result.Summary != "" {
		sb.WriteString(fmt.Sprintf("Summary: %s\n\n", result.Summary))
	}

	// Issues section
	if len(result.Issues) > 0 {
		sb.WriteString(renderSectionHeader("Issues"))
		sb.WriteString("\n")
		for _, issue := range result.Issues {
			sb.WriteString(formatIssue(issue))
			sb.WriteString("\n")
		}
	}

	// Action items section
	if len(result.ActionItems) > 0 {
		sb.WriteString(renderSectionHeader("Action Items"))
		sb.WriteString("\n")
		for _, item := range result.ActionItems {
			checkbox := "[ ]"
			sb.WriteString(fmt.Sprintf("%s %s\n", checkbox, item.Description))
		}
		sb.WriteString("\n")
	}

	// Generated code section
	if result.GeneratedCode != "" {
		sb.WriteString(renderSectionHeader("Generated Code"))
		sb.WriteString("\n")
		sb.WriteString("```\n")
		sb.WriteString(result.GeneratedCode)
		sb.WriteString("\n```\n\n")
		if result.Explanation != "" {
			sb.WriteString(result.Explanation)
			sb.WriteString("\n\n")
		}
	}

	// Documentation sections
	if len(result.Sections) > 0 {
		for _, section := range result.Sections {
			sb.WriteString(renderSectionHeader(section.Title))
			sb.WriteString("\n")
			sb.WriteString(section.Content)
			sb.WriteString("\n\n")
		}
	}

	// Raw response fallback
	if result.RawResponse != "" && len(result.Issues) == 0 && len(result.ActionItems) == 0 &&
		result.GeneratedCode == "" && len(result.Sections) == 0 {
		sb.WriteString(result.RawResponse)
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString(tui.MutedStyle.Render("───────────────────────────────────────"))
	sb.WriteString("\n")

	return sb.String()
}

func renderBox(title, subtitle string) string {
	width := 61
	topBorder := "┌" + strings.Repeat("─", width-2) + "┐"
	bottomBorder := "└" + strings.Repeat("─", width-2) + "┘"

	titlePadded := padCenter(title, width-4)
	subtitlePadded := padCenter(subtitle, width-4)

	return fmt.Sprintf("%s\n│  %s  │\n│  %s  │\n%s",
		topBorder, titlePadded, subtitlePadded, bottomBorder)
}

func padCenter(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	left := (width - len(s)) / 2
	right := width - len(s) - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func renderSectionHeader(title string) string {
	return fmt.Sprintf("%s\n%s", title, strings.Repeat("═", len(title)))
}

func formatIssue(issue Issue) string {
	var sb strings.Builder

	// Severity indicator
	var indicator string
	switch issue.Severity {
	case SeverityHigh:
		indicator = "🔴 HIGH"
	case SeverityMedium:
		indicator = "🟡 MEDIUM"
	case SeverityLow:
		indicator = "🟢 LOW"
	default:
		indicator = "ℹ️ INFO"
	}

	// Title with line number
	title := issue.Title
	if issue.Line > 0 {
		title = fmt.Sprintf("%s (Line %d)", title, issue.Line)
	}

	sb.WriteString(fmt.Sprintf("%s: %s\n", indicator, title))

	// Description
	if issue.Description != "" {
		sb.WriteString(fmt.Sprintf("   %s\n", issue.Description))
	}

	// Suggestion
	if issue.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\n   Suggestion: %s\n", issue.Suggestion))
	}

	// Code snippet
	if issue.CodeSnippet != "" {
		sb.WriteString("\n   ```\n")
		for _, line := range strings.Split(issue.CodeSnippet, "\n") {
			sb.WriteString(fmt.Sprintf("   %s\n", line))
		}
		sb.WriteString("   ```\n")
	}

	return sb.String()
}

// ParseAgentResponse attempts to parse a raw LLM response into structured output
// This uses heuristics to extract issues, action items, and code blocks
func ParseAgentResponse(raw string, taskType string) *AgentResult {
	result := &AgentResult{
		TaskType:    taskType,
		Timestamp:   time.Now(),
		RawResponse: raw,
	}

	// Extract issues based on severity markers
	result.Issues = extractIssues(raw)

	// Count by severity
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityHigh:
			result.HighCount++
		case SeverityMedium:
			result.MediumCount++
		case SeverityLow:
			result.LowCount++
		}
	}
	result.IssueCount = len(result.Issues)

	// Extract action items
	result.ActionItems = extractActionItems(raw)

	// Extract code blocks
	if code := extractCodeBlock(raw); code != "" {
		result.GeneratedCode = code
	}

	// Generate summary
	if result.IssueCount > 0 {
		result.Summary = fmt.Sprintf("%d issues found (%d high, %d medium, %d low)",
			result.IssueCount, result.HighCount, result.MediumCount, result.LowCount)
	} else if result.GeneratedCode != "" {
		result.Summary = "Code generated successfully"
	} else if len(result.ActionItems) > 0 {
		result.Summary = fmt.Sprintf("%d action items identified", len(result.ActionItems))
	}

	return result
}

// extractIssues finds issues in text based on common patterns
func extractIssues(text string) []Issue {
	var issues []Issue

	// Pattern: numbered items with severity indicators
	patterns := []struct {
		regex    *regexp.Regexp
		severity Severity
	}{
		{regexp.MustCompile(`(?i)(critical|high|severe|bug|error):\s*(.+)`), SeverityHigh},
		{regexp.MustCompile(`(?i)(warning|medium|caution):\s*(.+)`), SeverityMedium},
		{regexp.MustCompile(`(?i)(low|minor|suggestion|style):\s*(.+)`), SeverityLow},
		{regexp.MustCompile(`(?i)(note|info|consider):\s*(.+)`), SeverityInfo},
	}

	for _, p := range patterns {
		matches := p.regex.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			if len(m) >= 3 {
				issues = append(issues, Issue{
					Severity: p.severity,
					Title:    strings.TrimSpace(m[2]),
				})
			}
		}
	}

	// Also look for numbered issues: "1. ", "2. "
	numberedPattern := regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+)$`)
	numberedMatches := numberedPattern.FindAllStringSubmatch(text, -1)
	for _, m := range numberedMatches {
		if len(m) >= 2 {
			title := strings.TrimSpace(m[1])
			// Skip if it's likely an action item
			if strings.HasPrefix(strings.ToLower(title), "add") ||
				strings.HasPrefix(strings.ToLower(title), "fix") ||
				strings.HasPrefix(strings.ToLower(title), "update") {
				continue
			}
			// Check if we already have this issue
			found := false
			for _, existing := range issues {
				if existing.Title == title {
					found = true
					break
				}
			}
			if !found {
				issues = append(issues, Issue{
					Severity: SeverityMedium,
					Title:    title,
				})
			}
		}
	}

	return issues
}

// extractActionItems finds action items from text
func extractActionItems(text string) []ActionItem {
	var items []ActionItem
	priority := 1

	// Pattern: "[ ] item" or "- [ ] item"
	checkboxPattern := regexp.MustCompile(`(?m)^\s*[-*]?\s*\[[ x]?\]\s*(.+)$`)
	matches := checkboxPattern.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) >= 2 {
			items = append(items, ActionItem{
				Priority:    priority,
				Description: strings.TrimSpace(m[1]),
			})
			priority++
		}
	}

	// Pattern: "TODO: " or "Action: "
	todoPattern := regexp.MustCompile(`(?i)(?:todo|action|fix|implement):\s*(.+)`)
	todoMatches := todoPattern.FindAllStringSubmatch(text, -1)
	for _, m := range todoMatches {
		if len(m) >= 2 {
			items = append(items, ActionItem{
				Priority:    priority,
				Description: strings.TrimSpace(m[1]),
			})
			priority++
		}
	}

	return items
}

// extractCodeBlock extracts the first code block from markdown text
func extractCodeBlock(text string) string {
	// Match ```language\ncode\n```
	pattern := regexp.MustCompile("(?s)```(?:\\w+)?\\n(.+?)\\n```")
	match := pattern.FindStringSubmatch(text)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}
