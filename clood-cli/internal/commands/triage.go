package commands

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// TriageResult holds catfight analysis for one issue
type TriageResult struct {
	Issue       GitHubIssue      `json:"issue"`
	Analysis    []CatfightResult `json:"analysis"`
	Winner      *CatfightResult  `json:"winner,omitempty"`
	Duration    time.Duration    `json:"duration_ns"`
	DurationSec float64          `json:"duration_sec"`
	Skipped     bool             `json:"skipped"`
	SkipReason  string           `json:"skip_reason,omitempty"`
	Error       string           `json:"error,omitempty"`
}

// TriageOutput is the full JSON output
type TriageOutput struct {
	Timestamp   string         `json:"timestamp"`
	Repo        string         `json:"repo"`
	Results     []TriageResult `json:"results"`
	Summary     TriageSummary  `json:"summary"`
	TotalTime   float64        `json:"total_time_sec"`
}

// TriageSummary provides aggregate stats
type TriageSummary struct {
	TotalIssues   int `json:"total_issues"`
	Processed     int `json:"processed"`
	Skipped       int `json:"skipped"`
	Failed        int `json:"failed"`
	CommentsAdded int `json:"comments_added"`
}

func TriageCmd() *cobra.Command {
	var repo string
	var issueNums string
	var models string
	var dryRun bool
	var fast bool
	var live bool
	var timeout int
	var delay int
	var resume bool

	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Run catfight analysis on GitHub issues",
		Long: `Process GitHub issues with multi-model catfight analysis.

For each issue, queries multiple models for analysis and posts
results as GitHub comments.

Features:
  - Health check before each issue (de-icing)
  - Rate limiting on GitHub comments (configurable delay)
  - Resume capability (skip issues with existing catfight comments)
  - Timeout per issue (default 15 min)

Examples:
  # Process all open issues
  clood triage --repo dirtybirdnj/clood

  # Process specific issues
  clood triage --issues 93,94,95

  # Dry run (no GitHub comments)
  clood triage --dry-run

  # Fast mode (only small models <8b)
  clood triage --fast

  # Watch progress live
  clood triage --live

  # Custom models
  clood triage --models "qwen2.5-coder:7b,llama3.1:8b"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine repo
			if repo == "" {
				// Try to detect from git remote
				detected, err := detectGitRepo()
				if err != nil {
					return fmt.Errorf("no repo specified and couldn't detect: %w", err)
				}
				repo = detected
			}

			if !output.JSONMode && !live {
				fmt.Println(tui.RenderHeader("TRIAGE"))
				fmt.Println()
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Repo:"), repo)
			}

			// Check Ollama health
			if !output.JSONMode && !live {
				fmt.Printf("  %s Checking Ollama health...\n", tui.AccentStyle.Render(""))
			}

			mgr := hosts.NewManager()
			mgr.AddHosts(hosts.DefaultHosts())
			statuses := mgr.CheckAllHosts()

			var onlineHost *hosts.HostStatus
			for _, s := range statuses {
				if s.Online && len(s.Models) > 0 {
					onlineHost = s
					break
				}
			}

			if onlineHost == nil {
				return fmt.Errorf("no online Ollama hosts found")
			}

			if !output.JSONMode && !live {
				fmt.Printf("  %s Using %s (%d models)\n\n",
					tui.SuccessStyle.Render(""),
					onlineHost.Host.Name,
					len(onlineHost.Models))
			}

			// Determine which models to use
			var modelsToUse []string
			if models != "" {
				modelsToUse = strings.Split(models, ",")
				for i := range modelsToUse {
					modelsToUse[i] = strings.TrimSpace(modelsToUse[i])
				}
			} else if fast {
				// Fast mode: only small models
				for _, m := range onlineHost.Models {
					if isSmallModel(m.Name) {
						modelsToUse = append(modelsToUse, m.Name)
					}
				}
				if len(modelsToUse) == 0 {
					modelsToUse = []string{"qwen2.5-coder:3b"}
				}
			} else {
				// Default: first 3 models
				for i, m := range onlineHost.Models {
					if i >= 3 {
						break
					}
					modelsToUse = append(modelsToUse, m.Name)
				}
			}

			// Fetch issues
			var issues []GitHubIssue
			var err error

			if issueNums != "" {
				issues, err = fetchSpecificIssues(repo, issueNums)
			} else {
				issues, err = fetchRepoIssues(repo)
			}
			if err != nil {
				return fmt.Errorf("fetching issues: %w", err)
			}

			if len(issues) == 0 {
				if !output.JSONMode {
					fmt.Println(tui.MutedStyle.Render("  No issues to process"))
				}
				return nil
			}

			if !output.JSONMode && !live {
				fmt.Printf("  %s Found %d issues\n", tui.AccentStyle.Render(""), len(issues))
				fmt.Printf("  %s Models: %s\n", tui.MutedStyle.Render("Using"), strings.Join(modelsToUse, ", "))
				if dryRun {
					fmt.Printf("  %s\n", tui.ErrorStyle.Render("DRY RUN - no comments will be posted"))
				}
				fmt.Println()
			}

			// Process each issue
			startTime := time.Now()
			var results []TriageResult
			summary := TriageSummary{TotalIssues: len(issues)}

			client := ollama.NewClient(onlineHost.Host.URL, time.Duration(timeout)*time.Minute)

			for i, issue := range issues {
				if !output.JSONMode && !live {
					fmt.Printf("  [%d/%d] #%d %s\n",
						i+1, len(issues),
						issue.Number,
						truncateTitle(issue.Title, 50))
				}

				result := TriageResult{Issue: issue}
				issueStart := time.Now()

				// Check for existing catfight comment if resume mode
				if resume {
					hasComment, err := hasCatfightComment(repo, issue.Number)
					if err == nil && hasComment {
						result.Skipped = true
						result.SkipReason = "existing catfight comment"
						summary.Skipped++
						results = append(results, result)
						if !output.JSONMode && !live {
							fmt.Printf("       %s (has existing analysis)\n",
								tui.MutedStyle.Render("SKIPPED"))
						}
						continue
					}
				}

				// Build the prompt for this issue
				issueBody, err := fetchIssueBody(repo, issue.Number)
				if err != nil {
					result.Error = err.Error()
					summary.Failed++
					results = append(results, result)
					continue
				}

				prompt := buildTriagePrompt(issue.Title, issueBody)

				// Run each model
				for _, modelName := range modelsToUse {
					modelStart := time.Now()
					resp, err := client.Generate(modelName, prompt)
					modelDuration := time.Since(modelStart)

					catResult := CatfightResult{
						Cat:         Cat{Name: modelToName(modelName), Model: modelName},
						Host:        onlineHost.Host.Name,
						Duration:    modelDuration,
						DurationSec: modelDuration.Seconds(),
					}

					if err != nil {
						catResult.ErrorStr = err.Error()
					} else {
						catResult.Response = resp.Response
						catResult.Tokens = resp.EvalCount
						if resp.EvalDuration > 0 {
							catResult.TokSec = float64(resp.EvalCount) / (float64(resp.EvalDuration) / 1e9)
						}
					}

					result.Analysis = append(result.Analysis, catResult)

					if !output.JSONMode && !live {
						if err != nil {
							fmt.Printf("       %s %s: %s\n",
								tui.ErrorStyle.Render(""),
								modelName,
								tui.ErrorStyle.Render("FAILED"))
						} else {
							fmt.Printf("       %s %s: %.1fs, %d tok\n",
								tui.SuccessStyle.Render(""),
								modelName,
								modelDuration.Seconds(),
								resp.EvalCount)
						}
					}
				}

				// Find winner (fastest successful response)
				for i := range result.Analysis {
					if result.Analysis[i].ErrorStr == "" {
						if result.Winner == nil || result.Analysis[i].DurationSec < result.Winner.DurationSec {
							result.Winner = &result.Analysis[i]
						}
					}
				}

				result.Duration = time.Since(issueStart)
				result.DurationSec = result.Duration.Seconds()

				// Post comment unless dry run
				if !dryRun && result.Winner != nil {
					comment := formatTriageComment(result)
					if err := postGitHubComment(repo, issue.Number, comment); err != nil {
						if !output.JSONMode {
							fmt.Printf("       %s Failed to post comment: %s\n",
								tui.ErrorStyle.Render(""),
								err.Error())
						}
					} else {
						summary.CommentsAdded++
						if !output.JSONMode && !live {
							fmt.Printf("       %s Comment posted\n",
								tui.SuccessStyle.Render(""))
						}
					}
				}

				summary.Processed++
				results = append(results, result)

				// Rate limit delay between issues
				if i < len(issues)-1 && delay > 0 {
					time.Sleep(time.Duration(delay) * time.Second)
				}
			}

			totalDuration := time.Since(startTime)

			// Output results
			triageOutput := TriageOutput{
				Timestamp: time.Now().Format(time.RFC3339),
				Repo:      repo,
				Results:   results,
				Summary:   summary,
				TotalTime: totalDuration.Seconds(),
			}

			if output.JSONMode {
				data, _ := json.MarshalIndent(triageOutput, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			// Summary
			fmt.Println()
			fmt.Println(tui.RenderHeader("SUMMARY"))
			fmt.Println()
			fmt.Printf("  %s %d issues processed\n",
				tui.SuccessStyle.Render(""),
				summary.Processed)
			if summary.Skipped > 0 {
				fmt.Printf("  %s %d issues skipped (existing analysis)\n",
					tui.MutedStyle.Render(""),
					summary.Skipped)
			}
			if summary.Failed > 0 {
				fmt.Printf("  %s %d issues failed\n",
					tui.ErrorStyle.Render(""),
					summary.Failed)
			}
			if !dryRun {
				fmt.Printf("  %s %d comments added\n",
					tui.AccentStyle.Render(""),
					summary.CommentsAdded)
			}
			fmt.Printf("  %s %.1fs total\n",
				tui.MutedStyle.Render("Time:"),
				totalDuration.Seconds())

			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo (owner/name)")
	cmd.Flags().StringVar(&issueNums, "issues", "", "Specific issue numbers (comma-separated)")
	cmd.Flags().StringVar(&models, "models", "", "Models to use (comma-separated)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Don't post comments to GitHub")
	cmd.Flags().BoolVar(&fast, "fast", false, "Use only small models (<8b)")
	cmd.Flags().BoolVar(&live, "live", false, "Live progress mode")
	cmd.Flags().IntVar(&timeout, "timeout", 15, "Timeout per issue in minutes")
	cmd.Flags().IntVar(&delay, "delay", 5, "Delay between issues in seconds")
	cmd.Flags().BoolVar(&resume, "resume", true, "Skip issues with existing catfight comments")

	return cmd
}

func detectGitRepo() (string, error) {
	cmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func fetchRepoIssues(repo string) ([]GitHubIssue, error) {
	cmd := exec.Command("gh", "issue", "list",
		"--repo", repo,
		"--state", "open",
		"--limit", "100",
		"--json", "number,title,state,labels,url")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var rawIssues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		URL string `json:"url"`
	}

	if err := json.Unmarshal(out, &rawIssues); err != nil {
		return nil, err
	}

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

func fetchSpecificIssues(repo, nums string) ([]GitHubIssue, error) {
	var issues []GitHubIssue
	for _, numStr := range strings.Split(nums, ",") {
		numStr = strings.TrimSpace(numStr)
		cmd := exec.Command("gh", "issue", "view", numStr,
			"--repo", repo,
			"--json", "number,title,state,labels,url")
		out, err := cmd.Output()
		if err != nil {
			continue
		}

		var raw struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			State  string `json:"state"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
			URL string `json:"url"`
		}

		if err := json.Unmarshal(out, &raw); err != nil {
			continue
		}

		labels := make([]string, len(raw.Labels))
		for j, l := range raw.Labels {
			labels[j] = l.Name
		}

		issues = append(issues, GitHubIssue{
			Number: raw.Number,
			Title:  raw.Title,
			State:  raw.State,
			Labels: labels,
			URL:    raw.URL,
		})
	}

	return issues, nil
}

