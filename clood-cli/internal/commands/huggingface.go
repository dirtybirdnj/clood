package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Popular GGUF models with their HuggingFace paths
var popularGGUFModels = map[string]GGUFModel{
	"tinyllama": {
		Name:     "tinyllama",
		Repo:     "TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF",
		File:     "tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf",
		Size:     "638MB",
		Params:   "1.1B",
		Desc:     "Ultra-fast tiny model for quick tasks",
	},
	"qwen2.5-coder:3b": {
		Name:     "qwen2.5-coder:3b",
		Repo:     "Qwen/Qwen2.5-Coder-3B-Instruct-GGUF",
		File:     "qwen2.5-coder-3b-instruct-q4_k_m.gguf",
		Size:     "1.9GB",
		Params:   "3B",
		Desc:     "Fast coding model",
	},
	"qwen2.5-coder:7b": {
		Name:     "qwen2.5-coder:7b",
		Repo:     "Qwen/Qwen2.5-Coder-7B-Instruct-GGUF",
		File:     "qwen2.5-coder-7b-instruct-q4_k_m.gguf",
		Size:     "4.4GB",
		Params:   "7B",
		Desc:     "Balanced coding model - great quality/speed",
	},
	"qwen2.5-coder:14b": {
		Name:     "qwen2.5-coder:14b",
		Repo:     "Qwen/Qwen2.5-Coder-14B-Instruct-GGUF",
		File:     "qwen2.5-coder-14b-instruct-q4_k_m.gguf",
		Size:     "8.9GB",
		Params:   "14B",
		Desc:     "High quality coding model",
	},
	"qwen2.5-coder:32b": {
		Name:     "qwen2.5-coder:32b",
		Repo:     "Qwen/Qwen2.5-Coder-32B-Instruct-GGUF",
		File:     "qwen2.5-coder-32b-instruct-q4_k_m.gguf",
		Size:     "19GB",
		Params:   "32B",
		Desc:     "Top-tier coding model",
	},
	"deepseek-coder:6.7b": {
		Name:     "deepseek-coder:6.7b",
		Repo:     "TheBloke/deepseek-coder-6.7B-instruct-GGUF",
		File:     "deepseek-coder-6.7b-instruct.Q4_K_M.gguf",
		Size:     "3.8GB",
		Params:   "6.7B",
		Desc:     "DeepSeek coding specialist",
	},
	"llama3.1:8b": {
		Name:     "llama3.1:8b",
		Repo:     "bartowski/Meta-Llama-3.1-8B-Instruct-GGUF",
		File:     "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf",
		Size:     "4.9GB",
		Params:   "8B",
		Desc:     "Meta's Llama 3.1 - great all-rounder",
	},
	"mistral:7b": {
		Name:     "mistral:7b",
		Repo:     "TheBloke/Mistral-7B-Instruct-v0.2-GGUF",
		File:     "mistral-7b-instruct-v0.2.Q4_K_M.gguf",
		Size:     "4.1GB",
		Params:   "7B",
		Desc:     "Mistral 7B - efficient and capable",
	},
	"codestral:22b": {
		Name:     "codestral:22b",
		Repo:     "bartowski/Codestral-22B-v0.1-GGUF",
		File:     "Codestral-22B-v0.1-Q4_K_M.gguf",
		Size:     "12.4GB",
		Params:   "22B",
		Desc:     "Mistral's code-focused model",
	},
}

// GGUFModel represents a downloadable GGUF model
type GGUFModel struct {
	Name   string `json:"name"`
	Repo   string `json:"repo"`
	File   string `json:"file"`
	Size   string `json:"size"`
	Params string `json:"params"`
	Desc   string `json:"description"`
}

// GGUFDownloadResult is the JSON output for downloads
type GGUFDownloadResult struct {
	Model    string `json:"model"`
	Path     string `json:"path"`
	Size     int64  `json:"size_bytes"`
	Duration string `json:"duration"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// Default model directory
const defaultModelDir = "/data/cache/llama-models"

func HuggingfaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "huggingface",
		Aliases: []string{"hf"},
		Short:   "Manage GGUF models from HuggingFace",
		Long: `Download and manage GGUF models from HuggingFace for use with llama.cpp.

This provides an "ollama pull" experience for llama.cpp models.

Examples:
  clood huggingface list                    # Show available models
  clood huggingface pull qwen2.5-coder:7b   # Download a model
  clood huggingface pull --all-coding       # Download all coding models
  clood huggingface search "deepseek"       # Search HuggingFace`,
	}

	cmd.AddCommand(hfListCmd())
	cmd.AddCommand(hfPullCmd())
	cmd.AddCommand(hfSearchCmd())
	cmd.AddCommand(hfPathCmd())

	return cmd
}

