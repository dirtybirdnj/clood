package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// RunResult captures the output of a run command for structured output
type RunResult struct {
	Host     string `json:"host"`
	Model    string `json:"model"`
	Response string `json:"response"`
	Tokens   int    `json:"tokens,omitempty"`
	Duration int64  `json:"duration_ms,omitempty"`
	Error    string `json:"error,omitempty"`
}

func RunCmd() *cobra.Command {
	var hostName string
	var model string
	var systemPrompt string
	var systemFile string
	var promptFile string
	var outputJSON bool
	var quiet bool
	var noStream bool

	cmd := &cobra.Command{
		Use:   "run [prompt]",
		Short: "Run a prompt on a specific host (for agent delegation)",
		Long: `Execute a prompt on a specific Ollama host with full control.

Unlike 'ask' which auto-routes, 'run' gives explicit control over:
  - Which host executes the prompt
  - Which model to use
  - System prompt / role definition
  - Output format (streaming, JSON, quiet)

Designed for agent delegation workflows where you need deterministic
routing and structured output.

Examples:
  # Run on ubuntu25 with a specific model
  clood run --host ubuntu25 --model llama3.1:8b "explain this code"

  # Use a system prompt to define agent role
  clood run --host ubuntu25 --system "You are a code reviewer" "review this function"

  # Load system prompt from file (agent role definition)
  clood run --host ubuntu25 --system-file roles/reviewer.md "review this"

  # Pipe input and get JSON output for parsing
  cat file.go | clood run --host ubuntu25 --prompt-file - --json

  # Quiet mode for scripts (only output response)
  clood run --host ubuntu25 -q "summarize"`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				outputError(outputJSON, quiet, "Error loading config: "+err.Error())
				return
			}

			// Get the prompt
			prompt, err := getPrompt(args, promptFile)
			if err != nil {
				outputError(outputJSON, quiet, err.Error())
				return
			}

			if prompt == "" {
				outputError(outputJSON, quiet, "No prompt provided")
				return
			}

			// Get system prompt if specified
			system, err := getSystemPrompt(systemPrompt, systemFile)
			if err != nil {
				outputError(outputJSON, quiet, err.Error())
				return
			}

			// Find the host
			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			// If no host specified, pick best available
			if hostName == "" {
				best := mgr.GetBestHost()
				if best == nil {
					outputError(outputJSON, quiet, "No hosts available")
					return
				}
				hostName = best.Host.Name
			}

			host := mgr.GetHost(hostName)
			if host == nil {
				outputError(outputJSON, quiet, "Host not found: "+hostName)
				return
			}

			status := mgr.CheckHost(host)
			if !status.Online {
				outputError(outputJSON, quiet, "Host is offline: "+hostName)
				return
			}

			client := mgr.GetClient(hostName)

			// Select model - use specified, or default from tier config
			if model == "" {
				model = cfg.Tiers.Fast.Model // Default to fast tier model
			}

			// Verify model is available on host
			if !hostHasModel(status, model) {
				// Try to find a suitable model on this host
				if len(status.Models) > 0 {
					model = status.Models[0].Name
					if !quiet && !outputJSON {
						fmt.Fprintf(os.Stderr, "%s Using available model: %s\n",
							tui.MutedStyle.Render("Note:"), model)
					}
				} else {
					outputError(outputJSON, quiet, "No models available on "+hostName)
					return
				}
			}

			// Print header unless quiet or JSON
			if !quiet && !outputJSON {
				fmt.Printf("%s %s on %s\n",
					tui.MutedStyle.Render("Running:"),
					model,
					hostName)
				fmt.Println()
			}

			// Execute
			result := RunResult{
				Host:  hostName,
				Model: model,
			}

			if noStream || outputJSON {
				// Blocking execution
				resp, err := client.GenerateWithSystem(model, system, prompt)
				if err != nil {
					result.Error = err.Error()
					outputResult(result, outputJSON, quiet)
					return
				}
				result.Response = resp.Response
				result.Tokens = resp.EvalCount
				result.Duration = resp.TotalDuration / 1_000_000 // Convert to ms
				outputResult(result, outputJSON, quiet)
			} else {
				// Streaming execution
				var response strings.Builder
				resp, err := client.GenerateStreamWithSystem(model, system, prompt, func(chunk ollama.GenerateResponse) {
					fmt.Print(chunk.Response)
					response.WriteString(chunk.Response)
				})

				if err != nil {
					fmt.Fprintln(os.Stderr)
					outputError(outputJSON, quiet, err.Error())
					return
				}
				fmt.Println() // Final newline

				// Show stats if not quiet
				if !quiet {
					fmt.Println()
					fmt.Printf("%s %d tokens in %dms\n",
						tui.MutedStyle.Render("Stats:"),
						resp.EvalCount,
						resp.TotalDuration/1_000_000)
				}
			}
		},
	}

	cmd.Flags().StringVarP(&hostName, "host", "H", "", "Target host (required for deterministic routing)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model to use (default: first available)")
	cmd.Flags().StringVarP(&systemPrompt, "system", "s", "", "System prompt (agent role definition)")
	cmd.Flags().StringVar(&systemFile, "system-file", "", "Load system prompt from file")
	cmd.Flags().StringVarP(&promptFile, "prompt-file", "f", "", "Load prompt from file (use - for stdin)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output result as JSON")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode - only output response")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")

	return cmd
}

func getPrompt(args []string, promptFile string) (string, error) {
	// If prompt file specified, read from it
	if promptFile != "" {
		if promptFile == "-" {
			// Read from stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", fmt.Errorf("error reading stdin: %w", err)
			}
			return string(data), nil
		}
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("error reading prompt file: %w", err)
		}
		return string(data), nil
	}

	// Otherwise use args
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	return "", nil
}

func getSystemPrompt(systemPrompt, systemFile string) (string, error) {
	if systemFile != "" {
		data, err := os.ReadFile(systemFile)
		if err != nil {
			return "", fmt.Errorf("error reading system file: %w", err)
		}
		return string(data), nil
	}
	return systemPrompt, nil
}

func hostHasModel(status *hosts.HostStatus, model string) bool {
	for _, m := range status.Models {
		if m.Name == model {
			return true
		}
	}
	return false
}

func outputError(outputJSON, quiet bool, msg string) {
	if outputJSON {
		result := RunResult{Error: msg}
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
	} else if !quiet {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+msg))
	}
}

func outputResult(result RunResult, outputJSON, quiet bool) {
	if outputJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else if quiet {
		fmt.Print(result.Response)
	} else {
		fmt.Print(result.Response)
		if result.Tokens > 0 {
			fmt.Println()
			fmt.Printf("%s %d tokens in %dms\n",
				tui.MutedStyle.Render("Stats:"),
				result.Tokens,
				result.Duration)
		}
	}
}
