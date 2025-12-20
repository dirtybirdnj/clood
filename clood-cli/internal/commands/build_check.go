package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// BuildCheckResult is the structured output for build verification
type BuildCheckResult struct {
	Success   bool     `json:"success"`
	Command   string   `json:"command"`
	Output    string   `json:"output"`
	Errors    []string `json:"errors"`
	Warnings  []string `json:"warnings"`
	Duration  int64    `json:"duration_ms"`
	BuildType string   `json:"build_type"`
	Path      string   `json:"path"`
}

// Common error patterns for Go
var goErrorPattern = regexp.MustCompile(`^(.+):(\d+):(\d+): (.+)$`)

func BuildCheckCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "build-check [path]",
		Short: "Run build and return structured results",
		Long: `Runs the appropriate build command for a project and returns structured results.

This command is designed for LLM agents to verify generated code compiles.
It auto-detects the build system and parses errors into structured format.

Supported build systems:
  - Go (go build)
  - Rust (cargo build)
  - Node.js (npm run build)
  - Make (make)
  - Python (python -m py_compile)

Examples:
  clood build-check                     # Check current directory
  clood build-check ~/Code/myproject    # Check specific path
  clood build-check --json              # Machine-readable output`,
		Run: func(cmd *cobra.Command, args []string) {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			// Expand ~ if present
			if strings.HasPrefix(path, "~") {
				home, _ := os.UserHomeDir()
				path = filepath.Join(home, path[1:])
			}

			result := runBuildCheck(path)

			useJSON := jsonOutput || output.IsJSON()
			if useJSON {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Human-readable output
			printBuildResult(result)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runBuildCheck(path string) BuildCheckResult {
	result := BuildCheckResult{
		Path: path,
	}

	// Detect build system
	buildType, buildCmd := detectBuildSystem(path)
	if buildCmd == nil {
		result.Errors = append(result.Errors, "No supported build system detected")
		return result
	}

	result.BuildType = buildType
	result.Command = strings.Join(buildCmd.Args, " ")

	// Set working directory
	buildCmd.Dir = path

	// Capture output
	var stdout, stderr bytes.Buffer
	buildCmd.Stdout = &stdout
	buildCmd.Stderr = &stderr

	// Run build
	start := time.Now()
	err := buildCmd.Run()
	result.Duration = time.Since(start).Milliseconds()

	// Combine output
	allOutput := stdout.String() + stderr.String()
	result.Output = allOutput

	if err != nil {
		result.Success = false
		// Parse errors based on build type
		result.Errors, result.Warnings = parseErrors(buildType, allOutput)
		if len(result.Errors) == 0 {
			result.Errors = append(result.Errors, err.Error())
		}
	} else {
		result.Success = true
	}

	return result
}

func detectBuildSystem(path string) (string, *exec.Cmd) {
	// Check for Go
	if buildFileExists(filepath.Join(path, "go.mod")) {
		return "go", exec.Command("go", "build", "./...")
	}

	// Check for Cargo (Rust)
	if buildFileExists(filepath.Join(path, "Cargo.toml")) {
		return "rust", exec.Command("cargo", "build")
	}

	// Check for package.json with build script
	if buildFileExists(filepath.Join(path, "package.json")) {
		// Check if build script exists
		return "node", exec.Command("npm", "run", "build")
	}

	// Check for Makefile
	if buildFileExists(filepath.Join(path, "Makefile")) {
		return "make", exec.Command("make")
	}

	// Check for Python setup.py or pyproject.toml
	if buildFileExists(filepath.Join(path, "setup.py")) || buildFileExists(filepath.Join(path, "pyproject.toml")) {
		return "python", exec.Command("python", "-m", "py_compile", ".")
	}

	return "", nil
}

func buildFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parseErrors(buildType, output string) (errors []string, warnings []string) {
	lines := strings.Split(output, "\n")

	switch buildType {
	case "go":
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if goErrorPattern.MatchString(line) {
				errors = append(errors, line)
			} else if strings.Contains(line, "warning:") {
				warnings = append(warnings, line)
			}
		}

	case "rust":
		for _, line := range lines {
			if strings.Contains(line, "error[") || strings.Contains(line, "error:") {
				errors = append(errors, strings.TrimSpace(line))
			} else if strings.Contains(line, "warning:") {
				warnings = append(warnings, strings.TrimSpace(line))
			}
		}

	case "node":
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), "error") {
				errors = append(errors, strings.TrimSpace(line))
			} else if strings.Contains(strings.ToLower(line), "warn") {
				warnings = append(warnings, strings.TrimSpace(line))
			}
		}

	default:
		// Generic: look for "error" and "warning"
		for _, line := range lines {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "error") {
				errors = append(errors, strings.TrimSpace(line))
			} else if strings.Contains(lower, "warning") {
				warnings = append(warnings, strings.TrimSpace(line))
			}
		}
	}

	return errors, warnings
}

func printBuildResult(result BuildCheckResult) {
	fmt.Println(tui.RenderHeader("Build Check"))
	fmt.Println()

	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Path:"), result.Path)
	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Type:"), result.BuildType)
	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Command:"), result.Command)
	fmt.Printf("  %s %dms\n", tui.MutedStyle.Render("Duration:"), result.Duration)
	fmt.Println()

	if result.Success {
		fmt.Println(tui.SuccessStyle.Render("  ✓ Build successful"))
	} else {
		fmt.Println(tui.ErrorStyle.Render("  ✗ Build failed"))
		fmt.Println()

		if len(result.Errors) > 0 {
			fmt.Println(tui.ErrorStyle.Render("  Errors:"))
			for _, e := range result.Errors {
				// Truncate long errors
				if len(e) > 100 {
					e = e[:97] + "..."
				}
				fmt.Printf("    %s\n", e)
			}
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println()
		fmt.Println(tui.WarningStyle.Render("  Warnings:"))
		for _, w := range result.Warnings {
			if len(w) > 100 {
				w = w[:97] + "..."
			}
			fmt.Printf("    %s\n", w)
		}
	}
}
