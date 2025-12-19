package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/sd"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Default ComfyUI host - can be overridden with --host or COMFYUI_HOST env
const defaultComfyUIHost = "http://localhost:8188"

func getComfyUIHost(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("COMFYUI_HOST"); env != "" {
		return env
	}
	return defaultComfyUIHost
}

// SDCmd returns the Stable Diffusion command tree.
func SDCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sd",
		Aliases: []string{"forge", "enso"},
		Short:   "Stable Diffusion image generation via ComfyUI",
		Long: `The Spirit of Image Generation - Ensō

Ensō connects to a ComfyUI backend to generate images using Stable Diffusion models.
Like the incomplete circle, each generation is a single brushstroke - one breath.

ComfyUI host can be set via:
  --host flag
  COMFYUI_HOST environment variable
  Default: http://localhost:8188

Examples:
  clood sd status                     # Check ComfyUI connection and models
  clood sd paint "a tortoise"         # Generate an image
  clood sd sketch "quick concept"     # Fast generation (fewer steps)
  clood sd anvil "test prompt"        # Multi-model comparison
  clood sd gallery                    # View generated images`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(sdStatusCmd())
	cmd.AddCommand(sdPaintCmd())
	cmd.AddCommand(sdSketchCmd())
	cmd.AddCommand(sdAnvilCmd())
	cmd.AddCommand(sdGalleryCmd())
	cmd.AddCommand(sdPresetsCmd())
	cmd.AddCommand(sdRemixCmd())
	cmd.AddCommand(sdDeconstructCmd())
	cmd.AddCommand(sdInventoryCmd())

	return cmd
}

