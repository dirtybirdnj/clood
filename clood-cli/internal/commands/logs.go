package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dirtybirdnj/clood/internal/logging"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func LogsCmd() *cobra.Command {
	var logType string
	var model string
	var host string
	var since string
	var limit int
	var tail bool
	var onlyErrors bool
	var showStats bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Query conversation logs",
		Long: `View and query structured conversation logs.

Logs are stored in ~/.clood/conversations.jsonl in JSON Lines format.

Examples:
  clood logs                          # Show recent logs
  clood logs --tail -n 20             # Last 20 entries
  clood logs --type ask               # Filter by command type
  clood logs --model qwen2.5-coder:7b # Filter by model
  clood logs --errors                 # Only show errors
  clood logs --stats                  # Show aggregate statistics
  clood logs --since 1h               # Entries from last hour
  clood logs --json                   # Output as JSON`,
		Run: func(cmd *cobra.Command, args []string) {
			// Stats mode
			if showStats {
				stats, err := logging.GetStats("")
				if err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+err.Error()))
					return
				}

				if jsonOutput {
					data, _ := json.MarshalIndent(stats, "", "  ")
					fmt.Println(string(data))
					return
				}

				fmt.Println(tui.RenderHeader("Log Statistics"))
				fmt.Printf("%s %d\n", tui.MutedStyle.Render("Total entries:"), stats.TotalEntries)
				fmt.Printf("%s %.1f%%\n", tui.MutedStyle.Render("Success rate:"), stats.SuccessRate)
				fmt.Printf("%s %d\n", tui.MutedStyle.Render("Total tokens:"), stats.TotalTokens)
				fmt.Printf("%s %.1fms\n", tui.MutedStyle.Render("Avg duration:"), stats.AvgDurationMs)
				if !stats.FirstEntry.IsZero() {
					fmt.Printf("%s %s\n", tui.MutedStyle.Render("First entry:"), stats.FirstEntry.Format(time.RFC3339))
					fmt.Printf("%s %s\n", tui.MutedStyle.Render("Last entry:"), stats.LastEntry.Format(time.RFC3339))
				}

				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("By Type:"))
				for t, count := range stats.ByType {
					fmt.Printf("  %s: %d\n", t, count)
				}

				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("By Model:"))
				for m, count := range stats.ByModel {
					fmt.Printf("  %s: %d\n", m, count)
				}

				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("By Host:"))
				for h, count := range stats.ByHost {
					fmt.Printf("  %s: %d\n", h, count)
				}
				return
			}

			// Query mode
			opts := logging.QueryOptions{
				Type:     logType,
				Model:    model,
				Host:     host,
				Limit:    limit,
				Tail:     tail,
				OnlyErrs: onlyErrors,
			}

			// Parse since duration
			if since != "" {
				dur, err := time.ParseDuration(since)
				if err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Invalid duration: "+since))
					return
				}
				opts.Since = time.Now().Add(-dur)
			}

			entries, err := logging.Query("", opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+err.Error()))
				return
			}

			if len(entries) == 0 {
				fmt.Println(tui.MutedStyle.Render("No log entries found."))
				return
			}

			// JSON output
			if jsonOutput {
				data, _ := json.MarshalIndent(entries, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Human-readable output
			for _, entry := range entries {
				status := tui.SuccessStyle.Render("✓")
				if !entry.Success {
					status = tui.ErrorStyle.Render("✗")
				}

				timestamp := entry.Timestamp.Format("15:04:05")
				duration := ""
				if entry.DurationSec > 0 {
					duration = fmt.Sprintf(" (%.1fs)", entry.DurationSec)
				}

				fmt.Printf("%s %s %s %s%s\n",
					status,
					tui.MutedStyle.Render(timestamp),
					tui.AccentStyle.Render(entry.Type),
					entry.Model,
					duration)

				// Show prompt (truncated)
				if entry.Prompt != "" {
					prompt := entry.Prompt
					if len(prompt) > 60 {
						prompt = prompt[:60] + "..."
					}
					fmt.Printf("   %s %s\n", tui.MutedStyle.Render("→"), prompt)
				}

				// Show error if present
				if entry.Error != "" {
					fmt.Printf("   %s\n", tui.ErrorStyle.Render(entry.Error))
				}
			}

			fmt.Println()
			fmt.Printf("%s %d entries\n", tui.MutedStyle.Render("Showing"), len(entries))
		},
	}

	cmd.Flags().StringVarP(&logType, "type", "t", "", "Filter by type (ask, generate, chat, etc.)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Filter by model")
	cmd.Flags().StringVarP(&host, "host", "H", "", "Filter by host")
	cmd.Flags().StringVar(&since, "since", "", "Only entries newer than duration (e.g., 1h, 24h)")
	cmd.Flags().IntVarP(&limit, "n", "n", 10, "Maximum entries to show")
	cmd.Flags().BoolVar(&tail, "tail", false, "Show last N entries instead of first N")
	cmd.Flags().BoolVarP(&onlyErrors, "errors", "e", false, "Only show entries with errors")
	cmd.Flags().BoolVar(&showStats, "stats", false, "Show aggregate statistics")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}
