package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/system"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// sendATCEvent sends an event to the ATC dashboard
func sendATCEvent(atcURL, eventType string, data interface{}) {
	if atcURL == "" {
		return
	}
	event := map[string]interface{}{
		"type": eventType,
		"data": data,
	}
	body, _ := json.Marshal(event)
	// Fire and forget - don't block on ATC
	go func() {
		resp, err := http.Post(atcURL+"/events", "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
		}
	}()
}

// Cat represents a model in the catfight
type Cat struct {
	Name  string
	Model string
}

// DefaultCats are the proven performers from Kitchen Stadium
var DefaultCats = []Cat{
	{Name: "Persian", Model: "deepseek-coder:6.7b"},
	{Name: "Tabby", Model: "mistral:7b"},
	{Name: "Siamese", Model: "qwen2.5-coder:3b"},
}

// CatfightResult holds the output from one cat
type CatfightResult struct {
	Cat         Cat           `json:"cat"`
	Host        string        `json:"host"`
	HostURL     string        `json:"host_url,omitempty"`
	Response    string        `json:"response"`
	Duration    time.Duration `json:"duration_ns"`
	DurationSec float64       `json:"duration_sec"`
	Tokens      int           `json:"tokens"`
	TokSec      float64       `json:"tokens_per_sec"`
	Error       error         `json:"-"`
	ErrorStr    string        `json:"error,omitempty"`
}

// CatfightOutput is the full JSON output structure
type CatfightOutput struct {
	Timestamp  string           `json:"timestamp"`
	Prompt     string           `json:"prompt"`
	PromptFile string           `json:"prompt_file,omitempty"`
	Hosts      []string         `json:"hosts"`
	Models     []string         `json:"models"`
	Results    []CatfightResult `json:"results"`
	Winner     *CatfightResult  `json:"winner,omitempty"`
	Summary    CatfightSummary  `json:"summary"`
}

// CatfightSummary provides aggregate stats
type CatfightSummary struct {
	TotalRuns    int     `json:"total_runs"`
	Successful   int     `json:"successful"`
	Failed       int     `json:"failed"`
	FastestTime  float64 `json:"fastest_time_sec"`
	AverageTime  float64 `json:"average_time_sec"`
	AverageSpeed float64 `json:"average_tokens_per_sec"`
}

// Spinner frames for the catfight animation
var catSpinnerFrames = []string{
	"🐱       ", " 🐱      ", "  🐱     ", "   🐱    ",
	"    🐱   ", "     🐱  ", "      🐱 ", "       🐱",
	"      🐱 ", "     🐱  ", "    🐱   ", "   🐱    ",
	"  🐱     ", " 🐱      ",
}

