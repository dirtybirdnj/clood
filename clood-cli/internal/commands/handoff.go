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

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Handoff represents session context for continuity
type Handoff struct {
	Timestamp    time.Time `json:"timestamp"`
	Summary      string    `json:"summary"`
	FilesChanged []string  `json:"files_changed"`
	NextSteps    []string  `json:"next_steps"`
	GitRef       string    `json:"git_ref"`
	Branch       string    `json:"branch"`
	ProjectPath  string    `json:"project_path"`
}

func HandoffCmd() *cobra.Command {
	var saveMode bool
	var loadMode bool
	var historyMode bool
	var diffMode bool
	var jsonOutput bool
	var noPush bool

	cmd := &cobra.Command{
		Use:   "handoff [summary]",
		Short: "Session context handoff for continuity",
		Long: `Save or load session context for seamless continuity across sessions.

The handoff command helps maintain context when:
  - Hitting context limits
  - Switching machines
  - Coming back the next day
  - Starting fresh sessions

Modes:
  --save    Save current context (default if summary provided)
  --load    Load and display latest context
  --history View handoff history
  --diff    Show changes since last handoff

Examples:
  clood handoff --save "Completed auth, next: add tests"
  clood handoff "Completed auth, next: add tests"    # Same as above
  clood handoff --load                                # Load latest context
  clood handoff --history                             # View history
  clood handoff --diff                                # Changes since last`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine mode
			if loadMode {
				return runHandoffLoad(jsonOutput)
			}
			if historyMode {
				return runHandoffHistory(jsonOutput)
			}
			if diffMode {
				return runHandoffDiff()
			}

			// Default to save if summary provided
			if len(args) > 0 || saveMode {
				summary := ""
				if len(args) > 0 {
					summary = strings.Join(args, " ")
				}
				return runHandoffSave(summary, noPush, jsonOutput)
			}

			// No mode specified, show help
			return cmd.Help()
		},
	}

	cmd.Flags().BoolVarP(&saveMode, "save", "s", false, "Save current session context")
	cmd.Flags().BoolVarP(&loadMode, "load", "l", false, "Load and display latest context")
	cmd.Flags().BoolVar(&historyMode, "history", false, "View handoff history")
	cmd.Flags().BoolVar(&diffMode, "diff", false, "Show changes since last handoff")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Save without pushing to remote")

	return cmd
}

