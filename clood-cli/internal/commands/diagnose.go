package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// DiagnoseOutput is the JSON output structure
type DiagnoseOutput struct {
	Timestamp    string           `json:"timestamp"`
	LinesRead    int              `json:"lines_read"`
	Analysis     string           `json:"analysis"`
	Model        string           `json:"model"`
	Duration     float64          `json:"duration_sec"`
	ErrorCount   int              `json:"error_count"`
	WarningCount int              `json:"warning_count"`
	Patterns     []PatternCluster `json:"patterns,omitempty"`
}

// PatternCluster represents a group of similar log entries
type PatternCluster struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
	Sample  string `json:"sample"`
}

func DiagnoseCmd() *cobra.Command {
	var model string
	var errorsOnly bool
	var maxLines int
	var live bool

	cmd := &cobra.Command{
		Use:   "diagnose [file]",
		Short: "AI-powered log analysis and pattern detection",
		Long: `Analyze logs with AI to detect patterns, cluster errors, and suggest fixes.

Pipe logs in or specify a file. The AI will:
- Cluster similar errors
- Detect causal chains
- Suggest root causes and fixes

Examples:
  # Analyze a log file
  clood diagnose /var/log/app.log

  # Pipe from tail
  tail -1000 /var/log/app.log | clood diagnose

  # Focus on errors only
  clood diagnose app.log --errors-only

  # Use a specific model
  clood diagnose app.log --model llama3.1:8b

  # Limit lines analyzed
  clood diagnose app.log --max-lines 500`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var reader io.Reader

			// Determine input source
			if len(args) > 0 {
				file, err := os.Open(args[0])
				if err != nil {
					return fmt.Errorf("opening file: %w", err)
				}
				defer file.Close()
				reader = file
			} else {
				// Check if stdin has data
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return fmt.Errorf("no input: provide a file or pipe logs via stdin")
				}
				reader = os.Stdin
			}

			// Read log lines
			var lines []string
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				if errorsOnly {
					lower := strings.ToLower(line)
					if !strings.Contains(lower, "error") &&
						!strings.Contains(lower, "fail") &&
						!strings.Contains(lower, "exception") &&
						!strings.Contains(lower, "panic") {
						continue
					}
				}
				lines = append(lines, line)
				if maxLines > 0 && len(lines) >= maxLines {
					break
				}
			}

			if len(lines) == 0 {
				if !output.JSONMode {
					fmt.Println(tui.MutedStyle.Render("No log lines to analyze"))
				}
				return nil
			}

			// Count errors and warnings for stats
			errorCount := 0
			warningCount := 0
			for _, line := range lines {
				lower := strings.ToLower(line)
				if strings.Contains(lower, "error") || strings.Contains(lower, "exception") || strings.Contains(lower, "fail") {
					errorCount++
				}
				if strings.Contains(lower, "warn") {
					warningCount++
				}
			}

			if !output.JSONMode {
				fmt.Println(tui.RenderHeader("DIAGNOSE"))
				fmt.Println()
				fmt.Printf("  %s %d lines\n", tui.MutedStyle.Render("Analyzing:"), len(lines))
				fmt.Printf("  %s %d errors, %d warnings detected\n",
					tui.MutedStyle.Render("Found:"),
					errorCount, warningCount)
				fmt.Println()
			}

			// Find an online host
			mgr := hosts.NewManager()
			mgr.AddHosts(hosts.DefaultHosts())
			statuses := mgr.CheckAllHosts()

			var onlineHost *hosts.HostStatus
			var selectedModel string

			for _, s := range statuses {
				if s.Online && len(s.Models) > 0 {
					onlineHost = s
					if model != "" {
						selectedModel = model
					} else {
						// Pick the first model
						selectedModel = s.Models[0].Name
					}
					break
				}
			}

			if onlineHost == nil {
				return fmt.Errorf("no online Ollama hosts found")
			}

			if !output.JSONMode {
				fmt.Printf("  %s %s on %s\n",
					tui.AccentStyle.Render(""),
					selectedModel,
					onlineHost.Host.Name)
				fmt.Println()
			}

			// Build the analysis prompt
			prompt := buildDiagnosePrompt(lines, errorCount, warningCount)

			// Query the model
			client := ollama.NewClient(onlineHost.Host.URL, 5*time.Minute)
			startTime := time.Now()

			if !output.JSONMode {
				fmt.Print(tui.MutedStyle.Render("  Analyzing..."))
			}

			resp, err := client.Generate(selectedModel, prompt)
			duration := time.Since(startTime)

			if !output.JSONMode {
				fmt.Print("\r                    \r")
			}

			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			// Output results
			diagOutput := DiagnoseOutput{
				Timestamp:    time.Now().Format(time.RFC3339),
				LinesRead:    len(lines),
				Analysis:     resp.Response,
				Model:        selectedModel,
				Duration:     duration.Seconds(),
				ErrorCount:   errorCount,
				WarningCount: warningCount,
			}

			if output.JSONMode {
				data, _ := json.MarshalIndent(diagOutput, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			// Pretty print the analysis
			fmt.Println(tui.RenderHeader("ANALYSIS"))
			fmt.Println()
			fmt.Println(resp.Response)
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("  Analyzed by %s in %.1fs", selectedModel, duration.Seconds())))

			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Model to use for analysis")
	cmd.Flags().BoolVar(&errorsOnly, "errors-only", false, "Only analyze error lines")
	cmd.Flags().IntVar(&maxLines, "max-lines", 1000, "Maximum lines to analyze")
	cmd.Flags().BoolVar(&live, "live", false, "Live streaming analysis (experimental)")

	return cmd
}

func buildDiagnosePrompt(lines []string, errorCount, warningCount int) string {
	// Take a sample of lines (first 100, last 100, and some from middle)
	sampleLines := sampleLogLines(lines, 300)

	return fmt.Sprintf(`You are a senior DevOps engineer analyzing application logs.

LOG STATISTICS:
- Total lines: %d
- Errors detected: %d
- Warnings detected: %d

LOG SAMPLE (representative lines):
%s

Provide a concise analysis:

1. **Error Clusters** - Group similar errors, show count and pattern
2. **Root Cause** - What's the likely root cause?
3. **Causal Chain** - If errors seem related, show the chain (A caused B caused C)
4. **Suggested Fixes** - Actionable steps to resolve

Keep the response focused and actionable. Use bullet points.`,
		len(lines), errorCount, warningCount,
		strings.Join(sampleLines, "\n"))
}

func sampleLogLines(lines []string, maxSample int) []string {
	if len(lines) <= maxSample {
		return lines
	}

	// Take first third, middle third sample, last third
	third := maxSample / 3
	sample := make([]string, 0, maxSample)

	// First portion
	for i := 0; i < third && i < len(lines); i++ {
		sample = append(sample, lines[i])
	}

	// Middle portion
	midStart := len(lines)/2 - third/2
	for i := 0; i < third && midStart+i < len(lines); i++ {
		sample = append(sample, lines[midStart+i])
	}

	// Last portion
	lastStart := len(lines) - third
	if lastStart < 0 {
		lastStart = 0
	}
	for i := lastStart; i < len(lines); i++ {
		sample = append(sample, lines[i])
	}

	return sample
}
