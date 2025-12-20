package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// AgentConfig holds agent configuration
type AgentConfig struct {
	Model      string
	MaxTurns   int
	Verbose    bool
	Host       string
	SystemMsg  string
}

// AgentResult captures the full agent interaction
type AgentResult struct {
	Turns      []AgentTurn `json:"turns"`
	FinalText  string      `json:"final_text"`
	ToolCalls  int         `json:"tool_calls_total"`
	Success    bool        `json:"success"`
}

// AgentTurn represents one round of the agent loop
type AgentTurn struct {
	UserPrompt   string              `json:"user_prompt,omitempty"`
	ModelReply   string              `json:"model_reply"`
	ToolCalls    []ToolCallResult    `json:"tool_calls,omitempty"`
}

// ToolCallResult captures tool execution
type ToolCallResult struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
	Success   bool   `json:"success"`
}

// Define available tools
var agentTools = []ollama.Tool{
	{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        "execute_shell",
			Description: "Execute a shell command and return output. Use for running programs, checking files, etc.",
			Parameters: ollama.ToolParameters{
				Type: "object",
				Properties: map[string]ollama.ToolProperty{
					"command": {
						Type:        "string",
						Description: "The shell command to execute",
					},
				},
				Required: []string{"command"},
			},
		},
	},
	{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        "draw_bonsai",
			Description: "Draw an ASCII bonsai tree using cbonsai. Returns the art.",
			Parameters: ollama.ToolParameters{
				Type: "object",
				Properties: map[string]ollama.ToolProperty{
					"message": {
						Type:        "string",
						Description: "Optional message to display with the bonsai",
					},
				},
				Required: []string{},
			},
		},
	},
	{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        "read_file",
			Description: "Read the contents of a file",
			Parameters: ollama.ToolParameters{
				Type: "object",
				Properties: map[string]ollama.ToolProperty{
					"path": {
						Type:        "string",
						Description: "Path to the file to read",
					},
				},
				Required: []string{"path"},
			},
		},
	},
	{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        "list_files",
			Description: "List files in a directory",
			Parameters: ollama.ToolParameters{
				Type: "object",
				Properties: map[string]ollama.ToolProperty{
					"path": {
						Type:        "string",
						Description: "Directory path to list",
					},
				},
				Required: []string{"path"},
			},
		},
	},
	{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        "write_file",
			Description: "Write content to a file",
			Parameters: ollama.ToolParameters{
				Type: "object",
				Properties: map[string]ollama.ToolProperty{
					"path": {
						Type:        "string",
						Description: "Path to the file to write",
					},
					"content": {
						Type:        "string",
						Description: "Content to write to the file",
					},
				},
				Required: []string{"path", "content"},
			},
		},
	},
}

func AgentCmd() *cobra.Command {
	var model string
	var maxTurns int
	var verbose bool
	var host string
	var systemPrompt string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "agent [prompt]",
		Short: "Run an agentic loop with tool-calling LLM",
		Long: `Runs an agentic conversation loop with a tool-calling model.

The agent can use tools to interact with the system:
  - execute_shell: Run shell commands
  - draw_bonsai: Draw ASCII art trees
  - read_file: Read file contents
  - list_files: List directory contents
  - write_file: Write to files

The loop continues until the model stops calling tools or max turns is reached.

Recommended models with tool support:
  - llama3-groq-tool-use:8b (optimized for tool calling)
  - qwen2.5-coder:7b (good tool support)
  - mistral:7b (basic tool support)

Examples:
  clood agent "Draw me a bonsai tree"
  clood agent "List all Go files in the current directory"
  clood agent "Read the README and summarize it" --model llama3-groq-tool-use:8b
  clood agent "Create a hello.txt file with a greeting" --verbose`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			prompt := strings.Join(args, " ")

			config := AgentConfig{
				Model:     model,
				MaxTurns:  maxTurns,
				Verbose:   verbose,
				Host:      host,
				SystemMsg: systemPrompt,
			}

			result := runAgent(prompt, config)

			useJSON := jsonOutput || output.IsJSON()
			if useJSON {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Human output - show final result
			printAgentResult(result, verbose)
		},
	}

	cmd.Flags().StringVarP(&model, "model", "m", "llama3-groq-tool-use:8b", "Model to use (should support tool calling)")
	cmd.Flags().IntVar(&maxTurns, "max-turns", 10, "Maximum conversation turns")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed turn-by-turn output")
	cmd.Flags().StringVar(&host, "host", "http://localhost:11434", "Ollama host")
	cmd.Flags().StringVar(&systemPrompt, "system", "", "Custom system prompt")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runAgent(prompt string, config AgentConfig) AgentResult {
	result := AgentResult{
		Turns: []AgentTurn{},
	}

	client := ollama.NewClient(config.Host, 5*time.Minute)

	// Build initial messages
	messages := []ollama.Message{}

	// System message
	systemMsg := config.SystemMsg
	if systemMsg == "" {
		systemMsg = `You are a helpful assistant with access to tools.
Use tools when they would help accomplish the user's request.
After using tools, summarize what you did for the user.
Be concise and efficient.`
	}
	messages = append(messages, ollama.Message{
		Role:    "system",
		Content: systemMsg,
	})

	// User message
	messages = append(messages, ollama.Message{
		Role:    "user",
		Content: prompt,
	})

	// Agent loop
	for turn := 0; turn < config.MaxTurns; turn++ {
		if config.Verbose {
			fmt.Printf("\n%s Turn %d\n", tui.MutedStyle.Render("▸"), turn+1)
		}

		// Call model with tools
		resp, err := client.ChatWithTools(config.Model, messages, agentTools)
		if err != nil {
			result.FinalText = fmt.Sprintf("Error: %v", err)
			return result
		}

		turnResult := AgentTurn{
			ModelReply: resp.Message.Content,
		}

		// Check for tool calls
		if resp.HasToolCalls() {
			toolCalls := resp.GetToolCalls()

			for _, tc := range toolCalls {
				result.ToolCalls++

				if config.Verbose {
					argsJSON, _ := json.Marshal(tc.Function.Arguments)
					fmt.Printf("  %s %s(%s)\n",
						tui.AccentStyle.Render("→"),
						tc.Function.Name,
						string(argsJSON))
				}

				// Execute tool
				toolResult := executeTool(tc.Function.Name, tc.Function.Arguments)

				turnResult.ToolCalls = append(turnResult.ToolCalls, ToolCallResult{
					Name:      tc.Function.Name,
					Arguments: fmt.Sprintf("%v", tc.Function.Arguments),
					Result:    toolResult,
					Success:   !strings.HasPrefix(toolResult, "Error:"),
				})

				if config.Verbose {
					// Show truncated result
					display := toolResult
					if len(display) > 200 {
						display = display[:197] + "..."
					}
					fmt.Printf("  %s %s\n", tui.MutedStyle.Render("←"), display)
				}

				// Add assistant message with tool call
				messages = append(messages, ollama.Message{
					Role:      "assistant",
					Content:   resp.Message.Content,
					ToolCalls: toolCalls,
				})

				// Add tool result
				messages = append(messages, ollama.Message{
					Role:    "tool",
					Content: toolResult,
				})
			}

			result.Turns = append(result.Turns, turnResult)
			// Continue the loop for next turn
			continue
		}

		// No tool calls - model is done
		result.Turns = append(result.Turns, turnResult)
		result.FinalText = resp.Message.Content
		result.Success = true
		break
	}

	if result.FinalText == "" && len(result.Turns) > 0 {
		result.FinalText = result.Turns[len(result.Turns)-1].ModelReply
		result.Success = true
	}

	return result
}

