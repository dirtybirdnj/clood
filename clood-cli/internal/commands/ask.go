package commands

import (
	"fmt"
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

	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask a question (auto-routes to appropriate tier/host)",
		Long: `Ask a question and clood will:
  1. Classify the query complexity (fast vs deep)
  2. Select the appropriate model tier
  3. Route to the best available host
  4. Stream the response

Use --show-route to see routing decisions without executing.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			question := strings.Join(args, " ")

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

			// Show routing info
			if showRoute {
				printRouteInfo(result)
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

			// Build prompt with context
			prompt := question
			if !noContext {
				context := getProjectContext()
				if context != "" {
					prompt = fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context, question)
				}
			}

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

	return cmd
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