func runHandoffSave(summary string, noPush, jsonOutput bool) error {
	// Prompt for summary if not provided
	if summary == "" {
		fmt.Print(tui.MutedStyle.Render("Session summary: "))
		var input string
		fmt.Scanln(&input)
		summary = strings.TrimSpace(input)
		if summary == "" {
			return fmt.Errorf("summary required")
		}
	}

	// Build handoff
	handoff := &Handoff{
		Timestamp:   time.Now(),
		Summary:     summary,
		GitRef:      getGitRef(),
		Branch:      getGitBranch(),
		ProjectPath: getCurrentDir(),
	}

	// Get changed files
	handoff.FilesChanged = getChangedFiles()

	// Parse next steps from summary (lines starting with "next:")
	handoff.NextSteps = parseNextSteps(summary)

	// Save to handoff storage
	if err := saveHandoffToStorage(handoff); err != nil {
		return fmt.Errorf("saving handoff: %w", err)
	}

	// Update LAST_SESSION.md
	if err := updateLastSession(handoff); err != nil {
		fmt.Println(tui.ErrorStyle.Render("Warning: couldn't update LAST_SESSION.md: " + err.Error()))
	}

	// Git operations
	if !noPush {
		if err := commitAndPush(handoff); err != nil {
			fmt.Println(tui.ErrorStyle.Render("Warning: git operations failed: " + err.Error()))
		}
	}

	// Output
	if jsonOutput {
		data, _ := json.MarshalIndent(handoff, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println(tui.SuccessStyle.Render("Session handoff saved."))
		fmt.Println()
		fmt.Printf("  Timestamp:  %s\n", handoff.Timestamp.Format("2006-01-02 15:04"))
		fmt.Printf("  Branch:     %s\n", handoff.Branch)
		fmt.Printf("  Commit:     %s\n", truncate(handoff.GitRef, 8))
		fmt.Printf("  Files:      %d changed\n", len(handoff.FilesChanged))
		fmt.Println()
		if !noPush {
			fmt.Println(tui.MutedStyle.Render("Changes committed and pushed."))
		}
	}

	return nil
}

func runHandoffLoad(jsonOutput bool) error {
	// Pull latest
	fmt.Println(tui.MutedStyle.Render("Pulling latest..."))
	exec.Command("git", "pull").Run()

	// Find latest handoff
	handoff, err := loadLatestHandoff()
	if err != nil {
		return fmt.Errorf("loading handoff: %w", err)
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(handoff, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display for human/LLM consumption
	fmt.Println()
	fmt.Println(tui.RenderHeader("Session Context"))
	fmt.Println()
	fmt.Printf("  Last session:  %s\n", handoff.Timestamp.Format("2006-01-02 15:04"))
	fmt.Printf("  Branch:        %s\n", handoff.Branch)
	fmt.Printf("  Commit:        %s\n", truncate(handoff.GitRef, 8))
	fmt.Println()

	fmt.Println(tui.MutedStyle.Render("  What was done:"))
	fmt.Printf("    %s\n", handoff.Summary)
	fmt.Println()

	if len(handoff.FilesChanged) > 0 {
		fmt.Println(tui.MutedStyle.Render("  Files changed:"))
		for _, f := range handoff.FilesChanged {
			fmt.Printf("    - %s\n", f)
		}
		fmt.Println()
	}

	if len(handoff.NextSteps) > 0 {
		fmt.Println(tui.MutedStyle.Render("  Next steps:"))
		for _, step := range handoff.NextSteps {
			fmt.Printf("    [ ] %s\n", step)
		}
		fmt.Println()
	}

	fmt.Println(tui.SuccessStyle.Render("Ready to continue."))

	return nil
}

func runHandoffHistory(jsonOutput bool) error {
	handoffs, err := loadAllHandoffs()
	if err != nil {
		return err
	}

	if len(handoffs) == 0 {
		fmt.Println(tui.MutedStyle.Render("No handoffs found."))
		return nil
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(handoffs, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(tui.RenderHeader("Handoff History"))
	fmt.Println()

	for i, h := range handoffs {
		if i >= 10 {
			fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("  ... and %d more", len(handoffs)-10)))
			break
		}

		age := formatAge(h.Timestamp)
		fmt.Printf("  %s  %s\n",
			tui.MutedStyle.Render(h.Timestamp.Format("Jan 02 15:04")),
			truncate(h.Summary, 50))
		fmt.Printf("    %s on %s (%s)\n",
			truncate(h.GitRef, 8),
			h.Branch,
			age)
		fmt.Println()
	}

	return nil
}

func runHandoffDiff() error {
	handoff, err := loadLatestHandoff()
	if err != nil {
		return fmt.Errorf("loading handoff: %w", err)
	}

	fmt.Println(tui.RenderHeader("Changes Since Last Handoff"))
	fmt.Printf("  Since: %s (%s)\n\n", handoff.Timestamp.Format("Jan 02 15:04"), truncate(handoff.GitRef, 8))

	// Show git diff stat since that commit
	cmd := exec.Command("git", "diff", "--stat", handoff.GitRef+"..HEAD")
	output, err := cmd.Output()
	if err != nil {
		// Try without specific commit
		cmd = exec.Command("git", "status", "--short")
		output, _ = cmd.Output()
	}

	if len(output) == 0 {
		fmt.Println(tui.MutedStyle.Render("  No changes since last handoff."))
	} else {
		fmt.Println(string(output))
	}

	return nil
}

// Helper functions

func getHandoffDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clood", "handoffs")
}

func saveHandoffToStorage(h *Handoff) error {
	dir := getHandoffDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := h.Timestamp.Format("2006-01-02T15-04-05") + ".json"
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, filename), data, 0644)
}

