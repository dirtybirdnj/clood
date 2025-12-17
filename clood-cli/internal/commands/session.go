// Package commands provides CLI command implementations.
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// SessionContext represents the CONTEXT.yaml session state
type SessionContext struct {
	Session   string              `yaml:"session"`
	Branch    string              `yaml:"branch"`
	Status    string              `yaml:"status"`
	Focus     SessionContextFocus `yaml:"focus"`
	Next      []string            `yaml:"next"`
	Summary   string              `yaml:"summary"`
	Blockers  []string            `yaml:"blockers,omitempty"`
	UpdatedBy string              `yaml:"updated_by,omitempty"`
}

// SessionContextFocus defines what files/issues are relevant
type SessionContextFocus struct {
	Files  []SessionFocusFile `yaml:"files,omitempty"`
	Issues []int              `yaml:"issues,omitempty"`
}

// SessionFocusFile represents a file with context on why it matters
type SessionFocusFile struct {
	Path string `yaml:"path"`
	Why  string `yaml:"why,omitempty"`
}

const sessionContextFile = "CONTEXT.yaml"

// SessionCmd creates the session command
func SessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Session context management (The Scholar Spirit)",
		Long: `The Scholar Spirit (学者の霊) maintains the garden's memory,
ensuring no wisdom is lost between sessions.

Session context is saved to CONTEXT.yaml and can be committed
to share across machines and agents.`,
	}

	cmd.AddCommand(sessionShowCmd())
	cmd.AddCommand(sessionLoadCmd())
	cmd.AddCommand(sessionSaveCmd())
	cmd.AddCommand(sessionInitCmd())

	return cmd
}

func sessionShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current session context",
		Run: func(cmd *cobra.Command, args []string) {
			if err := sessionShow(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func sessionLoadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load",
		Short: "Load context (markdown format for agents)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := sessionLoad(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func sessionSaveCmd() *cobra.Command {
	var commitPush bool
	var addFiles string
	var addIssues string

	cmd := &cobra.Command{
		Use:   "save [summary]",
		Short: "Save session context",
		Long: `Save session context to CONTEXT.yaml.

Use "next:" in summary to define next steps:
  clood session save "Fixed bug, next: test hosts, add docs"

Use --commit to automatically commit and push.`,
		Run: func(cmd *cobra.Command, args []string) {
			summary := strings.Join(args, " ")
			if err := sessionSave(summary, commitPush, addFiles, addIssues); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVarP(&commitPush, "commit", "c", false, "Commit and push after saving")
	cmd.Flags().StringVarP(&addFiles, "files", "f", "", "Comma-separated focus files")
	cmd.Flags().StringVarP(&addIssues, "issues", "i", "", "Comma-separated issue numbers")

	return cmd
}

func sessionInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize new CONTEXT.yaml",
		Run: func(cmd *cobra.Command, args []string) {
			if err := sessionInit(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func sessionShow() error {
	ctx, err := loadSessionContext()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No CONTEXT.yaml found.")
			fmt.Println("Run 'clood session init' to create one.")
			return nil
		}
		return err
	}

	fmt.Println("╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║           The Scholar Spirit's Memory                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Printf("\n  Session: %s\n", ctx.Session)
	fmt.Printf("  Branch:  %s\n", ctx.Branch)
	fmt.Printf("  Status:  %s\n", ctx.Status)

	if ctx.Summary != "" {
		fmt.Printf("\n📝 Summary:\n   %s\n", ctx.Summary)
	}

	if len(ctx.Focus.Files) > 0 {
		fmt.Println("\n📁 Focus Files:")
		for _, f := range ctx.Focus.Files {
			if f.Why != "" {
				fmt.Printf("   • %s (%s)\n", f.Path, f.Why)
			} else {
				fmt.Printf("   • %s\n", f.Path)
			}
		}
	}

	if len(ctx.Focus.Issues) > 0 {
		fmt.Printf("\n🎫 Related Issues: #%v\n", formatIssues(ctx.Focus.Issues))
	}

	if len(ctx.Next) > 0 {
		fmt.Println("\n⏭️  Next Steps:")
		for i, step := range ctx.Next {
			fmt.Printf("   %d. %s\n", i+1, step)
		}
	}

	if len(ctx.Blockers) > 0 {
		fmt.Println("\n🚧 Blockers:")
		for _, b := range ctx.Blockers {
			fmt.Printf("   • %s\n", b)
		}
	}

	fmt.Println()
	return nil
}

func sessionLoad() error {
	ctx, err := loadSessionContext()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No CONTEXT.yaml found.")
			return nil
		}
		return err
	}

	// Output in markdown format for agent consumption
	fmt.Println("# Session Context")
	fmt.Println()
	fmt.Printf("**Branch:** %s | **Status:** %s\n\n", ctx.Branch, ctx.Status)

	if ctx.Summary != "" {
		fmt.Printf("## Summary\n%s\n\n", ctx.Summary)
	}

	if len(ctx.Next) > 0 {
		fmt.Println("## Next Steps")
		for _, step := range ctx.Next {
			fmt.Printf("- [ ] %s\n", step)
		}
		fmt.Println()
	}

	if len(ctx.Focus.Files) > 0 {
		fmt.Println("## Focus Files")
		for _, f := range ctx.Focus.Files {
			fmt.Printf("- `%s`", f.Path)
			if f.Why != "" {
				fmt.Printf(" — %s", f.Why)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	if len(ctx.Focus.Issues) > 0 {
		fmt.Printf("## Related Issues\n")
		for _, issue := range ctx.Focus.Issues {
			fmt.Printf("- #%d\n", issue)
		}
		fmt.Println()
	}

	if len(ctx.Blockers) > 0 {
		fmt.Println("## Blockers")
		for _, b := range ctx.Blockers {
			fmt.Printf("- ⚠️ %s\n", b)
		}
	}

	return nil
}

func sessionSave(summary string, commitPush bool, filesArg string, issuesArg string) error {
	// Get current branch
	branch := "main"
	if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		branch = strings.TrimSpace(string(out))
	}

	// Try to load existing context or create new
	ctx, err := loadSessionContext()
	if err != nil {
		ctx = &SessionContext{
			Status: "in_progress",
			Focus:  SessionContextFocus{},
		}
	}

	// Update fields
	ctx.Session = time.Now().Format(time.RFC3339)
	ctx.Branch = branch
	if summary != "" {
		ctx.Summary = summary
	}
	ctx.UpdatedBy = "clood session save"

	// Parse "next:" from summary if present
	if strings.Contains(strings.ToLower(summary), "next:") {
		parts := strings.SplitN(summary, "next:", 2)
		if len(parts) < 2 {
			parts = strings.SplitN(summary, "Next:", 2)
		}
		if len(parts) == 2 {
			nextPart := strings.TrimSpace(parts[1])
			steps := strings.Split(nextPart, ",")
			ctx.Next = []string{}
			for _, s := range steps {
				s = strings.TrimSpace(s)
				if s != "" {
					ctx.Next = append(ctx.Next, s)
				}
			}
			// Update summary to just the first part
			ctx.Summary = strings.TrimSpace(parts[0])
		}
	}

	// Add files if specified
	if filesArg != "" {
		files := strings.Split(filesArg, ",")
		for _, f := range files {
			f = strings.TrimSpace(f)
			if f != "" {
				ctx.Focus.Files = append(ctx.Focus.Files, SessionFocusFile{Path: f})
			}
		}
	}

	// Add issues if specified
	if issuesArg != "" {
		issues := strings.Split(issuesArg, ",")
		for _, i := range issues {
			i = strings.TrimSpace(i)
			var num int
			if _, err := fmt.Sscanf(i, "%d", &num); err == nil {
				ctx.Focus.Issues = append(ctx.Focus.Issues, num)
			}
		}
	}

	// Save to file
	if err := saveSessionContext(ctx); err != nil {
		return err
	}

	fmt.Println("✓ Context saved to CONTEXT.yaml")

	if commitPush {
		fmt.Println("\nCommitting and pushing...")
		exec.Command("git", "add", sessionContextFile).Run()
		exec.Command("git", "commit", "-m", "Update session context").Run()
		if err := exec.Command("git", "push").Run(); err != nil {
			fmt.Println("⚠️  Push failed - you may need to pull first")
		} else {
			fmt.Println("✓ Pushed to remote")
		}
	} else {
		fmt.Println("\nTo share with other sessions:")
		fmt.Println("  git add CONTEXT.yaml && git commit -m \"Update session context\" && git push")
	}

	return nil
}

func sessionInit() error {
	if _, err := os.Stat(sessionContextFile); err == nil {
		fmt.Println("CONTEXT.yaml already exists.")
		fmt.Println("Use 'clood session save' to update it.")
		return nil
	}

	branch := "main"
	if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		branch = strings.TrimSpace(string(out))
	}

	ctx := &SessionContext{
		Session: time.Now().Format(time.RFC3339),
		Branch:  branch,
		Status:  "starting",
		Focus: SessionContextFocus{
			Files:  []SessionFocusFile{},
			Issues: []int{},
		},
		Next:      []string{"Review codebase", "Define initial tasks"},
		Summary:   "New session initialized",
		UpdatedBy: "clood session init",
	}

	if err := saveSessionContext(ctx); err != nil {
		return err
	}

	fmt.Println("✓ Created CONTEXT.yaml")
	fmt.Println()
	fmt.Println("Update with:")
	fmt.Println("  clood session save \"your summary, next: task1, task2\"")
	fmt.Println()
	fmt.Println("Add focus files:")
	fmt.Println("  clood session save -f \"src/main.go,src/lib.go\" \"working on core\"")

	return nil
}

func loadSessionContext() (*SessionContext, error) {
	path := findSessionContextFile()
	if path == "" {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ctx SessionContext
	if err := yaml.Unmarshal(data, &ctx); err != nil {
		return nil, err
	}

	return &ctx, nil
}

func saveSessionContext(ctx *SessionContext) error {
	data, err := yaml.Marshal(ctx)
	if err != nil {
		return err
	}

	header := `# Session Context — The Scholar Spirit's Memory
#
# This file passes context between agents and sessions.
# Commit and push to share across machines.
#
# Update: clood session save "summary, next: tasks"
# View:   clood session show
# Load:   clood session load
#
# "The garden remembers what the gardener forgets."

`
	return os.WriteFile(sessionContextFile, []byte(header+string(data)), 0644)
}

func findSessionContextFile() string {
	// Check current directory
	if _, err := os.Stat(sessionContextFile); err == nil {
		return sessionContextFile
	}

	// Check git root
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		root := strings.TrimSpace(string(out))
		path := filepath.Join(root, sessionContextFile)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func formatIssues(issues []int) string {
	strs := make([]string, len(issues))
	for i, num := range issues {
		strs[i] = fmt.Sprintf("%d", num)
	}
	return strings.Join(strs, ", #")
}