func hfListCmd() *cobra.Command {
	var showInstalled bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available GGUF models",
		Run: func(cmd *cobra.Command, args []string) {
			modelDir := getModelDir()
			installed := getInstalledModels(modelDir)

			if output.IsJSON() {
				type ModelList struct {
					ModelDir  string      `json:"model_dir"`
					Models    []GGUFModel `json:"models"`
					Installed []string    `json:"installed"`
				}
				var models []GGUFModel
				for _, m := range popularGGUFModels {
					models = append(models, m)
				}
				data, _ := json.MarshalIndent(ModelList{
					ModelDir:  modelDir,
					Models:    models,
					Installed: installed,
				}, "", "  ")
				fmt.Println(string(data))
				return
			}

			fmt.Println(tui.RenderHeader("Available GGUF Models"))
			fmt.Println()
			fmt.Printf("  Model directory: %s\n\n", tui.MutedStyle.Render(modelDir))

			// Group by size
			small := []GGUFModel{}
			medium := []GGUFModel{}
			large := []GGUFModel{}

			for _, m := range popularGGUFModels {
				switch {
				case strings.Contains(m.Params, "1") || strings.Contains(m.Params, "3B"):
					small = append(small, m)
				case strings.Contains(m.Params, "6") || strings.Contains(m.Params, "7") || strings.Contains(m.Params, "8"):
					medium = append(medium, m)
				default:
					large = append(large, m)
				}
			}

			printModelGroup("Small (1-3B) - Fast", small, installed)
			printModelGroup("Medium (6-8B) - Balanced", medium, installed)
			printModelGroup("Large (14B+) - Quality", large, installed)

			fmt.Println(tui.MutedStyle.Render("\n  Use: clood huggingface pull <model>"))
		},
	}

	cmd.Flags().BoolVar(&showInstalled, "installed", false, "Show only installed models")
	return cmd
}

func printModelGroup(title string, models []GGUFModel, installed []string) {
	if len(models) == 0 {
		return
	}
	fmt.Printf("  %s:\n", tui.SuccessStyle.Render(title))
	for _, m := range models {
		status := "○"
		for _, inst := range installed {
			if strings.Contains(strings.ToLower(inst), strings.ToLower(strings.Split(m.Name, ":")[0])) {
				status = tui.SuccessStyle.Render("●")
				break
			}
		}
		fmt.Printf("    %s %-22s %6s  %s\n", status, m.Name, m.Size, tui.MutedStyle.Render(m.Desc))
	}
	fmt.Println()
}

func hfPullCmd() *cobra.Command {
	var modelDir string
	var quant string
	var allCoding bool

	cmd := &cobra.Command{
		Use:   "pull <model>",
		Short: "Download a GGUF model from HuggingFace",
		Long: `Download a GGUF model from HuggingFace.

Models are stored in the llama.cpp model directory (default: /data/cache/llama-models).

Examples:
  clood huggingface pull qwen2.5-coder:7b
  clood huggingface pull tinyllama
  clood huggingface pull --dir /path/to/models qwen2.5-coder:14b`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if modelDir == "" {
				modelDir = getModelDir()
			}

			// Ensure directory exists
			if err := os.MkdirAll(modelDir, 0755); err != nil {
				return fmt.Errorf("creating model directory: %w", err)
			}

			if allCoding {
				codingModels := []string{"qwen2.5-coder:3b", "qwen2.5-coder:7b", "deepseek-coder:6.7b"}
				for _, m := range codingModels {
					if err := pullGGUFModel(m, modelDir, quant); err != nil {
						fmt.Printf("  %s %s: %v\n", tui.ErrorStyle.Render("✗"), m, err)
					}
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("specify a model name (use 'clood huggingface list' to see available)")
			}

			return pullGGUFModel(args[0], modelDir, quant)
		},
	}

	cmd.Flags().StringVar(&modelDir, "dir", "", "Model directory (default: /data/cache/llama-models)")
	cmd.Flags().StringVar(&quant, "quant", "Q4_K_M", "Quantization level (Q4_K_M, Q5_K_M, Q8_0)")
	cmd.Flags().BoolVar(&allCoding, "all-coding", false, "Download all coding models (3B, 7B, 6.7B)")

	return cmd
}

