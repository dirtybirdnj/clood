package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func HostsCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "hosts",
		Short: "List and check Ollama hosts",
		Long: `Lists all configured Ollama hosts and checks their status.
Shows which hosts are online, their latency, and available models.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading config: " + err.Error()))
				return
			}

			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			fmt.Println(tui.MutedStyle.Render("Checking hosts..."))
			fmt.Println()

			statuses := mgr.CheckAllHosts()

			// Detect if localhost is same as a named host
			localAlias := detectLocalAlias(statuses)

			if jsonOutput {
				printHostsJSON(statuses)
				return
			}

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
