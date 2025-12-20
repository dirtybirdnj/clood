package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// PRReviewResult holds the review analysis
type PRReviewResult struct {
	PRNumber     int            `json:"pr_number"`
	Title        string         `json:"title"`
	Author       string         `json:"author"`
	FilesChanged int            `json:"files_changed"`
	Additions    int            `json:"additions"`
	Deletions    int            `json:"deletions"`
	Issues       []ReviewIssue  `json:"issues"`
	Model        string         `json:"model"`
	Duration     float64        `json:"duration_seconds"`
}

type ReviewIssue struct {
	Severity    string `json:"severity"`    // critical, warning, note
	Category    string `json:"category"`    // security, style, logic, performance
	File        string `json:"file"`
	Line        int    `json:"line,omitempty"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
}

func ReviewPRCmd() *cobra.Command {
	var dryRun bool
	var securityOnly bool
	var model string
	var host string

	cmd := &cobra.Command{
		Use:   "review-pr [PR_NUMBER]",
		Short: "Automated PR review with local LLMs",
		Long: `Fetch a PR diff and analyze it with local models.

Reviews check for:
  - Security issues (injection, auth, secrets)
  - Code style (naming, formatting, idioms)
  - Logic problems (edge cases, error handling)
  - Performance concerns (N+1 queries, allocations)

Examples:
  clood review-pr 123               # Review PR #123
  clood review-pr 123 --security    # Security focus only
  clood review-pr 123 --dry-run     # Preview without posting
  clood review-pr 123 --model llama3.1:8b  # Use specific model`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			prNumber := args[0]
			result, err := reviewPR(prNumber, model, host, securityOnly, dryRun)
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			printReviewResult(result)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview review without posting")
	cmd.Flags().BoolVar(&securityOnly, "security", false, "Focus on security issues only")
	cmd.Flags().StringVarP(&model, "model", "m", "qwen2.5-coder:7b", "Model to use for review")
	cmd.Flags().StringVar(&host, "host", "http://localhost:11434", "Ollama host")

	return cmd
}

func reviewPR(prNumber, model, host string, securityOnly, dryRun bool) (*PRReviewResult, error) {
	startTime := time.Now()

	// Get PR info
	prInfo, err := getPRInfo(prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %w", err)
	}

	// Get PR diff
	diff, err := getPRDiff(prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR diff: %w", err)
	}

	// Truncate if too long
	if len(diff) > 15000 {
		diff = diff[:15000] + "\n... (truncated)"
	}

	// Build review prompt
	prompt := buildReviewPrompt(prInfo, diff, securityOnly)

	// Call model
	reviewText, err := callModel(host, model, prompt)
	if err != nil {
		return nil, fmt.Errorf("model review failed: %w", err)
	}

	// Parse issues from response
	issues := parseReviewIssues(reviewText)

	result := &PRReviewResult{
		PRNumber:     prInfo.Number,
		Title:        prInfo.Title,
		Author:       prInfo.Author,
		FilesChanged: prInfo.Files,
		Additions:    prInfo.Additions,
		Deletions:    prInfo.Deletions,
		Issues:       issues,
		Model:        model,
		Duration:     time.Since(startTime).Seconds(),
	}

	return result, nil
}

type prInfo struct {
	Number    int
	Title     string
	Author    string
	Files     int
	Additions int
	Deletions int
}

func getPRInfo(prNumber string) (*prInfo, error) {
	out, err := exec.Command("gh", "pr", "view", prNumber, "--json",
		"number,title,author,files,additions,deletions").Output()
	if err != nil {
		return nil, err
	}

	var data struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Author    struct {
			Login string `json:"login"`
		} `json:"author"`
		Files     []interface{} `json:"files"`
		Additions int           `json:"additions"`
		Deletions int           `json:"deletions"`
	}

	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}

	return &prInfo{
		Number:    data.Number,
		Title:     data.Title,
		Author:    data.Author.Login,
		Files:     len(data.Files),
		Additions: data.Additions,
		Deletions: data.Deletions,
	}, nil
}

