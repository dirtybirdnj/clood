package commands

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/system"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// DiagnosticResult represents a single diagnostic check
type DiagnosticResult struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	Status      string `json:"status"` // "ok", "warning", "error"
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

// DoctorReport is the full diagnostic report
type DoctorReport struct {
	Hardware    HardwareDiag  `json:"hardware"`
	Ollama      OllamaDiag    `json:"ollama"`
	Config      ConfigDiag    `json:"config"`
	CLITools    []ToolDiag    `json:"cli_tools"`
	Diagnostics []DiagnosticResult `json:"diagnostics"`
	Summary     SummaryDiag   `json:"summary"`
}

type HardwareDiag struct {
	CPU       string  `json:"cpu"`
	Cores     int     `json:"cores"`
	MemoryGB  float64 `json:"memory_gb"`
	GPUType   string  `json:"gpu_type"`
	VRAMGB    float64 `json:"vram_gb"`
	Status    string  `json:"status"`
}

type OllamaDiag struct {
	Installed    bool     `json:"installed"`
	Running      bool     `json:"running"`
	HostsOnline  int      `json:"hosts_online"`
	HostsTotal   int      `json:"hosts_total"`
	ModelsCount  int      `json:"models_count"`
	Models       []string `json:"models,omitempty"`
	Status       string   `json:"status"`
}

type ConfigDiag struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	Valid     bool   `json:"valid"`
	FastModel string `json:"fast_model"`
	DeepModel string `json:"deep_model"`
	Status    string `json:"status"`
}

type ToolDiag struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
}

type SummaryDiag struct {
	Status    string `json:"status"` // "healthy", "degraded", "unhealthy"
	OkCount   int    `json:"ok_count"`
	WarnCount int    `json:"warning_count"`
	ErrCount  int    `json:"error_count"`
}

func DoctorCmd() *cobra.Command {
	var jsonOutput bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose clood setup and suggest fixes",
		Long: `Comprehensive diagnostic of your clood setup.

Checks:
  • Hardware (CPU, memory, GPU/VRAM)
  • Ollama (installed, running, hosts reachable)
  • Config (valid, models available)
  • CLI tools (mods, gh, ollama, etc.)

Provides actionable recommendations with exact commands to fix issues.`,
		Run: func(cmd *cobra.Command, args []string) {
			useJSON := jsonOutput || output.IsJSON()
			report := runDiagnostics(verbose)

			if useJSON {
				data, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(data))
				return
			}

			printDoctorReport(report, verbose)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed diagnostics")

	return cmd
}

