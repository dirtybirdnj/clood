package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/system"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// VerifyResult represents a single verification test result
type VerifyResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
	TimeMs  int64  `json:"time_ms"`
}

// VerifyReport is the full verification report
type VerifyReport struct {
	Timestamp string         `json:"timestamp"`
	Tests     []VerifyResult `json:"tests"`
	Passed    int            `json:"passed"`
	Failed    int            `json:"failed"`
	Total     int            `json:"total"`
}

func VerifyCmd() *cobra.Command {
	var jsonOutput bool
	var quick bool

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify clood installation works correctly",
		Long: `Run automated tests to verify clood installation.

Tests:
  1. Hardware Detection - Can detect CPU/memory/GPU
  2. Config Loading - Config loads without errors
  3. Ollama Connection - At least one host reachable
  4. Model Inference - Can run a simple prompt (unless --quick)

Use --quick to skip the inference test for faster verification.`,
		Run: func(cmd *cobra.Command, args []string) {
			useJSON := jsonOutput || output.IsJSON()

			report := runVerifyTests(quick, useJSON)

			if useJSON {
				data, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(data))
				return
			}

			printVerifyReport(report)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&quick, "quick", false, "Skip inference test for faster verification")

	return cmd
}

func runVerifyTests(quick bool, silent bool) VerifyReport {
	report := VerifyReport{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Test 1: Hardware Detection
	if !silent {
		fmt.Print("  [1/4] Hardware Detection... ")
	}
	start := time.Now()
	hw, err := system.DetectHardware()
	result := VerifyResult{
		Name:   "Hardware Detection",
		TimeMs: time.Since(start).Milliseconds(),
	}
	if err != nil {
		result.Passed = false
		result.Message = err.Error()
	} else if hw.CPUCores == 0 {
		result.Passed = false
		result.Message = "Failed to detect CPU cores"
	} else {
		result.Passed = true
		result.Message = fmt.Sprintf("%s, %d cores, %.0fGB RAM", hw.CPUModel, hw.CPUCores, hw.MemoryGB)
	}
	report.Tests = append(report.Tests, result)
	if !silent {
		printTestResult(result)
	}

	// Test 2: Config Loading
	if !silent {
		fmt.Print("  [2/4] Config Loading... ")
	}
	start = time.Now()
	cfg, err := config.Load()
	result = VerifyResult{
		Name:   "Config Loading",
		TimeMs: time.Since(start).Milliseconds(),
	}
	if err != nil {
		result.Passed = false
		result.Message = err.Error()
	} else {
		result.Passed = true
		if config.Exists() {
			result.Message = fmt.Sprintf("Loaded from %s", config.ConfigPath())
		} else {
			result.Message = "Using defaults (no config file)"
		}
	}
	report.Tests = append(report.Tests, result)
	if !silent {
		printTestResult(result)
	}

	// Test 3: Ollama Connection
	if !silent {
		fmt.Print("  [3/4] Ollama Connection... ")
	}
	start = time.Now()
	result = VerifyResult{
		Name:   "Ollama Connection",
		TimeMs: time.Since(start).Milliseconds(),
	}

	if cfg != nil && len(cfg.Hosts) > 0 {
		mgr := hosts.NewManager()
		mgr.AddHosts(cfg.Hosts)
		statuses := mgr.CheckAllHosts()

		var onlineHosts []string
		for _, s := range statuses {
			if s.Online {
				onlineHosts = append(onlineHosts, s.Host.Name)
			}
		}

		result.TimeMs = time.Since(start).Milliseconds()

		if len(onlineHosts) > 0 {
			result.Passed = true
			if len(onlineHosts) == 1 {
				result.Message = fmt.Sprintf("Connected to %s", onlineHosts[0])
			} else {
				result.Message = fmt.Sprintf("Connected to %d hosts", len(onlineHosts))
			}
		} else {
			result.Passed = false
			result.Message = fmt.Sprintf("No hosts online (0/%d)", len(cfg.Hosts))
		}
	} else {
		result.Passed = false
		result.Message = "No hosts configured"
	}
	report.Tests = append(report.Tests, result)
	if !silent {
		printTestResult(result)
	}

	// Test 4: Model Inference (skip if --quick)
	if quick {
		if !silent {
			fmt.Println("  [4/4] Model Inference... " + tui.MutedStyle.Render("SKIPPED (--quick)"))
		}
		result = VerifyResult{
			Name:    "Model Inference",
			Passed:  true,
			Message: "Skipped (--quick flag)",
			TimeMs:  0,
		}
		report.Tests = append(report.Tests, result)
	} else {
		if !silent {
			fmt.Print("  [4/4] Model Inference... ")
		}
		start = time.Now()
		result = VerifyResult{
			Name: "Model Inference",
		}

		// Find first available host and model
		if cfg != nil && len(cfg.Hosts) > 0 {
			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			var testHost *hosts.HostStatus
			statuses := mgr.CheckAllHosts()
			for _, s := range statuses {
				if s.Online && len(s.Models) > 0 {
					testHost = s
					break
				}
			}

			if testHost != nil {
				// Pick a small model preferably
				modelName := testHost.Models[0].Name
				for _, m := range testHost.Models {
					if m.Name == "qwen2.5-coder:1.5b" || m.Name == "tinyllama" {
						modelName = m.Name
						break
					}
				}

				// Run simple inference
				client := ollama.NewClient(testHost.Host.URL, 30*time.Second)

				_, err := client.Generate(modelName, "Say 'Hello' in one word:")
				result.TimeMs = time.Since(start).Milliseconds()

				if err != nil {
					result.Passed = false
					result.Message = "Inference failed: " + err.Error()
				} else {
					result.Passed = true
					result.Message = fmt.Sprintf("Ran %s on %s (%dms)", modelName, testHost.Host.Name, result.TimeMs)
				}
			} else {
				result.Passed = false
				result.Message = "No hosts with models available"
				result.TimeMs = time.Since(start).Milliseconds()
			}
		} else {
			result.Passed = false
			result.Message = "No hosts configured"
			result.TimeMs = time.Since(start).Milliseconds()
		}
		report.Tests = append(report.Tests, result)
		if !silent {
			printTestResult(result)
		}
	}

	// Calculate totals
	for _, t := range report.Tests {
		report.Total++
		if t.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
	}

	return report
}

func printTestResult(r VerifyResult) {
	if r.Passed {
		fmt.Printf("%s", tui.SuccessStyle.Render("OK"))
		if r.Message != "" {
			fmt.Printf(" %s", tui.MutedStyle.Render("("+r.Message+")"))
		}
	} else {
		fmt.Printf("%s", tui.ErrorStyle.Render("FAIL"))
		if r.Message != "" {
			fmt.Printf(" %s", tui.MutedStyle.Render("("+r.Message+")"))
		}
	}
	fmt.Println()
}

func printVerifyReport(report VerifyReport) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("CLOOD VERIFY"))
	fmt.Println()

	// Results already printed during test run

	fmt.Println()
	fmt.Println("  ─────────────────────────────────────────────────────────")

	if report.Failed == 0 {
		fmt.Printf("  %s %d/%d tests passed\n\n",
			tui.SuccessStyle.Render("✓"),
			report.Passed,
			report.Total)
	} else {
		fmt.Printf("  %s %d/%d tests passed, %d failed\n\n",
			tui.ErrorStyle.Render("✗"),
			report.Passed,
			report.Total,
			report.Failed)
	}
}
