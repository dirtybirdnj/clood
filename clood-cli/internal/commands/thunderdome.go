package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// ThunderdomeResult holds result from one model on one host
type ThunderdomeResult struct {
	Host        string        `json:"host"`
	Model       string        `json:"model"`
	ModelHost   string        `json:"model_host"` // "model@host" format
	Response    string        `json:"response"`
	Duration    time.Duration `json:"duration_ns"`
	DurationSec float64       `json:"duration_sec"`
	Tokens      int           `json:"tokens"`
	TokSec      float64       `json:"tokens_per_sec"`
	Error       string        `json:"error,omitempty"`
}

// ThunderdomeOutput is the full JSON output
type ThunderdomeOutput struct {
	Timestamp string              `json:"timestamp"`
	Prompt    string              `json:"prompt"`
	Hosts     []string            `json:"hosts"`
	Results   []ThunderdomeResult `json:"results"`
	Champion  *ThunderdomeResult  `json:"champion,omitempty"`
	Summary   ThunderdomeSummary  `json:"summary"`
}

// ThunderdomeSummary provides aggregate stats
type ThunderdomeSummary struct {
	TotalHosts     int     `json:"total_hosts"`
	OnlineHosts    int     `json:"online_hosts"`
	TotalModels    int     `json:"total_models"`
	TotalRuns      int     `json:"total_runs"`
	Successful     int     `json:"successful"`
	Failed         int     `json:"failed"`
	FastestTime    float64 `json:"fastest_time_sec"`
	TotalTokens    int     `json:"total_tokens"`
	TotalDuration  float64 `json:"total_duration_sec"`
	ParallelFactor float64 `json:"parallel_factor"` // speedup vs sequential
}

// hostResult is used internally during parallel execution
type hostResult struct {
	host    string
	results []ThunderdomeResult
	err     error
}

