package commands

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// GitHubIssue represents an issue from gh CLI
type GitHubIssue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
	URL    string   `json:"url"`
}

// IssueCategory groups issues by status
type IssueCategory struct {
	ReadyToClose []GitHubIssue `json:"ready_to_close"`
	ReadyToBuild []GitHubIssue `json:"ready_to_build"`
	NotStarted   []GitHubIssue `json:"not_started"`
	Bugs         []GitHubIssue `json:"bugs"`
	InProgress   []GitHubIssue `json:"in_progress"`
}

// IssueSummary provides counts
type IssueSummary struct {
	Total       int `json:"total"`
	ReadyClose  int `json:"ready_to_close"`
	ReadyBuild  int `json:"ready_to_build"`
	NotStarted  int `json:"not_started"`
	Bugs        int `json:"bugs"`
	InProgress  int `json:"in_progress"`
}

// IssuesOutput is the full JSON output
type IssuesOutput struct {
	Categories IssueCategory `json:"categories"`
	Summary    IssueSummary  `json:"summary"`
}

func IssuesCmd() *cobra.Command {
	var jsonOutput bool
	var readyOnly bool
	var buildableOnly bool
	var bugsOnly bool
	var labelFilter string

	cmd := &cobra.Command{
		Use:   "issues",
		Short: "Show project issue status dashboard",
		Long: `Display GitHub issues organized by status.

Categories:
  ✅ Ready to Close    - labeled 'agent-review-complete'
  🏗️  Ready to Build    - design resolved, ready for implementation
  🔄 In Progress       - labeled 'agent-in-progress'
  🐛 Bugs              - labeled 'bug'
  🔲 Not Started       - everything else

Examples:
  clood issues              # Full dashboard
  clood issues --json       # JSON for agents
  clood issues --ready      # Only ready to close
  clood issues --buildable  # Only ready to build
  clood issues --bugs       # Only bugs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if gh CLI is available
			if _, err := exec.LookPath("gh"); err != nil {
				return fmt.Errorf("gh CLI not found - install from https://cli.github.com")
			}

			// Fetch issues from GitHub
			issues, err := fetchGitHubIssues()
			if err != nil {
				return fmt.Errorf("fetching issues: %w", err)
			}

			// Filter by label if specified
			if labelFilter != "" {
				issues = filterByLabel(issues, labelFilter)
			}

			// Categorize issues
			categories := categorizeIssues(issues)

			// Apply category filters
			if readyOnly {
				categories = IssueCategory{ReadyToClose: categories.ReadyToClose}
			} else if buildableOnly {
				categories = IssueCategory{ReadyToBuild: categories.ReadyToBuild}
			} else if bugsOnly {
				categories = IssueCategory{Bugs: categories.Bugs}
			}

			// Output
			if jsonOutput {
				return outputJSON(categories)
			}
			return outputTUI(categories)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON for agents")
	cmd.Flags().BoolVar(&readyOnly, "ready", false, "Show only ready to close")
	cmd.Flags().BoolVar(&buildableOnly, "buildable", false, "Show only ready to build")
	cmd.Flags().BoolVar(&bugsOnly, "bugs", false, "Show only bugs")
	cmd.Flags().StringVarP(&labelFilter, "label", "l", "", "Filter by label")

	return cmd
}

func fetchGitHubIssues() ([]GitHubIssue, error) {
	// Use gh CLI to fetch issues
	cmd := exec.Command("gh", "issue", "list", "--state", "open", "--limit", "100", "--json", "number,title,state,labels,url")
	output, err := cmd.Output()
	if err != nil {
		// Try to get more info about the error
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	// Parse the JSON output
	var rawIssues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		URL string `json:"url"`
	}

	if err := json.Unmarshal(output, &rawIssues); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	// Convert to our format
	issues := make([]GitHubIssue, len(rawIssues))
	for i, raw := range rawIssues {
		labels := make([]string, len(raw.Labels))
		for j, l := range raw.Labels {
			labels[j] = l.Name
		}
		issues[i] = GitHubIssue{
			Number: raw.Number,
			Title:  raw.Title,
			State:  raw.State,
			Labels: labels,
			URL:    raw.URL,
		}
	}

	return issues, nil
}

func filterByLabel(issues []GitHubIssue, label string) []GitHubIssue {
	var filtered []GitHubIssue
	for _, issue := range issues {
		for _, l := range issue.Labels {
			if strings.EqualFold(l, label) {
				filtered = append(filtered, issue)
				break
			}
		}
	}
	return filtered
}

func categorizeIssues(issues []GitHubIssue) IssueCategory {
	var categories IssueCategory

	for _, issue := range issues {
		categorized := false

		for _, label := range issue.Labels {
			switch strings.ToLower(label) {
			case "agent-review-complete":
				categories.ReadyToClose = append(categories.ReadyToClose, issue)
				categorized = true
			case "agent-in-progress":
				categories.InProgress = append(categories.InProgress, issue)
				categorized = true
			case "bug":
				categories.Bugs = append(categories.Bugs, issue)
				categorized = true
			case "design-complete", "ready-to-build":
				categories.ReadyToBuild = append(categories.ReadyToBuild, issue)
				categorized = true
			}
			if categorized {
				break
			}
		}

		if !categorized {
			categories.NotStarted = append(categories.NotStarted, issue)
		}
	}

	return categories
}

func outputJSON(categories IssueCategory) error {
	output := IssuesOutput{
		Categories: categories,
		Summary: IssueSummary{
			Total:      len(categories.ReadyToClose) + len(categories.ReadyToBuild) + len(categories.NotStarted) + len(categories.Bugs) + len(categories.InProgress),
			ReadyClose: len(categories.ReadyToClose),
			ReadyBuild: len(categories.ReadyToBuild),
			NotStarted: len(categories.NotStarted),
			Bugs:       len(categories.Bugs),
			InProgress: len(categories.InProgress),
		},
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func outputTUI(categories IssueCategory) error {
	fmt.Println(tui.RenderHeader("Issue Status"))
	fmt.Println()

	// Ready to Close
	if len(categories.ReadyToClose) > 0 {
		fmt.Println(tui.SuccessStyle.Render("  ✅ Ready to Close (implemented)"))
		for _, issue := range categories.ReadyToClose {
			fmt.Printf("     #%-3d %s\n", issue.Number, issue.Title)
		}
		fmt.Println()
	}

	// Ready to Build
	if len(categories.ReadyToBuild) > 0 {
		fmt.Println(tui.TierDeepStyle.Render("  🏗️  Ready to Build (design complete)"))
		for _, issue := range categories.ReadyToBuild {
			fmt.Printf("     #%-3d %s\n", issue.Number, issue.Title)
		}
		fmt.Println()
	}

	// In Progress
	if len(categories.InProgress) > 0 {
		fmt.Println(tui.TierFastStyle.Render("  🔄 In Progress"))
		for _, issue := range categories.InProgress {
			fmt.Printf("     #%-3d %s\n", issue.Number, issue.Title)
		}
		fmt.Println()
	}

	// Bugs
	if len(categories.Bugs) > 0 {
		fmt.Println(tui.ErrorStyle.Render("  🐛 Bugs"))
		for _, issue := range categories.Bugs {
			fmt.Printf("     #%-3d %s\n", issue.Number, issue.Title)
		}
		fmt.Println()
	}

	// Not Started
	if len(categories.NotStarted) > 0 {
		fmt.Println(tui.MutedStyle.Render("  🔲 Not Started"))
		for _, issue := range categories.NotStarted {
			fmt.Printf("     #%-3d %s\n", issue.Number, issue.Title)
		}
		fmt.Println()
	}

	// Summary
	total := len(categories.ReadyToClose) + len(categories.ReadyToBuild) + len(categories.NotStarted) + len(categories.Bugs) + len(categories.InProgress)
	fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("  Total: %d open issues", total)))

	return nil
}
