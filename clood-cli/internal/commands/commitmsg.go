package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

const commitMsgSystemPrompt = `You are a commit message generator. Analyze the git diff and generate a clear, concise commit message.

Rules:
1. First line: imperative mood, max 72 chars (e.g., "Add feature" not "Added feature")
2. If needed, blank line then detailed description
3. Focus on WHY, not just WHAT changed
4. Use conventional commit prefixes when appropriate: feat:, fix:, docs:, refactor:, test:, chore:

Output ONLY the commit message, no explanations or markdown.`

const commitMsgWithHaikuPrompt = `You are a commit message generator for a creative codebase. Analyze the git diff and generate a clear commit message WITH a haiku.

Rules:
1. First line: imperative mood, max 72 chars (e.g., "Add feature" not "Added feature")
2. If needed, blank line then detailed description
3. Focus on WHY, not just WHAT changed
4. Use conventional commit prefixes: feat:, fix:, docs:, refactor:, test:, chore:
5. End with a blank line and a haiku (5-7-5 syllables) reflecting the change

Output ONLY the commit message with haiku, no explanations or markdown.`

// CommitMsgResult is the JSON output structure
type CommitMsgResult struct {
	Message   string `json:"message"`
	Model     string `json:"model"`
	Host      string `json:"host"`
	TimeMs    int64  `json:"time_ms"`
	DiffLines int    `json:"diff_lines"`
}

func CommitMsgCmd() *cobra.Command {
	var jsonOutput bool
	var dryRun bool
	var apply bool
	var haiku bool
	var conventional bool

	cmd := &cobra.Command{
		Use:   "commit-msg",
		Short: "Generate commit message from staged changes",
		Long: `Generate a commit message from git diff using local LLM.

Examples:
  git diff --staged | clood commit-msg     # Pipe diff, get message
  clood commit-msg                         # Auto-reads staged changes
  clood commit-msg --haiku                 # Include a haiku
  clood commit-msg --apply                 # Generate and commit
  clood commit-msg --dry-run               # Preview only`,
		Run: func(cmd *cobra.Command, args []string) {
			useJSON := jsonOutput || output.IsJSON()

			// Get diff - from stdin or git
			diff, err := getDiff()
			if err != nil {
				if useJSON {
					fmt.Printf(`{"error": %q}`, err.Error())
				} else {
					fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
				}
				return
			}

			if strings.TrimSpace(diff) == "" {
				if useJSON {
					fmt.Println(`{"error": "No staged changes found"}`)
				} else {
					fmt.Println(tui.WarningStyle.Render("No staged changes. Stage files with 'git add' first."))
				}
				return
			}

			diffLines := len(strings.Split(diff, "\n"))

			// Load config and find a host
			cfg, err := config.Load()
			if err != nil {
				if useJSON {
					fmt.Printf(`{"error": %q}`, err.Error())
				} else {
					fmt.Println(tui.ErrorStyle.Render("Error loading config: " + err.Error()))
				}
				return
			}

			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			// Find first available host with a model
			var targetHost *hosts.HostStatus
			statuses := mgr.CheckAllHosts()
			for _, s := range statuses {
				if s.Online && len(s.Models) > 0 {
					targetHost = s
					break
				}
			}

			if targetHost == nil {
				if useJSON {
					fmt.Println(`{"error": "No Ollama hosts available"}`)
				} else {
					fmt.Println(tui.ErrorStyle.Render("No Ollama hosts available. Run 'clood health' to check."))
				}
				return
			}

			// Pick model - prefer fast tier
			modelName := cfg.Tiers.Fast.Model
			allModels := mgr.GetAllModels()
			if _, found := allModels[modelName]; !found {
				// Fall back to first available model
				modelName = targetHost.Models[0].Name
			}

			// Build prompt
			systemPrompt := commitMsgSystemPrompt
			if haiku {
				systemPrompt = commitMsgWithHaikuPrompt
			}
			if conventional {
				systemPrompt += "\nALWAYS use conventional commit format (feat:, fix:, docs:, etc.)"
			}

			userPrompt := fmt.Sprintf("Generate a commit message for this diff:\n\n```diff\n%s\n```", diff)

			// Truncate if diff is too long
			if len(userPrompt) > 8000 {
				userPrompt = userPrompt[:8000] + "\n... (truncated)"
			}

			if !useJSON && !dryRun {
				fmt.Printf("%s Generating commit message with %s...\n",
					tui.MutedStyle.Render("●"),
					modelName)
			}

			// Generate
			client := ollama.NewClient(targetHost.Host.URL, 60*time.Second)
			start := time.Now()

			resp, err := client.GenerateWithSystem(modelName, systemPrompt, userPrompt)
			elapsed := time.Since(start)

			if err != nil {
				if useJSON {
					fmt.Printf(`{"error": %q}`, err.Error())
				} else {
					fmt.Println(tui.ErrorStyle.Render("Generation failed: " + err.Error()))
				}
				return
			}

			message := strings.TrimSpace(resp.Response)

			// Add model attribution as Co-Authored-By
			attribution := fmt.Sprintf("\n\nCo-Authored-By: %s (%s) <inference@%s>",
				modelName, targetHost.Host.Name, strings.TrimPrefix(strings.TrimPrefix(targetHost.Host.URL, "http://"), "https://"))
			message = message + attribution

			// JSON output
			if useJSON {
				result := CommitMsgResult{
					Message:   message,
					Model:     modelName,
					Host:      targetHost.Host.Name,
					TimeMs:    elapsed.Milliseconds(),
					DiffLines: diffLines,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Human output
			fmt.Println()
			fmt.Println(tui.RenderHeader("Suggested Commit Message"))
			fmt.Println()
			fmt.Println(tui.BoxStyle.Render(message))
			fmt.Println()
			fmt.Printf("%s Generated by %s on %s (%dms, %d lines analyzed)\n",
				tui.MutedStyle.Render("●"),
				modelName,
				targetHost.Host.Name,
				elapsed.Milliseconds(),
				diffLines)

			// Apply if requested
			if apply && !dryRun {
				fmt.Println()
				fmt.Print(tui.MutedStyle.Render("Applying commit... "))

				commitCmd := exec.Command("git", "commit", "-m", message)
				commitCmd.Stdout = os.Stdout
				commitCmd.Stderr = os.Stderr

				if err := commitCmd.Run(); err != nil {
					fmt.Println(tui.ErrorStyle.Render("FAILED"))
					fmt.Println(tui.ErrorStyle.Render("Commit failed: " + err.Error()))
				} else {
					fmt.Println(tui.SuccessStyle.Render("DONE"))
				}
			} else if !apply {
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("Use --apply to commit with this message"))
			}
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview only, don't commit")
	cmd.Flags().BoolVar(&apply, "apply", false, "Apply the generated commit message")
	cmd.Flags().BoolVar(&haiku, "haiku", false, "Include a haiku in the commit message")
	cmd.Flags().BoolVar(&conventional, "conventional", false, "Use conventional commits format")

	return cmd
}

// getDiff reads diff from stdin or runs git diff --staged
func getDiff() (string, error) {
	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Stdin has data - read it
		reader := bufio.NewReader(os.Stdin)
		var sb strings.Builder
		for {
			line, err := reader.ReadString('\n')
			sb.WriteString(line)
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
		}
		return sb.String(), nil
	}

	// No stdin - run git diff --staged
	cmd := exec.Command("git", "diff", "--staged")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	return string(out), nil
}
