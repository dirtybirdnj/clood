package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func HostsCmd() *cobra.Command {
	var jsonOutput bool
	var gardenView bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "hosts",
		Short: "List and check Ollama hosts",
		Long: `Lists all configured Ollama hosts and checks their status.
Shows which hosts are online, their latency, and available models.

Use --garden for ASCII art visualization of the Server Garden topology.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading config: " + err.Error()))
				return
			}

			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			// Only show progress message in human mode
			if !jsonOutput && !output.IsJSON() {
				fmt.Println(tui.MutedStyle.Render("Checking hosts..."))
				fmt.Println()
			}

			statuses := mgr.CheckAllHosts()

			// Detect if localhost is same as a named host
			localAlias := detectLocalAlias(statuses)

			// Check both local --json flag and global -j flag
			if jsonOutput || output.IsJSON() {
				if gardenView {
					printGardenJSON(statuses, verbose)
				} else {
					printHostsJSON(statuses)
				}
				return
			}

			// Garden view - ASCII art topology
			if gardenView {
				printGardenVisual(statuses, verbose)
				return
			}

			// Default list view
			fmt.Println(tui.RenderHeader("Ollama Hosts"))
			fmt.Println()

			for _, status := range statuses {
				printHostStatusWithAlias(status, localAlias)
			}

			// Summary - don't double-count if localhost is an alias
			online := 0
			totalModels := 0
			modelsSeen := make(map[string]bool)
			for _, s := range statuses {
				if s.Online {
					// Skip localhost in count if it's an alias
					if localAlias != "" && s.Host.Name == "localhost" {
						continue
					}
					online++
					for _, m := range s.Models {
						if !modelsSeen[m.Name] {
							modelsSeen[m.Name] = true
							totalModels++
						}
					}
				}
			}

			fmt.Println()
			totalHosts := len(statuses)
			if localAlias != "" {
				totalHosts-- // Don't count localhost separately if it's an alias
			}
			fmt.Printf("  %s %d/%d hosts online, %d unique models\n",
				tui.MutedStyle.Render("Summary:"),
				online,
				totalHosts,
				totalModels)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&gardenView, "garden", false, "Show ASCII art garden visualization")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show all model details")

	return cmd
}

// detectLocalAlias checks if localhost is the same Ollama instance as a named host
func detectLocalAlias(statuses []*hosts.HostStatus) string {
	// First, check by hostname match
	hostname, _ := os.Hostname()
	hostname = strings.ToLower(strings.Split(hostname, ".")[0]) // Get short hostname

	// Find localhost and other host statuses
	var localStatus *hosts.HostStatus
	var namedStatuses []*hosts.HostStatus

	for _, s := range statuses {
		if s.Host.Name == "localhost" || strings.HasPrefix(s.Host.Name, "local") {
			localStatus = s
		} else {
			namedStatuses = append(namedStatuses, s)
		}
	}

	if localStatus == nil || !localStatus.Online {
		return ""
	}

	// Check if hostname matches a named host
	for _, s := range namedStatuses {
		if strings.ToLower(s.Host.Name) == hostname {
			return s.Host.Name
		}
	}

	// Fallback: check if version and model count match (same instance)
	for _, s := range namedStatuses {
		if s.Online && s.Version == localStatus.Version && len(s.Models) == len(localStatus.Models) {
			// Same version and model count - likely same instance
			return s.Host.Name
		}
	}

	return ""
}

func printHostStatus(status *hosts.HostStatus) {
	printHostStatusWithAlias(status, "")
}

func printHostStatusWithAlias(status *hosts.HostStatus, localAlias string) {
	var indicator, statusText string

	if status.Online {
		indicator = tui.SuccessStyle.Render("●")
		statusText = tui.SuccessStyle.Render("online")
	} else {
		indicator = tui.ErrorStyle.Render("○")
		statusText = tui.ErrorStyle.Render("offline")
	}

	// Check if this localhost entry is an alias
	isAlias := localAlias != "" && status.Host.Name == "localhost"

	// Host name and URL
	if isAlias {
		fmt.Printf("  %s %s %s\n", indicator, status.Host.Name, tui.MutedStyle.Render(fmt.Sprintf("(= %s)", localAlias)))
	} else {
		fmt.Printf("  %s %s\n", indicator, status.Host.Name)
	}
	fmt.Printf("    %s\n", tui.MutedStyle.Render(status.Host.URL))

	if status.Online {
		// If it's an alias, just show brief info
		if isAlias {
			fmt.Printf("    %s\n", tui.MutedStyle.Render("Same instance as "+localAlias))
			fmt.Println()
			return
		}

		// Latency
		fmt.Printf("    Latency: %s\n", tui.MutedStyle.Render(fmt.Sprintf("%dms", status.Latency.Milliseconds())))

		// Version
		if status.Version != "" {
			fmt.Printf("    Version: %s\n", tui.MutedStyle.Render(status.Version))
		}

		// Models count
		if len(status.Models) > 0 {
			fmt.Printf("    Models:  %s\n", tui.MutedStyle.Render(fmt.Sprintf("%d available", len(status.Models))))
			// Show first few models
			for i, m := range status.Models {
				if i >= 3 {
					fmt.Printf("             %s\n", tui.MutedStyle.Render(fmt.Sprintf("... and %d more", len(status.Models)-3)))
					break
				}
				fmt.Printf("             %s\n", m.Name)
			}
		}
	} else {
		fmt.Printf("    Status: %s\n", statusText)
		if status.Error != nil {
			fmt.Printf("    Error:  %s\n", tui.MutedStyle.Render(status.Error.Error()))
		}
	}

	fmt.Println()
}

func printHostsJSON(statuses []*hosts.HostStatus) {
	type hostJSON struct {
		Name    string   `json:"name"`
		URL     string   `json:"url"`
		Online  bool     `json:"online"`
		Latency int64    `json:"latency_ms,omitempty"`
		Version string   `json:"version,omitempty"`
		Models  []string `json:"models,omitempty"`
		Error   string   `json:"error,omitempty"`
	}

	var result []hostJSON
	for _, s := range statuses {
		h := hostJSON{
			Name:   s.Host.Name,
			URL:    s.Host.URL,
			Online: s.Online,
		}
		if s.Online {
			h.Latency = s.Latency.Milliseconds()
			h.Version = s.Version
			for _, m := range s.Models {
				h.Models = append(h.Models, m.Name)
			}
		}
		if s.Error != nil {
			h.Error = s.Error.Error()
		}
		result = append(result, h)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

// Garden view types and functions

// GardenStatus represents host status for garden JSON output
type GardenStatus struct {
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Online     bool     `json:"online"`
	LatencyMs  int64    `json:"latency_ms"`
	ModelCount int      `json:"model_count"`
	Models     []string `json:"models,omitempty"`
}

// GardenOutput represents the full garden status with driver info
type GardenOutput struct {
	Driver  string         `json:"driver"`
	Hosts   []GardenStatus `json:"hosts"`
	Summary struct {
		Online       int `json:"online"`
		Total        int `json:"total"`
		UniqueModels int `json:"unique_models"`
	} `json:"summary"`
}

func printGardenVisual(statuses []*hosts.HostStatus, verbose bool) {
	fmt.Println()

	// ASCII garden visualization
	fmt.Println("  ┌─────────────────────────────────────────────────────────┐")
	fmt.Println("  │                    🌳 Server Garden 🌳                   │")
	fmt.Println("  └─────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Driver (local machine)
	hostname, _ := os.Hostname()
	fmt.Printf("  ┌──────────────────┐\n")
	fmt.Printf("  │  🖥️  %-12s │  DRIVER\n", truncateGarden(hostname, 12))
	fmt.Printf("  │  clood hosts     │\n")
	fmt.Printf("  └────────┬─────────┘\n")
	fmt.Println("           │")

	// Count online hosts
	online := 0
	for _, s := range statuses {
		if s.Online {
			online++
		}
	}

	if len(statuses) == 0 {
		fmt.Println("           │")
		fmt.Println("      (no hosts configured)")
		return
	}

	// Connection lines
	if len(statuses) > 1 {
		fmt.Println("     ┌─────┴─────┐")
	} else {
		fmt.Println("           │")
	}

	// Draw hosts
	for i, status := range statuses {
		if i > 0 {
			fmt.Println()
		}
		printHostBox(status, verbose)
	}

	// Summary
	fmt.Println()
	fmt.Println("  ─────────────────────────────────────────────────────────")

	uniqueModels := make(map[string]bool)
	for _, s := range statuses {
		for _, m := range s.Models {
			uniqueModels[m.Name] = true
		}
	}

	statusText := fmt.Sprintf("%d/%d hosts online", online, len(statuses))
	if online == len(statuses) {
		fmt.Printf("  %s  •  %d unique models\n",
			tui.SuccessStyle.Render(statusText), len(uniqueModels))
	} else if online > 0 {
		fmt.Printf("  %s  •  %d unique models\n",
			tui.MutedStyle.Render(statusText), len(uniqueModels))
	} else {
		fmt.Printf("  %s\n", tui.ErrorStyle.Render(statusText))
	}
	fmt.Println()
}

func printHostBox(status *hosts.HostStatus, verbose bool) {
	var statusColor string
	if status.Online {
		statusColor = tui.SuccessStyle.Render("● ONLINE")
	} else {
		statusColor = tui.ErrorStyle.Render("○ OFFLINE")
	}

	fmt.Printf("  ┌──────────────────┐\n")
	fmt.Printf("  │  %-14s  │\n", truncateGarden(status.Host.Name, 14))
	fmt.Printf("  │  %s       │\n", statusColor)

	if status.Online {
		fmt.Printf("  │  %d models        │\n", len(status.Models))
		fmt.Printf("  │  %dms latency    │\n", status.Latency.Milliseconds())
		if status.Version != "" {
			fmt.Printf("  │  v%-13s │\n", truncateGarden(status.Version, 13))
		}
	}
	fmt.Printf("  └──────────────────┘\n")

	if verbose && status.Online && len(status.Models) > 0 {
		fmt.Println("     Models:")
		for _, m := range status.Models {
			fmt.Printf("       • %s\n", m.Name)
		}
	}
}

func printGardenJSON(statuses []*hosts.HostStatus, verbose bool) {
	output := GardenOutput{}

	hostname, _ := os.Hostname()
	output.Driver = hostname

	uniqueModels := make(map[string]bool)

	for _, s := range statuses {
		gs := GardenStatus{
			Name:       s.Host.Name,
			URL:        s.Host.URL,
			Online:     s.Online,
			LatencyMs:  s.Latency.Milliseconds(),
			ModelCount: len(s.Models),
		}

		if verbose {
			for _, m := range s.Models {
				gs.Models = append(gs.Models, m.Name)
				uniqueModels[m.Name] = true
			}
		} else {
			for _, m := range s.Models {
				uniqueModels[m.Name] = true
			}
		}

		if s.Online {
			output.Summary.Online++
		}

		output.Hosts = append(output.Hosts, gs)
	}

	output.Summary.Total = len(statuses)
	output.Summary.UniqueModels = len(uniqueModels)

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func truncateGarden(s string, max int) string {
	if len(s) <= max {
		return s + strings.Repeat(" ", max-len(s))
	}
	return s[:max-1] + "…"
}
