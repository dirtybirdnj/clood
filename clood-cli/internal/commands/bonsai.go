package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dirtybirdnj/clood/internal/bonsai"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Size presets for cbonsai: [life, multiplier]
var sizePresets = map[string][2]int{
	"tiny":    {10, 2},
	"small":   {20, 3},
	"medium":  {32, 5},
	"large":   {60, 8},
	"ancient": {100, 12},
}

// BonsaiCmd returns a command that generates ASCII bonsai trees
func BonsaiCmd() *cobra.Command {
	var size string
	var message string
	var outputFile string
	var outputFormat string // "terminal", "ascii", "svg"
	var fontName string
	var seed int

	cmd := &cobra.Command{
		Use:   "bonsai",
		Short: "Generate ASCII bonsai tree using cbonsai",
		Long: `Generate beautiful ASCII bonsai trees for the Server Garden.

The bonsai represents careful cultivation of local resources.
Each tree is unique, grown from the seeds of your configuration.

DEFAULTS:
  Format: ascii (colored output, works everywhere)
  Size:   medium (life=32, multiplier=5)
  Seed:   random (use --seed for reproducible trees)

OUTPUT FORMATS:
  ascii     Colored ASCII with ANSI codes (default)
  plain     Plain ASCII without colors (for piping/logs)
  terminal  Direct ncurses passthrough (unreliable)
  svg       SVG with single-line fonts for pen plotters

SIZE PRESETS:
  tiny      Compact tree (life=10)
  small     Small tree (life=20)
  medium    Standard tree (life=32) [default]
  large     Large tree (life=60)
  ancient   Massive tree (life=100)

SVG FONTS (--font):
  HersheySans1    Classic single-stroke sans-serif [default]
  EMSDelight      Elegant cursive script
  EMSCasualHand   Casual handwritten style

EXAMPLES:
  # Default: colored ASCII output
  clood bonsai

  # Tiny tree with a message
  clood bonsai --size tiny --message "Hello"

  # Plain text (no colors) for piping/logs
  clood bonsai --format plain > tree.txt

  # SVG output to file
  clood bonsai -f svg -o tree.svg
  clood bonsai -f svg > tree.svg

  # SVG with seed and font, piped to file
  clood bonsai -f svg --seed 42 --font EMSDelight > fancy.svg

  # Pipe to other tools
  clood bonsai -f svg | rat-king fill -p lines -o filled.svg`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to medium if no size specified
			if size == "" {
				size = "medium"
			}

			preset, ok := sizePresets[size]
			if !ok {
				return fmt.Errorf("invalid size: %s (use tiny/small/medium/large/ancient)", size)
			}

			life := preset[0]
			multiplier := preset[1]

			// Build cbonsai arguments
			cbArgs := []string{
				"-p", // Static mode (no animation)
				fmt.Sprintf("-L%d", life),
				fmt.Sprintf("-M%d", multiplier),
			}

			if message != "" {
				cbArgs = append(cbArgs, "-m", message)
			}

			if seed > 0 {
				cbArgs = append(cbArgs, fmt.Sprintf("-s%d", seed))
			}

			// Check if cbonsai is installed first
			if _, lookErr := exec.LookPath("cbonsai"); lookErr != nil {
				fmt.Println(tui.ErrorStyle.Render("cbonsai not found"))
				fmt.Println(tui.MutedStyle.Render("Install with: brew install cbonsai"))
				return nil
			}

			// Default to ascii format
			if outputFormat == "" {
				outputFormat = "ascii"
			}

			// For terminal output, use direct output to preserve ncurses behavior
			if outputFormat == "terminal" {
				cbCmd := exec.Command("cbonsai", cbArgs...)
				cbCmd.Stdout = os.Stdout
				cbCmd.Stderr = os.Stderr
				if err := cbCmd.Run(); err != nil {
					return fmt.Errorf("cbonsai error: %v", err)
				}
				return nil
			}

			// For other formats, capture and process the output
			cbCmd := exec.Command("cbonsai", cbArgs...)
			output, err := cbCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("cbonsai error: %v", err)
			}

			// Parse the output with color extraction
			result := bonsai.ParseANSIWithColors(string(output))

			var finalOutput string
			switch outputFormat {
			case "ascii":
				// Use colored output for terminal display
				finalOutput = result.Colored

			case "plain":
				// Plain ASCII without colors (for piping/logs)
				finalOutput = result.ASCII

			case "svg":
				if fontName == "" {
					fontName = "HersheySans1"
				}
				gen, err := bonsai.NewSVGGenerator(fontName)
				if err != nil {
					return fmt.Errorf("failed to load font %s: %v", fontName, err)
				}
				finalOutput = gen.GenerateSVG(result)

			default:
				return fmt.Errorf("unknown format: %s (use ascii/plain/terminal/svg)", outputFormat)
			}

			// Write output
			if outputFile != "" && outputFile != "-" {
				if err := os.WriteFile(outputFile, []byte(finalOutput), 0644); err != nil {
					return fmt.Errorf("failed to write file: %v", err)
				}
				fmt.Printf("Wrote %s (%d bytes)\n", outputFile, len(finalOutput))
			} else {
				// Add spacing for terminal display
				fmt.Print("\n")
				fmt.Println(finalOutput)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&size, "size", "s", "", "Size preset: tiny/small/medium/large/ancient (default: medium)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message to display in the pot")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "", "Output format: ascii/plain/terminal/svg (default: ascii)")
	cmd.Flags().StringVar(&fontName, "font", "", "SVG font: HersheySans1/EMSDelight/EMSCasualHand (default: HersheySans1)")
	cmd.Flags().IntVar(&seed, "seed", 0, "Random seed for reproducible trees (default: random)")

	return cmd
}
