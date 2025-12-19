package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

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
  clood build bcbc               # Shorthand for the above

The spirit of Xzibit is pleased.`,
		Example: `  clood build clood              # Pull latest and build to ~/bin/clood
  clood build clood --skip-pull  # Just build, no git pull
  clood build clood -o /path     # Custom output path
  clood build clood build clood  # Summon the Council of Wojacks
  clood build bcbc               # bcbc = build clood build clood`,
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// Check for the recursive easter egg: clood build clood build clood
			// args would be ["clood", "build", "clood"]
			// Also support shorthand: clood bcbc (build clood build clood)
			if len(args) == 3 && args[0] == "clood" && args[1] == "build" && args[2] == "clood" {
				showCouncilOfWojacks()
				return
			}

			// Shorthand: bcbc = build clood build clood
			if len(args) == 1 && args[0] == "bcbc" {
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

// showCouncilOfWojacks displays the easter egg and self-improvement prompt
func showCouncilOfWojacks() {
	// The scene: Two README wojacks try to use clood before the Council
	// Setting: Kill Bill Charlie Brown Establishment
	fmt.Println()
	fmt.Println(tui.AccentStyle.Render(`
  ╔═══════════════════════════════════════════════════════════════════════════╗
  ║                     THE COUNCIL OF WOJACKS CONVENES                       ║
  ║                                                                           ║
  ║   ┌─────────────────────────────────────────────────────────────────┐     ║
  ║   │  clood build clood build clood                                  │     ║
  ║   │  clood build bcbc                                               │     ║
  ║   │                                                                 │     ║
  ║   │  POINTING WOJACK: "You need to build clood."                    │     ║
  ║   │  CONFUSED WOJACK: "clood clood build?"                          │     ║
  ║   │  POINTING WOJACK: "No! clood build clood!"                      │     ║
  ║   │  CONFUSED WOJACK: "So clood... builds... clood?"                │     ║
  ║   │  POINTING WOJACK: "YES!"                                        │     ║
  ║   │  CONFUSED WOJACK: "Then what does bcbc do?"                     │     ║
  ║   │  POINTING WOJACK: "THE SAME THING"                              │     ║
  ║   │  CONFUSED WOJACK: "WHY ARE THERE TWO WAYS"                      │     ║
  ║   │                                                                 │     ║
  ║   │     😐  😔  🤔  😶  😑                                           │     ║
  ║   │     Bloomer  Doomer  Thinker  NPC  Zoomer                       │     ║
  ║   │                                                                 │     ║
  ║   │  O-REN ISHII WOJACK: "The recursion tail grows."                │     ║
  ║   └─────────────────────────────────────────────────────────────────┘     ║
  ║                                                                           ║
  ║   🗡️  The Council has spoken. Self-improvement shall commence. 🗡️         ║
  ╚═══════════════════════════════════════════════════════════════════════════╝
`))

	fmt.Println()
	fmt.Println(tui.RenderHeader("AUTOMATED SYSTEM DEVELOPMENT"))
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("  The Council demands introspection. Feed this to Claude:"))
	fmt.Println()

	// The actual self-improvement prompt
	prompt := `┌──────────────────────────────────────────────────────────────────────────────┐
│  COUNCIL MANDATE: CLOOD USAGE ANALYSIS                                       │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  I want you to analyze this project to help Claude use clood better during   │
│  coding sessions.                                                            │
│                                                                              │
│  DIRECTIVE:                                                                  │
│  1. Parse the codebase using clood tools (eat your own dogfood)              │
│  2. Check how much YOU used clood vs built-in tools this session             │
│  3. Report findings on local-first compliance                                │
│  4. Suggest improvements to maximize filesystem traversal and system info    │
│  5. Propose next-step improvements to push local-first to the max            │
│                                                                              │
│  CONSTRAINTS:                                                                │
│  - Use clood_preflight first                                                 │
│  - Use clood_grep instead of Grep                                            │
│  - Use clood_tree instead of Glob                                            │
│  - Use clood_ask for coding questions (if Ollama online)                     │
│  - Avoid network requests at all costs                                       │
│                                                                              │
│  OUTPUT:                                                                     │
│  - Self-assessment of clood usage this session                               │
│  - Specific improvements to CLAUDE.md or MCP tools                           │
│  - Code changes to implement (if any)                                        │
│                                                                              │
│  MEME: Poking with a stick - "c'mon, build something"                        │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘`

	fmt.Println(tui.AccentStyle.Render(prompt))
	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("  Copy the above and paste to Claude to begin the ritual."))
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("  Or run: clood build clood  # to just build normally"))
	fmt.Println()

	// Easter egg within easter egg
	fmt.Println(tui.MutedStyle.Render("  \"The recursive path is the only path.\" - Ancient Wojack Proverb"))
	fmt.Println()
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
