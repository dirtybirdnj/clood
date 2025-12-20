package commands

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// PreflightResult captures dependency check results
type PreflightResult struct {
	FilesToModify   []string          `json:"files_to_modify"`
	ConflictingPRs  []ConflictingPR   `json:"conflicting_prs,omitempty"`
	HotFiles        []string          `json:"hot_files,omitempty"`
	Standalone      bool              `json:"standalone"`
	Recommendation  string            `json:"recommendation"`
	SuggestedLabels []string          `json:"suggested_labels,omitempty"`
}

// ConflictingPR represents a PR that touches the same files
type ConflictingPR struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Files  []string `json:"files"`
}

// Hot files that commonly cause conflicts
var hotFiles = []string{
	"cmd/clood/main.go",
	"internal/tui/styles.go",
	"internal/config/config.go",
	"go.mod",
	"go.sum",
}

func AgentPreflightCmd() *cobra.Command {
	var files []string
	var issueNum int
	var addComment bool

	cmd := &cobra.Command{
		Use:   "agent-preflight",
		Short: "Check for conflicts before agent work begins",
		Long: `Run dependency checks before an agent starts modifying files.

This prevents merge conflicts by detecting:
- Open PRs that touch the same files
- Hot files that commonly cause conflicts
- Suggested merge ordering

Examples:
  clood agent-preflight -f cmd/clood/main.go -f internal/commands/new.go
  clood agent-preflight -f internal/router/ --issue 123 --comment
  clood agent-preflight -f *.go`,
		Run: func(cmd *cobra.Command, args []string) {
			result := runAgentPreflight(files)

			if output.IsJSON() {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			printPreflightResult(result)

			// Add comment to issue if requested
			if addComment && issueNum > 0 {
				addPreflightComment(issueNum, result)
			}
		},
	}

	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "Files that will be modified")
	cmd.Flags().IntVar(&issueNum, "issue", 0, "Issue number to add preflight comment")
	cmd.Flags().BoolVar(&addComment, "comment", false, "Add preflight results as issue comment")

	return cmd
}

func runAgentPreflight(files []string) *PreflightResult {
	result := &PreflightResult{
		FilesToModify:   files,
		Standalone:      true,
		SuggestedLabels: []string{},
	}

	// Expand globs
	var expandedFiles []string
	for _, f := range files {
		if strings.Contains(f, "*") {
			matches, err := filepath.Glob(f)
			if err == nil {
				expandedFiles = append(expandedFiles, matches...)
			}
		} else {
			expandedFiles = append(expandedFiles, f)
		}
	}
	result.FilesToModify = expandedFiles

	// Check for hot files
	for _, f := range expandedFiles {
		for _, hot := range hotFiles {
			if strings.HasSuffix(f, hot) || f == hot {
				result.HotFiles = append(result.HotFiles, f)
				result.SuggestedLabels = append(result.SuggestedLabels, "touches:"+filepath.Base(hot))
			}
		}
	}

	// Check open PRs for conflicts
	result.ConflictingPRs = findConflictingPRs(expandedFiles)

	// Determine if standalone
	if len(result.ConflictingPRs) > 0 || len(result.HotFiles) > 0 {
		result.Standalone = false
	}

	// Generate recommendation
	if result.Standalone {
		result.Recommendation = "No conflicts detected. Safe to proceed."
		result.SuggestedLabels = append(result.SuggestedLabels, "standalone")
	} else if len(result.ConflictingPRs) > 0 {
		prs := make([]string, len(result.ConflictingPRs))
		for i, pr := range result.ConflictingPRs {
			prs[i] = fmt.Sprintf("#%d", pr.Number)
		}
		result.Recommendation = fmt.Sprintf("Conflicts with %s. Coordinate merge order.", strings.Join(prs, ", "))
		result.SuggestedLabels = append(result.SuggestedLabels, "merge-order:pending")
	} else {
		result.Recommendation = "Hot files detected. Extra care needed with merges."
	}

	return result
}

