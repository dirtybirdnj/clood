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
	var showStorage bool

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

			// If --storage --json, just output storage report
			if showStorage && useJSON {
				printStorageReport(mgr, true)
				return
			}

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

			// Show storage if requested
			if showStorage {
				fmt.Println()
				printStorageReport(mgr, useJSON)
			}
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVarP(&hostFilter, "host", "H", "", "Show models for specific host")
	cmd.Flags().BoolVar(&showStorage, "storage", false, "Show storage usage by host")

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

// StorageReport holds storage information for JSON output
type StorageReport struct {
	Hosts      []HostStorage `json:"hosts"`
	TotalBytes int64         `json:"total_bytes"`
	TotalGB    float64       `json:"total_gb"`
}

// HostStorage holds per-host storage info
type HostStorage struct {
	Name       string  `json:"name"`
	Online     bool    `json:"online"`
	ModelCount int     `json:"model_count"`
	Bytes      int64   `json:"bytes"`
	GB         float64 `json:"gb"`
}

func printStorageReport(mgr *hosts.Manager, jsonOutput bool) {
	statuses := mgr.CheckAllHosts()

	var report StorageReport
	var totalBytes int64

	for _, status := range statuses {
		hs := HostStorage{
			Name:   status.Host.Name,
			Online: status.Online,
		}

		if status.Online {
			var hostBytes int64
			for _, model := range status.Models {
				hostBytes += model.Size
			}
			hs.ModelCount = len(status.Models)
			hs.Bytes = hostBytes
			hs.GB = float64(hostBytes) / (1024 * 1024 * 1024)
			totalBytes += hostBytes
		}

		report.Hosts = append(report.Hosts, hs)
	}

	report.TotalBytes = totalBytes
	report.TotalGB = float64(totalBytes) / (1024 * 1024 * 1024)

	if jsonOutput {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println(tui.RenderHeader("Storage Usage"))
	fmt.Println()

	// Per-host storage
	for _, hs := range report.Hosts {
		if !hs.Online {
			fmt.Printf("  %s %s\n",
				tui.ErrorStyle.Render("✗"),
				tui.MutedStyle.Render(hs.Name+" (offline)"))
			continue
		}

		// Create visual bar
		bar := renderStorageBar(hs.GB, 50) // max 50GB for bar scale
		fmt.Printf("  %s %s\n", hs.Name, tui.MutedStyle.Render(fmt.Sprintf("(%d models)", hs.ModelCount)))
		fmt.Printf("    %s %.1f GB\n", bar, hs.GB)
	}

	fmt.Println()
	fmt.Printf("  %s %.1f GB\n",
		tui.HeaderStyle.Render("Total:"),
		report.TotalGB)
}

func renderStorageBar(gb float64, maxGB float64) string {
	const barWidth = 20
	filled := int((gb / maxGB) * barWidth)
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}