// sdStatusCmd checks ComfyUI connectivity and lists available models.
func sdStatusCmd() *cobra.Command {
	var host string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check ComfyUI connection and list available models",
		Long: `Check if ComfyUI is running and discover available checkpoints/LoRAs.

Examples:
  clood sd status
  clood sd status --host http://192.168.4.64:8188
  clood sd status --json`,
		Run: func(cmd *cobra.Command, args []string) {
			host := getComfyUIHost(host)
			client := sd.NewClient(host)

			// Check connection
			if !jsonOutput {
				fmt.Printf("%s Checking ComfyUI at %s...\n", tui.MutedStyle.Render(">>>"), host)
			}

			if err := client.Ping(); err != nil {
				if jsonOutput {
					json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
						"connected": false,
						"error":     err.Error(),
						"host":      host,
					})
				} else {
					fmt.Printf("%s ComfyUI not reachable: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
					fmt.Println()
					fmt.Println(tui.MutedStyle.Render("Make sure ComfyUI is running:"))
					fmt.Println(tui.MutedStyle.Render("  cd ~/stable-diffusion-webui && python main.py"))
				}
				return
			}

			// Get system stats
			stats, err := client.GetSystemStats()
			if err != nil && !jsonOutput {
				fmt.Printf("%s Could not get system stats: %v\n", tui.WarningStyle.Render("WARN:"), err)
			}

			// Get checkpoints
			checkpoints, err := client.GetCheckpoints()
			if err != nil && !jsonOutput {
				fmt.Printf("%s Could not list checkpoints: %v\n", tui.WarningStyle.Render("WARN:"), err)
			}

			if jsonOutput {
				output := map[string]interface{}{
					"connected":   true,
					"host":        host,
					"checkpoints": checkpoints,
				}
				if stats != nil {
					output["system"] = stats.System
					output["devices"] = stats.Devices
				}
				json.NewEncoder(os.Stdout).Encode(output)
				return
			}

			// Pretty output
			fmt.Printf("%s Connected!\n\n", tui.SuccessStyle.Render("OK"))

			if stats != nil {
				fmt.Println(tui.RenderHeader("SYSTEM"))
				fmt.Printf("  OS: %s\n", stats.System.OS)
				fmt.Printf("  Python: %s\n", stats.System.PythonVersion)
				for _, dev := range stats.Devices {
					vramGB := float64(dev.VRAM) / (1024 * 1024 * 1024)
					freeGB := float64(dev.VRAMFree) / (1024 * 1024 * 1024)
					fmt.Printf("  %s: %.1f GB VRAM (%.1f GB free)\n", dev.Name, vramGB, freeGB)
				}
				fmt.Println()
			}

			if len(checkpoints) > 0 {
				fmt.Println(tui.RenderHeader("CHECKPOINTS"))
				aquarium := sd.ExampleAquarium()
				for _, ckpt := range checkpoints {
					fish := aquarium.GetFishByCheckpoint(ckpt)
					if fish != nil {
						fmt.Printf("  %s %s (%s)\n",
							tui.AccentStyle.Render(fish.Name+":"),
							ckpt,
							fish.Species)
					} else {
						fmt.Printf("  %s\n", ckpt)
					}
				}
				fmt.Println()
			}
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ComfyUI host URL")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// sdPaintCmd generates a full-quality image.
func sdPaintCmd() *cobra.Command {
	var host string
	var checkpoint string
	var negative string
	var width, height, steps int
	var cfgScale float64
	var seed int64
	var lora string
	var loraWeight float64
	var outputDir string
	var open bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "paint [prompt]",
		Short: "Generate a full-quality image",
		Long: `Generate an image using ComfyUI with full quality settings.

Examples:
  clood sd paint "a tortoise with spectacles, ghibli style"
  clood sd paint "sunset over mountains" --checkpoint sdxl_base.safetensors
  clood sd paint "castle" --lora ghibli_style --lora-weight 0.8
  clood sd paint "portrait" --steps 40 --cfg 8.0
  clood sd paint "test" --open  # Open image after generation`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			promptText := strings.Join(args, " ")
			host := getComfyUIHost(host)
			client := sd.NewClient(host)

			// Set output directory
			if outputDir != "" {
				client.OutputDir = outputDir
			}

			// Check connection first
			if err := client.Ping(); err != nil {
				fmt.Printf("%s ComfyUI not reachable at %s: %v\n",
					tui.ErrorStyle.Render("ERROR:"), host, err)
				return
			}

			// Build prompt
			prompt := sd.NewPrompt(promptText)
			if negative != "" {
				prompt.WithNegative(negative)
			}
			if lora != "" {
				prompt.WithLoRA(lora, loraWeight)
			}
			if seed > 0 {
				prompt.WithSeed(seed)
			}

			// Build workflow config
			cfg := sd.DefaultWorkflowConfig()
			cfg.Prompt = prompt
			if checkpoint != "" {
				cfg.Checkpoint = checkpoint
			} else {
				// Try to get first available checkpoint
				checkpoints, err := client.GetCheckpoints()
				if err != nil || len(checkpoints) == 0 {
					fmt.Printf("%s No checkpoints available. Specify with --checkpoint\n",
						tui.ErrorStyle.Render("ERROR:"))
					return
				}
				cfg.Checkpoint = checkpoints[0]
			}
			if width > 0 {
				cfg.Width = width
			}
			if height > 0 {
				cfg.Height = height
			}
			if steps > 0 {
				cfg.Steps = steps
			}
			if cfgScale > 0 {
				cfg.CFGScale = cfgScale
			}

			if !jsonOutput {
				fmt.Println(tui.RenderHeader("ENSŌ - PAINT"))
				fmt.Printf("%s %s\n", tui.MutedStyle.Render("Prompt:"), promptText)
				fmt.Printf("%s %s\n", tui.MutedStyle.Render("Checkpoint:"), cfg.Checkpoint)
				fmt.Printf("%s %dx%d, %d steps, CFG %.1f\n",
					tui.MutedStyle.Render("Settings:"),
					cfg.Width, cfg.Height, cfg.Steps, cfg.CFGScale)
				if lora != "" {
					fmt.Printf("%s %s @ %.2f\n", tui.MutedStyle.Render("LoRA:"), lora, loraWeight)
				}
				fmt.Println()
				fmt.Printf("%s Generating...\n", tui.AccentStyle.Render(">>>"))
			}

			// Generate
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			result, err := client.Generate(ctx, cfg)
			if err != nil {
				if jsonOutput {
					json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
						"success": false,
						"error":   err.Error(),
					})
				} else {
					fmt.Printf("%s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				}
				return
			}

			if jsonOutput {
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"success":    true,
					"prompt_id":  result.PromptID,
					"images":     result.ImagePaths,
					"duration":   result.Duration.Seconds(),
					"checkpoint": cfg.Checkpoint,
					"prompt":     promptText,
				})
				return
			}

			fmt.Printf("%s Generated in %.1fs\n", tui.SuccessStyle.Render("DONE"), result.Duration.Seconds())
			for _, path := range result.ImagePaths {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Saved:"), path)

				// Open image if requested
				if open {
					openImage(path)
				}
			}
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ComfyUI host URL")
	cmd.Flags().StringVarP(&checkpoint, "checkpoint", "c", "", "Model checkpoint to use")
	cmd.Flags().StringVarP(&negative, "negative", "n", "", "Negative prompt")
	cmd.Flags().IntVarP(&width, "width", "W", 0, "Image width (default: 1024)")
	cmd.Flags().IntVarP(&height, "height", "H", 0, "Image height (default: 1024)")
	cmd.Flags().IntVarP(&steps, "steps", "s", 0, "Sampling steps (default: 25)")
	cmd.Flags().Float64Var(&cfgScale, "cfg", 0, "CFG scale (default: 7.0)")
	cmd.Flags().Int64Var(&seed, "seed", 0, "Random seed (0 for random)")
	cmd.Flags().StringVar(&lora, "lora", "", "LoRA model name")
	cmd.Flags().Float64Var(&loraWeight, "lora-weight", 0.8, "LoRA weight")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory")
	cmd.Flags().BoolVar(&open, "open", false, "Open image after generation")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// sdSketchCmd generates a quick/low-quality image.