func executeTool(name string, args map[string]interface{}) string {
	switch name {
	case "execute_shell":
		command, ok := args["command"].(string)
		if !ok {
			return "Error: command argument required"
		}
		return executeShell(command)

	case "draw_bonsai":
		message, _ := args["message"].(string)
		return drawBonsai(message)

	case "read_file":
		path, ok := args["path"].(string)
		if !ok {
			return "Error: path argument required"
		}
		return readFileContents(path)

	case "list_files":
		path, ok := args["path"].(string)
		if !ok {
			path = "."
		}
		return listFiles(path)

	case "write_file":
		path, ok := args["path"].(string)
		if !ok {
			return "Error: path argument required"
		}
		content, ok := args["content"].(string)
		if !ok {
			return "Error: content argument required"
		}
		return agentWriteFile(path, content)

	default:
		return fmt.Sprintf("Error: unknown tool %s", name)
	}
}

func executeShell(command string) string {
	// Safety check - block dangerous commands
	dangerous := []string{"rm -rf /", "rm -rf ~", "mkfs", "dd if=", ":(){", "fork bomb"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "Error: command blocked for safety"
		}
	}

	cmd := exec.Command("bash", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if err != nil {
		if output != "" {
			return fmt.Sprintf("Error: %v\nOutput: %s", err, output)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	if output == "" {
		return "(no output)"
	}

	// Truncate very long output
	if len(output) > 4000 {
		output = output[:4000] + "\n... (truncated)"
	}

	return output
}

func drawBonsai(message string) string {
	args := []string{"-p", "-L", "16"}
	if message != "" {
		args = append(args, "-m", message)
	}

	cmd := exec.Command("cbonsai", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: cbonsai failed - %v", err)
	}
	return string(output)
}

func readFileContents(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	content := string(data)
	if len(content) > 8000 {
		content = content[:8000] + "\n... (truncated, file too large)"
	}

	return content
}

func listFiles(path string) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var lines []string
	for _, e := range entries {
		prefix := "  "
		if e.IsDir() {
			prefix = "📁"
		}
		lines = append(lines, fmt.Sprintf("%s %s", prefix, e.Name()))
	}

	return strings.Join(lines, "\n")
}

func agentWriteFile(path, content string) string {
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)
}

func printAgentResult(result AgentResult, verbose bool) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("Agent Result"))
	fmt.Println()

	if result.ToolCalls > 0 {
		fmt.Printf("  %s %d tool call(s)\n", tui.MutedStyle.Render("Tools used:"), result.ToolCalls)

		if verbose && len(result.Turns) > 0 {
			for i, turn := range result.Turns {
				if len(turn.ToolCalls) > 0 {
					fmt.Printf("\n  %s\n", tui.MutedStyle.Render(fmt.Sprintf("Turn %d:", i+1)))
					for _, tc := range turn.ToolCalls {
						status := tui.SuccessStyle.Render("✓")
						if !tc.Success {
							status = tui.ErrorStyle.Render("✗")
						}
						fmt.Printf("    %s %s\n", status, tc.Name)
					}
				}
			}
		}
		fmt.Println()
	}

	if result.FinalText != "" {
		fmt.Println(tui.AccentStyle.Render("  Response:"))
		// Indent the response
		for _, line := range strings.Split(result.FinalText, "\n") {
			fmt.Printf("    %s\n", line)
		}
	}

	fmt.Println()
	if result.Success {
		fmt.Println(tui.SuccessStyle.Render("  ✓ Agent completed successfully"))
	} else {
		fmt.Println(tui.ErrorStyle.Render("  ✗ Agent encountered errors"))
	}
}
