package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Checkpoint represents a saved session state
type Checkpoint struct {
	Name      string    `yaml:"name" json:"name"`
	Created   time.Time `yaml:"created" json:"created"`
	Machine   string    `yaml:"machine" json:"machine"`
	Directory string    `yaml:"directory" json:"directory"`

	Context CheckpointContext `yaml:"context" json:"context"`
	Git     GitState          `yaml:"git" json:"git"`
	Todos   []string          `yaml:"todos,omitempty" json:"todos,omitempty"`
}

// CheckpointContext holds the session context
type CheckpointContext struct {
	Summary       string           `yaml:"summary" json:"summary"`
	RecentActions []string         `yaml:"recent_actions,omitempty" json:"recent_actions,omitempty"`
	NextSteps     []string         `yaml:"next_steps,omitempty" json:"next_steps,omitempty"`
	OpenIssues    []IssueRef       `yaml:"open_issues,omitempty" json:"open_issues,omitempty"`
	FocusGoal     string           `yaml:"focus_goal,omitempty" json:"focus_goal,omitempty"`
	Notes         string           `yaml:"notes,omitempty" json:"notes,omitempty"`
}

// IssueRef is a reference to a GitHub issue
type IssueRef struct {
	Number int    `yaml:"number" json:"number"`
	Title  string `yaml:"title" json:"title"`
	Labels string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// GitState captures git repository state
type GitState struct {
	Branch     string `yaml:"branch" json:"branch"`
	LastCommit string `yaml:"last_commit" json:"last_commit"`
	Status     string `yaml:"status,omitempty" json:"status,omitempty"`
}

func CheckpointCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Save and restore session checkpoints",
		Long: `Manage session checkpoints for portable Claude sessions.

Checkpoints capture:
- Session summary and notes
- Recent actions and next steps
- Git state (branch, last commit)
- Open issues being worked on

Examples:
  clood checkpoint save "planning-dec17"
  clood checkpoint load "planning-dec17"
  clood checkpoint list
  clood checkpoint show "planning-dec17"`,
	}

	cmd.AddCommand(checkpointSaveCmd())
	cmd.AddCommand(checkpointLoadCmd())
	cmd.AddCommand(checkpointListCmd())
	cmd.AddCommand(checkpointShowCmd())
	cmd.AddCommand(checkpointDeleteCmd())

	return cmd
}

func checkpointSaveCmd() *cobra.Command {
	var summary string
	var notes string
	var nextSteps []string
	var recentActions []string

	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save current session state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			checkpoint := &Checkpoint{
				Name:      name,
				Created:   time.Now(),
				Machine:   getHostname(),
				Directory: getCwd(),
				Context: CheckpointContext{
					Summary:       summary,
					Notes:         notes,
					NextSteps:     nextSteps,
					RecentActions: recentActions,
				},
				Git: getGitState(),
			}

			// Get focus state if available
			if focus := GetFocusSummary(); focus != "" {
				parts := strings.SplitN(focus, " | ", 2)
				if len(parts) > 0 && strings.HasPrefix(parts[0], "Goal: ") {
					checkpoint.Context.FocusGoal = strings.TrimPrefix(parts[0], "Goal: ")
				}
			}

			// Get recent closed issues
			checkpoint.Context.OpenIssues = getRecentIssues()

			if err := saveCheckpoint(checkpoint); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error saving checkpoint: " + err.Error()))
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(checkpoint, "", "  ")
				fmt.Println(string(data))
				return
			}

			fmt.Println()
			fmt.Println(tui.SuccessStyle.Render("✓ Saved checkpoint: " + name))
			fmt.Printf("  📁 Location: %s\n", getCheckpointPath(name))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("Contains:"))
			fmt.Printf("  • Machine: %s\n", checkpoint.Machine)
			fmt.Printf("  • Branch: %s\n", checkpoint.Git.Branch)
			if len(checkpoint.Context.OpenIssues) > 0 {
				fmt.Printf("  • Open issues: %d\n", len(checkpoint.Context.OpenIssues))
			}
			if len(checkpoint.Context.NextSteps) > 0 {
				fmt.Printf("  • Next steps: %d\n", len(checkpoint.Context.NextSteps))
			}
		},
	}

	cmd.Flags().StringVarP(&summary, "summary", "s", "", "Session summary")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "Additional notes")
	cmd.Flags().StringArrayVar(&nextSteps, "next", nil, "Next steps (can specify multiple)")
	cmd.Flags().StringArrayVar(&recentActions, "action", nil, "Recent actions (can specify multiple)")

	return cmd
}

func checkpointLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load <name>",
		Short: "Load and display a checkpoint",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			checkpoint, err := loadCheckpoint(name)
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading checkpoint: " + err.Error()))
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(checkpoint, "", "  ")
				fmt.Println(string(data))
				return
			}

			printCheckpointResume(checkpoint)
		},
	}

	return cmd
}

func checkpointListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all checkpoints",
		Run: func(cmd *cobra.Command, args []string) {
			checkpoints, err := listCheckpoints()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error listing checkpoints: " + err.Error()))
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(checkpoints, "", "  ")
				fmt.Println(string(data))
				return
			}

			if len(checkpoints) == 0 {
				fmt.Println(tui.MutedStyle.Render("No checkpoints saved."))
				fmt.Println(tui.MutedStyle.Render("Use 'clood checkpoint save <name>' to create one."))
				return
			}

			fmt.Println()
			fmt.Println(tui.RenderHeader("Checkpoints"))
			fmt.Println()

			for _, cp := range checkpoints {
				age := time.Since(cp.Created).Round(time.Hour)
				ageStr := formatCheckpointAge(age)
				fmt.Printf("  %s  %s  %s\n",
					tui.MutedStyle.Render(cp.Name),
					tui.MutedStyle.Render("│"),
					ageStr)
				if cp.Context.Summary != "" {
					fmt.Printf("           %s\n", tui.MutedStyle.Render(truncateCheckpointStr(cp.Context.Summary, 50)))
				}
			}
		},
	}

	return cmd
}

func checkpointShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show checkpoint details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			checkpoint, err := loadCheckpoint(name)
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading checkpoint: " + err.Error()))
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(checkpoint, "", "  ")
				fmt.Println(string(data))
				return
			}

			printCheckpointFull(checkpoint)
		},
	}

	return cmd
}

func checkpointDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a checkpoint",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			path := getCheckpointPath(name)

			if err := os.Remove(path); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error deleting checkpoint: " + err.Error()))
				return
			}

			fmt.Println(tui.SuccessStyle.Render("✓ Deleted checkpoint: " + name))
		},
	}

	return cmd
}

func getCheckpointDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clood", "checkpoints")
}

func getCheckpointPath(name string) string {
	return filepath.Join(getCheckpointDir(), name+".yaml")
}

func saveCheckpoint(cp *Checkpoint) error {
	dir := getCheckpointDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cp)
	if err != nil {
		return err
	}

	return os.WriteFile(getCheckpointPath(cp.Name), data, 0644)
}

func loadCheckpoint(name string) (*Checkpoint, error) {
	data, err := os.ReadFile(getCheckpointPath(name))
	if err != nil {
		return nil, err
	}

	var cp Checkpoint
	if err := yaml.Unmarshal(data, &cp); err != nil {
		return nil, err
	}

	return &cp, nil
}

func listCheckpoints() ([]*Checkpoint, error) {
	dir := getCheckpointDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var checkpoints []*Checkpoint
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			name := strings.TrimSuffix(entry.Name(), ".yaml")
			cp, err := loadCheckpoint(name)
			if err == nil {
				checkpoints = append(checkpoints, cp)
			}
		}
	}

	// Sort by creation time, newest first
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Created.After(checkpoints[j].Created)
	})

	return checkpoints, nil
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func getCwd() string {
	dir, _ := os.Getwd()
	return dir
}

func getGitState() GitState {
	state := GitState{}

	// Get current branch
	if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		state.Branch = strings.TrimSpace(string(out))
	}

	// Get last commit
	if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		state.LastCommit = strings.TrimSpace(string(out))
	}

	// Get status summary
	if out, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			state.Status = fmt.Sprintf("%d files changed", len(lines))
		} else {
			state.Status = "clean"
		}
	}

	return state
}

func getRecentIssues() []IssueRef {
	var issues []IssueRef

	out, err := exec.Command("gh", "issue", "list", "--state", "open", "--limit", "10", "--json", "number,title,labels").Output()
	if err != nil {
		return issues
	}

	var ghIssues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	if err := json.Unmarshal(out, &ghIssues); err != nil {
		return issues
	}

	for _, gi := range ghIssues {
		labels := make([]string, len(gi.Labels))
		for i, l := range gi.Labels {
			labels[i] = l.Name
		}
		issues = append(issues, IssueRef{
			Number: gi.Number,
			Title:  gi.Title,
			Labels: strings.Join(labels, ", "),
		})
	}

	return issues
}

func printCheckpointResume(cp *Checkpoint) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("📜 Loading checkpoint: " + cp.Name))
	fmt.Println()

	// Session info
	fmt.Printf("  Created: %s on %s\n", cp.Created.Format("Jan 02, 2006 15:04"), cp.Machine)
	fmt.Printf("  Directory: %s\n", cp.Directory)
	fmt.Println()

	// Summary
	if cp.Context.Summary != "" {
		fmt.Println(tui.MutedStyle.Render("Session Summary"))
		fmt.Printf("  %s\n\n", cp.Context.Summary)
	}

	// Focus
	if cp.Context.FocusGoal != "" {
		fmt.Printf("🎯 Focus: %s\n\n", cp.Context.FocusGoal)
	}

	// Git state
	fmt.Println(tui.MutedStyle.Render("Git State"))
	fmt.Printf("  Branch: %s @ %s (%s)\n\n", cp.Git.Branch, cp.Git.LastCommit, cp.Git.Status)

	// Next steps
	if len(cp.Context.NextSteps) > 0 {
		fmt.Println(tui.MutedStyle.Render("Suggested Next Steps"))
		for i, step := range cp.Context.NextSteps {
			fmt.Printf("  %d. %s\n", i+1, step)
		}
		fmt.Println()
	}

	// Open issues
	if len(cp.Context.OpenIssues) > 0 {
		fmt.Println(tui.MutedStyle.Render("Open Issues"))
		for _, issue := range cp.Context.OpenIssues[:minInt(5, len(cp.Context.OpenIssues))] {
			fmt.Printf("  #%d: %s\n", issue.Number, issue.Title)
		}
		if len(cp.Context.OpenIssues) > 5 {
			fmt.Printf("  ... and %d more\n", len(cp.Context.OpenIssues)-5)
		}
	}
}

func printCheckpointFull(cp *Checkpoint) {
	data, _ := yaml.Marshal(cp)
	fmt.Println()
	fmt.Println(tui.RenderHeader("Checkpoint: " + cp.Name))
	fmt.Println()
	fmt.Println(string(data))
}

func formatCheckpointAge(d time.Duration) string {
	hours := int(d.Hours())
	if hours < 1 {
		return "< 1 hour ago"
	}
	if hours < 24 {
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := hours / 24
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

func truncateCheckpointStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
