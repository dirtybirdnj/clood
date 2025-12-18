package commands

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/system"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func SystemCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "system",
		Short: "Display hardware information and recommendations",
		Long: `Detects and displays local hardware information relevant to LLM inference:
  - CPU model and cores
  - Memory (RAM)
  - GPU type and VRAM
  - Available disk space
  - Model recommendations based on hardware`,
		Run: func(cmd *cobra.Command, args []string) {
			hw, err := system.DetectHardware()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error detecting hardware: " + err.Error()))
				return
			}

			// Check both local --json and global -j flag
			if jsonOutput || output.IsJSON() {
				data, _ := json.MarshalIndent(hw.JSON(), "", "  ")
				fmt.Println(string(data))
				return
			}

			// Pretty output
			fmt.Println(tui.RenderHeader("Hardware Profile"))
			fmt.Println()
			fmt.Println(renderBox(hw.Summary()))

			// Recommendations
			models := hw.RecommendedModels()
			if len(models) > 0 {
				fmt.Println()
				fmt.Println(tui.RenderHeader("Recommended Models"))
				fmt.Println()
				for i, model := range models {
					if i == 0 {
						fmt.Printf("  %s %s %s\n",
							tui.SuccessStyle.Render("★"),
							model,
							tui.MutedStyle.Render("(optimal)"))
					} else {
						fmt.Printf("  %s %s\n",
							tui.MutedStyle.Render("○"),
							model)
					}
				}
			}

			// VRAM analysis
			fmt.Println()
			fmt.Println(tui.RenderHeader("Capacity Analysis"))
			fmt.Println()
			printCapacity("qwen2.5-coder:1.5b", 1.5, hw.OllamaVRAM)
			printCapacity("qwen2.5-coder:3b", 3, hw.OllamaVRAM)
			printCapacity("qwen2.5-coder:7b", 7, hw.OllamaVRAM)
			printCapacity("qwen2.5-coder:14b", 14, hw.OllamaVRAM)
			printCapacity("qwen2.5-coder:32b", 32, hw.OllamaVRAM)
			printCapacity("llama3.1:70b", 70, hw.OllamaVRAM)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func printCapacity(model string, sizeB float64, vram float64) {
	// Estimate VRAM needed (0.6 GB per billion params for Q4)
	needed := sizeB * 0.6

	var status string
	if vram >= needed*1.5 {
		status = tui.SuccessStyle.Render("✓ comfortable")
	} else if vram >= needed {
		status = tui.SuccessStyle.Render("✓ fits")
	} else if vram >= needed*0.8 {
		status = lipgloss.NewStyle().Foreground(tui.ColorWarning).Render("⚠ tight")
	} else {
		status = tui.ErrorStyle.Render("✗ won't fit")
	}

	fmt.Printf("  %-22s %4.1fGB needed  %s\n", model, needed, status)
}

func renderBox(content string) string {
	return tui.BoxStyle.Render(content)
}