func getPRDiff(prNumber string) (string, error) {
	out, err := exec.Command("gh", "pr", "diff", prNumber).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func buildReviewPrompt(info *prInfo, diff string, securityOnly bool) string {
	var focus string
	if securityOnly {
		focus = `Focus ONLY on security issues:
- SQL/Command injection
- Authentication/authorization flaws
- Hardcoded secrets or credentials
- XSS vulnerabilities
- Insecure data handling`
	} else {
		focus = `Review for:
1. SECURITY: Injection, auth issues, secrets, XSS
2. LOGIC: Edge cases, error handling, null checks
3. STYLE: Naming, formatting, code organization
4. PERFORMANCE: N+1 queries, unnecessary allocations`
	}

	return fmt.Sprintf(`You are a senior code reviewer. Review this pull request diff.

PR: %s (#%d) by %s
Files: %d | +%d -%d

%s

For each issue found, respond in this exact format:
[SEVERITY] CATEGORY: FILE:LINE - Description
Suggestion: How to fix

Where SEVERITY is: CRITICAL, WARNING, or NOTE
And CATEGORY is: SECURITY, LOGIC, STYLE, or PERFORMANCE

If no issues found, say "No issues found."

Here's the diff:
%s

Review:`, info.Title, info.Number, info.Author, info.Files, info.Additions, info.Deletions, focus, diff)
}

func callModel(host, model, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": 1000,
			"temperature": 0.3,
		},
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(host+"/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}

func parseReviewIssues(text string) []ReviewIssue {
	var issues []ReviewIssue

	// Pattern: [SEVERITY] CATEGORY: FILE:LINE - Description
	pattern := regexp.MustCompile(`\[(CRITICAL|WARNING|NOTE)\]\s*(SECURITY|LOGIC|STYLE|PERFORMANCE):\s*([^\s:]+)(?::(\d+))?\s*[-–]\s*(.+?)(?:\nSuggestion:\s*(.+?))?(?:\n\n|\n\[|$)`)

	matches := pattern.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		issue := ReviewIssue{
			Severity:    strings.ToLower(m[1]),
			Category:    strings.ToLower(m[2]),
			File:        m[3],
			Description: strings.TrimSpace(m[5]),
		}
		if m[4] != "" {
			fmt.Sscanf(m[4], "%d", &issue.Line)
		}
		if len(m) > 6 && m[6] != "" {
			issue.Suggestion = strings.TrimSpace(m[6])
		}
		issues = append(issues, issue)
	}

	// If pattern didn't match but there's content, try simpler extraction
	if len(issues) == 0 && !strings.Contains(strings.ToLower(text), "no issues found") {
		// Fall back to treating the whole response as a note
		if len(strings.TrimSpace(text)) > 10 {
			issues = append(issues, ReviewIssue{
				Severity:    "note",
				Category:    "general",
				File:        "general",
				Description: strings.TrimSpace(text),
			})
		}
	}

	return issues
}

func printReviewResult(result *PRReviewResult) {
	fmt.Println()
	fmt.Printf("🔍 PR #%d: %s\n", result.PRNumber, result.Title)
	fmt.Printf("   Author: %s | Files: %d | +%d -%d\n",
		result.Author, result.FilesChanged, result.Additions, result.Deletions)
	fmt.Println()
	fmt.Println(strings.Repeat("━", 50))
	fmt.Println()

	if len(result.Issues) == 0 {
		fmt.Println(tui.SuccessStyle.Render("✓ No issues found"))
	} else {
		// Group by category
		bySeverity := map[string][]ReviewIssue{
			"critical": {},
			"warning":  {},
			"note":     {},
		}
		for _, issue := range result.Issues {
			bySeverity[issue.Severity] = append(bySeverity[issue.Severity], issue)
		}

		// Print critical first
		if len(bySeverity["critical"]) > 0 {
			fmt.Println(tui.ErrorStyle.Render(fmt.Sprintf("🔴 CRITICAL (%d)", len(bySeverity["critical"]))))
			for _, issue := range bySeverity["critical"] {
				printIssue(issue)
			}
			fmt.Println()
		}

		// Then warnings
		if len(bySeverity["warning"]) > 0 {
			fmt.Println(tui.WarningStyle.Render(fmt.Sprintf("🟡 WARNING (%d)", len(bySeverity["warning"]))))
			for _, issue := range bySeverity["warning"] {
				printIssue(issue)
			}
			fmt.Println()
		}

		// Then notes
		if len(bySeverity["note"]) > 0 {
			fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("🟢 NOTE (%d)", len(bySeverity["note"]))))
			for _, issue := range bySeverity["note"] {
				printIssue(issue)
			}
			fmt.Println()
		}
	}

	fmt.Println(strings.Repeat("━", 50))

	// Summary
	critCount := 0
	warnCount := 0
	noteCount := 0
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "critical":
			critCount++
		case "warning":
			warnCount++
		case "note":
			noteCount++
		}
	}

	fmt.Printf("Summary: %d critical, %d warning, %d note\n", critCount, warnCount, noteCount)
	fmt.Printf("Reviewed by: %s (%.1fs)\n", result.Model, result.Duration)
	fmt.Println()
}

func printIssue(issue ReviewIssue) {
	location := issue.File
	if issue.Line > 0 {
		location = fmt.Sprintf("%s:%d", issue.File, issue.Line)
	}

	fmt.Printf("   %s\n", tui.AccentStyle.Render(location))
	fmt.Printf("   │ %s\n", issue.Description)
	if issue.Suggestion != "" {
		fmt.Printf("   │ Suggestion: %s\n", tui.MutedStyle.Render(issue.Suggestion))
	}
	fmt.Println()
}
