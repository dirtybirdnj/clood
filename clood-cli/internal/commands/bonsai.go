package commands

import (
	"fmt"
	"os/exec"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Size presets for cbonsai
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

	cmd := &cobra.Command{
		Use:   "bonsai",
		Short: "Generate ASCII bonsai tree using cbonsai",
		Long: `Generate beautiful ASCII bonsai trees for the Server Garden.

The bonsai represents careful cultivation of local resources.
Each tree is unique, grown from the seeds of your configuration.`,
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

			// Check if cbonsai is installed first
			if _, lookErr := exec.LookPath("cbonsai"); lookErr != nil {
				fmt.Println(tui.ErrorStyle.Render("cbonsai not found"))
				fmt.Println(tui.MutedStyle.Render("Install with: brew install cbonsai"))
				return nil
			}

			// Run cbonsai
			cbCmd := exec.Command("cbonsai", cbArgs...)
			output, err := cbCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("cbonsai error: %v\n%s", err, output)
			}

			fmt.Print(string(output))
			// Reset terminal attributes after cbonsai (it may leave escape codes)
			fmt.Print("\033[0m")
			return nil
		},
	}

	cmd.Flags().StringVarP(&size, "size", "s", "", "Size preset (tiny/small/medium/large/ancient)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Custom message to display")

	return cmd
}
