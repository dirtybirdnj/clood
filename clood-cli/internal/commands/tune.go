package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/spf13/cobra"
)

// TuneRecommendation contains tuning advice
type TuneRecommendation struct {
	Parameter   string `json:"parameter"`
	Current     string `json:"current,omitempty"`
	Recommended string `json:"recommended"`
	Reason      string `json:"reason"`
}

// TuneReport contains the full tuning analysis
type TuneReport struct {
	Host            string               `json:"host"`
	GPUMemoryMB     int                  `json:"gpu_memory_mb,omitempty"`
	SystemMemoryMB  int                  `json:"system_memory_mb,omitempty"`
	Recommendations []TuneRecommendation `json:"recommendations"`
	Modelfiles      map[string]string    `json:"modelfiles,omitempty"`
}

func TuneCmd() *cobra.Command {
	var jsonOutput bool
	var showModelfiles bool
	var targetHost string

	cmd := &cobra.Command{
		Use:   "tune",
		Short: "Analyze and recommend Ollama performance tuning",
		Long: `Analyze your Ollama setup and provide performance tuning recommendations.

This command helps optimize Ollama for code analysis by:
- Detecting hardware capabilities (GPU memory, system RAM)
- Recommending context window sizes (num_ctx)
- Suggesting response length settings (num_predict)
- Generating optimized Modelfiles for common tasks

Examples:
  clood tune                    # Analyze local Ollama
  clood tune --host ubuntu25    # Analyze remote host
  clood tune --modelfiles       # Show optimized modelfile templates
  clood tune --json             # Output as JSON for scripting`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showModelfiles {
				printModelfiles()
				return nil
			}

			return runTuneAnalysis(targetHost, jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVar(&showModelfiles, "modelfiles", false, "Show optimized modelfile templates")
	cmd.Flags().StringVarP(&targetHost, "host", "H", "", "Target host to analyze")

	return cmd
}

func runTuneAnalysis(targetHost string, jsonOutput bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	mgr := hosts.NewManager()
	mgr.AddHosts(cfg.Hosts)

	// Determine which host to analyze
	var hostToAnalyze *hosts.Host
	if targetHost != "" {
		hostToAnalyze = mgr.GetHost(targetHost)
		if hostToAnalyze == nil {
			return fmt.Errorf("host not found: %s", targetHost)
		}
	} else {
		// Find first online host
		for _, h := range cfg.Hosts {
			status := mgr.CheckHost(h)
			if status.Online {
				hostToAnalyze = h
				break
			}
		}
	}

	if hostToAnalyze == nil {
		return fmt.Errorf("no online hosts found")
	}

	report := TuneReport{
		Host:            hostToAnalyze.Name,
		Recommendations: []TuneRecommendation{},
	}

	// Get system info
	sysMemMB := getSystemMemoryMB()
	report.SystemMemoryMB = sysMemMB

	// Build recommendations
	report.Recommendations = buildRecommendations(sysMemMB)

	// Output
	if jsonOutput {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
	} else {
		printTuneReport(report)
	}

	return nil
}

func buildRecommendations(sysMemMB int) []TuneRecommendation {
	recs := []TuneRecommendation{}

	// Context window recommendation
	var ctxRec string
	var ctxReason string
	if sysMemMB >= 64000 {
		ctxRec = "32768"
		ctxReason = "64GB+ RAM supports large context for multi-file analysis"
	} else if sysMemMB >= 32000 {
		ctxRec = "16384"
		ctxReason = "32GB RAM supports 16K context comfortably"
	} else if sysMemMB >= 16000 {
		ctxRec = "8192"
		ctxReason = "16GB RAM - use 8K context, larger may cause swapping"
	} else {
		ctxRec = "4096"
		ctxReason = "Limited RAM - keep context small to avoid OOM"
	}

	recs = append(recs, TuneRecommendation{
		Parameter:   "num_ctx",
		Current:     "2048 (Ollama default)",
		Recommended: ctxRec,
		Reason:      ctxReason,
	})

	// Response length recommendation
	recs = append(recs, TuneRecommendation{
		Parameter:   "num_predict",
		Current:     "128-512 (varies by model)",
		Recommended: "4096",
		Reason:      "Code analysis needs longer responses for detailed feedback",
	})

	// Temperature recommendation
	recs = append(recs, TuneRecommendation{
		Parameter:   "temperature",
		Current:     "0.7-0.8 (default)",
		Recommended: "0.2",
		Reason:      "Lower temperature for more deterministic code analysis",
	})

	// Environment variable recommendations
	recs = append(recs, TuneRecommendation{
		Parameter:   "OLLAMA_FLASH_ATTENTION",
		Current:     "unset",
		Recommended: "1",
		Reason:      "Reduces VRAM usage and increases inference speed",
	})

	recs = append(recs, TuneRecommendation{
		Parameter:   "OLLAMA_KV_CACHE_TYPE",
		Current:     "unset (fp16)",
		Recommended: "q8_0",
		Reason:      "Halves KV cache memory with minimal quality loss",
	})

	return recs
}

func printTuneReport(report TuneReport) {
	fmt.Println("=== Ollama Performance Tuning ===")
	fmt.Println()
	fmt.Printf("Host: %s\n", report.Host)
	if report.SystemMemoryMB > 0 {
		fmt.Printf("System RAM: %d MB (%.1f GB)\n", report.SystemMemoryMB, float64(report.SystemMemoryMB)/1024)
	}
	fmt.Println()

	fmt.Println("=== Recommendations ===")
	fmt.Println()

	for _, rec := range report.Recommendations {
		fmt.Printf("%s:\n", rec.Parameter)
		if rec.Current != "" {
			fmt.Printf("  Current:     %s\n", rec.Current)
		}
		fmt.Printf("  Recommended: %s\n", rec.Recommended)
		fmt.Printf("  Reason:      %s\n", rec.Reason)
		fmt.Println()
	}

	fmt.Println("=== Quick Start ===")
	fmt.Println()
	fmt.Println("1. Set environment variables (add to ~/.bashrc or ~/.zshrc):")
	fmt.Println()
	fmt.Println("   export OLLAMA_FLASH_ATTENTION=1")
	fmt.Println("   export OLLAMA_KV_CACHE_TYPE=q8_0")
	fmt.Println()
	fmt.Println("2. Create optimized model (run on Ollama host):")
	fmt.Println()
	fmt.Println("   clood tune --modelfiles > ~/.ollama/Modelfile.analysis")
	fmt.Println("   ollama create qwen-analysis -f ~/.ollama/Modelfile.analysis")
	fmt.Println()
	fmt.Println("3. Use optimized model:")
	fmt.Println()
	fmt.Println("   clood analyze file.go --model qwen-analysis")
	fmt.Println()
}

func printModelfiles() {
	fmt.Println("# ===========================================")
	fmt.Println("# Ollama Modelfiles for Code Analysis")
	fmt.Println("# ===========================================")
	fmt.Println("#")
	fmt.Println("# Save the appropriate section to a file and run:")
	fmt.Println("#   ollama create <name> -f <filename>")
	fmt.Println("#")
	fmt.Println()

	// Code Analysis Model
	fmt.Println("# ----- qwen-coder-analysis -----")
	fmt.Println("# Good for: Code review, bug finding, security analysis")
	fmt.Println("# Save to: Modelfile.qwen-analysis")
	fmt.Println()
	fmt.Println("FROM qwen2.5-coder:7b")
	fmt.Println()
	fmt.Println("# Large context for multi-file analysis")
	fmt.Println("PARAMETER num_ctx 16384")
	fmt.Println()
	fmt.Println("# Allow detailed responses")
	fmt.Println("PARAMETER num_predict 4096")
	fmt.Println()
	fmt.Println("# Deterministic for consistent analysis")
	fmt.Println("PARAMETER temperature 0.2")
	fmt.Println()
	fmt.Println("# Stop tokens")
	fmt.Println("PARAMETER stop <|im_end|>")
	fmt.Println("PARAMETER stop <|endoftext|>")
	fmt.Println()
	fmt.Println(`SYSTEM """You are a senior code reviewer. When analyzing code:
1. Be specific - reference line numbers and function names
2. Focus on bugs, edge cases, and security issues
3. Explain WHY something is problematic, not just WHAT
4. Provide actionable fix suggestions
5. If the code is good, say so briefly and explain why
"""`)
	fmt.Println()

	// Reasoning Model
	fmt.Println("# ----- llama-reasoning -----")
	fmt.Println("# Good for: Complex problem solving, architecture decisions")
	fmt.Println("# Save to: Modelfile.llama-reasoning")
	fmt.Println()
	fmt.Println("FROM llama3.1:8b")
	fmt.Println()
	fmt.Println("PARAMETER num_ctx 16384")
	fmt.Println("PARAMETER num_predict 4096")
	fmt.Println("PARAMETER temperature 0.3")
	fmt.Println()
	fmt.Println(`SYSTEM """You are a software architect helping with complex decisions.
Think step by step. Consider trade-offs. Be thorough but concise.
When asked about code, focus on the underlying design and architecture.
"""`)
	fmt.Println()

	// Fast Coding Model
	fmt.Println("# ----- qwen-coder-fast -----")
	fmt.Println("# Good for: Quick code generation, simple queries")
	fmt.Println("# Save to: Modelfile.qwen-fast")
	fmt.Println()
	fmt.Println("FROM qwen2.5-coder:3b")
	fmt.Println()
	fmt.Println("PARAMETER num_ctx 8192")
	fmt.Println("PARAMETER num_predict 2048")
	fmt.Println("PARAMETER temperature 0.4")
	fmt.Println()
	fmt.Println(`SYSTEM """You are a helpful coding assistant. Be concise and direct.
Output code without excessive explanation unless asked.
"""`)
	fmt.Println()

	// Documentation Model
	fmt.Println("# ----- llama-docs -----")
	fmt.Println("# Good for: Documentation, PR descriptions, explanations")
	fmt.Println("# Save to: Modelfile.llama-docs")
	fmt.Println()
	fmt.Println("FROM llama3.1:8b")
	fmt.Println()
	fmt.Println("PARAMETER num_ctx 8192")
	fmt.Println("PARAMETER num_predict 2048")
	fmt.Println("PARAMETER temperature 0.5")
	fmt.Println()
	fmt.Println(`SYSTEM """You are a technical writer. Write clear, concise documentation.
Use markdown formatting. Focus on what the reader needs to know.
Avoid jargon unless necessary. Include examples where helpful.
"""`)
}

func getSystemMemoryMB() int {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0
		}
		var bytes int64
		fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &bytes)
		return int(bytes / 1024 / 1024)

	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				var kb int
				fmt.Sscanf(line, "MemTotal: %d kB", &kb)
				return kb / 1024
			}
		}
	}
	return 0
}

// GetModelfileDir returns the directory for custom modelfiles
func GetModelfileDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ollama"
	}
	return filepath.Join(home, ".ollama", "modelfiles")
}