func pullGGUFModel(modelName, modelDir, quant string) error {
	model, ok := popularGGUFModels[modelName]
	if !ok {
		// Try to find partial match
		for name, m := range popularGGUFModels {
			if strings.Contains(name, modelName) || strings.Contains(modelName, strings.Split(name, ":")[0]) {
				model = m
				ok = true
				break
			}
		}
	}

	if !ok {
		return fmt.Errorf("unknown model: %s (use 'clood huggingface list' to see available)", modelName)
	}

	// Build download URL
	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s?download=true", model.Repo, model.File)

	// Determine output filename
	outputFile := filepath.Join(modelDir, model.File)

	// Check if already exists
	if _, err := os.Stat(outputFile); err == nil {
		if output.IsJSON() {
			result := GGUFDownloadResult{
				Model:   model.Name,
				Path:    outputFile,
				Success: true,
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("  %s %s already exists at %s\n", tui.SuccessStyle.Render("✓"), model.Name, outputFile)
		}
		return nil
	}

	if !output.IsJSON() {
		fmt.Printf("  Downloading %s (%s)...\n", model.Name, model.Size)
		fmt.Printf("  From: %s\n", tui.MutedStyle.Render(model.Repo))
		fmt.Printf("  To:   %s\n\n", tui.MutedStyle.Render(outputFile))
	}

	startTime := time.Now()

	// Use curl for download with progress
	curlCmd := exec.Command("curl", "-L", "-o", outputFile, "--progress-bar", url)
	curlCmd.Stdout = os.Stdout
	curlCmd.Stderr = os.Stderr

	if err := curlCmd.Run(); err != nil {
		os.Remove(outputFile) // Clean up partial download
		if output.IsJSON() {
			result := GGUFDownloadResult{
				Model:   model.Name,
				Success: false,
				Error:   err.Error(),
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		}
		return fmt.Errorf("download failed: %w", err)
	}

	duration := time.Since(startTime)

	// Get file size
	info, _ := os.Stat(outputFile)
	var size int64
	if info != nil {
		size = info.Size()
	}

	if output.IsJSON() {
		result := GGUFDownloadResult{
			Model:    model.Name,
			Path:     outputFile,
			Size:     size,
			Duration: duration.Round(time.Second).String(),
			Success:  true,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println()
		fmt.Printf("  %s Downloaded %s in %v\n", tui.SuccessStyle.Render("✓"), model.Name, duration.Round(time.Second))
		fmt.Printf("  Path: %s\n", outputFile)
	}

	return nil
}

func hfSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search HuggingFace for GGUF models",
		Long: `Search HuggingFace for GGUF models.

Examples:
  clood huggingface search "qwen coder"
  clood huggingface search "deepseek"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("specify a search query")
			}

			query := strings.Join(args, " ")
			return searchHuggingFace(query)
		},
	}
	return cmd
}

func searchHuggingFace(query string) error {
	// Use HuggingFace API to search for GGUF models
	searchURL := fmt.Sprintf("https://huggingface.co/api/models?search=%s+gguf&sort=downloads&direction=-1&limit=10",
		strings.ReplaceAll(query, " ", "+"))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(searchURL)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var results []struct {
		ID        string `json:"id"`
		Downloads int    `json:"downloads"`
		Likes     int    `json:"likes"`
	}

	if err := json.Unmarshal(body, &results); err != nil {
		return fmt.Errorf("parsing results: %w", err)
	}

	if output.IsJSON() {
		fmt.Println(string(body))
		return nil
	}

	fmt.Println(tui.RenderHeader("HuggingFace GGUF Models"))
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("  No results found")
		return nil
	}

	for _, r := range results {
		fmt.Printf("  %s\n", r.ID)
		fmt.Printf("    Downloads: %d  Likes: %d\n", r.Downloads, r.Likes)
		fmt.Printf("    %s\n\n", tui.MutedStyle.Render(fmt.Sprintf("https://huggingface.co/%s", r.ID)))
	}

	fmt.Println(tui.MutedStyle.Render("  To download, find the GGUF file and use:"))
	fmt.Println(tui.MutedStyle.Render("  curl -L -o model.gguf 'https://huggingface.co/<repo>/resolve/main/<file>.gguf'"))

	return nil
}

func hfPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show the GGUF model directory path",
		Run: func(cmd *cobra.Command, args []string) {
			modelDir := getModelDir()
			if output.IsJSON() {
				data, _ := json.MarshalIndent(map[string]string{"path": modelDir}, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Println(modelDir)
			}
		},
	}
	return cmd
}

func getModelDir() string {
	// Check environment variable first
	if dir := os.Getenv("LLAMACPP_MODELS"); dir != "" {
		return dir
	}
	// Check if /data exists (ubuntu25 setup)
	if _, err := os.Stat("/data/cache/llama-models"); err == nil {
		return "/data/cache/llama-models"
	}
	// Fallback to home directory
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "llama-models")
}

func getInstalledModels(dir string) []string {
	var installed []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return installed
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".gguf") {
			installed = append(installed, f.Name())
		}
	}
	return installed
}
