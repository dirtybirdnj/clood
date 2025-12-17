package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Recommended models by tier
var recommendedModels = map[string][]string{
	"fast":     {"qwen2.5-coder:3b", "tinyllama:latest"},
	"deep":     {"qwen2.5-coder:7b", "llama3.1:8b"},
	"analysis": {"deepseek-r1:14b", "llama3.1:8b"},
	"writing":  {"llama3.1:8b", "mistral:7b"},
}

// Estimated model sizes in GB (approximate, varies by quantization)
var modelSizes = map[string]float64{
	"tinyllama:latest":      0.6,
	"qwen2.5-coder:3b":      1.9,
	"llama3.2:3b":           2.0,
	"qwen2.5-coder:7b":      4.4,
	"llama3.1:8b":           4.7,
	"mistral:7b":            4.1,
	"llama3-groq-tool-use:8b": 4.7,
	"deepseek-coder:6.7b":   3.8,
	"deepseek-r1:14b":       8.5,
	"phi4-reasoning:14b":    8.0,
	"qwen2.5-coder:14b":     8.9,
	"qwen2.5-coder:32b":     19.0,
}

// DiskInfo contains disk usage information
type DiskInfo struct {
	Total     int64
	Used      int64
	Available int64
	UsedPct   int
	Path      string
}

// getDiskInfo returns disk usage for a path (or default Ollama models path)
func getDiskInfo(path string) (*DiskInfo, error) {
	if path == "" {
		path = "/"
	}

	// Use df command for cross-platform compatibility
	out, err := exec.Command("df", "-k", path).Output()
	if err != nil {
		return nil, fmt.Errorf("df failed: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected df output")
	}

	// Parse second line (first is header)
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return nil, fmt.Errorf("unexpected df format")
	}

	// Fields: Filesystem, 1024-blocks, Used, Available, Capacity/Use%, ...
	total, _ := strconv.ParseInt(fields[1], 10, 64)
	used, _ := strconv.ParseInt(fields[2], 10, 64)
	avail, _ := strconv.ParseInt(fields[3], 10, 64)
	usedPct := 0
	// Look for percentage in remaining fields (position varies by OS)
	for i := 4; i < len(fields); i++ {
		if strings.HasSuffix(fields[i], "%") {
			pct := strings.TrimSuffix(fields[i], "%")
			usedPct, _ = strconv.Atoi(pct)
			break
		}
	}

	return &DiskInfo{
		Total:     total * 1024, // Convert from KB to bytes
		Used:      used * 1024,
		Available: avail * 1024,
		UsedPct:   usedPct,
		Path:      path,
	}, nil
}

// formatBytes formats bytes as human-readable string
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// getModelSize returns estimated size for a model
func getModelSize(model string) float64 {
	if size, ok := modelSizes[model]; ok {
		return size
	}
	// Estimate based on parameter count in name
	if strings.Contains(model, "70b") || strings.Contains(model, "72b") {
		return 40.0
	}
	if strings.Contains(model, "32b") || strings.Contains(model, "34b") {
		return 19.0
	}
	if strings.Contains(model, "14b") || strings.Contains(model, "13b") {
		return 8.0
	}
	if strings.Contains(model, "7b") || strings.Contains(model, "8b") {
		return 4.5
	}
	if strings.Contains(model, "3b") {
		return 2.0
	}
	if strings.Contains(model, "1b") {
		return 0.7
	}
	return 5.0 // Default estimate
}

func PullCmd() *cobra.Command {
	var hostName string
	var tier string
	var showRecommend bool

	cmd := &cobra.Command{
		Use:   "pull [MODEL]",
		Short: "Pull a model to a host",
		Long: `Pull (download) a model to an Ollama host.

Examples:
  clood pull qwen2.5-coder:7b              # Pull to best available host
  clood pull --host ubuntu25 llama3.1:8b   # Pull to specific host
  clood pull --tier analysis               # Pull all models for analysis tier
  clood pull --recommend                   # Show recommended models`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Show recommendations
			if showRecommend {
				return showRecommendations(cfg)
			}

			// Pull tier models
			if tier != "" {
				return pullTierModels(cfg, tier, hostName)
			}

			// Need a model name
			if len(args) == 0 {
				return fmt.Errorf("specify a model name or use --tier/--recommend")
			}

			model := args[0]
			return pullModel(cfg, model, hostName)
		},
	}

	cmd.Flags().StringVar(&hostName, "host", "", "Target host (default: best available)")
	cmd.Flags().StringVar(&tier, "tier", "", "Pull all models for tier (fast, deep, analysis, writing)")
	cmd.Flags().BoolVar(&showRecommend, "recommend", false, "Show recommended models")

	return cmd
}