func findConflictingPRs(files []string) []ConflictingPR {
	var conflicts []ConflictingPR

	// Try to get open PRs with gh
	out, err := exec.Command("gh", "pr", "list", "--state", "open", "--json", "number,title,files", "--limit", "50").Output()
	if err != nil {
		return conflicts
	}

	var prs []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Files  []struct {
			Path string `json:"path"`
		} `json:"files"`
	}

	if err := json.Unmarshal(out, &prs); err != nil {
		return conflicts
	}

	// Check each PR for file overlaps
	for _, pr := range prs {
		var overlapping []string
		for _, prFile := range pr.Files {
			for _, targetFile := range files {
				if prFile.Path == targetFile || strings.HasPrefix(prFile.Path, targetFile) ||
					strings.HasPrefix(targetFile, prFile.Path) {
					overlapping = append(overlapping, prFile.Path)
				}
			}
		}
		if len(overlapping) > 0 {
			conflicts = append(conflicts, ConflictingPR{
				Number: pr.Number,
				Title:  pr.Title,
				Files:  overlapping,
			})
		}
	}

	return conflicts
}

func printPreflightResult(result *PreflightResult) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("Agent Preflight Check"))
	fmt.Println()

	// Files to modify
	fmt.Println(tui.MutedStyle.Render("Files to modify:"))
	for _, f := range result.FilesToModify {
		isHot := false
		for _, h := range result.HotFiles {
			if f == h {
				isHot = true
				break
			}
		}
		if isHot {
			fmt.Printf("  ⚠️  %s %s\n", f, tui.WarningStyle.Render("(hot file)"))
		} else {
			fmt.Printf("  •  %s\n", f)
		}
	}
	fmt.Println()

	// Conflicting PRs
	if len(result.ConflictingPRs) > 0 {
		fmt.Println(tui.WarningStyle.Render("Conflicting PRs:"))
		for _, pr := range result.ConflictingPRs {
			fmt.Printf("  #%d: %s\n", pr.Number, pr.Title)
			for _, f := range pr.Files {
				fmt.Printf("       └─ %s\n", f)
			}
		}
		fmt.Println()
	}

	// Status
	if result.Standalone {
		fmt.Println(tui.SuccessStyle.Render("✓ Standalone - safe to proceed"))
	} else {
		fmt.Println(tui.WarningStyle.Render("⚠ Coordination needed"))
	}

	// Recommendation
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("Recommendation: ") + result.Recommendation)

	// Suggested labels
	if len(result.SuggestedLabels) > 0 {
		fmt.Printf("\n%s %s\n", tui.MutedStyle.Render("Suggested labels:"), strings.Join(result.SuggestedLabels, ", "))
	}
}

func addPreflightComment(issueNum int, result *PreflightResult) {
	var sb strings.Builder

	sb.WriteString("## Preflight Dependency Check\n\n")

	sb.WriteString("**Files to modify:**\n")
	for _, f := range result.FilesToModify {
		sb.WriteString(fmt.Sprintf("- `%s`", f))
		for _, h := range result.HotFiles {
			if f == h {
				sb.WriteString(" ⚠️ (hot file)")
				break
			}
		}
		sb.WriteString("\n")
	}

	if len(result.ConflictingPRs) > 0 {
		sb.WriteString("\n**Conflicting PRs:**\n")
		for _, pr := range result.ConflictingPRs {
			sb.WriteString(fmt.Sprintf("- PR #%d: %s\n", pr.Number, pr.Title))
		}
	} else {
		sb.WriteString("\n**Conflicting PRs:** None\n")
	}

	sb.WriteString(fmt.Sprintf("\n**Status:** %s\n", result.Recommendation))

	if len(result.SuggestedLabels) > 0 {
		sb.WriteString(fmt.Sprintf("\n**Suggested labels:** %s\n", strings.Join(result.SuggestedLabels, ", ")))
	}

	sb.WriteString("\n---\n*Proceeding with implementation.*")

	// Add comment via gh
	cmd := exec.Command("gh", "issue", "comment", fmt.Sprintf("%d", issueNum), "--body", sb.String())
	if err := cmd.Run(); err != nil {
		fmt.Println(tui.ErrorStyle.Render("Failed to add comment: " + err.Error()))
	} else {
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Preflight comment added to issue #%d", issueNum)))
	}
}
