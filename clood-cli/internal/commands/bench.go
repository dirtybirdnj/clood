package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func BenchCmd() *cobra.Command {
	var jsonOutput bool
	var hostName string
	var prompt string

	cmd := &cobra.Command{
		Use:   "bench [model]",
		Short: "Benchmark a model's performance",
		Long: `Runs a simple benchmark on the specified model to measure:
  - Tokens per second (generation speed)
  - Time to first token (latency)
  - Total generation time

If no model is specified, benchmarks the default fast tier model.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading config: " + err.Error()))
				return
			}

			// Determine model
			model := cfg.Tiers.Fast.Model
			if len(args) > 0 {
				model = args[0]
			}

			// Find host with this model
			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			var targetHost *hosts.HostStatus
			if hostName != "" {
				host := mgr.GetHost(hostName)
				if host == nil {
					fmt.Println(tui.ErrorStyle.Render("Host not found: " + hostName))
					return
				}
				targetHost = mgr.CheckHost(host)
				if !targetHost.Online {
					fmt.Println(tui.ErrorStyle.Render("Host is offline: " + hostName))
					return
				}
			} else {
				// Find best host with this model
				found := mgr.FindModel(model)
				if len(found) == 0 {
					// Try any online host
					targetHost = mgr.GetBestHost()
					if targetHost == nil {
						fmt.Println(tui.ErrorStyle.Render("No online hosts found"))
						return
					}
					fmt.Printf("%s Model %s not found, trying on %s anyway\n",
						tui.MutedStyle.Render("Warning:"),
						model,
						targetHost.Host.Name)
				} else {
					targetHost = found[0]
				}
			}

			// Default prompt
			if prompt == "" {
				prompt = "Write a haiku about programming. Be creative and concise."
			}

			fmt.Printf("%s Benchmarking %s on %s\n",
				tui.MutedStyle.Render("Running:"),
				model,
				targetHost.Host.Name)
			fmt.Println()

			client := ollama.NewClient(targetHost.Host.URL, 120*time.Second)

			result, err := client.Benchmark(model, prompt)
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Benchmark failed: " + err.Error()))
				return
			}

			if jsonOutput {
				printBenchJSON(result, targetHost)
				return
			}

			printBenchResult(result, targetHost)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVarP(&hostName, "host", "H", "", "Run on specific host")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Custom prompt for benchmark")

	return cmd
}

func printBenchResult(result *ollama.BenchmarkResult, host *hosts.HostStatus) {
	fmt.Println(tui.RenderHeader("Benchmark Results"))
	fmt.Println()

	fmt.Printf("  Model:     %s\n", result.Model)
	fmt.Printf("  Host:      %s\n", host.Host.Name)
	fmt.Println()

	fmt.Printf("  %s\n", tui.HeaderStyle.Render("Performance"))
	fmt.Printf("    Generation:  %s tok/s\n", formatTokPerSec(result.GenerateTokPerSec))
	fmt.Printf("    Prompt eval: %s tok/s\n", formatTokPerSec(result.PromptTokPerSec))
	fmt.Println()

	fmt.Printf("  %s\n", tui.HeaderStyle.Render("Timing"))
	fmt.Printf("    Total:       %s\n", result.TotalDuration.Round(time.Millisecond))
	fmt.Printf("    Model load:  %s\n", result.LoadDuration.Round(time.Millisecond))
	fmt.Println()

	fmt.Printf("  %s\n", tui.HeaderStyle.Render("Tokens"))
	fmt.Printf("    Prompt:      %d tokens\n", result.PromptTokens)
	fmt.Printf("    Generated:   %d tokens\n", result.GeneratedTokens)

	// Quality assessment
	fmt.Println()
	assessPerformance(result.GenerateTokPerSec)
}

func formatTokPerSec(tps float64) string {
	if tps >= 100 {
		return tui.SuccessStyle.Render(fmt.Sprintf("%.1f", tps))
	} else if tps >= 30 {
		return fmt.Sprintf("%.1f", tps)
	} else if tps >= 10 {
		return fmt.Sprintf("%.1f", tps)
	}
	return tui.ErrorStyle.Render(fmt.Sprintf("%.1f", tps))
}

func assessPerformance(tps float64) {
	var assessment string
	if tps >= 100 {
		assessment = tui.SuccessStyle.Render("Excellent") + " - Very fast generation"
	} else if tps >= 50 {
		assessment = tui.SuccessStyle.Render("Good") + " - Responsive for interactive use"
	} else if tps >= 20 {
		assessment = "Moderate - Acceptable for most tasks"
	} else if tps >= 10 {
		assessment = lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("Slow") + " - May feel slow for long responses"
	} else {
		assessment = tui.ErrorStyle.Render("Very slow") + " - Consider a smaller model"
	}

	fmt.Printf("  %s %s\n", tui.HeaderStyle.Render("Assessment:"), assessment)
}

func printBenchJSON(result *ollama.BenchmarkResult, host *hosts.HostStatus) {
	data := map[string]interface{}{
		"model":              result.Model,
		"host":               host.Host.Name,
		"host_url":           host.Host.URL,
		"total_duration_ms":  result.TotalDuration.Milliseconds(),
		"load_duration_ms":   result.LoadDuration.Milliseconds(),
		"prompt_tokens":      result.PromptTokens,
		"generated_tokens":   result.GeneratedTokens,
		"prompt_tok_per_sec": result.PromptTokPerSec,
		"gen_tok_per_sec":    result.GenerateTokPerSec,
	}

	out, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(out))
}