func sdSketchCmd() *cobra.Command {
	var host string
	var checkpoint string
	var open bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "sketch [prompt]",
		Short: "Quick sketch - fast, rough generation",
		Long: `Generate a quick sketch with reduced steps for fast iteration.

Uses 10 steps instead of 25 for ~2.5x faster generation.

Examples:
  clood sd sketch "quick concept art"
  clood sd sketch "layout test" --open`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			promptText := strings.Join(args, " ")
			host := getComfyUIHost(host)
			client := sd.NewClient(host)

			// Check connection
			if err := client.Ping(); err != nil {
				fmt.Printf("%s ComfyUI not reachable: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			// Build config with sketch settings
			prompt := sd.NewPrompt(promptText)
			cfg := sd.DefaultWorkflowConfig()
			cfg.Prompt = prompt
			cfg.Steps = 10 // Quick sketch
			cfg.Width = 512
			cfg.Height = 512

			if checkpoint != "" {
				cfg.Checkpoint = checkpoint
			} else {
				checkpoints, err := client.GetCheckpoints()
				if err != nil || len(checkpoints) == 0 {
					fmt.Printf("%s No checkpoints available\n", tui.ErrorStyle.Render("ERROR:"))
					return
				}
				cfg.Checkpoint = checkpoints[0]
			}

			if !jsonOutput {
				fmt.Printf("%s Quick sketch: %s\n", tui.AccentStyle.Render(">>>"), promptText)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			result, err := client.Generate(ctx, cfg)
			if err != nil {
				fmt.Printf("%s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			if jsonOutput {
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"success": true,
					"images":  result.ImagePaths,
				})
				return
			}

			fmt.Printf("%s Done in %.1fs\n", tui.SuccessStyle.Render("OK"), result.Duration.Seconds())
			for _, path := range result.ImagePaths {
				fmt.Printf("  %s\n", path)
				if open {
					openImage(path)
				}
			}
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ComfyUI host URL")
	cmd.Flags().StringVarP(&checkpoint, "checkpoint", "c", "", "Model checkpoint")
	cmd.Flags().BoolVar(&open, "open", false, "Open image after generation")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// sdAnvilCmd runs multi-model comparison (the catfight for images).
func sdAnvilCmd() *cobra.Command {
	var host string
	var checkpoints string
	var outputDir string
	var openGallery bool

	cmd := &cobra.Command{
		Use:   "anvil [prompt]",
		Short: "Multi-model comparison - the catfight for images",
		Long: `Generate the same prompt with multiple checkpoints for comparison.

Like the catfight command, but for image generation.
Outputs an HTML gallery comparing all results.

Examples:
  clood sd anvil "a beach at sunset"
  clood sd anvil "portrait" --checkpoints "sdxl_base,dreamshaper"
  clood sd anvil "test" --open`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			promptText := strings.Join(args, " ")
			host := getComfyUIHost(host)
			client := sd.NewClient(host)

			// Check connection
			if err := client.Ping(); err != nil {
				fmt.Printf("%s ComfyUI not reachable: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			// Get checkpoints to compare
			var ckptList []string
			if checkpoints != "" {
				ckptList = strings.Split(checkpoints, ",")
				for i := range ckptList {
					ckptList[i] = strings.TrimSpace(ckptList[i])
				}
			} else {
				// Use all available checkpoints
				var err error
				ckptList, err = client.GetCheckpoints()
				if err != nil || len(ckptList) == 0 {
					fmt.Printf("%s No checkpoints available\n", tui.ErrorStyle.Render("ERROR:"))
					return
				}
				// Limit to first 4 for reasonable time
				if len(ckptList) > 4 {
					ckptList = ckptList[:4]
				}
			}

			fmt.Println(tui.RenderHeader("ANVIL - MODEL COMPARISON"))
			fmt.Printf("%s %s\n", tui.MutedStyle.Render("Prompt:"), promptText)
			fmt.Printf("%s %d models\n", tui.MutedStyle.Render("Testing:"), len(ckptList))
			for _, ckpt := range ckptList {
				fmt.Printf("  - %s\n", ckpt)
			}
			fmt.Println()

			// Set up output directory
			if outputDir == "" {
				outputDir = filepath.Join(os.Getenv("HOME"), ".clood", "gallery",
					fmt.Sprintf("anvil-%s", time.Now().Format("20060102-150405")))
			}
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				fmt.Printf("%s Create output dir: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}
			client.OutputDir = outputDir

			// Build batch config
			prompt := sd.NewPrompt(promptText)
			prompt.WithSeed(42069) // Fixed seed for fair comparison
			batch := sd.NewBatchConfig("anvil", prompt)
			batch.OutputDir = outputDir
			batch.Description = fmt.Sprintf("Anvil comparison: %s", promptText)

			for _, ckpt := range ckptList {
				batch.AddVariation(sd.Variation{
					Name:       ckpt,
					Checkpoint: ckpt,
				})
			}

			// Run generations
			startTime := time.Now()
			var results []sd.VariationResult

			for i, ckpt := range ckptList {
				fmt.Printf("%s [%d/%d] %s\n",
					tui.AccentStyle.Render(">>>"),
					i+1, len(ckptList), ckpt)

				cfg := sd.DefaultWorkflowConfig()
				cfg.Prompt = prompt
				cfg.Checkpoint = ckpt
				cfg.OutputPrefix = fmt.Sprintf("anvil_%s", sanitizeFilename(ckpt))

				genStart := time.Now()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				result, err := client.Generate(ctx, cfg)
				cancel()

				vr := sd.VariationResult{
					Variation: sd.Variation{
						Name:       ckpt,
						Checkpoint: ckpt,
					},
					GenerateTime: time.Since(genStart),
					Metadata: sd.ImageMetadata{
						Checkpoint: ckpt,
						Prompt:     promptText,
						Seed:       prompt.Seed,
						Steps:      cfg.Steps,
						CFGScale:   cfg.CFGScale,
					},
				}

				if err != nil {
					vr.Success = false
					vr.Error = err.Error()
					fmt.Printf("    %s %v\n", tui.ErrorStyle.Render("FAILED:"), err)
				} else {
					vr.Success = true
					if len(result.ImagePaths) > 0 {
						vr.OutputPath = result.ImagePaths[0]
					}
					fmt.Printf("    %s %.1fs\n", tui.SuccessStyle.Render("DONE"), vr.GenerateTime.Seconds())
				}

				results = append(results, vr)
			}

			// Build batch result
			batchResult := &sd.BatchResult{
				Config:    batch,
				StartTime: startTime,
				EndTime:   time.Now(),
				Results:   results,
				TotalTime: time.Since(startTime),
			}

			// Generate comparison gallery
			gallery := sd.NewCompareGallery(batchResult)
			htmlPath := filepath.Join(outputDir, "compare.html")
			f, err := os.Create(htmlPath)
			if err != nil {
				fmt.Printf("%s Create gallery: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}
			if err := gallery.Render(f); err != nil {
				f.Close()
				fmt.Printf("%s Render gallery: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}
			f.Close()

			// Also generate markdown report
			mdPath := filepath.Join(outputDir, "report.md")
			os.WriteFile(mdPath, []byte(sd.MarkdownReport(batchResult)), 0644)

			fmt.Println()
			fmt.Println(tui.RenderHeader("RESULTS"))
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Gallery:"), htmlPath)
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Report:"), mdPath)
			fmt.Printf("  %s %.1fs\n", tui.MutedStyle.Render("Total time:"), batchResult.TotalTime.Seconds())

			if openGallery {
				openImage(htmlPath)
			}
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ComfyUI host URL")
	cmd.Flags().StringVar(&checkpoints, "checkpoints", "", "Comma-separated checkpoint list")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory")
	cmd.Flags().BoolVar(&openGallery, "open", false, "Open gallery after generation")

	return cmd
}

// sdGalleryCmd shows generated images.
func sdGalleryCmd() *cobra.Command {
	var limit int
	var openLatest bool

	cmd := &cobra.Command{
		Use:   "gallery",
		Short: "View generated images",
		Long: `List and view images in the clood gallery.

Examples:
  clood sd gallery           # List recent images
  clood sd gallery --open    # Open most recent image
  clood sd gallery -n 20     # Show last 20 images`,
		Run: func(cmd *cobra.Command, args []string) {
			galleryDir := filepath.Join(os.Getenv("HOME"), ".clood", "gallery")

			if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
				fmt.Println(tui.MutedStyle.Render("No images yet. Generate some with 'clood sd paint'"))
				return
			}

			entries, err := os.ReadDir(galleryDir)
			if err != nil {
				fmt.Printf("%s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			// Filter and sort images
			var images []os.DirEntry
			for _, e := range entries {
				if !e.IsDir() && (strings.HasSuffix(e.Name(), ".png") ||
					strings.HasSuffix(e.Name(), ".jpg") ||
					strings.HasSuffix(e.Name(), ".webp")) {
					images = append(images, e)
				}
			}

			if len(images) == 0 {
				fmt.Println(tui.MutedStyle.Render("No images yet. Generate some with 'clood sd paint'"))
				return
			}

			// Limit output
			if limit > 0 && len(images) > limit {
				images = images[len(images)-limit:]
			}

			fmt.Println(tui.RenderHeader("GALLERY"))
			fmt.Printf("%s %s\n\n", tui.MutedStyle.Render("Location:"), galleryDir)

			for _, img := range images {
				info, _ := img.Info()
				size := ""
				if info != nil {
					size = fmt.Sprintf("%.1f KB", float64(info.Size())/1024)
				}
				fmt.Printf("  %s  %s\n", img.Name(), tui.MutedStyle.Render(size))
			}

			if openLatest && len(images) > 0 {
				latest := filepath.Join(galleryDir, images[len(images)-1].Name())
				openImage(latest)
			}
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Number of images to show")
	cmd.Flags().BoolVar(&openLatest, "open", false, "Open most recent image")

	return cmd
}

// sdPresetsCmd shows available presets.
func sdPresetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "presets",
		Short: "Show available style presets",
		Long:  `Display the built-in style presets like Ghibli, Pulp Fiction, etc.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(tui.RenderHeader("STYLE PRESETS"))
			fmt.Println()

			fmt.Printf("%s\n", tui.AccentStyle.Render("Ghibli Style"))
			fmt.Println("  Studio Ghibli aesthetic - watercolor, soft lighting, whimsical")
			fmt.Println("  LoRA: ghibli_style_offset @ 0.8")
			fmt.Println("  Usage: clood sd paint \"subject\" --lora ghibli_style")
			fmt.Println()

			fmt.Printf("%s\n", tui.AccentStyle.Render("Pulp Fiction × Ghibli"))
			fmt.Println("  The briefcase scene reimagined by Miyazaki")
			fmt.Println("  Seed: 42069 (the sacred seed)")
			fmt.Println("  For catfight comparison use: clood sd anvil")
			fmt.Println()

			aquarium := sd.ExampleAquarium()
			fmt.Printf("%s\n", tui.AccentStyle.Render("The Aquarium (Model Taxonomy)"))
			fmt.Println()
			fmt.Println("  Fish (Base Models):")
			for _, fish := range aquarium.Fish {
				fmt.Printf("    %s - %s (%s)\n",
					tui.AccentStyle.Render(string(fish.Species)),
					fish.Name,
					fish.Checkpoint)
			}
			fmt.Println()
			fmt.Println("  Shrimp (LoRAs):")
			for _, shrimp := range aquarium.Shrimp {
				fmt.Printf("    %s - %s (%.1f)\n",
					tui.AccentStyle.Render(string(shrimp.Species)),
					shrimp.Name,
					shrimp.DefaultWeight)
			}
		},
	}
}

// openImage opens an image or HTML file with the system default application.
func openImage(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		return
	}
	cmd.Start()
}

// sdRemixCmd parses external source and attempts reproduction.
func sdRemixCmd() *cobra.Command {
	var host string
	var bestEffort bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "remix [url-or-params]",
		Short: "Remix a CivitAI/HuggingFace image with local resources",
		Long: `Parse generation parameters from CivitAI, HuggingFace, or pasted text,
analyze what can be reproduced locally, and generate with available resources.

Supported sources:
  - CivitAI image URLs: https://civitai.com/images/12345
  - HuggingFace models: https://huggingface.co/owner/model
  - Pasted A1111/ComfyUI generation parameters
  - Raw prompt text

Examples:
  clood sd remix "https://civitai.com/images/12345"
  clood sd remix --best-effort  # Skip prompts, generate immediately

  # Paste generation parameters directly:
  clood sd remix "masterpiece, best quality, 1girl..."`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := strings.Join(args, " ")
			host := getComfyUIHost(host)
			client := sd.NewClient(host)

			// Check ComfyUI connection
			if err := client.Ping(); err != nil {
				fmt.Printf("%s ComfyUI not reachable: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			// Parse the input
			parser := sd.NewMultiSourceParser()
			source, err := parser.Parse(input)
			if err != nil {
				fmt.Printf("%s Could not parse input: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			if !jsonOutput {
				fmt.Println(tui.RenderHeader("REMIX - STACK ANALYSIS"))
				fmt.Printf("%s %s\n", tui.MutedStyle.Render("Source:"), source.Source)
				if source.SourceURL != "" {
					fmt.Printf("%s %s\n", tui.MutedStyle.Render("URL:"), source.SourceURL)
				}
				fmt.Println()
			}

			// Build local inventory from ComfyUI
			inventory := sd.NewLocalInventory()
			if err := inventory.FromComfyUIAPI(client); err != nil {
				fmt.Printf("%s Could not get inventory: %v\n", tui.WarningStyle.Render("WARN:"), err)
			}

			// Build stack and analyze
			stack := sd.NewStackFromSource(source)
			analysis := sd.AnalyzeStack(stack, inventory)

			if jsonOutput {
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"source":          source,
					"overall":         analysis.Overall,
					"can_generate":    analysis.Overall.CanGenerate,
					"recovery":        analysis.Overall.OverallRecovery,
					"blocking_issues": analysis.Overall.BlockingIssues,
				})
				return
			}

			// Print analysis report
			fmt.Print(analysis.FormatReport())
			fmt.Println()

			// If can't generate and not best-effort, offer options
			if !analysis.Overall.CanGenerate {
				fmt.Println(tui.WarningStyle.Render("Cannot generate - missing critical resources"))
				for _, issue := range analysis.Overall.BlockingIssues {
					fmt.Printf("  • %s\n", issue)
				}
				return
			}

			if !bestEffort {
				// Show options and ask user
				fmt.Println(tui.MutedStyle.Render("Options:"))
				for i, opt := range analysis.Options {
					fmt.Printf("  [%d] %s (%.0f%% fidelity)\n", i+1, opt.Label, opt.Recovery*100)
				}
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("Use --best-effort to generate immediately"))
				return
			}

			// Best effort generation
			fmt.Printf("%s Generating with %.0f%% expected fidelity...\n",
				tui.AccentStyle.Render(">>>"),
				analysis.Overall.OverallRecovery*100)

			// Find first available checkpoint
			var checkpoint string
			for _, layer := range analysis.Layers {
				if layer.Layer == sd.LayerCheckpoint && layer.Available != nil {
					if ckpt, ok := layer.Available.(sd.LocalCheckpoint); ok {
						checkpoint = ckpt.Name
						break
					}
				}
			}
			if checkpoint == "" && len(inventory.Checkpoints) > 0 {
				checkpoint = inventory.Checkpoints[0].Name
			}

			// Build workflow config
			cfg := sd.DefaultWorkflowConfig()
			cfg.Checkpoint = checkpoint
			cfg.Prompt = sd.NewPrompt(source.Prompt)
			if source.NegativePrompt != "" {
				cfg.Prompt.WithNegative(source.NegativePrompt)
			}
			if source.Seed != 0 {
				cfg.Prompt.WithSeed(source.Seed)
			}
			if source.Sampler.Steps > 0 {
				cfg.Steps = source.Sampler.Steps
			}
			if source.Sampler.CFGScale > 0 {
				cfg.CFGScale = source.Sampler.CFGScale
			}
			if source.Dimensions.Width > 0 {
				cfg.Width = source.Dimensions.Width
			}
			if source.Dimensions.Height > 0 {
				cfg.Height = source.Dimensions.Height
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			result, err := client.Generate(ctx, cfg)
			if err != nil {
				fmt.Printf("%s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			fmt.Printf("%s Generated in %.1fs\n", tui.SuccessStyle.Render("DONE"), result.Duration.Seconds())
			for _, path := range result.ImagePaths {
				fmt.Printf("  %s\n", path)
			}
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ComfyUI host URL")
	cmd.Flags().BoolVar(&bestEffort, "best-effort", false, "Generate immediately with available resources")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// sdDeconstructCmd analyzes a generation without generating.
func sdDeconstructCmd() *cobra.Command {
	var host string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "deconstruct [url-or-params]",
		Short: "Analyze a generation stack without generating",
		Long: `Parse and analyze generation parameters without actually generating.
Shows layer-by-layer breakdown of what's required vs available.

Perfect for:
  - Checking if you can reproduce an image before downloading
  - Understanding what models/LoRAs make up a generation
  - Planning which resources to download

Examples:
  clood sd deconstruct "https://civitai.com/images/12345"
  clood sd deconstruct "masterpiece, 1girl, <lora:ghibli:0.8>"`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := strings.Join(args, " ")
			host := getComfyUIHost(host)
			client := sd.NewClient(host)

			// Parse input
			parser := sd.NewMultiSourceParser()
			source, err := parser.Parse(input)
			if err != nil {
				fmt.Printf("%s Could not parse: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				return
			}

			if !jsonOutput {
				fmt.Println(tui.RenderHeader("DECONSTRUCT - STACK ANALYSIS"))
			}

			// Build inventory (optional - we can analyze without ComfyUI)
			inventory := sd.NewLocalInventory()
			if client.Ping() == nil {
				inventory.FromComfyUIAPI(client)
			}

			// Build and analyze stack
			stack := sd.NewStackFromSource(source)
			analysis := sd.AnalyzeStack(stack, inventory)

			if jsonOutput {
				// Detailed JSON output
				layers := make([]map[string]interface{}, 0)
				for _, l := range analysis.Layers {
					layers = append(layers, map[string]interface{}{
						"layer":       l.Layer.String(),
						"required":    l.Required,
						"match":       l.Match.String(),
						"recovery":    l.Recovery,
						"workaround":  l.Workaround,
						"downloadURL": l.DownloadURL,
					})
				}
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"source": map[string]interface{}{
						"type":       source.Source,
						"url":        source.SourceURL,
						"prompt":     source.Prompt,
						"checkpoint": source.Checkpoint,
						"loras":      source.LoRAs,
					},
					"layers":  layers,
					"overall": analysis.Overall,
					"options": analysis.Options,
				})
				return
			}

			// Print the formatted report
			fmt.Print(analysis.FormatReport())

			// Show extracted prompt details
			if source.Prompt != "" {
				fmt.Println()
				fmt.Println(tui.RenderHeader("PROMPT DETAILS"))
				if len(source.Prompt) > 200 {
					fmt.Printf("%s %s...\n", tui.MutedStyle.Render("Positive:"), source.Prompt[:200])
				} else {
					fmt.Printf("%s %s\n", tui.MutedStyle.Render("Positive:"), source.Prompt)
				}
				if source.NegativePrompt != "" {
					if len(source.NegativePrompt) > 100 {
						fmt.Printf("%s %s...\n", tui.MutedStyle.Render("Negative:"), source.NegativePrompt[:100])
					} else {
						fmt.Printf("%s %s\n", tui.MutedStyle.Render("Negative:"), source.NegativePrompt)
					}
				}
			}

			// Show download suggestions if missing pieces
			var missingDownloads []sd.DownloadItem
			for _, opt := range analysis.Options {
				if opt.ID == "download_missing" {
					missingDownloads = opt.Downloads
					break
				}
			}

			if len(missingDownloads) > 0 {
				fmt.Println()
				fmt.Println(tui.RenderHeader("DOWNLOAD SUGGESTIONS"))
				for _, dl := range missingDownloads {
					fmt.Printf("  %s %s\n", tui.AccentStyle.Render(dl.Type+":"), dl.URL)
				}
			}
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ComfyUI host URL (optional)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// sdInventoryCmd shows local model inventory.
func sdInventoryCmd() *cobra.Command {
	var host string
	var scanPath string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Show local model inventory",
		Long: `Display all locally available checkpoints, LoRAs, VAEs, and embeddings.

Can scan from ComfyUI API or directly from filesystem.

Examples:
  clood sd inventory                    # From ComfyUI API
  clood sd inventory --scan ~/ComfyUI   # Scan filesystem
  clood sd inventory --json             # Machine-readable`,
		Run: func(cmd *cobra.Command, args []string) {
			inventory := sd.NewLocalInventory()

			if scanPath != "" {
				// Scan filesystem
				if !jsonOutput {
					fmt.Printf("%s Scanning %s...\n", tui.AccentStyle.Render(">>>"), scanPath)
				}
				if err := inventory.ScanComfyUI(scanPath); err != nil {
					fmt.Printf("%s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
					return
				}
			} else {
				// Get from ComfyUI API
				host := getComfyUIHost(host)
				client := sd.NewClient(host)

				if !jsonOutput {
					fmt.Printf("%s Getting inventory from %s...\n", tui.AccentStyle.Render(">>>"), host)
				}

				if err := client.Ping(); err != nil {
					fmt.Printf("%s ComfyUI not reachable: %v\n", tui.ErrorStyle.Render("ERROR:"), err)
					fmt.Println(tui.MutedStyle.Render("Use --scan to scan filesystem directly"))
					return
				}

				if err := inventory.FromComfyUIAPI(client); err != nil {
					fmt.Printf("%s %v\n", tui.WarningStyle.Render("WARN:"), err)
				}
			}

			if jsonOutput {
				data, _ := inventory.ToJSON()
				os.Stdout.Write(data)
				return
			}

			// Pretty output
			fmt.Println()
			fmt.Println(tui.RenderHeader("LOCAL INVENTORY"))

			// Hardware
			if inventory.Hardware.GPUName != "" {
				fmt.Println()
				fmt.Println(tui.AccentStyle.Render("Hardware:"))
				vramGB := float64(inventory.Hardware.TotalVRAM) / (1024 * 1024 * 1024)
				fmt.Printf("  %s (%.1fGB VRAM, %s)\n",
					inventory.Hardware.GPUName,
					vramGB,
					inventory.Hardware.Backend)
			}

			// Checkpoints
			if len(inventory.Checkpoints) > 0 {
				fmt.Println()
				fmt.Printf("%s (%d)\n", tui.AccentStyle.Render("Checkpoints:"), len(inventory.Checkpoints))
				for _, ckpt := range inventory.Checkpoints {
					baseInfo := ""
					if ckpt.BaseModel != "" && ckpt.BaseModel != "unknown" {
						baseInfo = fmt.Sprintf(" [%s]", ckpt.BaseModel)
					}
					fmt.Printf("  • %s%s\n", ckpt.Name, tui.MutedStyle.Render(baseInfo))
				}
			}

			// LoRAs
			if len(inventory.LoRAs) > 0 {
				fmt.Println()
				fmt.Printf("%s (%d)\n", tui.AccentStyle.Render("LoRAs:"), len(inventory.LoRAs))
				for _, lora := range inventory.LoRAs {
					fmt.Printf("  • %s\n", lora.Name)
				}
			}

			// VAEs
			if len(inventory.VAEs) > 0 {
				fmt.Println()
				fmt.Printf("%s (%d)\n", tui.AccentStyle.Render("VAEs:"), len(inventory.VAEs))
				for _, vae := range inventory.VAEs {
					fmt.Printf("  • %s\n", vae.Name)
				}
			}

			// Embeddings
			if len(inventory.Embeddings) > 0 {
				fmt.Println()
				fmt.Printf("%s (%d)\n", tui.AccentStyle.Render("Embeddings:"), len(inventory.Embeddings))
				for _, emb := range inventory.Embeddings {
					fmt.Printf("  • %s\n", emb.Name)
				}
			}

			fmt.Println()
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ComfyUI host URL")
	cmd.Flags().StringVar(&scanPath, "scan", "", "Scan ComfyUI directory instead of API")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}