func ThunderdomeCmd() *cobra.Command {
	var promptFile string
	var models string
	var hostFilter string
	var fast bool
	var topN int

	cmd := &cobra.Command{
		Use:   "thunderdome [prompt]",
		Short: "Orchestrate catfights across ALL hosts in parallel",
		Long: `Two hosts enter, one model wins - THUNDERDOME!

Unlike catfight which runs sequentially, thunderdome:
- Auto-discovers all online Ollama hosts
- Runs catfights in PARALLEL across all hosts
- Aggregates into unified leaderboard
- Crown a single champion (fastest model@host)

Examples:
  clood thunderdome "Write fizzbuzz in Rust"
  clood thunderdome --hosts "mac-mini,ubuntu25" -p "Explain monads"
  clood thunderdome --fast "Debug this code"   # Only fast models
  clood thunderdome --top 5 "Write a haiku"    # Show top 5 only
  clood thunderdome -f prompt.txt --json       # Machine-readable`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get prompt
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

			// Discover hosts
			mgr := hosts.NewManager()
			mgr.AddHosts(hosts.DefaultHosts())

			var targetHosts []*hosts.Host
			if hostFilter != "" {
				// Use specified hosts
				for _, name := range strings.Split(hostFilter, ",") {
					name = strings.TrimSpace(name)
					if h := mgr.GetHost(name); h != nil {
						targetHosts = append(targetHosts, h)
					}
				}
			} else {
				// Use all hosts
				targetHosts = mgr.GetAllHosts()
			}

			if !output.JSONMode {
				fmt.Println(tui.RenderHeader("THUNDERDOME"))
				fmt.Println()
				fmt.Printf("  %s Scanning hosts...\n", tui.AccentStyle.Render(""))
			}

			// Check which hosts are online (parallel)
			statuses := mgr.CheckAllHosts()
			var onlineHosts []*hosts.HostStatus
			for _, s := range statuses {
				// Only include if in our target list
				isTarget := false
				for _, t := range targetHosts {
					if t.Name == s.Host.Name {
						isTarget = true
						break
					}
				}
				if isTarget && s.Online && len(s.Models) > 0 {
					onlineHosts = append(onlineHosts, s)
				}
			}

			if len(onlineHosts) == 0 {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("No online hosts with models found"))
				return
			}

			if !output.JSONMode {
				fmt.Printf("  %s %d hosts online, %d total models\n\n",
					tui.SuccessStyle.Render(""),
					len(onlineHosts),
					countTotalModels(onlineHosts))

				for _, h := range onlineHosts {
					modelCount := len(h.Models)
					if fast {
						modelCount = min(3, modelCount) // Fast mode: top 3 models per host
					}
					fmt.Printf("  %s %s (%d models, %dms latency)\n",
						tui.SuccessStyle.Render(""),
						h.Host.Name,
						modelCount,
						h.Latency.Milliseconds())
				}
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("  Prompt: " + truncatePrompt(prompt, 60)))
				fmt.Println()
				fmt.Printf("  %s\n\n", tui.AccentStyle.Render("ALLEZ CUISINE!"))
			}

			// Run catfights in parallel across all hosts
			startTime := time.Now()
			resultsChan := make(chan hostResult, len(onlineHosts))
			var wg sync.WaitGroup

			for _, hostStatus := range onlineHosts {
				wg.Add(1)
				go func(hs *hosts.HostStatus) {
					defer wg.Done()
					results := runHostCatfight(hs, prompt, models, fast)
					resultsChan <- hostResult{host: hs.Host.Name, results: results}
				}(hostStatus)
			}

			// Wait for all hosts to complete
			go func() {
				wg.Wait()
				close(resultsChan)
			}()

			// Collect results
			var allResults []ThunderdomeResult
			hostsCompleted := 0
			for hr := range resultsChan {
				hostsCompleted++
				if !output.JSONMode {
					successCount := 0
					for _, r := range hr.results {
						if r.Error == "" {
							successCount++
						}
					}
					fmt.Printf("  [%d/%d] %s completed (%d/%d successful)\n",
						hostsCompleted, len(onlineHosts),
						hr.host,
						successCount, len(hr.results))
				}
				allResults = append(allResults, hr.results...)
			}

			totalDuration := time.Since(startTime)

			// Sort by duration (fastest first)
			sort.Slice(allResults, func(i, j int) bool {
				// Errors go to the end
				if allResults[i].Error != "" && allResults[j].Error == "" {
					return false
				}
				if allResults[i].Error == "" && allResults[j].Error != "" {
					return true
				}
				return allResults[i].DurationSec < allResults[j].DurationSec
			})

			// Find champion
			var champion *ThunderdomeResult
			for i := range allResults {
				if allResults[i].Error == "" {
					champion = &allResults[i]
					break
				}
			}

			// Calculate summary stats
			successful := 0
			failed := 0
			totalTokens := 0
			sequentialTime := 0.0
			for _, r := range allResults {
				if r.Error == "" {
					successful++
					totalTokens += r.Tokens
					sequentialTime += r.DurationSec
				} else {
					failed++
				}
			}

			summary := ThunderdomeSummary{
				TotalHosts:     len(targetHosts),
				OnlineHosts:    len(onlineHosts),
				TotalModels:    countTotalModels(onlineHosts),
				TotalRuns:      len(allResults),
				Successful:     successful,
				Failed:         failed,
				TotalTokens:    totalTokens,
				TotalDuration:  totalDuration.Seconds(),
				ParallelFactor: sequentialTime / totalDuration.Seconds(),
			}
			if champion != nil {
				summary.FastestTime = champion.DurationSec
			}

			// Build output
			hostNames := make([]string, len(onlineHosts))
			for i, h := range onlineHosts {
				hostNames[i] = h.Host.Name
			}

			thunderdomeOutput := ThunderdomeOutput{
				Timestamp: time.Now().Format(time.RFC3339),
				Prompt:    prompt,
				Hosts:     hostNames,
				Results:   allResults,
				Champion:  champion,
				Summary:   summary,
			}

			if output.JSONMode {
				data, _ := json.MarshalIndent(thunderdomeOutput, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Display leaderboard
			fmt.Println()
			fmt.Println(tui.RenderHeader("LEADERBOARD"))
			fmt.Println()

			displayCount := len(allResults)
			if topN > 0 && topN < displayCount {
				displayCount = topN
			}

			for i := 0; i < displayCount; i++ {
				r := allResults[i]
				rank := i + 1
				var rankStyle string
				switch rank {
				case 1:
					rankStyle = tui.SuccessStyle.Render(fmt.Sprintf("%2d.", rank))
				case 2, 3:
					rankStyle = tui.AccentStyle.Render(fmt.Sprintf("%2d.", rank))
				default:
					rankStyle = tui.MutedStyle.Render(fmt.Sprintf("%2d.", rank))
				}

				if r.Error != "" {
					fmt.Printf("  %s %-30s %s\n",
						rankStyle,
						r.ModelHost,
						tui.ErrorStyle.Render("ERROR"))
				} else {
					fmt.Printf("  %s %-30s %6.1fs  %5d tok  %6.1f tok/s\n",
						rankStyle,
						r.ModelHost,
						r.DurationSec,
						r.Tokens,
						r.TokSec)
				}
			}

			if topN > 0 && len(allResults) > topN {
				fmt.Printf("\n  %s\n", tui.MutedStyle.Render(fmt.Sprintf("... and %d more", len(allResults)-topN)))
			}

			// Champion announcement
			if champion != nil {
				fmt.Println()
				fmt.Println(strings.Repeat("=", 50))
				fmt.Printf("  %s %s\n",
					tui.SuccessStyle.Render("THUNDERDOME CHAMPION:"),
					champion.ModelHost)
				fmt.Printf("  %s %.1fs | %d tokens | %.1f tok/s\n",
					tui.MutedStyle.Render("Stats:"),
					champion.DurationSec,
					champion.Tokens,
					champion.TokSec)
				fmt.Println(strings.Repeat("=", 50))
			}

			// Summary
			fmt.Println()
			fmt.Printf("  %s %d runs across %d hosts in %.1fs (%.1fx parallel speedup)\n",
				tui.MutedStyle.Render("Summary:"),
				summary.TotalRuns,
				summary.OnlineHosts,
				summary.TotalDuration,
				summary.ParallelFactor)
		},
	}

	cmd.Flags().StringVarP(&promptFile, "file", "f", "", "Read prompt from file")
	cmd.Flags().StringVarP(&models, "models", "m", "", "Specific models to test (comma-separated)")
	cmd.Flags().StringVar(&hostFilter, "hosts", "", "Specific hosts to use (comma-separated)")
	cmd.Flags().BoolVar(&fast, "fast", false, "Fast mode: only top 3 models per host")
	cmd.Flags().IntVar(&topN, "top", 0, "Show only top N results (0 = all)")

	return cmd
}