func runDiagnostics(verbose bool) DoctorReport {
	var report DoctorReport
	var diagnostics []DiagnosticResult

	// Hardware check
	hw, err := system.DetectHardware()
	if err == nil {
		gpuType := "none"
		if hw.GPU != nil {
			gpuType = hw.GPU.Type
		}
		report.Hardware = HardwareDiag{
			CPU:      hw.CPUModel,
			Cores:    hw.CPUCores,
			MemoryGB: hw.MemoryGB,
			GPUType:  gpuType,
			VRAMGB:   hw.OllamaVRAM,
		}

		// Hardware diagnostics
		if hw.OllamaVRAM >= 8 {
			report.Hardware.Status = "ok"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category: "Hardware",
				Name:     "GPU/VRAM",
				Status:   "ok",
				Message:  fmt.Sprintf("%.0fGB available - can run 7B+ models", hw.OllamaVRAM),
			})
		} else if hw.OllamaVRAM >= 4 {
			report.Hardware.Status = "ok"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category: "Hardware",
				Name:     "GPU/VRAM",
				Status:   "ok",
				Message:  fmt.Sprintf("%.0fGB available - can run 3B models", hw.OllamaVRAM),
			})
		} else if hw.OllamaVRAM > 0 {
			report.Hardware.Status = "warning"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category:    "Hardware",
				Name:        "GPU/VRAM",
				Status:      "warning",
				Message:     fmt.Sprintf("%.0fGB limited - stick to small models", hw.OllamaVRAM),
				Remediation: "Use qwen2.5-coder:1.5b or tinyllama",
			})
		} else {
			report.Hardware.Status = "warning"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category:    "Hardware",
				Name:        "GPU/VRAM",
				Status:      "warning",
				Message:     "No GPU detected - CPU inference will be slow",
				Remediation: "Consider running on a machine with GPU",
			})
		}

		if hw.MemoryGB >= 16 {
			diagnostics = append(diagnostics, DiagnosticResult{
				Category: "Hardware",
				Name:     "Memory",
				Status:   "ok",
				Message:  fmt.Sprintf("%.0fGB RAM available", hw.MemoryGB),
			})
		} else if hw.MemoryGB >= 8 {
			diagnostics = append(diagnostics, DiagnosticResult{
				Category: "Hardware",
				Name:     "Memory",
				Status:   "warning",
				Message:  fmt.Sprintf("%.0fGB RAM - may limit context size", hw.MemoryGB),
			})
		} else {
			diagnostics = append(diagnostics, DiagnosticResult{
				Category:    "Hardware",
				Name:        "Memory",
				Status:      "error",
				Message:     fmt.Sprintf("%.0fGB RAM - insufficient for most models", hw.MemoryGB),
				Remediation: "Upgrade RAM or use remote hosts",
			})
		}
	} else {
		report.Hardware.Status = "error"
		diagnostics = append(diagnostics, DiagnosticResult{
			Category: "Hardware",
			Name:     "Detection",
			Status:   "error",
			Message:  "Failed to detect hardware: " + err.Error(),
		})
	}

	// Ollama check
	_, ollamaErr := exec.LookPath("ollama")
	report.Ollama.Installed = ollamaErr == nil

	if !report.Ollama.Installed {
		report.Ollama.Status = "error"
		diagnostics = append(diagnostics, DiagnosticResult{
			Category:    "Ollama",
			Name:        "Installation",
			Status:      "error",
			Message:     "Ollama not found in PATH",
			Remediation: "Install: curl -fsSL https://ollama.ai/install.sh | sh",
		})
	} else {
		diagnostics = append(diagnostics, DiagnosticResult{
			Category: "Ollama",
			Name:     "Installation",
			Status:   "ok",
			Message:  "Ollama is installed",
		})
	}

	// Config check
	cfg, cfgErr := config.Load()
	report.Config.Path = config.ConfigPath()
	report.Config.Exists = config.Exists()

	if cfgErr != nil {
		report.Config.Valid = false
		report.Config.Status = "error"
		diagnostics = append(diagnostics, DiagnosticResult{
			Category:    "Config",
			Name:        "Parsing",
			Status:      "error",
			Message:     "Config error: " + cfgErr.Error(),
			Remediation: "Run: clood discover > ~/.config/clood/config.yaml",
		})
	} else {
		report.Config.Valid = true
		report.Config.FastModel = cfg.Tiers.Fast.Model
		report.Config.DeepModel = cfg.Tiers.Deep.Model

		if !report.Config.Exists {
			report.Config.Status = "warning"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category:    "Config",
				Name:        "File",
				Status:      "warning",
				Message:     "Using default config (no config.yaml)",
				Remediation: "Run: clood discover to generate config",
			})
		} else {
			report.Config.Status = "ok"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category: "Config",
				Name:     "File",
				Status:   "ok",
				Message:  "Config loaded from " + report.Config.Path,
			})
		}
	}

	// Host connectivity check
	if cfg != nil {
		mgr := hosts.NewManager()
		mgr.AddHosts(cfg.Hosts)
		statuses := mgr.CheckAllHosts()

		report.Ollama.HostsTotal = len(statuses)
		modelsSeen := make(map[string]bool)

		for _, s := range statuses {
			if s.Online {
				report.Ollama.HostsOnline++
				report.Ollama.Running = true
				for _, m := range s.Models {
					if !modelsSeen[m.Name] {
						modelsSeen[m.Name] = true
						if verbose {
							report.Ollama.Models = append(report.Ollama.Models, m.Name)
						}
					}
				}
			}
		}
		report.Ollama.ModelsCount = len(modelsSeen)

		if report.Ollama.HostsOnline == 0 {
			report.Ollama.Status = "error"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category:    "Ollama",
				Name:        "Connectivity",
				Status:      "error",
				Message:     fmt.Sprintf("No hosts online (0/%d)", report.Ollama.HostsTotal),
				Remediation: "Start Ollama: ollama serve",
			})
		} else if report.Ollama.HostsOnline < report.Ollama.HostsTotal {
			report.Ollama.Status = "warning"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category: "Ollama",
				Name:     "Connectivity",
				Status:   "warning",
				Message:  fmt.Sprintf("%d/%d hosts online", report.Ollama.HostsOnline, report.Ollama.HostsTotal),
			})
		} else {
			report.Ollama.Status = "ok"
			diagnostics = append(diagnostics, DiagnosticResult{
				Category: "Ollama",
				Name:     "Connectivity",
				Status:   "ok",
				Message:  fmt.Sprintf("All %d hosts online with %d models", report.Ollama.HostsOnline, report.Ollama.ModelsCount),
			})
		}

		// Check tier models availability
		if report.Ollama.ModelsCount > 0 {
			allModels := mgr.GetAllModels()
			if _, found := allModels[cfg.Tiers.Fast.Model]; !found {
				diagnostics = append(diagnostics, DiagnosticResult{
					Category:    "Config",
					Name:        "Fast Model",
					Status:      "warning",
					Message:     fmt.Sprintf("Fast tier model '%s' not found on any host", cfg.Tiers.Fast.Model),
					Remediation: fmt.Sprintf("Pull it: ollama pull %s", cfg.Tiers.Fast.Model),
				})
			}
			if _, found := allModels[cfg.Tiers.Deep.Model]; !found {
				diagnostics = append(diagnostics, DiagnosticResult{
					Category:    "Config",
					Name:        "Deep Model",
					Status:      "warning",
					Message:     fmt.Sprintf("Deep tier model '%s' not found on any host", cfg.Tiers.Deep.Model),
					Remediation: fmt.Sprintf("Pull it: ollama pull %s", cfg.Tiers.Deep.Model),
				})
			}
		}
	}

	// CLI tools check
	cliTools := []string{"mods", "gh", "ollama"}
	for _, tool := range cliTools {
		path, err := exec.LookPath(tool)
		td := ToolDiag{Name: tool, Installed: err == nil}
		if err == nil {
			td.Path = path
		}
		report.CLITools = append(report.CLITools, td)
	}

	report.Diagnostics = diagnostics

	// Calculate summary
	for _, d := range diagnostics {
		switch d.Status {
		case "ok":
			report.Summary.OkCount++
		case "warning":
			report.Summary.WarnCount++
		case "error":
			report.Summary.ErrCount++
		}
	}

	if report.Summary.ErrCount > 0 {
		report.Summary.Status = "unhealthy"
	} else if report.Summary.WarnCount > 0 {
		report.Summary.Status = "degraded"
	} else {
		report.Summary.Status = "healthy"
	}

	return report
}

