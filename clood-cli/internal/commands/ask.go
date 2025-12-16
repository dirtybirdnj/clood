package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dirtybirdnj/clood/internal/router"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func AskCmd() *cobra.Command {
	var forceTier int
	var noContext bool

	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask a question (auto-routes to appropriate tier)",
		Long: `Ask a question and clood will route it to the appropriate tier:
  - Tier 1 (mods): Fast responses for simple questions
  - Tier 2 (crush): Deep reasoning for complex tasks`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			question := strings.Join(args, " ")

			// Determine tier
			tier := forceTier
			if tier == 0 {
				tier = router.ClassifyQuery(question)
			}

			fmt.Println(tui.RenderTier(tier))
			fmt.Println()

			// Inject project context if available and not disabled
			var context string
			if !noContext {
				context = getProjectContext()
			}

			if tier == 1 {
				runMods(question, context)
			} else {
				runCrush(question, context)
			}
		},
	}

	cmd.Flags().IntVarP(&forceTier, "tier", "T", 0, "Force specific tier (1 or 2)")
	cmd.Flags().BoolVar(&noContext, "no-context", false, "Skip project context injection")

	return cmd
}

func getProjectContext() string {
	// Check for project context files
	contextFiles := []string{"CLAUDE.md", "AGENTS.md", "README.md"}

	for _, file := range contextFiles {
		if content, err := os.ReadFile(file); err == nil {
			// Truncate if too long
			s := string(content)
			if len(s) > 2000 {
				s = s[:2000] + "\n... (truncated)"
			}
			return s
		}
	}
	return ""
}

func runMods(question string, context string) {
	// Check if mods is available
	if _, err := exec.LookPath("mods"); err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: 'mods' not found in PATH"))
		fmt.Fprintln(os.Stderr, tui.MutedStyle.Render("Install: go install github.com/charmbracelet/mods@latest"))
		return
	}

	// Build the prompt
	prompt := question
	if context != "" {
		prompt = fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context, question)
	}

	// Run mods
	cmd := exec.Command("mods", prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error running mods: "+err.Error()))
	}
}

func runCrush(question string, context string) {
	// Check if crush is available
	if _, err := exec.LookPath("crush"); err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: 'crush' not found in PATH"))
		fmt.Fprintln(os.Stderr, tui.MutedStyle.Render("Install: brew install charmbracelet/tap/crush"))
		return
	}

	// For crush, we launch it interactively
	// The context would ideally be passed via MCP or initial prompt
	fmt.Println(tui.MutedStyle.Render("Launching Crush for deep reasoning..."))
	if context != "" {
		fmt.Println(tui.MutedStyle.Render("(Project context detected)"))
	}
	fmt.Println()

	cmd := exec.Command("crush")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error running crush: "+err.Error()))
	}
}
