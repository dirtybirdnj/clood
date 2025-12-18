package commands

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func ModelsCmd() *cobra.Command {
	var jsonOutput bool
	var hostFilter string

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List available models across all hosts",
		Long: `Lists all models available across all configured Ollama hosts.
Shows which hosts have each model and model details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading config: " + err.Error()))
				return
			}

			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			// Check both local --json and global -j flag
			useJSON := jsonOutput || output.IsJSON()

			// Only show progress in human mode
			if !useJSON {
				fmt.Println(tui.MutedStyle.Render("Scanning hosts for models..."))
				fmt.Println()
			}

			if hostFilter != "" {
				// Show models for specific host
				host := mgr.GetHost(hostFilter)
				if host == nil {
					fmt.Println(tui.ErrorStyle.Render("Host not found: " + hostFilter))
					return
				}
				status := mgr.CheckHost(host)
				if !status.Online {
					fmt.Println(tui.ErrorStyle.Render("Host is offline: " + hostFilter))
					return
				}
				printHostModels(status, useJSON)
				return
			}

			// Show all models across all hosts
			allModels := mgr.GetAllModels()

			if useJSON {
				data, _ := json.MarshalIndent(allModels, "", "  ")
				fmt.Println(string(data))
				return
			}

			fmt.Println(tui.RenderHeader("Available Models"))
			fmt.Println()

			if len(allModels) == 0 {
				fmt.Println(tui.MutedStyle.Render("  No models found on any host"))
				return
			}

			// Sort model names
			var names []string
			for name := range allModels {
				names = append(names, name)
			}
			sort.Strings(names)

			// Group by model family
			families := groupByFamily(names)

			for family, models := range families {
				fmt.Printf("  %s\n", tui.HeaderStyle.Render(family))
				for _, model := range models {
					hosts := allModels[model]
					hostStr := ""
					for i, h := range hosts {
						if i > 0 {
							hostStr += ", "
						}
						hostStr += h
					}
					fmt.Printf("    %s %s\n", model, tui.MutedStyle.Render("("+hostStr+")"))
				}
				fmt.Println()
			}

			// Show tier configuration
			fmt.Println(tui.RenderHeader("Tier Configuration"))
			fmt.Println()
			fmt.Printf("  Fast (Tier 1): %s\n", cfg.Tiers.Fast.Model)
			fmt.Printf("  Deep (Tier 2): %s\n", cfg.Tiers.Deep.Model)

			// Check if tier models are available
			fmt.Println()
			checkTierModel(allModels, "Fast", cfg.Tiers.Fast.Model)
			checkTierModel(allModels, "Deep", cfg.Tiers.Deep.Model)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVarP(&hostFilter, "host", "H", "", "Show models for specific host")

	return cmd
}

func printHostModels(status *hosts.HostStatus, jsonOutput bool) {
	if jsonOutput {
		var models []string
		for _, m := range status.Models {
			models = append(models, m.Name)
		}
		data, _ := json.MarshalIndent(models, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("%s Models on %s\n", tui.RenderHeader(""), status.Host.Name)
	fmt.Println()

	if len(status.Models) == 0 {
		fmt.Println(tui.MutedStyle.Render("  No models installed"))
		return
	}

	for _, m := range status.Models {
		sizeGB := float64(m.Size) / (1024 * 1024 * 1024)
		fmt.Printf("  %s\n", m.Name)
		fmt.Printf("    %s\n", tui.MutedStyle.Render(fmt.Sprintf("%.1f GB, %s, %s",
			sizeGB,
			m.Details.ParameterSize,
			m.Details.QuantizationLevel)))
	}
}

func groupByFamily(names []string) map[string][]string {
	families := make(map[string][]string)

	for _, name := range names {
		family := extractFamily(name)
		families[family] = append(families[family], name)
	}

	return families
}

func extractFamily(name string) string {
	// Extract family from model name like "qwen2.5-coder:7b" -> "qwen2.5-coder"
	// or "llama3.1:8b" -> "llama3.1"
	for i, c := range name {
		if c == ':' {
			return name[:i]
		}
	}
	return name
}

func checkTierModel(allModels map[string][]string, tier, model string) {
	hosts, found := allModels[model]
	if found {
		fmt.Printf("  %s %s tier: %s available on %v\n",
			tui.SuccessStyle.Render("✓"),
			tier,
			model,
			hosts)
	} else {
		fmt.Printf("  %s %s tier: %s not found on any host\n",
			tui.ErrorStyle.Render("✗"),
			tier,
			model)
	}
}
