package commands

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/system"
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

// ModelInfo holds enriched model information for display
type ModelInfo struct {
	Name              string   `json:"name"`
	SizeGB            float64  `json:"size_gb"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
	Family            string   `json:"family"`
	Category          string   `json:"category"`
	BestFor           string   `json:"best_for,omitempty"`
	Hosts             []string `json:"hosts,omitempty"`
	TokPerSec         float64  `json:"tok_per_sec,omitempty"`  // From catfight benchmarks
	BenchmarkSource   string   `json:"benchmark_source,omitempty"`
}

func printHostModels(status *hosts.HostStatus, jsonOutput bool) {
	// Load benchmark store to enrich with performance data
	benchStore, _ := system.NewBenchmarkStore()

	// Calculate total library size
	var totalBytes int64
	for _, m := range status.Models {
		totalBytes += m.Size
	}
	totalGB := float64(totalBytes) / (1024 * 1024 * 1024)

	if jsonOutput {
		var models []ModelInfo
		for _, m := range status.Models {
			class := system.ClassifyModel(m.Name)
			info := ModelInfo{
				Name:              m.Name,
				SizeGB:            float64(m.Size) / (1024 * 1024 * 1024),
				ParameterSize:     m.Details.ParameterSize,
				QuantizationLevel: m.Details.QuantizationLevel,
				Family:            m.Details.Family,
				Category:          string(class.Category),
				BestFor:           class.BestFor,
			}
			// Add benchmark data if available
			if benchStore != nil {
				if bm := benchStore.GetBenchmarkForHost(m.Name, status.Host.Name); bm != nil {
					info.TokPerSec = bm.TokPerSec
					info.BenchmarkSource = bm.Source
				}
			}
			models = append(models, info)
		}
		output := map[string]interface{}{
			"host":          status.Host.Name,
			"model_count":   len(models),
			"total_size_gb": totalGB,
			"models":        models,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("%s Models on %s\n", tui.RenderHeader(""), status.Host.Name)
	fmt.Println()

	if len(status.Models) == 0 {
		fmt.Println(tui.MutedStyle.Render("  No models installed"))
		return
	}

	// Show library summary
	fmt.Printf("  %s %d models, %.1f GB total\n\n",
		tui.AccentStyle.Render("Library:"),
		len(status.Models),
		totalGB)

	// Group models by category
	byCategory := make(map[system.ModelCategory][]struct {
		model   interface{}
		sizeGB  float64
		params  string
		quant   string
		tokSec  string
		bestFor string
	})

	// Check if we have any benchmarks
	hasBenchmarks := false
	if benchStore != nil {
		for _, m := range status.Models {
			if bm := benchStore.GetBenchmarkForHost(m.Name, status.Host.Name); bm != nil {
				hasBenchmarks = true
				break
			}
		}
	}

	// Categorize all models
	for _, m := range status.Models {
		class := system.ClassifyModel(m.Name)
		sizeGB := float64(m.Size) / (1024 * 1024 * 1024)
		paramSize := m.Details.ParameterSize
		if paramSize == "" {
			paramSize = "-"
		}
		quantLevel := m.Details.QuantizationLevel
		if quantLevel == "" {
			quantLevel = "-"
		}
		tokSec := "-"
		if benchStore != nil {
			if bm := benchStore.GetBenchmarkForHost(m.Name, status.Host.Name); bm != nil {
				tokSec = fmt.Sprintf("%.1f", bm.TokPerSec)
			}
		}

		byCategory[class.Category] = append(byCategory[class.Category], struct {
			model   interface{}
			sizeGB  float64
			params  string
			quant   string
			tokSec  string
			bestFor string
		}{
			model:   m.Name,
			sizeGB:  sizeGB,
			params:  paramSize,
			quant:   quantLevel,
			tokSec:  tokSec,
			bestFor: class.BestFor,
		})
	}

	// Display order for categories
	categoryOrder := []system.ModelCategory{
		system.CategoryCoding,
		system.CategoryReasoning,
		system.CategoryVision,
		system.CategoryGeneral,
	}

	for _, cat := range categoryOrder {
		models, ok := byCategory[cat]
		if !ok || len(models) == 0 {
			continue
		}

		info := system.GetCategoryInfo(cat)
		fmt.Printf("  %s %s %s\n", info.Emoji, tui.HeaderStyle.Render(info.Name), tui.MutedStyle.Render("- "+info.Description))

		if hasBenchmarks {
			fmt.Printf("    %-26s %6s %7s %5s %8s  %s\n",
				"MODEL", "SIZE", "PARAMS", "QUANT", "TOK/S", "BEST FOR")
			fmt.Printf("    %s\n", tui.MutedStyle.Render("─────────────────────────────────────────────────────────────────────────────"))
		} else {
			fmt.Printf("    %-26s %6s %7s %5s  %s\n",
				"MODEL", "SIZE", "PARAMS", "QUANT", "BEST FOR")
			fmt.Printf("    %s\n", tui.MutedStyle.Render("───────────────────────────────────────────────────────────────────"))
		}

		for _, m := range models {
			bestFor := m.bestFor
			if len(bestFor) > 25 {
				bestFor = bestFor[:22] + "..."
			}
			if hasBenchmarks {
				fmt.Printf("    %-26s %5.1fGB %7s %5s %8s  %s\n",
					m.model, m.sizeGB, m.params, m.quant, m.tokSec, tui.MutedStyle.Render(bestFor))
			} else {
				fmt.Printf("    %-26s %5.1fGB %7s %5s  %s\n",
					m.model, m.sizeGB, m.params, m.quant, tui.MutedStyle.Render(bestFor))
			}
		}
		fmt.Println()
	}

	if !hasBenchmarks {
		fmt.Println(tui.MutedStyle.Render("  Tip: Run 'clood catfight' to benchmark models and see tok/s"))
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
