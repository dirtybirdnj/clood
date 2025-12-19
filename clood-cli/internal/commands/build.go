package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func BuildCmd() *cobra.Command {
	var skipPull bool
	var outputPath string

	cmd := &cobra.Command{
		Use:   "build [target]",
		Short: "Build clood from source",
		Long: `Yo dawg, I heard you like clood, so we put a build in your clood
so you can build while you clood.

  clood build clood    # git pull && go build -o ~/bin/clood

The spirit of Xzibit is pleased.`,
		Example: `  clood build clood           # Pull latest and build to ~/bin/clood
  clood build clood --skip-pull  # Just build, no git pull
  clood build clood -o /usr/local/bin/clood  # Custom output path`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := "clood"
			if len(args) > 0 {
				target = args[0]
			}

			if target != "clood" {
				fmt.Println(tui.ErrorStyle.Render(fmt.Sprintf("Unknown target: %s", target)))
				fmt.Println(tui.MutedStyle.Render("Currently only 'clood build clood' is supported"))
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
				outputPath = filepath.Join(homeDir, "bin", "clood")
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
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Run:"), outputPath)
		},
	}

	cmd.Flags().BoolVar(&skipPull, "skip-pull", false, "Skip git pull, just build")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path (default ~/bin/clood)")

	return cmd
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

	// Try common locations
	homeDir, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(homeDir, "Code", "clood", "clood-cli"),
		filepath.Join(homeDir, "code", "clood", "clood-cli"),
		filepath.Join(homeDir, "src", "clood", "clood-cli"),
		"/Users/mgilbert/Code/clood/clood-cli",
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
