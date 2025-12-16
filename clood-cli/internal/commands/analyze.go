package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/router"
	"github.com/spf13/cobra"
)

// AnalysisResult represents structured analysis output
type AnalysisResult struct {
	File     string   `json:"file"`
	Model    string   `json:"model"`
	Focus    string   `json:"focus,omitempty"`
	Analysis string   `json:"analysis"`
	Issues   []string `json:"issues,omitempty"`
}

func AnalyzeCmd() *cobra.Command {
	var focus string
	var jsonOutput bool
	var fromStdin bool
	var model string

	cmd := &cobra.Command{
		Use:   "analyze [FILE]",
		Short: "Analyze code using reasoning model",
		Long: `Analyze code files using a reasoning model (deepseek-r1, phi4-reasoning).

Unlike code generation, analysis focuses on understanding:
- Finding bugs and edge cases
- Identifying security issues
- Explaining complex logic
- Reviewing for best practices

Examples:
  clood analyze internal/router/router.go
  clood analyze internal/config/ --focus security
  cat file.go | clood analyze --stdin
  git diff | clood analyze --stdin --focus "review changes"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var code string
			var filename string

			// Get code from stdin or file
			if fromStdin {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				code = string(data)
				filename = "stdin"
			} else if len(args) > 0 {
				target := args[0]
				info, err := os.Stat(target)
				if err != nil {
					return fmt.Errorf("stat %s: %w", target, err)
				}

				if info.IsDir() {
					// Analyze directory - concatenate all Go files
					code, filename, err = readDirectory(target)
					if err != nil {
						return err
					}
				} else {
					// Single file
					data, err := os.ReadFile(target)
					if err != nil {
						return fmt.Errorf("reading %s: %w", target, err)
					}
					code = string(data)
					filename = target
				}
			} else {
				return fmt.Errorf("provide a file path or use --stdin")
			}

			if code == "" {
				return fmt.Errorf("no code to analyze")
			}

			// Load config and setup client
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Use analysis tier by default
			r := router.NewRouter(cfg)
			result, err := r.Route("analyze", router.TierAnalysis, model)
			if err != nil {
				return fmt.Errorf("routing: %w", err)
			}

			if result.Client == nil {
				return fmt.Errorf("no available host with analysis model")
			}

			// Build analysis prompt
			prompt := buildAnalysisPrompt(code, filename, focus)

			// Execute analysis with tuned options
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "Analyzing %s with %s (tuned: num_ctx=16384, num_predict=4096)...\n", filename, result.Model)
			}

			opts := ollama.DefaultAnalysisOptions()
			response, err := result.Client.GenerateWithOptions(result.Model, prompt, opts)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			// Output results
			if jsonOutput {
				output := AnalysisResult{
					File:     filename,
					Model:    result.Model,
					Focus:    focus,
					Analysis: response.Response,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Println(response.Response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&focus, "focus", "f", "", "Focus area (security, performance, bugs, style)")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read code from stdin")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Override model (default: analysis tier)")

	return cmd
}

func readDirectory(dir string) (string, string, error) {
	var builder strings.Builder
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only Go files for now
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(dir, path)
		builder.WriteString(fmt.Sprintf("// === %s ===\n", relPath))
		builder.Write(data)
		builder.WriteString("\n\n")
		files = append(files, relPath)

		return nil
	})

	if err != nil {
		return "", "", err
	}

	filename := fmt.Sprintf("%s (%d files)", dir, len(files))
	return builder.String(), filename, nil
}

func buildAnalysisPrompt(code, filename, focus string) string {
	var sb strings.Builder

	sb.WriteString("You are a code reviewer analyzing code for issues and improvements.\n\n")

	if focus != "" {
		sb.WriteString(fmt.Sprintf("Focus your analysis on: %s\n\n", focus))
	}

	sb.WriteString("Analyze the following code and provide:\n")
	sb.WriteString("1. A brief summary of what the code does\n")
	sb.WriteString("2. Potential bugs or edge cases\n")
	sb.WriteString("3. Security concerns (if any)\n")
	sb.WriteString("4. Suggestions for improvement\n\n")

	sb.WriteString("Be specific - reference line numbers or function names.\n")
	sb.WriteString("If the code looks good, say so briefly.\n\n")

	sb.WriteString(fmt.Sprintf("File: %s\n", filename))
	sb.WriteString("```\n")
	sb.WriteString(code)
	sb.WriteString("\n```\n")

	return sb.String()
}

// AnalyzeStdin is a helper for piping - reads from stdin and returns analysis
func AnalyzeStdin(focus string) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var code strings.Builder
	for scanner.Scan() {
		code.WriteString(scanner.Text())
		code.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	// This would need to be called with proper config setup
	// For now, return the prompt that would be sent
	return buildAnalysisPrompt(code.String(), "stdin", focus), nil
}
