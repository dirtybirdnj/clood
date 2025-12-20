package commands

import (
	"encoding/json"
	"fmt"

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
  - All disk storage with usage
  - Ollama models directory location and headroom
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

			// Disk storage
			fmt.Println()
			fmt.Println(tui.RenderHeader("Storage"))
			fmt.Println()
			fmt.Print(hw.DiskSummary())

			// Recommendations by category
			fmt.Println()
			fmt.Println(tui.RenderHeader("Best Models For Your Hardware"))
			fmt.Println()

			recommendations := system.RecommendedByCategory(hw.OllamaVRAM)
			categoryOrder := []system.ModelCategory{
				system.CategoryCoding,
				system.CategoryReasoning,
				system.CategoryVision,
				system.CategoryGeneral,
			}

			for _, cat := range categoryOrder {
				models, ok := recommendations[cat]
				if !ok || len(models) == 0 {
					continue
				}
				info := system.GetCategoryInfo(cat)
				// Show first (best) model for each category
				fmt.Printf("  %s %-10s %s\n",
					info.Emoji,
					tui.HeaderStyle.Render(info.Name+":"),
					models[0])
			}

			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Run 'clood models' for detailed model info by category"))
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func renderBox(content string) string {
	return tui.BoxStyle.Render(content)
}