// runHostCatfight runs catfight on a single host
func runHostCatfight(hs *hosts.HostStatus, prompt string, modelFilter string, fast bool) []ThunderdomeResult {
	client := ollama.NewClient(hs.Host.URL, 5*time.Minute)
	var results []ThunderdomeResult

	// Determine which models to run
	var modelsToRun []ollama.Model
	if modelFilter != "" {
		// Use specified models
		for _, m := range strings.Split(modelFilter, ",") {
			m = strings.TrimSpace(m)
			for _, hm := range hs.Models {
				if hm.Name == m {
					modelsToRun = append(modelsToRun, hm)
					break
				}
			}
		}
	} else {
		modelsToRun = hs.Models
	}

	// Fast mode: limit to 3 models
	if fast && len(modelsToRun) > 3 {
		modelsToRun = modelsToRun[:3]
	}

	for _, model := range modelsToRun {
		result := ThunderdomeResult{
			Host:      hs.Host.Name,
			Model:     model.Name,
			ModelHost: fmt.Sprintf("%s@%s", model.Name, hs.Host.Name),
		}

		start := time.Now()
		resp, err := client.Generate(model.Name, prompt)
		duration := time.Since(start)

		result.Duration = duration
		result.DurationSec = duration.Seconds()

		if err != nil {
			result.Error = err.Error()
		} else {
			result.Response = resp.Response
			result.Tokens = resp.EvalCount
			if resp.EvalDuration > 0 {
				result.TokSec = float64(resp.EvalCount) / (float64(resp.EvalDuration) / 1e9)
			}
		}

		results = append(results, result)
	}

	return results
}

func countTotalModels(hosts []*hosts.HostStatus) int {
	total := 0
	for _, h := range hosts {
		total += len(h.Models)
	}
	return total
}

func truncatePrompt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