func fetchIssueBody(repo string, number int) (string, error) {
	cmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", number),
		"--repo", repo,
		"--json", "body",
		"-q", ".body")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func hasCatfightComment(repo string, number int) (bool, error) {
	cmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", number),
		"--repo", repo,
		"--json", "comments",
		"-q", ".comments[].body")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), "CATFIGHT TRIAGE"), nil
}

func buildTriagePrompt(title, body string) string {
	return fmt.Sprintf(`You are a senior software engineer reviewing a GitHub issue.

ISSUE TITLE: %s

ISSUE BODY:
%s

Provide a brief analysis:
1. Is this issue well-defined? (yes/no with reason)
2. Estimated complexity (low/medium/high)
3. Key implementation considerations (2-3 bullet points)
4. Suggested approach (1-2 sentences)

Keep response under 200 words.`, title, body)
}

func formatTriageComment(result TriageResult) string {
	var sb strings.Builder
	sb.WriteString("## CATFIGHT TRIAGE\n\n")
	sb.WriteString(fmt.Sprintf("*Analyzed by %d models*\n\n", len(result.Analysis)))

	for _, analysis := range result.Analysis {
		if analysis.ErrorStr != "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s (%s)\n", analysis.Cat.Name, analysis.Cat.Model))
		sb.WriteString(fmt.Sprintf("*%.1fs, %d tokens, %.1f tok/s*\n\n",
			analysis.DurationSec, analysis.Tokens, analysis.TokSec))
		sb.WriteString(analysis.Response)
		sb.WriteString("\n\n---\n\n")
	}

	if result.Winner != nil {
		sb.WriteString(fmt.Sprintf("**Fastest:** %s (%.1fs)\n",
			result.Winner.Cat.Model, result.Winner.DurationSec))
	}

	sb.WriteString("\n*Generated by clood triage*")
	return sb.String()
}

func postGitHubComment(repo string, number int, comment string) error {
	cmd := exec.Command("gh", "issue", "comment", fmt.Sprintf("%d", number),
		"--repo", repo,
		"--body", comment)
	return cmd.Run()
}

func isSmallModel(name string) bool {
	smallSuffixes := []string{":1b", ":1.5b", ":3b", ":1.8b", ":2b"}
	name = strings.ToLower(name)
	for _, suffix := range smallSuffixes {
		if strings.Contains(name, suffix) {
			return true
		}
	}
	return false
}

func truncateTitle(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