func showRecommendations(cfg *config.Config) error {
	fmt.Println(tui.RenderHeader("Recommended Models"))
	fmt.Println()

	// Check what's available
	mgr := hosts.NewManager()
	mgr.AddHosts(cfg.Hosts)
	statuses := mgr.CheckAllHosts()

	// Build set of available models
	available := make(map[string][]string)
	for _, status := range statuses {
		if !status.Online {
			continue
		}
		for _, m := range status.Models {
			available[m.Name] = append(available[m.Name], status.Host.Name)
		}
	}

	// Show by tier with sizes
	tiers := []string{"fast", "deep", "analysis", "writing"}
	var missingSize float64
	for _, t := range tiers {
		models := recommendedModels[t]
		fmt.Printf("  %s tier:\n", tui.SuccessStyle.Render(t))
		for _, m := range models {
			size := getModelSize(m)
			sizeStr := fmt.Sprintf("~%.1fGB", size)
			if hosts, ok := available[m]; ok {
				fmt.Printf("    ✓ %-25s %s %s\n", m, tui.MutedStyle.Render(sizeStr), tui.MutedStyle.Render(fmt.Sprintf("(%v)", hosts)))
			} else {
				fmt.Printf("    ○ %-25s %s %s\n", m, tui.MutedStyle.Render(sizeStr), tui.MutedStyle.Render("(not installed)"))
				missingSize += size
			}
		}
		fmt.Println()
	}

	// Show disk info
	disk, err := getDiskInfo("")
	if err == nil {
		fmt.Println(tui.RenderHeader("Disk Usage"))
		fmt.Println()
		fmt.Printf("  Available: %s\n", formatBytes(disk.Available))
		fmt.Printf("  Used:      %s (%d%%)\n", formatBytes(disk.Used), disk.UsedPct)
		fmt.Printf("  Total:     %s\n", formatBytes(disk.Total))
		fmt.Println()

		if missingSize > 0 {
			neededBytes := int64(missingSize * 1024 * 1024 * 1024)
			if disk.Available > neededBytes {
				fmt.Printf("  %s ~%.1fGB needed for missing models, %s available\n",
					tui.SuccessStyle.Render("✓"),
					missingSize,
					formatBytes(disk.Available))
			} else {
				fmt.Printf("  %s ~%.1fGB needed but only %s available\n",
					tui.ErrorStyle.Render("⚠"),
					missingSize,
					formatBytes(disk.Available))
			}
		}
		fmt.Println()
	}

	fmt.Println(tui.MutedStyle.Render("  Use 'clood pull --tier <name>' to install tier models"))
	return nil
}

func pullTierModels(cfg *config.Config, tier, hostName string) error {
	models, ok := recommendedModels[tier]
	if !ok {
		return fmt.Errorf("unknown tier: %s (use: fast, deep, analysis, writing)", tier)
	}

	fmt.Printf("Pulling %d models for %s tier...\n\n", len(models), tier)

	for _, model := range models {
		if err := pullModel(cfg, model, hostName); err != nil {
			fmt.Printf("  %s %s: %v\n", tui.ErrorStyle.Render("✗"), model, err)
		}
	}

	return nil
}

func pullModel(cfg *config.Config, model, hostName string) error {
	mgr := hosts.NewManager()
	mgr.AddHosts(cfg.Hosts)

	var client *ollama.Client
	var targetHost string

	if hostName != "" {
		// Use specific host
		host := mgr.GetHost(hostName)
		if host == nil {
			return fmt.Errorf("host not found: %s", hostName)
		}
		client = mgr.GetClient(hostName)
		targetHost = hostName
	} else {
		// Use best available host
		best := mgr.GetBestHost()
		if best == nil {
			return fmt.Errorf("no hosts available")
		}
		client = mgr.GetClient(best.Host.Name)
		targetHost = best.Host.Name
	}

	if client == nil {
		return fmt.Errorf("could not get client for host")
	}

	// Check if already installed
	has, _ := client.HasModel(model)
	if has {
		fmt.Printf("  %s %s already installed on %s\n", tui.SuccessStyle.Render("✓"), model, targetHost)
		return nil
	}

	// Check disk space (only for localhost pulls, can't check remote)
	if targetHost == "localhost" || strings.HasPrefix(targetHost, "local") {
		modelSize := getModelSize(model)
		neededBytes := int64(modelSize * 1024 * 1024 * 1024)

		disk, err := getDiskInfo("")
		if err == nil {
			if disk.Available < neededBytes {
				return fmt.Errorf("insufficient disk space: need ~%.1fGB, only %s available",
					modelSize, formatBytes(disk.Available))
			}
			if disk.UsedPct > 90 {
				fmt.Printf("  %s Disk is %d%% full (%s available)\n",
					tui.ErrorStyle.Render("⚠"),
					disk.UsedPct,
					formatBytes(disk.Available))
			}
		}
	}

	modelSize := getModelSize(model)
	fmt.Printf("  Pulling %s (~%.1fGB) to %s...\n", model, modelSize, targetHost)

	var lastStatus string
	var lastPercent int
	startTime := time.Now()

	err := client.Pull(model, func(status string, completed, total int64) {
		if status != lastStatus {
			if lastStatus != "" {
				fmt.Println() // New line for new status
			}
			lastStatus = status
			lastPercent = -1
		}

		if total > 0 {
			percent := int(completed * 100 / total)
			if percent != lastPercent && percent%10 == 0 {
				fmt.Printf("\r    %s: %d%%", status, percent)
				lastPercent = percent
			}
		} else if status != "" {
			fmt.Printf("\r    %s", status)
		}
	})

	elapsed := time.Since(startTime)
	fmt.Println()

	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	fmt.Printf("  %s %s pulled to %s in %v\n", tui.SuccessStyle.Render("✓"), model, targetHost, elapsed.Round(time.Second))
	return nil
}
