package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/system"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// SetupConfig represents the generated configuration
type SetupConfig struct {
	Hosts   []SetupHost   `yaml:"hosts"`
	Routing SetupRouting  `yaml:"routing"`
}

type SetupHost struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Type string `yaml:"type"`
}

type SetupRouting struct {
	Tier1Model string `yaml:"tier1_model"`
	Tier2Model string `yaml:"tier2_model"`
	Tier3Model string `yaml:"tier3_model"`
	Tier4Model string `yaml:"tier4_model"`
	Fallback   bool   `yaml:"fallback"`
}

func SetupCmd() *cobra.Command {
	var nonInteractive bool
	var host string
	var force bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive first-run setup wizard",
		Long: `Configure clood for your local hardware.

The setup wizard will:
1. Detect your hardware (CPU, memory, GPU/VRAM)
2. Find running Ollama instances
3. Generate an optimized configuration
4. Offer to pull recommended models

Examples:
  clood setup                           # Interactive wizard
  clood setup --no-interactive          # Auto-detect and configure
  clood setup --host http://server:11434 # Configure with specific host`,
		Run: func(cmd *cobra.Command, args []string) {
			if output.IsJSON() {
				runSetupJSON(host)
				return
			}

			if nonInteractive {
				runSetupNonInteractive(host, force)
				return
			}

			runSetupInteractive(host, force)
		},
	}

	cmd.Flags().BoolVar(&nonInteractive, "no-interactive", false, "Run without prompts (auto-detect and configure)")
	cmd.Flags().StringVar(&host, "host", "", "Ollama host URL (default: auto-detect)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing config")

	return cmd
}

func runSetupJSON(hostURL string) {
	result := map[string]interface{}{}

	// Detect hardware
	hw, err := system.DetectHardware()
	if err != nil {
		result["error"] = err.Error()
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}
	result["hardware"] = hw.JSON()

	// Find Ollama
	ollamaURL := hostURL
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	ollamaOnline := checkOllama(ollamaURL)
	result["ollama"] = map[string]interface{}{
		"url":    ollamaURL,
		"online": ollamaOnline,
	}

	// Get recommended models
	result["recommended_models"] = hw.RecommendedModels()

	// Generate config
	config := generateConfig(hw, ollamaURL)
	result["config"] = config

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func runSetupNonInteractive(hostURL string, force bool) {
	fmt.Println(tui.RenderHeader("clood setup"))
	fmt.Println()

	// Check existing config
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Println(tui.WarningStyle.Render("Config already exists: " + configPath))
		fmt.Println(tui.MutedStyle.Render("Use --force to overwrite"))
		return
	}

	// Step 1: Detect hardware
	fmt.Print("Detecting hardware... ")
	hw, err := system.DetectHardware()
	if err != nil {
		fmt.Println(tui.ErrorStyle.Render("failed"))
		fmt.Println(tui.ErrorStyle.Render(err.Error()))
		return
	}
	fmt.Println(tui.SuccessStyle.Render("done"))
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render(hw.Summary()))

	// Step 2: Find Ollama
	ollamaURL := hostURL
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	fmt.Printf("Checking Ollama at %s... ", ollamaURL)
	if checkOllama(ollamaURL) {
		fmt.Println(tui.SuccessStyle.Render("online"))
	} else {
		fmt.Println(tui.WarningStyle.Render("offline"))
		fmt.Println(tui.MutedStyle.Render("  Ollama not running. Start with: ollama serve"))
	}
	fmt.Println()

	// Step 3: Generate config
	config := generateConfig(hw, ollamaURL)
	if err := saveConfig(configPath, config); err != nil {
		fmt.Println(tui.ErrorStyle.Render("Failed to save config: " + err.Error()))
		return
	}

	fmt.Println(tui.SuccessStyle.Render("✓ Config saved to: " + configPath))
	fmt.Println()

	// Step 4: Show recommended models
	models := hw.RecommendedModels()
	if len(models) > 0 {
		fmt.Println("Recommended models for your hardware:")
		for _, m := range models {
			fmt.Printf("  • %s\n", m)
		}
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("Pull with: ollama pull <model>"))
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("Setup complete! Run 'clood preflight' to verify."))
}

