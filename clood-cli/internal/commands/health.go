package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

type ServiceStatus struct {
	Name    string
	URL     string
	Status  string
	Latency time.Duration
}

// HealthReport is the JSON output structure
type HealthReport struct {
	Timestamp   string            `json:"timestamp"`
	OllamaHosts []HostHealth      `json:"ollama_hosts"`
	Services    []ServiceHealth   `json:"services"`
	CLITools    []CLIToolHealth   `json:"cli_tools"`
	Config      ConfigHealth      `json:"config"`
}

type HostHealth struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Online    bool   `json:"online"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type ServiceHealth struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
}

type CLIToolHealth struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
}

type ConfigHealth struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	FastModel string `json:"fast_model"`
	DeepModel string `json:"deep_model"`
}

func HealthCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check status of all clood services",
		Long:  "Comprehensive health check of Ollama hosts, services, and CLI tools.",
		Run: func(cmd *cobra.Command, args []string) {
			// Check both local --json and global -j flag
			useJSON := jsonOutput || output.IsJSON()

			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error loading config: " + err.Error()))
				return
			}

			// Build report structure
			report := HealthReport{
				Timestamp: time.Now().Format(time.RFC3339),
			}

			// Ollama Hosts
			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)
			statuses := mgr.CheckAllHosts()

			for _, status := range statuses {
				hh := HostHealth{
					Name:      status.Host.Name,
					URL:       status.Host.URL,
					Online:    status.Online,
					LatencyMs: status.Latency.Milliseconds(),
				}
				if status.Error != nil {
					hh.Error = status.Error.Error()
				}
				report.OllamaHosts = append(report.OllamaHosts, hh)
			}

			// Additional Services
			services := []struct {
				name string
				url  string
			}{
				{"SearXNG", "http://192.168.4.63:8888/healthz"},
				{"LiteLLM", "http://localhost:4000/health"},
			}

			for _, svc := range services {
				status := checkService(svc.url)
				report.Services = append(report.Services, ServiceHealth{
					Name:      svc.name,
					URL:       svc.url,
					Status:    status.Status,
					LatencyMs: status.Latency.Milliseconds(),
				})
			}

			// CLI Tools
			cliTools := []string{"mods", "crush", "aider", "gh", "ollama"}
			for _, tool := range cliTools {
				path, err := exec.LookPath(tool)
				th := CLIToolHealth{
					Name:      tool,
					Installed: err == nil,
				}
				if err == nil {
					th.Path = path
				}
				report.CLITools = append(report.CLITools, th)
			}

			// Config Status
			report.Config = ConfigHealth{
				Path:      config.ConfigPath(),
				Exists:    config.Exists(),
				FastModel: cfg.Tiers.Fast.Model,
				DeepModel: cfg.Tiers.Deep.Model,
			}

			// JSON output
			if useJSON {
				data, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Human-readable output
			fmt.Println(tui.RenderHeader("Ollama Hosts"))
			fmt.Println()
			for _, hh := range report.OllamaHosts {
				var err error
				if hh.Error != "" {
					err = fmt.Errorf("%s", hh.Error)
				}
				printServiceStatusCompact(hh.Name, hh.Online, time.Duration(hh.LatencyMs)*time.Millisecond, err)
			}

			fmt.Println()
			fmt.Println(tui.RenderHeader("Additional Services"))
			fmt.Println()
			for _, sh := range report.Services {
				printServiceStatusCompact(sh.Name, sh.Status == "healthy", time.Duration(sh.LatencyMs)*time.Millisecond, nil)
			}

			fmt.Println()
			fmt.Println(tui.RenderHeader("CLI Tools"))
			fmt.Println()
			for _, th := range report.CLITools {
				if th.Installed {
					fmt.Printf("  %s %s: %s\n",
						tui.SuccessStyle.Render("●"),
						th.Name,
						tui.MutedStyle.Render(th.Path))
				} else {
					fmt.Printf("  %s %s: %s\n",
						tui.ErrorStyle.Render("○"),
						th.Name,
						tui.ErrorStyle.Render("not found"))
				}
			}

			fmt.Println()
			fmt.Println(tui.RenderHeader("Configuration"))
			fmt.Println()
			fmt.Printf("  Config file:  %s\n", report.Config.Path)
			if report.Config.Exists {
				fmt.Printf("  Status:       %s\n", tui.SuccessStyle.Render("exists"))
			} else {
				fmt.Printf("  Status:       %s\n", tui.MutedStyle.Render("using defaults"))
			}
			fmt.Printf("  Fast model:   %s\n", report.Config.FastModel)
			fmt.Printf("  Deep model:   %s\n", report.Config.DeepModel)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

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

