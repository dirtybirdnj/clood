package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/router"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func AskCmd() *cobra.Command {
	var forceTier int
	var forceModel string
	var forceHost string
	var noStream bool
	var noContext bool
	var showRoute bool
	var verbose bool
	var jsonOutput bool
	var stdinMode bool

	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask a question (auto-routes to appropriate tier/host)",
		Long: `Ask a question and clood will:
  1. Classify the query complexity (fast vs deep)
  2. Select the appropriate model tier
  3. Route to the best available host
  4. Stream the response

Use --show-route to see routing decisions without executing.
Use --stdin to read additional content from stdin (for piping).

Examples:
  clood ask "What is Go?"
  cat issue.txt | clood ask "Rate this issue" --stdin
  gh issue view 42 --json body | clood ask "Scope this" --stdin`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			question := strings.Join(args, " ")

			// Read from stdin if flag is set
			if stdinMode {
				stdinContent, err := readStdin()
				if err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error reading stdin: "+err.Error()))
					return
				}
				if stdinContent != "" {
					question = fmt.Sprintf("%s\n\n---\n%s", question, stdinContent)
				}
			}

			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error loading config: "+err.Error()))
				return
			}

			// Create router
			r := router.NewRouter(cfg)

			// Route the query
			result, err := r.Route(question, forceTier, forceModel)
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Routing error: "+err.Error()))
				return
			}

			// Override host if specified
			if forceHost != "" {
				mgr := r.GetManager()
				host := mgr.GetHost(forceHost)
				if host == nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Host not found: "+forceHost))
					return
				}
				status := mgr.CheckHost(host)
				if !status.Online {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Host is offline: "+forceHost))
					return
				}
				result.Host = status
				result.Client = mgr.GetClient(forceHost)
			}

			// Show routing info only (exit without executing)
			if showRoute {
				printRouteInfo(result)
				return
			}

			// Show routing info with verbose (continue to execute)
			if verbose {
				printRouteInfo(result)
				fmt.Println()
			}

			// Build prompt with context (needed for both JSON and normal output)
			prompt := question
			if !noContext {
				context := getProjectContext()
				if context != "" {
					prompt = fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context, question)
				}
			}

			// JSON output mode - clean machine-readable output, no TUI
			if jsonOutput {
				executeJSON(result, prompt)
				return
			}

			// Check we have a host
			if result.Host == nil || result.Client == nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("No available host found"))
				fmt.Fprintln(os.Stderr, tui.MutedStyle.Render("Run 'clood hosts' to check host status"))
				return
			}

			// Print routing header
			fmt.Println(tui.RenderTier(result.Tier))
			fmt.Printf("%s %s on %s\n",
				tui.MutedStyle.Render("Model:"),
				result.Model,
				result.Host.Host.Name)
			fmt.Println()

			// Execute query
			stream := cfg.Defaults.Stream && !noStream

			if stream {
				executeStreaming(result.Client, result.Model, prompt)
			} else {
				executeBlocking(result.Client, result.Model, prompt)
			}
		},
	}

	cmd.Flags().IntVarP(&forceTier, "tier", "T", 0, "Force specific tier (1=fast, 2=deep, 3=analysis, 4=writing)")
	cmd.Flags().StringVarP(&forceModel, "model", "m", "", "Force specific model")
	cmd.Flags().StringVarP(&forceHost, "host", "H", "", "Force specific host")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&noContext, "no-context", false, "Skip project context injection")
	cmd.Flags().BoolVar(&showRoute, "show-route", false, "Show routing decision without executing")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show routing decisions before executing")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output response as JSON")
	cmd.Flags().BoolVar(&stdinMode, "stdin", false, "Read additional content from stdin")

	return cmd
}

// readStdin reads all content from stdin.
func readStdin() (string, error) {
	// Check if stdin has data (not a terminal)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// stdin is a terminal, no piped data
		return "", nil
	}

	reader := bufio.NewReader(os.Stdin)
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func printRouteInfo(result *router.RouteResult) {
	fmt.Println(tui.RenderHeader("Routing Decision"))
	fmt.Println()

	tierName := router.TierName(result.Tier)
	fmt.Printf("  Tier:       %s (Tier %d) (%.0f%% confidence)\n", tierName, result.Tier, result.Confidence*100)
	fmt.Printf("  Model:      %s\n", result.Model)

	if result.Host != nil {
		fmt.Printf("  Host:       %s\n", result.Host.Host.Name)
		fmt.Printf("  URL:        %s\n", result.Host.Host.URL)
		fmt.Printf("  Latency:    %dms\n", result.Host.Latency.Milliseconds())
	} else {
		fmt.Printf("  Host:       %s\n", tui.ErrorStyle.Render("none available"))
	}
}

func executeStreaming(client *ollama.Client, model, prompt string) {
	_, err := client.GenerateStream(model, prompt, func(chunk ollama.GenerateResponse) {
		fmt.Print(chunk.Response)
	})

	if err != nil {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+err.Error()))
		return
	}

	fmt.Println() // Final newline
}

func executeBlocking(client *ollama.Client, model, prompt string) {
	resp, err := client.Generate(model, prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+err.Error()))
		return
	}

	fmt.Println(resp.Response)
}

// AskResponse represents the JSON output format for ask command
type AskResponse struct {
	Routing struct {
		Tier       int     `json:"tier"`
		TierName   string  `json:"tier_name"`
		Confidence float64 `json:"confidence"`
		Model      string  `json:"model"`
		Host       string  `json:"host,omitempty"`
		URL        string  `json:"url,omitempty"`
	} `json:"routing"`
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func executeJSON(result *router.RouteResult, prompt string) {
	output := AskResponse{}

	// Fill routing info
	output.Routing.Tier = result.Tier
	output.Routing.TierName = router.TierName(result.Tier)
	output.Routing.Confidence = result.Confidence
	output.Routing.Model = result.Model
	if result.Host != nil {
		output.Routing.Host = result.Host.Host.Name
		output.Routing.URL = result.Host.Host.URL
	}

	// Check for host availability
	if result.Host == nil || result.Client == nil {
		output.Error = "no available host found"
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Execute query (non-streaming for JSON)
	resp, err := result.Client.Generate(result.Model, prompt)
	if err != nil {
		output.Error = err.Error()
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return
	}

	output.Response = resp.Response
	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func getProjectContext() string {
	// Check for project context files
	contextFiles := []string{"CLAUDE.md", "AGENTS.md", "README.md"}

	for _, file := range contextFiles {
		if content, err := os.ReadFile(file); err == nil {
			// Truncate if too long
			s := string(content)
			if len(s) > 2000 {
				s = s[:2000] + "\n... (truncated)"
			}
			return s
		}
	}
	return ""
}