func runSetupInteractive(hostURL string, force bool) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(tui.RenderHeader("clood setup wizard"))
	fmt.Println()
	fmt.Println("This wizard will configure clood for your hardware.")
	fmt.Println()

	// Check existing config
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Println(tui.WarningStyle.Render("Config already exists: " + configPath))
		fmt.Print("Overwrite? [y/N] ")
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("Setup cancelled.")
			return
		}
		fmt.Println()
	}

	// Step 1: Detect hardware
	fmt.Println(tui.AccentStyle.Render("Step 1: Hardware Detection"))
	fmt.Print("Detecting hardware... ")
	hw, err := system.DetectHardware()
	if err != nil {
		fmt.Println(tui.ErrorStyle.Render("failed"))
		fmt.Println(tui.ErrorStyle.Render(err.Error()))
		return
	}
	fmt.Println(tui.SuccessStyle.Render("done"))
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render(hw.Summary()))
	fmt.Println()

	// Step 2: Find Ollama
	fmt.Println(tui.AccentStyle.Render("Step 2: Ollama Detection"))

	ollamaURL := hostURL
	if ollamaURL == "" {
		// Try common locations
		candidates := []string{
			"http://localhost:11434",
			"http://127.0.0.1:11434",
		}

		for _, url := range candidates {
			fmt.Printf("Checking %s... ", url)
			if checkOllama(url) {
				fmt.Println(tui.SuccessStyle.Render("found"))
				ollamaURL = url
				break
			} else {
				fmt.Println(tui.MutedStyle.Render("no"))
			}
		}

		if ollamaURL == "" {
			fmt.Println()
			fmt.Print("Enter Ollama URL [http://localhost:11434]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				ollamaURL = "http://localhost:11434"
			} else {
				ollamaURL = input
			}
		}
	}

	// Verify final URL
	ollamaOnline := checkOllama(ollamaURL)
	if !ollamaOnline {
		fmt.Println()
		fmt.Println(tui.WarningStyle.Render("⚠ Ollama not responding at " + ollamaURL))
		fmt.Println(tui.MutedStyle.Render("  Config will be created but you'll need to start Ollama."))
	}
	fmt.Println()

	// Step 3: Model recommendations
	fmt.Println(tui.AccentStyle.Render("Step 3: Model Recommendations"))
	models := hw.RecommendedModels()
	if len(models) == 0 {
		fmt.Println(tui.WarningStyle.Render("No models recommended for available VRAM."))
		fmt.Println(tui.MutedStyle.Render("Consider adding more memory or using CPU inference."))
	} else {
		fmt.Println("Based on your hardware, these models should work well:")
		fmt.Println()
		for i, m := range models {
			if i < 5 { // Show top 5
				fmt.Printf("  %d. %s\n", i+1, m)
			}
		}
		if len(models) > 5 {
			fmt.Printf("  ... and %d more\n", len(models)-5)
		}
	}
	fmt.Println()

	// Step 4: Generate and save config
	fmt.Println(tui.AccentStyle.Render("Step 4: Configuration"))
	config := generateConfig(hw, ollamaURL)

	fmt.Println("Generated configuration:")
	fmt.Println()
	configYAML, _ := yaml.Marshal(config)
	for _, line := range strings.Split(string(configYAML), "\n") {
		fmt.Println(tui.MutedStyle.Render("  " + line))
	}
	fmt.Println()

	fmt.Print("Save configuration? [Y/n] ")
	answer, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(answer)) == "n" {
		fmt.Println("Setup cancelled.")
		return
	}

	if err := saveConfig(configPath, config); err != nil {
		fmt.Println(tui.ErrorStyle.Render("Failed to save config: " + err.Error()))
		return
	}

	fmt.Println(tui.SuccessStyle.Render("✓ Config saved to: " + configPath))
	fmt.Println()

	// Step 5: Offer to pull models
	if ollamaOnline && len(models) > 0 {
		fmt.Println(tui.AccentStyle.Render("Step 5: Pull Models"))
		fmt.Print("Would you like to pull recommended models now? [y/N] ")
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) == "y" {
			pullModels(models[:min(3, len(models))]) // Pull top 3
		}
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("✓ Setup complete!"))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  clood preflight    Verify configuration")
	fmt.Println("  clood hosts        Check host status")
	fmt.Println("  clood ask \"test\"   Test a query")
	fmt.Println()
}

func checkOllama(url string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url + "/api/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func generateConfig(hw *system.HardwareInfo, ollamaURL string) SetupConfig {
	config := SetupConfig{
		Hosts: []SetupHost{
			{
				Name: "local",
				URL:  ollamaURL,
				Type: "local",
			},
		},
		Routing: SetupRouting{
			Fallback: true,
		},
	}

	// Assign models based on VRAM
	models := hw.RecommendedModels()

	if len(models) >= 4 {
		// Have plenty of models to choose from
		config.Routing.Tier1Model = selectModel(models, "qwen2.5-coder:3b", "tinyllama")
		config.Routing.Tier2Model = selectModel(models, "qwen2.5-coder:7b", "qwen2.5-coder:3b")
		config.Routing.Tier3Model = selectModel(models, "llama3.1:8b", "qwen2.5-coder:7b")
		config.Routing.Tier4Model = selectModel(models, "qwen2.5-coder:7b", "qwen2.5-coder:3b")
	} else if len(models) >= 2 {
		// Limited selection
		config.Routing.Tier1Model = models[len(models)-1] // Smallest
		config.Routing.Tier2Model = models[0]              // Largest
		config.Routing.Tier3Model = models[0]
		config.Routing.Tier4Model = models[0]
	} else if len(models) == 1 {
		// Only one model fits
		config.Routing.Tier1Model = models[0]
		config.Routing.Tier2Model = models[0]
		config.Routing.Tier3Model = models[0]
		config.Routing.Tier4Model = models[0]
	} else {
		// Fallback defaults
		config.Routing.Tier1Model = "tinyllama"
		config.Routing.Tier2Model = "tinyllama"
		config.Routing.Tier3Model = "tinyllama"
		config.Routing.Tier4Model = "tinyllama"
	}

	return config
}

func selectModel(available []string, preferred, fallback string) string {
	for _, m := range available {
		if m == preferred {
			return preferred
		}
	}
	for _, m := range available {
		if m == fallback {
			return fallback
		}
	}
	if len(available) > 0 {
		return available[0]
	}
	return fallback
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clood", "config.yaml")
}

func saveConfig(path string, config SetupConfig) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Add header comment
	header := "# clood configuration\n# Generated by 'clood setup'\n# Edit as needed\n\n"
	return os.WriteFile(path, []byte(header+string(data)), 0644)
}

func pullModels(models []string) {
	fmt.Println()
	for _, model := range models {
		fmt.Printf("Pulling %s... ", model)
		// Note: This is a placeholder - actual implementation would use ollama API
		fmt.Println(tui.MutedStyle.Render("(run 'ollama pull " + model + "' manually)"))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
