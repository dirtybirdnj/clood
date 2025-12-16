package commands

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

type ServiceStatus struct {
	Name    string
	URL     string
	Status  string
	Latency time.Duration
}

func HealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check status of all clood services",
		Long:  "Comprehensive health check of Ollama hosts, services, and CLI tools.",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading config: " + err.Error()))
				return
			}

			// Ollama Hosts
			fmt.Println(tui.RenderHeader("Ollama Hosts"))
			fmt.Println()

			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			statuses := mgr.CheckAllHosts()
			for _, status := range statuses {
				printServiceStatusCompact(status.Host.Name, status.Online, status.Latency, status.Error)
			}

			// Additional Services
			fmt.Println()
			fmt.Println(tui.RenderHeader("Additional Services"))
			fmt.Println()

			services := []struct {
				name string
				url  string
			}{
				{"SearXNG", "http://192.168.4.63:8888/healthz"},
				{"LiteLLM", "http://localhost:4000/health"},
			}

			for _, svc := range services {
				status := checkService(svc.url)
				printServiceStatusCompact(svc.name, status.Status == "healthy", status.Latency, nil)
			}

			// CLI Tools
			fmt.Println()
			fmt.Println(tui.RenderHeader("CLI Tools"))
			fmt.Println()
			checkCLIToolReal("mods")
			checkCLIToolReal("crush")
			checkCLIToolReal("aider")
			checkCLIToolReal("gh")
			checkCLIToolReal("ollama")

			// Config Status
			fmt.Println()
			fmt.Println(tui.RenderHeader("Configuration"))
			fmt.Println()
			fmt.Printf("  Config file:  %s\n", config.ConfigPath())
			if config.Exists() {
				fmt.Printf("  Status:       %s\n", tui.SuccessStyle.Render("exists"))
			} else {
				fmt.Printf("  Status:       %s\n", tui.MutedStyle.Render("using defaults"))
			}
			fmt.Printf("  Fast model:   %s\n", cfg.Tiers.Fast.Model)
			fmt.Printf("  Deep model:   %s\n", cfg.Tiers.Deep.Model)
		},
	}

	return cmd
}

func checkService(url string) ServiceStatus {
	status := ServiceStatus{URL: url}

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(url)
	status.Latency = time.Since(start)

	if err != nil {
		status.Status = "offline"
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		status.Status = "healthy"
	} else {
		status.Status = fmt.Sprintf("error (%d)", resp.StatusCode)
	}

	return status
}

func printServiceStatusCompact(name string, online bool, latency time.Duration, err error) {
	var indicator string

	if online {
		indicator = tui.SuccessStyle.Render("●")
		latencyStr := tui.MutedStyle.Render(fmt.Sprintf("(%dms)", latency.Milliseconds()))
		fmt.Printf("  %s %-20s %s %s\n", indicator, name, tui.SuccessStyle.Render("online"), latencyStr)
	} else {
		indicator = tui.ErrorStyle.Render("○")
		errStr := ""
		if err != nil {
			errStr = tui.MutedStyle.Render(fmt.Sprintf(" - %s", err.Error()))
		}
		fmt.Printf("  %s %-20s %s%s\n", indicator, name, tui.ErrorStyle.Render("offline"), errStr)
	}
}

func checkCLIToolReal(name string) {
	path, err := exec.LookPath(name)
	if err != nil {
		fmt.Printf("  %s %s: %s\n",
			tui.ErrorStyle.Render("○"),
			name,
			tui.ErrorStyle.Render("not found"))
		return
	}

	fmt.Printf("  %s %s: %s\n",
		tui.SuccessStyle.Render("●"),
		name,
		tui.MutedStyle.Render(path))
}
