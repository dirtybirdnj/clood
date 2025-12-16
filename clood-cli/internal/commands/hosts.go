package commands

import (
	"encoding/json"
	"fmt"

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

			if jsonOutput {
				printHostsJSON(statuses)
				return
			}

			fmt.Println(tui.RenderHeader("Ollama Hosts"))
			fmt.Println()

			for _, status := range statuses {
				printHostStatus(status)
			}

			// Summary
			online := 0
			totalModels := 0
			for _, s := range statuses {
				if s.Online {
					online++
					totalModels += len(s.Models)
				}
			}

			fmt.Println()
			fmt.Printf("  %s %d/%d hosts online, %d models available\n",
				tui.MutedStyle.Render("Summary:"),
				online,
				len(statuses),
				totalModels)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func printHostStatus(status *hosts.HostStatus) {
	var indicator, statusText string

	if status.Online {
		indicator = tui.SuccessStyle.Render("●")
		statusText = tui.SuccessStyle.Render("online")
	} else {
		indicator = tui.ErrorStyle.Render("○")
		statusText = tui.ErrorStyle.Render("offline")
	}

	// Host name and URL
	fmt.Printf("  %s %s\n", indicator, status.Host.Name)
	fmt.Printf("    %s\n", tui.MutedStyle.Render(status.Host.URL))

	if status.Online {
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
