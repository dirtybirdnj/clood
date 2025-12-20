package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/agents"
	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// DelegateResult captures the full result of a delegation
type DelegateResult struct {
	Agent      string   `json:"agent"`
	Model      string   `json:"model"`
	Host       string   `json:"host"`
	Task       string   `json:"task"`
	Files      []string `json:"files,omitempty"`
	Response   string   `json:"response"`
	Tokens     int      `json:"tokens,omitempty"`
	DurationMs int64    `json:"duration_ms"`
	Error      string   `json:"error,omitempty"`
}

func DelegateCmd() *cobra.Command {
	var agentName string
	var hostName string
	var model string
	var files []string
	var outputJSON bool
	var quiet bool
	var formatOutput bool
	var taskType string

	cmd := &cobra.Command{
		Use:   "delegate [task]",
		Short: "Delegate a task to a remote agent",
		Long: `Send a task to a remote LLM agent and get structured results.

Unlike 'run' which executes raw prompts, 'delegate' is task-oriented:
  - Uses preconfigured agent roles
  - Automatically includes file context
  - Produces formatted, structured output
  - Shows metadata (agent, model, host, time)

Examples:
  # Delegate a code review
  clood delegate --agent reviewer "Review internal/router/router.go"

  # Include specific files as context
  clood delegate --agent coder --file broken.go "Fix the bug in this file"

  # Delegate to any host without specifying agent
  clood delegate --host ubuntu25 "Summarize the codebase structure"

  # Get JSON output for scripts
  clood delegate --agent reviewer --json "Review internal/"`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			start := time.Now()

			task := strings.Join(args, " ")

			cfg, err := config.Load()
			if err != nil {
				outputDelegateError(outputJSON, quiet, "Error loading config: "+err.Error())
				return
			}

			result := DelegateResult{
				Task: task,
			}

			// Load agent configuration
			var agent *agents.Agent
			var system string

			if agentName != "" {
				agentCfg := agents.LoadConfigWithFallback()
				agent = agentCfg.GetAgent(agentName)
				if agent == nil {
					outputDelegateError(outputJSON, quiet, "Agent not found: "+agentName+"\nRun 'clood agents' to list available agents")
					return
				}
				result.Agent = agentName

				// Apply agent settings
				if hostName == "" && agent.Host != "" {
					hostName = agent.Host
				}
				if model == "" && agent.Model != "" {
					model = agent.Model
				}
				system = agent.System
			} else {
				result.Agent = "default"
			}

			// Setup host manager
			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			// Find host
			if hostName == "" {
				best := mgr.GetBestHost()
				if best == nil {
					outputDelegateError(outputJSON, quiet, "No hosts available")
					return
				}
				hostName = best.Host.Name
			}

			host := mgr.GetHost(hostName)
			if host == nil {
				outputDelegateError(outputJSON, quiet, "Host not found: "+hostName)
				return
			}

			status := mgr.CheckHost(host)
			if !status.Online {
				outputDelegateError(outputJSON, quiet, "Host is offline: "+hostName)
				return
			}

			result.Host = hostName
			client := mgr.GetClient(hostName)

			// Select model
			if model == "" {
				model = cfg.Tiers.Fast.Model
			}

			// Verify model or use first available
			if !hostHasModel(status, model) {
				if len(status.Models) > 0 {
					model = status.Models[0].Name
				} else {
					outputDelegateError(outputJSON, quiet, "No models available on "+hostName)
					return
				}
			}
			result.Model = model

			// Build prompt with file context
			prompt := buildDelegatePrompt(task, files)
			result.Files = files

			// Show progress unless quiet/JSON
			if !quiet && !outputJSON {
				agentDisplay := result.Agent
				if agentDisplay == "default" {
					agentDisplay = "general"
				}
				fmt.Printf("%s Delegating to %s (%s on %s)...\n\n",
					tui.MutedStyle.Render("🔍"),
					agentDisplay,
					model,
					hostName)
			}

			// Execute
			resp, err := client.GenerateWithSystem(model, system, prompt)
			if err != nil {
				result.Error = err.Error()
				result.DurationMs = time.Since(start).Milliseconds()
				outputDelegateResult(result, outputJSON, quiet, false, taskType)
				return
			}

			result.Response = resp.Response
			result.Tokens = resp.EvalCount
			result.DurationMs = time.Since(start).Milliseconds()

			outputDelegateResult(result, outputJSON, quiet, formatOutput, taskType)
		},
	}

	cmd.Flags().StringVarP(&agentName, "agent", "a", "", "Agent role to use (see 'clood agents')")
	cmd.Flags().StringVarP(&hostName, "host", "H", "", "Target host (overrides agent config)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model to use (overrides agent config)")
	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "Include file(s) as context")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output result as JSON")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode - only output response")
	cmd.Flags().BoolVar(&formatOutput, "format", false, "Parse response into structured output with issues/actions")
	cmd.Flags().StringVar(&taskType, "task-type", "review", "Task type for formatting: review, generate, document, analyze")

	return cmd
}