func printDoctorReport(report DoctorReport, verbose bool) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("CLOOD DOCTOR"))
	fmt.Println()

	// Hardware section
	fmt.Println(tui.AccentStyle.Render("  HARDWARE"))
	printDiagLine("CPU", report.Hardware.CPU, report.Hardware.Status)
	printDiagLine("Cores", fmt.Sprintf("%d", report.Hardware.Cores), "ok")
	printDiagLine("Memory", fmt.Sprintf("%.0fGB", report.Hardware.MemoryGB), statusFromValue(report.Hardware.MemoryGB, 16, 8))
	if report.Hardware.GPUType != "" {
		printDiagLine("GPU", report.Hardware.GPUType, "ok")
	}
	printDiagLine("VRAM", fmt.Sprintf("%.0fGB", report.Hardware.VRAMGB), statusFromValue(report.Hardware.VRAMGB, 8, 4))
	fmt.Println()

	// Ollama section
	fmt.Println(tui.AccentStyle.Render("  OLLAMA"))
	printDiagBool("Installed", report.Ollama.Installed)
	if report.Ollama.HostsTotal > 0 {
		printDiagLine("Hosts", fmt.Sprintf("%d/%d online", report.Ollama.HostsOnline, report.Ollama.HostsTotal), report.Ollama.Status)
		printDiagLine("Models", fmt.Sprintf("%d available", report.Ollama.ModelsCount), statusFromValue(float64(report.Ollama.ModelsCount), 3, 1))
	}
	fmt.Println()

	// Config section
	fmt.Println(tui.AccentStyle.Render("  CONFIG"))
	printDiagBool("Exists", report.Config.Exists)
	if report.Config.Valid {
		printDiagLine("Fast tier", report.Config.FastModel, "ok")
		printDiagLine("Deep tier", report.Config.DeepModel, "ok")
	}
	fmt.Println()

	// CLI Tools section
	fmt.Println(tui.AccentStyle.Render("  CLI TOOLS"))
	for _, t := range report.CLITools {
		printDiagBool(t.Name, t.Installed)
	}
	fmt.Println()

	// Issues and recommendations
	hasIssues := false
	for _, d := range report.Diagnostics {
		if d.Status != "ok" {
			if !hasIssues {
				fmt.Println(tui.AccentStyle.Render("  ISSUES & RECOMMENDATIONS"))
				hasIssues = true
			}
			var icon string
			if d.Status == "error" {
				icon = tui.ErrorStyle.Render("✗")
			} else {
				icon = tui.WarningStyle.Render("⚠")
			}
			fmt.Printf("    %s %s: %s\n", icon, d.Name, d.Message)
			if d.Remediation != "" {
				fmt.Printf("      → %s\n", tui.MutedStyle.Render(d.Remediation))
			}
		}
	}

	if !hasIssues {
		fmt.Println(tui.SuccessStyle.Render("  ✓ No issues found - clood is healthy!"))
	}

	fmt.Println()

	// Summary bar
	var statusStr string
	switch report.Summary.Status {
	case "healthy":
		statusStr = tui.SuccessStyle.Render(report.Summary.Status)
	case "degraded":
		statusStr = tui.WarningStyle.Render(report.Summary.Status)
	default:
		statusStr = tui.ErrorStyle.Render(report.Summary.Status)
	}

	fmt.Println("  ─────────────────────────────────────────────────────────")
	fmt.Printf("  Status: %s  │  ", statusStr)
	fmt.Printf("%s %d ok  ", tui.SuccessStyle.Render("●"), report.Summary.OkCount)
	if report.Summary.WarnCount > 0 {
		fmt.Printf("%s %d warnings  ", tui.WarningStyle.Render("●"), report.Summary.WarnCount)
	}
	if report.Summary.ErrCount > 0 {
		fmt.Printf("%s %d errors", tui.ErrorStyle.Render("●"), report.Summary.ErrCount)
	}
	fmt.Println()
	fmt.Println()
}

func printDiagLine(name, value, status string) {
	var icon string
	switch status {
	case "ok":
		icon = tui.SuccessStyle.Render("✓")
	case "warning":
		icon = tui.WarningStyle.Render("⚠")
	case "error":
		icon = tui.ErrorStyle.Render("✗")
	default:
		icon = tui.MutedStyle.Render("○")
	}
	fmt.Printf("    %s %-12s %s\n", icon, name+":", value)
}

func printDiagBool(name string, ok bool) {
	if ok {
		fmt.Printf("    %s %-12s %s\n", tui.SuccessStyle.Render("✓"), name+":", "yes")
	} else {
		fmt.Printf("    %s %-12s %s\n", tui.ErrorStyle.Render("✗"), name+":", "no")
	}
}

func statusFromValue(value, good, ok float64) string {
	if value >= good {
		return "ok"
	} else if value >= ok {
		return "warning"
	}
	return "error"
}
