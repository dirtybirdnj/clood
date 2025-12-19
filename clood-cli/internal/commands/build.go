package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dirtybirdnj/clood/internal/analyze"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// BcbcCmd is the shorthand for "build clood build clood" - summons the Council
func BcbcCmd() *cobra.Command {
	var dryRun bool
	var runTests bool
	var noLaunch bool

	cmd := &cobra.Command{
		Use:   "bcbc",
		Short: "Build Clood Build Clood - launch Claude with static analysis context",
		Long: `Shorthand for "clood build clood build clood"

bcbc = Build Clood Build Clood

Runs static analysis on the codebase and launches Claude Code with:
1. Pre-computed context (go vet, build status, TODOs, symbols)
2. MCP server configured for clood tools
3. A prompt to analyze and improve clood

The Council of Wojacks approves.
"The recursion tail grows." - O-Ren Ishii Wojack`,
		Run: func(cmd *cobra.Command, args []string) {
			runBcbc(dryRun, runTests, noLaunch)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show analysis and prompt without launching Claude")
	cmd.Flags().BoolVar(&runTests, "tests", false, "Include test results in analysis (slower)")
	cmd.Flags().BoolVar(&noLaunch, "no-launch", false, "Run analysis but don't launch Claude")

	return cmd
}

func BuildCmd() *cobra.Command {
	var skipPull bool
	var outputPath string

	cmd := &cobra.Command{
		Use:   "build [target...]",
		Short: "Build clood from source",
		Long: `Yo dawg, I heard you like clood, so we put a build in your clood
so you can build while you clood.

  clood build clood              # git pull && go build -o ~/bin/clood
  clood build clood build clood  # The Council convenes...
  clood bcbc                     # Shorthand for the above

The spirit of Xzibit is pleased.`,
		Example: `  clood build clood              # Pull latest and build to ~/bin/clood
  clood build clood --skip-pull  # Just build, no git pull
  clood build clood -o /path     # Custom output path
  clood build clood build clood  # Summon the Council of Wojacks
  clood bcbc                     # bcbc = build clood build clood`,
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// Check for the recursive easter egg: clood build clood build clood
			// args would be ["clood", "build", "clood"]
			// Also support shorthand: clood bcbc (build clood build clood)
			if len(args) == 3 && args[0] == "clood" && args[1] == "build" && args[2] == "clood" {
				showCouncilOfWojacks()
				return
			}


			target := "clood"
			if len(args) > 0 {
				target = args[0]
			}

			if target != "clood" {
				fmt.Println(tui.ErrorStyle.Render(fmt.Sprintf("Unknown target: %s", target)))
				fmt.Println(tui.MutedStyle.Render("Currently only 'clood build clood' is supported"))
				fmt.Println(tui.MutedStyle.Render("Or try: clood build clood build clood"))
				return
			}

			// Find the clood-cli directory
			cloodDir, err := findCloodDir()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Could not find clood-cli directory: " + err.Error()))
				return
			}

			// Determine output path
			if outputPath == "" {
				homeDir, _ := os.UserHomeDir()
				if runtime.GOOS == "windows" {
					outputPath = filepath.Join(homeDir, "bin", "clood.exe")
				} else {
					outputPath = filepath.Join(homeDir, "bin", "clood")
				}
			}

			// Ensure output directory exists
			outputDir := filepath.Dir(outputPath)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Could not create output directory: " + err.Error()))
				return
			}

			fmt.Println(tui.RenderHeader("Building clood"))
			fmt.Println()
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Source:"), cloodDir)
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Output:"), outputPath)
			fmt.Println()

			// Step 1: Git pull (unless skipped)
			if !skipPull {
				fmt.Printf("  %s git pull...\n", tui.MutedStyle.Render("→"))
				gitCmd := exec.Command("git", "pull")
				gitCmd.Dir = filepath.Dir(cloodDir) // Parent of clood-cli
				gitCmd.Stdout = os.Stdout
				gitCmd.Stderr = os.Stderr
				if err := gitCmd.Run(); err != nil {
					fmt.Println(tui.ErrorStyle.Render("  ✗ git pull failed: " + err.Error()))
					return
				}
				fmt.Println(tui.SuccessStyle.Render("  ✓ pulled latest"))
			} else {
				fmt.Println(tui.MutedStyle.Render("  ○ skipping git pull"))
			}

			// Step 2: Go build
			fmt.Printf("  %s go build -o %s ...\n", tui.MutedStyle.Render("→"), outputPath)
			buildCmd := exec.Command("go", "build", "-o", outputPath, "./cmd/clood")
			buildCmd.Dir = cloodDir
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				fmt.Println(tui.ErrorStyle.Render("  ✗ go build failed: " + err.Error()))
				return
			}
			fmt.Println(tui.SuccessStyle.Render("  ✓ built successfully"))

			// Victory message
			fmt.Println()
			fmt.Println(tui.SuccessStyle.Render("  Yo dawg, clood built clood. 🎤"))
			fmt.Println()

			// Show build info
			showBuildInfo(outputPath)
		},
	}

	cmd.Flags().BoolVar(&skipPull, "skip-pull", false, "Skip git pull, just build")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path (default ~/bin/clood)")

	return cmd
}

