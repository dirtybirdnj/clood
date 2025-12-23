package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// BatchInput represents a single input for batch processing
type BatchInput struct {
	ID     string `json:"id,omitempty"`
	Prompt string `json:"prompt"`
}

// BatchResult represents a single output from batch processing
type BatchResult struct {
	ID       string  `json:"id,omitempty"`
	Prompt   string  `json:"prompt"`
	Response string  `json:"response"`
	Model    string  `json:"model"`
	Duration float64 `json:"duration_sec"`
	Tokens   int     `json:"tokens"`
	Error    string  `json:"error,omitempty"`
}

// BatchOutput is the full batch processing output
type BatchOutput struct {
	Timestamp  string        `json:"timestamp"`
	Model      string        `json:"model"`
	Host       string        `json:"host"`
	Results    []BatchResult `json:"results"`
	Summary    BatchSummary  `json:"summary"`
	OutputFile string        `json:"output_file,omitempty"`
}

// BatchSummary provides aggregate stats
type BatchSummary struct {
	Total       int     `json:"total"`
	Successful  int     `json:"successful"`
	Failed      int     `json:"failed"`
	TotalTime   float64 `json:"total_time_sec"`
	TotalTokens int     `json:"total_tokens"`
	AvgTime     float64 `json:"avg_time_sec"`
}

func BatchCmd() *cobra.Command {
	var inputFile string
	var outputFile string
	var prompt string
	var model string
	var parallel int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Process multiple prompts through LLMs",
		Long: `Run batch processing of prompts through local models.

Input can be:
- A JSONL file with {"prompt": "..."} per line
- A text file with one prompt per line
- Piped input

Examples:
  # Process prompts from file
  clood batch --input prompts.jsonl --output results.jsonl

  # Apply same prompt to each line of a text file
  clood batch --input items.txt --prompt "Summarize: {input}" --output summaries.jsonl

  # Process piped input
  cat prompts.txt | clood batch --output results.jsonl

  # Use specific model
  clood batch --input prompts.jsonl --model llama3.1:8b --output results.jsonl

  # Dry run to check setup
  clood batch --input prompts.jsonl --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var inputs []BatchInput

			// Determine input source
			if inputFile != "" {
				var err error
				inputs, err = readBatchInputs(inputFile, prompt)
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
			} else {
				// Check stdin
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return fmt.Errorf("no input: use --input or pipe data")
				}
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" {
						continue
					}
					inputs = append(inputs, BatchInput{
						ID:     fmt.Sprintf("%d", len(inputs)+1),
						Prompt: applyTemplate(prompt, line),
					})
				}
			}

			if len(inputs) == 0 {
				return fmt.Errorf("no inputs to process")
			}

			if !output.JSONMode {
				fmt.Println(tui.RenderHeader("BATCH"))
				fmt.Println()
				fmt.Printf("  %s %d inputs\n", tui.MutedStyle.Render("Processing:"), len(inputs))
			}

			if dryRun {
				if !output.JSONMode {
					fmt.Println(tui.ErrorStyle.Render("\n  DRY RUN - showing first 3 inputs:\n"))
					for i := 0; i < 3 && i < len(inputs); i++ {
						fmt.Printf("  [%s] %s\n", inputs[i].ID, truncateTitle(inputs[i].Prompt, 60))
					}
				}
				return nil
			}

			// Find online host
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
						selectedModel = s.Models[0].Name
					}
					break
				}
			}

			if onlineHost == nil {
				return fmt.Errorf("no online Ollama hosts found")
			}

			if !output.JSONMode {
				fmt.Printf("  %s %s on %s\n\n",
					tui.AccentStyle.Render(""),
					selectedModel,
					onlineHost.Host.Name)
			}

			// Process inputs
			client := ollama.NewClient(onlineHost.Host.URL, 5*time.Minute)
			startTime := time.Now()
			var results []BatchResult
			summary := BatchSummary{Total: len(inputs)}

			for i, input := range inputs {
				if !output.JSONMode {
					fmt.Printf("\r  Processing %d/%d...", i+1, len(inputs))
				}

				result := BatchResult{
					ID:     input.ID,
					Prompt: input.Prompt,
					Model:  selectedModel,
				}

				itemStart := time.Now()
				resp, err := client.Generate(selectedModel, input.Prompt)
				result.Duration = time.Since(itemStart).Seconds()

				if err != nil {
					result.Error = err.Error()
					summary.Failed++
				} else {
					result.Response = resp.Response
					result.Tokens = resp.EvalCount
					summary.Successful++
					summary.TotalTokens += resp.EvalCount
				}

				results = append(results, result)
			}

			summary.TotalTime = time.Since(startTime).Seconds()
			if summary.Successful > 0 {
				summary.AvgTime = summary.TotalTime / float64(summary.Successful)
			}

			if !output.JSONMode {
				fmt.Print("\r                              \r")
			}

			// Build output
			batchOutput := BatchOutput{
				Timestamp:  time.Now().Format(time.RFC3339),
				Model:      selectedModel,
				Host:       onlineHost.Host.Name,
				Results:    results,
				Summary:    summary,
				OutputFile: outputFile,
			}

			// Write output file if specified
			if outputFile != "" {
				file, err := os.Create(outputFile)
				if err != nil {
					return fmt.Errorf("creating output file: %w", err)
				}
				defer file.Close()

				encoder := json.NewEncoder(file)
				for _, r := range results {
					encoder.Encode(r)
				}

				if !output.JSONMode {
					fmt.Printf("  %s Wrote %d results to %s\n",
						tui.SuccessStyle.Render(""),
						len(results),
						outputFile)
				}
			}

			if output.JSONMode {
				data, _ := json.MarshalIndent(batchOutput, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			// Summary
			fmt.Println()
			fmt.Println(tui.RenderHeader("SUMMARY"))
			fmt.Println()
			fmt.Printf("  %s %d processed, %d successful, %d failed\n",
				tui.SuccessStyle.Render(""),
				summary.Total,
				summary.Successful,
				summary.Failed)
			fmt.Printf("  %s %.1fs total, %.1fs avg per item\n",
				tui.MutedStyle.Render("Time:"),
				summary.TotalTime,
				summary.AvgTime)
			fmt.Printf("  %s %d total\n",
				tui.MutedStyle.Render("Tokens:"),
				summary.TotalTokens)

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file (JSONL or text)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (JSONL)")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Prompt template (use {input} for placeholder)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model to use")
	cmd.Flags().IntVar(&parallel, "parallel", 1, "Parallel workers (future)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show inputs without processing")

	return cmd
}

func readBatchInputs(filename, promptTemplate string) ([]BatchInput, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var inputs []BatchInput
	scanner := bufio.NewScanner(file)
	lineNum := 0

	ext := strings.ToLower(filepath.Ext(filename))
	isJSONL := ext == ".jsonl" || ext == ".json"

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var input BatchInput
		input.ID = fmt.Sprintf("%d", lineNum)

		if isJSONL {
			// Try to parse as JSON
			if err := json.Unmarshal([]byte(line), &input); err != nil {
				// Fall back to treating as plain text
				input.Prompt = applyTemplate(promptTemplate, line)
			}
		} else {
			// Plain text file - use template if provided
			input.Prompt = applyTemplate(promptTemplate, line)
		}

		if input.Prompt != "" {
			inputs = append(inputs, input)
		}
	}

	return inputs, scanner.Err()
}

func applyTemplate(template, input string) string {
	if template == "" {
		return input
	}
	if strings.Contains(template, "{input}") {
		return strings.ReplaceAll(template, "{input}", input)
	}
	return template + " " + input
}