func loadLatestHandoff() (*Handoff, error) {
	handoffs, err := loadAllHandoffs()
	if err != nil {
		return nil, err
	}

	if len(handoffs) == 0 {
		return nil, fmt.Errorf("no handoffs found")
	}

	return &handoffs[0], nil
}

func loadAllHandoffs() ([]Handoff, error) {
	dir := getHandoffDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var handoffs []Handoff
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var h Handoff
		if err := json.Unmarshal(data, &h); err != nil {
			continue
		}

		handoffs = append(handoffs, h)
	}

	// Sort by timestamp descending
	sort.Slice(handoffs, func(i, j int) bool {
		return handoffs[i].Timestamp.After(handoffs[j].Timestamp)
	})

	return handoffs, nil
}

func updateLastSession(h *Handoff) error {
	// Find LAST_SESSION.md in current or parent directories
	path := findLastSessionFile()
	if path == "" {
		// Create in current directory
		path = "LAST_SESSION.md"
	}

	content := fmt.Sprintf(`# Session Handoff - %s

## Summary
%s

---

## Details

- **Branch:** %s
- **Commit:** %s
- **Project:** %s

## Files Changed
%s

## Next Steps
%s

---

*Handoff by clood CLI*
`,
		h.Timestamp.Format("2006-01-02 15:04"),
		h.Summary,
		h.Branch,
		h.GitRef,
		h.ProjectPath,
		formatFileList(h.FilesChanged),
		formatNextStepsList(h.NextSteps),
	)

	return os.WriteFile(path, []byte(content), 0644)
}

func findLastSessionFile() string {
	// Check current directory
	if _, err := os.Stat("LAST_SESSION.md"); err == nil {
		return "LAST_SESSION.md"
	}

	// Check parent (for monorepos)
	if _, err := os.Stat("../LAST_SESSION.md"); err == nil {
		return "../LAST_SESSION.md"
	}

	return ""
}

func commitAndPush(h *Handoff) error {
	// Add LAST_SESSION.md
	exec.Command("git", "add", "LAST_SESSION.md").Run()
	exec.Command("git", "add", "../LAST_SESSION.md").Run() // Try parent too

	// Commit
	msg := fmt.Sprintf("Session handoff: %s", truncate(h.Summary, 50))
	cmd := exec.Command("git", "commit", "-m", msg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	// Push
	cmd = exec.Command("git", "push")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	return nil
}

func getGitRef() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func getGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func getCurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}

func getChangedFiles() []string {
	// Get recently changed files (last commit)
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~1")
	output, err := cmd.Output()
	if err != nil {
		// Try uncommitted changes
		cmd = exec.Command("git", "status", "--porcelain")
		output, _ = cmd.Output()
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Handle porcelain format (status prefix)
		if len(line) > 3 && line[2] == ' ' {
			line = line[3:]
		}
		files = append(files, line)
	}

	return files
}

func parseNextSteps(summary string) []string {
	var steps []string

	// Look for "next:" pattern
	lower := strings.ToLower(summary)
	if idx := strings.Index(lower, "next:"); idx != -1 {
		nextPart := summary[idx+5:]
		// Split by comma or semicolon
		parts := strings.FieldsFunc(nextPart, func(r rune) bool {
			return r == ',' || r == ';'
		})
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				steps = append(steps, p)
			}
		}
	}

	return steps
}

func formatFileList(files []string) string {
	if len(files) == 0 {
		return "- (none)"
	}

	var lines []string
	for _, f := range files {
		lines = append(lines, "- "+f)
	}
	return strings.Join(lines, "\n")
}

func formatNextStepsList(steps []string) string {
	if len(steps) == 0 {
		return "- (none specified)"
	}

	var lines []string
	for _, s := range steps {
		lines = append(lines, "- [ ] "+s)
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