// showBuildInfo displays version, ollama, and system info after build
func showBuildInfo(binaryPath string) {
	// Get version from newly built binary
	versionCmd := exec.Command(binaryPath, "--version")
	if versionOut, err := versionCmd.Output(); err == nil {
		// Extract just the version line
		lines := strings.Split(string(versionOut), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Version:") {
				fmt.Printf("  %s\n", strings.TrimSpace(line))
				break
			}
		}
	}

	// Get ollama version
	ollamaCmd := exec.Command("ollama", "--version")
	if ollamaOut, err := ollamaCmd.Output(); err == nil {
		ollamaVer := strings.TrimSpace(string(ollamaOut))
		fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Ollama:"), ollamaVer)
	} else {
		fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Ollama:"), tui.ErrorStyle.Render("not found"))
	}

	// System one-liner
	hostname, _ := os.Hostname()
	fmt.Printf("  %s %s/%s on %s\n", tui.MutedStyle.Render("System:"), runtime.GOOS, runtime.GOARCH, hostname)
	fmt.Println()
}

// runBcbc performs static analysis and launches Claude with context
func runBcbc(dryRun, runTests, noLaunch bool) {
	// Show the banner
	fmt.Println()
	fmt.Println(tui.AccentStyle.Render(`
  ╔═══════════════════════════════════════════════════════════════════════════╗
  ║                     THE COUNCIL OF WOJACKS CONVENES                       ║
  ║                                                                           ║
  ║   🗡️  Static analysis engaged. Claude shall receive enlightenment. 🗡️      ║
  ╚═══════════════════════════════════════════════════════════════════════════╝
`))

	// Find project root
	projectRoot, err := findCloodDir()
	if err != nil {
		// Try current directory
		projectRoot, _ = os.Getwd()
	} else {
		// Go up one level from clood-cli to clood root
		projectRoot = filepath.Dir(projectRoot)
	}

	fmt.Println(tui.RenderHeader("RUNNING STATIC ANALYSIS"))
	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Project:"), projectRoot)
	fmt.Println()

	// Run analysis
	fmt.Printf("  %s Running go vet, build check, symbol scan...\n", tui.MutedStyle.Render("→"))
	analysis, err := analyze.RunAnalysis(filepath.Join(projectRoot, "clood-cli"), runTests)
	if err != nil {
		fmt.Println(tui.ErrorStyle.Render("  Analysis failed: " + err.Error()))
		return
	}

	// Show summary
	fmt.Println(tui.SuccessStyle.Render("  ✓ Analysis complete"))
	fmt.Printf("  %s %s\n\n", tui.MutedStyle.Render("Summary:"), analysis.FormatSummary())

	// Format context for Claude
	analysisContext := analysis.FormatForClaude()

	// The improvement prompt
	improvementPrompt := `COUNCIL MANDATE: CLOOD SELF-IMPROVEMENT

You have been summoned by "clood bcbc" to improve clood. Static analysis has been pre-computed to save you discovery time.

DIRECTIVE:
1. Review the static analysis context above (build status, vet issues, TODOs)
2. Use clood MCP tools for any additional exploration (clood_grep, clood_tree, clood_symbols)
3. Identify concrete improvements to the codebase
4. Implement fixes for any build/vet issues first
5. Then address TODOs and propose enhancements

CONSTRAINTS:
- Use clood_preflight first to confirm MCP tools are available
- Prefer clood_* tools over built-in Grep/Glob/Task
- Focus on actionable improvements, not documentation

OUTPUT:
- Fix any build or vet issues
- Address high-priority TODOs
- Propose one enhancement to clood's capabilities

"The recursive path is the only path." - Ancient Wojack Proverb`

	if dryRun {
		// Just show what would be sent
		fmt.Println(tui.RenderHeader("ANALYSIS CONTEXT (would be injected)"))
		fmt.Println(analysisContext)
		fmt.Println(tui.RenderHeader("INITIAL PROMPT"))
		fmt.Println(improvementPrompt)
		return
	}

	if noLaunch {
		fmt.Println(tui.RenderHeader("ANALYSIS CONTEXT"))
		fmt.Println(analysisContext)
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("  Use --dry-run=false to launch Claude"))
		return
	}

	// Check if claude is available
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Println(tui.ErrorStyle.Render("  Claude CLI not found. Install with: npm install -g @anthropic-ai/claude-code"))
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("  Showing context for manual use:"))
		fmt.Println()
		fmt.Println(analysisContext)
		fmt.Println(improvementPrompt)
		return
	}

	// Build the system prompt with analysis context
	systemPromptAddition := fmt.Sprintf(`
<static-analysis-context>
%s
</static-analysis-context>

IMPORTANT: The above static analysis was pre-computed by "clood bcbc". Use it to skip discovery and go straight to improvements.
`, analysisContext)

	// Launch Claude with context
	fmt.Println(tui.RenderHeader("LAUNCHING CLAUDE"))
	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Claude:"), claudePath)
	fmt.Printf("  %s injecting %d bytes of analysis context\n", tui.MutedStyle.Render("→"), len(systemPromptAddition))
	fmt.Println()

	// Build claude command
	claudeCmd := exec.Command(claudePath,
		"--append-system-prompt", systemPromptAddition,
		improvementPrompt,
	)
	claudeCmd.Dir = projectRoot
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	// Run interactively
	if err := claudeCmd.Run(); err != nil {
		fmt.Println(tui.ErrorStyle.Render("  Claude exited with error: " + err.Error()))
	}
}