func CatfightCmd() *cobra.Command {
	var promptFile string
	var models string
	var outputDir string
	var host string
	var hostNames string
	var allHosts bool
	var quiet bool
	var jsonOutput bool
	var markdownOutput bool
	var crossHost bool
	var streamOutput bool
	var createIssue bool
	var issueLabels string
	var atcURL string

	cmd := &cobra.Command{
		Use:   "catfight [prompt]",
		Short: "Compare multiple models on the same prompt",
		Long: `Compare multiple LLM models against the same prompt.

SINGLE MACHINE (default):
  Runs on localhost. Perfect for standalone workstations.

MULTI-HOST OPTIONS:
  --host URL        Run on a specific Ollama instance
  --hosts "a,b"     Run on specific named hosts from config
  --all-hosts       Run on ALL online hosts in parallel (garden mode)

MODELS:
  Default: qwen2.5-coder:3b, mistral:7b, deepseek-coder:6.7b
  Override with --models "model1,model2,model3"

Examples:
  # Single machine (default)
  clood catfight "Write hello world in Go"
  clood catfight -f prompt.txt
  clood catfight --models "llama3.1:8b,qwen2.5-coder:7b" "Explain recursion"

  # Specific host
  clood catfight --host http://192.168.1.100:11434 "prompt"

  # Multiple specific hosts
  clood catfight --hosts "mac-mini,ubuntu25" "prompt"

  # All online hosts (garden mode)
  clood catfight --all-hosts "Write fizzbuzz"
  clood catfight --all-hosts --json -f prompt.txt

  # Post results to GitHub issue
  clood catfight --issue "Compare sorting algorithms"`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get the prompt
			var prompt string
			if promptFile != "" {
				data, err := os.ReadFile(promptFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error reading prompt file: "+err.Error()))
					return
				}
				prompt = string(data)
			} else if len(args) > 0 {
				prompt = strings.Join(args, " ")
			} else {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("No prompt provided. Use -f <file> or pass prompt as argument."))
				return
			}

			// Determine which cats to use
			cats := DefaultCats
			if models != "" {
				cats = []Cat{}
				for _, m := range strings.Split(models, ",") {
					m = strings.TrimSpace(m)
					cats = append(cats, Cat{Name: modelToName(m), Model: m})
				}
			}

			// Set up output directory
			if outputDir != "" {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error creating output dir: "+err.Error()))
					return
				}
			}

			// Build host/client map - support multi-host mode
			type hostClient struct {
				name   string
				url    string
				client *ollama.Client
			}
			var hostClients []hostClient

			if allHosts {
				// Garden mode: discover and use ALL online hosts
				cfg, err := config.Load()
				if err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error loading config: "+err.Error()))
					return
				}
				mgr := hosts.NewManager()
				mgr.AddHosts(cfg.Hosts)
				// Also add default hosts for discovery
				mgr.AddHosts(hosts.DefaultHosts())

				statuses := mgr.CheckAllHosts()
				for _, s := range statuses {
					if s.Online && len(s.Models) > 0 {
						hostClients = append(hostClients, hostClient{
							name:   s.Host.Name,
							url:    s.Host.URL,
							client: mgr.GetClient(s.Host.Name),
						})
					}
				}

				if len(hostClients) == 0 {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("No online hosts found. Check 'clood hosts' for status."))
					return
				}

				if !jsonOutput && !markdownOutput {
					fmt.Printf("%s Garden mode: %d hosts online\n", tui.AccentStyle.Render("🌿"), len(hostClients))
				}
			} else if hostNames != "" {
				// Multi-host mode: use named hosts from config
				cfg, err := config.Load()
				if err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error loading config: "+err.Error()))
					return
				}
				mgr := hosts.NewManager()
				mgr.AddHosts(cfg.Hosts)

				for _, name := range strings.Split(hostNames, ",") {
					name = strings.TrimSpace(name)
					h := mgr.GetHost(name)
					if h == nil {
						fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Host not found: "+name))
						return
					}
					hostClients = append(hostClients, hostClient{
						name:   h.Name,
						url:    h.URL,
						client: mgr.GetClient(h.Name),
					})
				}
			} else if host != "" {
				// Single host URL mode
				hostClients = append(hostClients, hostClient{
					name:   "custom",
					url:    host,
					client: ollama.NewClient(host, 5*time.Minute),
				})
			} else {
				// Default: localhost only
				hostClients = append(hostClients, hostClient{
					name:   "localhost",
					url:    "http://localhost:11434",
					client: ollama.NewClient("http://localhost:11434", 5*time.Minute),
				})
			}

			// Collect host names and model names for output
			var hostNameList []string
			var modelNameList []string
			for _, hc := range hostClients {
				hostNameList = append(hostNameList, hc.name)
			}
			for _, cat := range cats {
				modelNameList = append(modelNameList, cat.Model)
			}

			// ALLEZ CUISINE! (unless JSON output)
			if !jsonOutput && !markdownOutput {
				fmt.Println(tui.RenderHeader("KITCHEN STADIUM - CATFIGHT"))
				fmt.Println()
				if len(hostClients) > 1 {
					fmt.Printf("%s Two-Kitchen Showdown!\n", tui.AccentStyle.Render("🏟️"))
					for _, hc := range hostClients {
						fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Kitchen:"), hc.name)
					}
					fmt.Println()
				}
				fmt.Printf("%s %d cats entering the arena\n", tui.MutedStyle.Render("Contenders:"), len(cats))
				for _, cat := range cats {
					fmt.Printf("  %s %s\n", tui.AccentStyle.Render(cat.Name+":"), cat.Model)
				}
				fmt.Println()

				if !quiet {
					fmt.Println(tui.MutedStyle.Render("Prompt:"))
					displayPrompt := prompt
					if len(displayPrompt) > 200 {
						displayPrompt = displayPrompt[:200] + "..."
					}
					fmt.Println(displayPrompt)
					fmt.Println()
				}
			}

			// Send ATC start event
			sendATCEvent(atcURL, "start", map[string]interface{}{
				"prompt": func() string {
					if len(prompt) > 100 {
						return prompt[:100] + "..."
					}
					return prompt
				}(),
				"models": modelNameList,
				"hosts":  hostNameList,
			})

			// Run each cat on each host
			results := []CatfightResult{}
			runCount := 0
			totalRuns := len(cats) * len(hostClients)

			for _, hc := range hostClients {
				if !jsonOutput && !markdownOutput && len(hostClients) > 1 {
					fmt.Printf("\n%s Kitchen: %s (%s)\n", tui.AccentStyle.Render("🍳"), hc.name, hc.url)
					fmt.Println(strings.Repeat("-", 50))
				}

				for _, cat := range cats {
					runCount++
					if !jsonOutput && !markdownOutput {
						fmt.Printf("%s [%d/%d] %s (%s)",
							tui.AccentStyle.Render(">>>"),
							runCount, totalRuns,
							cat.Name,
							cat.Model)
						if len(hostClients) > 1 {
							fmt.Printf(" on %s", hc.name)
						}
						fmt.Println()
					}

					start := time.Now()

					result := CatfightResult{
						Cat:     cat,
						Host:    hc.name,
						HostURL: hc.url,
					}

					var resp *ollama.GenerateResponse
					var err error
					var responseBuilder strings.Builder

					if streamOutput && !jsonOutput && !markdownOutput {
						// Streaming mode with live spinner
						fmt.Printf("    ")
						tokenCount := 0
						spinnerIdx := 0
						lastSpinnerUpdate := time.Now()

						resp, err = hc.client.GenerateStream(cat.Model, prompt, func(chunk ollama.GenerateResponse) {
							responseBuilder.WriteString(chunk.Response)
							tokenCount++

							// Show spinner animation every 100ms
							if time.Since(lastSpinnerUpdate) > 100*time.Millisecond {
								fmt.Printf("\r    %s %s %d tokens...",
									catSpinnerFrames[spinnerIdx%len(catSpinnerFrames)],
									tui.MutedStyle.Render("generating"),
									tokenCount)
								spinnerIdx++
								lastSpinnerUpdate = time.Now()
							}
						})

						// Clear the spinner line
						fmt.Printf("\r                                                    \r")

						if resp != nil {
							resp.Response = responseBuilder.String()
						}
					} else {
						// Non-streaming mode (original behavior)
						resp, err = hc.client.Generate(cat.Model, prompt)
					}

					duration := time.Since(start)
					result.Duration = duration
					result.DurationSec = duration.Seconds()

					if err != nil {
						result.Error = err
						result.ErrorStr = err.Error()
						if !jsonOutput && !markdownOutput {
							fmt.Printf("    %s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
						}
						// Send ATC error event
						sendATCEvent(atcURL, "progress", map[string]interface{}{
							"model":   cat.Model,
							"host":    hc.name,
							"status":  "error",
							"message": err.Error(),
						})
					} else {
						result.Response = resp.Response
						result.Tokens = resp.EvalCount
						if resp.EvalDuration > 0 {
							result.TokSec = float64(resp.EvalCount) / (float64(resp.EvalDuration) / 1e9)
						}
						if !jsonOutput && !markdownOutput {
							fmt.Printf("    %s %.1fs | %d tokens | %.1f tok/s\n",
								tui.SuccessStyle.Render("DONE"),
								duration.Seconds(),
								result.Tokens,
								result.TokSec)

							// Save output if outputDir specified
							if outputDir != "" {
								filename := filepath.Join(outputDir, hc.name+"_"+sanitizeFilename(cat.Model)+".txt")
								if err := os.WriteFile(filename, []byte(resp.Response), 0644); err != nil {
									fmt.Printf("    %s %v\n", tui.WarningStyle.Render("Save error:"), err)
								} else {
									fmt.Printf("    %s %s\n", tui.MutedStyle.Render("Saved:"), filename)
								}
							}
						}
						// Send ATC progress event
						sendATCEvent(atcURL, "progress", map[string]interface{}{
							"model":      cat.Model,
							"host":       hc.name,
							"status":     "complete",
							"time_sec":   result.DurationSec,
							"tokens":     result.Tokens,
							"tokens_sec": result.TokSec,
						})
					}
					results = append(results, result)
				}
			}

			// Record benchmarks for successful runs
			if benchStore, err := system.NewBenchmarkStore(); err == nil {
				for _, r := range results {
					if r.Error == nil && r.TokSec > 0 {
						benchStore.Record(r.Cat.Model, r.Host, "catfight", r.TokSec)
					}
				}
			}

			// Find winner and calculate summary
			var fastest *CatfightResult
			var totalTime, totalSpeed float64
			var successful, failed int
			for i := range results {
				r := &results[i]
				if r.Error != nil {
					failed++
				} else {
					successful++
					totalTime += r.DurationSec
					totalSpeed += r.TokSec
					if fastest == nil || r.Duration < fastest.Duration {
						fastest = r
					}
				}
			}

			summary := CatfightSummary{
				TotalRuns:  len(results),
				Successful: successful,
				Failed:     failed,
			}
			if fastest != nil {
				summary.FastestTime = fastest.DurationSec
			}
			if successful > 0 {
				summary.AverageTime = totalTime / float64(successful)
				summary.AverageSpeed = totalSpeed / float64(successful)
			}

			// Send ATC complete event
			completeData := map[string]interface{}{
				"successful": successful,
				"failed":     failed,
				"avg_speed":  summary.AverageSpeed,
			}
			if fastest != nil {
				completeData["winner"] = fastest.Cat.Model
				completeData["winner_time"] = fastest.DurationSec
				completeData["winner_host"] = fastest.Host
			}
			sendATCEvent(atcURL, "complete", completeData)

			// JSON output
			if jsonOutput {
				output := CatfightOutput{
					Timestamp:  time.Now().Format(time.RFC3339),
					Prompt:     prompt,
					PromptFile: promptFile,
					Hosts:      hostNameList,
					Models:     modelNameList,
					Results:    results,
					Winner:     fastest,
					Summary:    summary,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Markdown output
			if markdownOutput {
				fmt.Println("## Kitchen Stadium - Catfight Results")
				fmt.Println()
				fmt.Printf("**Timestamp:** %s\n\n", time.Now().Format(time.RFC3339))
				if len(hostClients) > 1 {
					fmt.Printf("**Kitchens:** %s\n\n", strings.Join(hostNameList, ", "))
				}
				fmt.Println("| Cat | Model | Host | Time | Tokens | Speed |")
				fmt.Println("|-----|-------|------|------|--------|-------|")
				for _, r := range results {
					if r.Error != nil {
						fmt.Printf("| %s | %s | %s | FAILED | - | - |\n", r.Cat.Name, r.Cat.Model, r.Host)
					} else {
						winner := ""
						if fastest != nil && r.Cat.Model == fastest.Cat.Model && r.Host == fastest.Host {
							winner = " **WINNER**"
						}
						fmt.Printf("| %s%s | %s | %s | %.1fs | %d | %.1f tok/s |\n",
							r.Cat.Name, winner, r.Cat.Model, r.Host, r.DurationSec, r.Tokens, r.TokSec)
					}
				}
				if fastest != nil {
					fmt.Printf("\n**Winner:** %s (%s) on %s with %.1fs\n", fastest.Cat.Name, fastest.Cat.Model, fastest.Host, fastest.DurationSec)
				}
				return
			}

			// Create GitHub issue with results
			if createIssue {
				var body strings.Builder
				body.WriteString("## Kitchen Stadium - Catfight Results\n\n")
				body.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", time.Now().Format(time.RFC3339)))
				body.WriteString(fmt.Sprintf("**Prompt:** %s\n\n", prompt))
				if len(hostClients) > 1 {
					body.WriteString(fmt.Sprintf("**Kitchens:** %s\n\n", strings.Join(hostNameList, ", ")))
				}
				body.WriteString("| Cat | Model | Host | Time | Tokens | Speed |\n")
				body.WriteString("|-----|-------|------|------|--------|-------|\n")
				for _, r := range results {
					if r.Error != nil {
						body.WriteString(fmt.Sprintf("| %s | %s | %s | FAILED | - | - |\n", r.Cat.Name, r.Cat.Model, r.Host))
					} else {
						winner := ""
						if fastest != nil && r.Cat.Model == fastest.Cat.Model && r.Host == fastest.Host {
							winner = " 🏆"
						}
						body.WriteString(fmt.Sprintf("| %s%s | %s | %s | %.1fs | %d | %.1f tok/s |\n",
							r.Cat.Name, winner, r.Cat.Model, r.Host, r.DurationSec, r.Tokens, r.TokSec))
					}
				}
				if fastest != nil {
					body.WriteString(fmt.Sprintf("\n**Winner:** %s (%s) on %s with %.1fs\n", fastest.Cat.Name, fastest.Cat.Model, fastest.Host, fastest.DurationSec))
				}

				// Add responses in collapsible sections
				body.WriteString("\n---\n\n## Responses\n\n")
				for _, r := range results {
					if r.Error == nil {
						body.WriteString(fmt.Sprintf("<details>\n<summary>%s (%s)</summary>\n\n```\n%s\n```\n\n</details>\n\n",
							r.Cat.Name, r.Cat.Model, r.Response))
					}
				}

				// Build title
				winnerName := "no winner"
				if fastest != nil {
					winnerName = fastest.Cat.Name
				}
				title := fmt.Sprintf("Catfight: %s wins! [%s]", winnerName, time.Now().Format("2006-01-02 15:04"))

				// Build gh command
				ghArgs := []string{"issue", "create", "--title", title, "--body", body.String()}
				if issueLabels != "" {
					ghArgs = append(ghArgs, "--label", issueLabels)
				}

				fmt.Printf("%s Creating GitHub issue...\n", tui.AccentStyle.Render(">>>"))
				ghCmd := exec.Command("gh", ghArgs...)
				output, err := ghCmd.CombinedOutput()
				if err != nil {
					fmt.Printf("    %s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
					fmt.Printf("    %s\n", string(output))
				} else {
					fmt.Printf("    %s %s", tui.SuccessStyle.Render("Created:"), string(output))
				}
				return
			}

			// Standard terminal output - Summary
			fmt.Println()
			fmt.Println(tui.RenderHeader("RESULTS"))
			fmt.Println()

			if len(hostClients) > 1 {
				fmt.Printf("%-12s %-20s %-12s %8s %8s %10s\n", "CAT", "MODEL", "HOST", "TIME", "TOKENS", "TOK/S")
				fmt.Println(strings.Repeat("-", 80))
			} else {
				fmt.Printf("%-12s %-25s %10s %8s %10s\n", "CAT", "MODEL", "TIME", "TOKENS", "TOK/S")
				fmt.Println(strings.Repeat("-", 70))
			}

			for _, r := range results {
				if r.Error != nil {
					if len(hostClients) > 1 {
						fmt.Printf("%-12s %-20s %-12s %s\n", r.Cat.Name, r.Cat.Model, r.Host, tui.ErrorStyle.Render("FAILED"))
					} else {
						fmt.Printf("%-12s %-25s %s\n", r.Cat.Name, r.Cat.Model, tui.ErrorStyle.Render("FAILED"))
					}
				} else {
					if len(hostClients) > 1 {
						fmt.Printf("%-12s %-20s %-12s %8.1fs %8d %10.1f\n", r.Cat.Name, r.Cat.Model, r.Host, r.DurationSec, r.Tokens, r.TokSec)
					} else {
						fmt.Printf("%-12s %-25s %8.1fs %8d %10.1f\n", r.Cat.Name, r.Cat.Model, r.DurationSec, r.Tokens, r.TokSec)
					}
				}
			}

			if fastest != nil {
				fmt.Println()
				if len(hostClients) > 1 {
					fmt.Printf("%s %s on %s wins with %.1fs!\n",
						tui.AccentStyle.Render("WINNER:"),
						fastest.Cat.Name,
						fastest.Host,
						fastest.DurationSec)
				} else {
					fmt.Printf("%s %s wins with %.1fs!\n",
						tui.AccentStyle.Render("WINNER:"),
						fastest.Cat.Name,
						fastest.DurationSec)
				}
			}

			// Show all responses for easy comparison / LLM consumption
			fmt.Println()
			fmt.Println(tui.RenderHeader("RESPONSES"))
			fmt.Println()
			for _, r := range results {
				if len(hostClients) > 1 {
					fmt.Printf("%s %s (%s) on %s\n",
						tui.AccentStyle.Render("###"),
						r.Cat.Name,
						r.Cat.Model,
						r.Host)
				} else {
					fmt.Printf("%s %s (%s)\n",
						tui.AccentStyle.Render("###"),
						r.Cat.Name,
						r.Cat.Model)
				}
				fmt.Println(strings.Repeat("-", 60))
				if r.Error != nil {
					fmt.Printf("%s\n", tui.ErrorStyle.Render("ERROR: "+r.Error.Error()))
				} else {
					fmt.Println(r.Response)
				}
				fmt.Println()
			}
		},
	}

	cmd.Flags().StringVarP(&promptFile, "file", "f", "", "Read prompt from file")
	cmd.Flags().StringVarP(&models, "models", "m", "", "Comma-separated list of models to compare")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Directory to save outputs")
	cmd.Flags().StringVarP(&host, "host", "H", "", "Ollama host URL (single host)")
	cmd.Flags().StringVar(&hostNames, "hosts", "", "Comma-separated host names (e.g., ubuntu25,mac-mini)")
	cmd.Flags().BoolVar(&allHosts, "all-hosts", false, "Run on ALL online hosts in parallel (garden mode)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Don't show prompt preview")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().BoolVar(&markdownOutput, "markdown", false, "Output results as Markdown")
	cmd.Flags().BoolVar(&crossHost, "cross-host", false, "Compare same model across hosts")
	cmd.Flags().BoolVarP(&streamOutput, "stream", "s", false, "Show live progress during generation")
	cmd.Flags().BoolVar(&createIssue, "issue", false, "Create GitHub issue with results")
	cmd.Flags().StringVar(&issueLabels, "labels", "", "Labels for GitHub issue (requires --issue)")
	cmd.Flags().StringVar(&atcURL, "atc", "", "ATC dashboard URL for live events (e.g., http://localhost:8080)")

	return cmd
}

// modelToName creates a friendly name from a model string
func modelToName(model string) string {
	// Check known cats
	knownCats := map[string]string{
		"deepseek-coder:6.7b": "Persian",
		"mistral:7b":          "Tabby",
		"qwen2.5-coder:3b":    "Siamese",
		"llama3.1:8b":         "Caracal",
		"deepseek-r1:14b":     "HouseLion",
		"starcoder2:7b":       "Bengal",
		"tinyllama":           "Kitten",
	}
	if name, ok := knownCats[model]; ok {
		return name
	}
	// Use first part of model name
	parts := strings.Split(model, ":")
	return strings.Title(parts[0])
}

// sanitizeFilename makes a model name safe for filenames
func sanitizeFilename(model string) string {
	return strings.ReplaceAll(strings.ReplaceAll(model, ":", "_"), "/", "_")
}