func buildDelegatePrompt(task string, files []string) string {
	var sb strings.Builder

	// Add file contents if specified
	if len(files) > 0 {
		sb.WriteString("Context files:\n\n")
		for _, file := range files {
			content, err := readFileContent(file)
			if err != nil {
				sb.WriteString(fmt.Sprintf("--- %s ---\n(Error reading: %s)\n\n", file, err))
				continue
			}
			sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", file, content))
		}
	}

	sb.WriteString("Task: ")
	sb.WriteString(task)

	return sb.String()
}

func readFileContent(path string) (string, error) {
	// Handle glob patterns
	if strings.Contains(path, "*") {
		matches, err := filepath.Glob(path)
		if err != nil {
			return "", err
		}
		var contents []string
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			contents = append(contents, fmt.Sprintf("--- %s ---\n%s", match, string(data)))
		}
		return strings.Join(contents, "\n\n"), nil
	}

	// Single file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func outputDelegateError(outputJSON, quiet bool, msg string) {
	if outputJSON {
		result := DelegateResult{Error: msg}
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
	} else if !quiet {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+msg))
	}
}

func outputDelegateResult(result DelegateResult, outputJSON, quiet, formatOutput bool, taskType string) {
	// Handle structured output format
	if formatOutput && result.Error == "" {
		parsed := output.ParseAgentResponse(result.Response, taskType)
		parsed.Agent = result.Agent
		parsed.Model = result.Model
		parsed.Host = result.Host
		parsed.DurationMs = result.DurationMs
		parsed.Tokens = result.Tokens
		if len(result.Files) > 0 {
			parsed.File = result.Files[0]
			parsed.Files = result.Files
		}

		if outputJSON {
			// Full structured JSON output (Strata-compatible)
			data, _ := json.MarshalIndent(parsed, "", "  ")
			fmt.Println(string(data))
		} else if !quiet {
			// Formatted terminal output
			fmt.Println(output.FormatAgentResult(parsed))
		} else {
			// Quiet mode - just summary
			if parsed.Summary != "" {
				fmt.Println(parsed.Summary)
			}
		}
		return
	}

	if outputJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	if result.Error != "" {
		if !quiet {
			fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+result.Error))
		}
		return
	}

	if quiet {
		fmt.Print(result.Response)
		return
	}

	// Formatted output
	fmt.Println(tui.RenderHeader("Results"))
	fmt.Println()
	fmt.Println(result.Response)
	fmt.Println()

	// Metadata footer
	fmt.Println(tui.MutedStyle.Render("───────────────────────────────────────"))
	agentDisplay := result.Agent
	if agentDisplay == "default" {
		agentDisplay = "general"
	}
	fmt.Printf("%s %s | %s %s | %s %s | %s %.1fs\n",
		tui.MutedStyle.Render("Agent:"),
		agentDisplay,
		tui.MutedStyle.Render("Model:"),
		result.Model,
		tui.MutedStyle.Render("Host:"),
		result.Host,
		tui.MutedStyle.Render("Time:"),
		float64(result.DurationMs)/1000)
}