// showCouncilOfWojacks displays the easter egg (preserved for build clood build clood)
func showCouncilOfWojacks() {
	// Redirect to the new function
	runBcbc(false, false, false)
}

// findCloodDir locates the clood-cli directory
func findCloodDir() (string, error) {
	// First, try relative to current executable
	execPath, err := os.Executable()
	if err == nil {
		// Walk up looking for clood-cli with go.mod
		dir := filepath.Dir(execPath)
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, "clood-cli")
			if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err == nil {
				return candidate, nil
			}
			candidate = filepath.Join(dir, "Code", "clood", "clood-cli")
			if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err == nil {
				return candidate, nil
			}
			dir = filepath.Dir(dir)
		}
	}

	// Try common locations based on OS
	homeDir, _ := os.UserHomeDir()
	candidates := []string{
		// Universal
		filepath.Join(homeDir, "Code", "clood", "clood-cli"),
		filepath.Join(homeDir, "code", "clood", "clood-cli"),
		filepath.Join(homeDir, "src", "clood", "clood-cli"),
		filepath.Join(homeDir, "projects", "clood", "clood-cli"),
		filepath.Join(homeDir, "repos", "clood", "clood-cli"),
		filepath.Join(homeDir, "git", "clood", "clood-cli"),
	}

	// Add Windows-specific paths
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(homeDir, "Documents", "GitHub", "clood", "clood-cli"),
			filepath.Join(homeDir, "source", "repos", "clood", "clood-cli"),
		)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err == nil {
			return candidate, nil
		}
	}

	// Try to find it via git
	gitCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if output, err := gitCmd.Output(); err == nil {
		repoRoot := strings.TrimSpace(string(output))
		candidate := filepath.Join(repoRoot, "clood-cli")
		if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not find clood-cli directory")
}
