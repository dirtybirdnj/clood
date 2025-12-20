package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dirtybirdnj/clood/internal/focus"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// FocusState persists across sessions
type FocusState struct {
	Goal       string   `yaml:"goal" json:"goal"`
	Keywords   []string `yaml:"keywords" json:"keywords"`
	DriftCount int      `yaml:"drift_count" json:"drift_count"`
	Status     string   `yaml:"status" json:"status"`
}

func FocusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "focus",
		Short: "Focus Guardian - drift detection for sessions (Gamera-kun)",
		Long: `The Focus Guardian (Gamera-kun, the slow tortoise) watches over your session
to ensure you stay on track with your stated goal.

When you drift from your goal for too long, the guardian gently reminds you.

Commands:
  clood focus set "implement auth"    Set session goal
  clood focus check "some message"    Check if message drifts from goal
  clood focus status                  Show current focus state
  clood focus reset                   Clear drift counter
  clood focus clear                   Remove goal entirely`,
	}

	cmd.AddCommand(focusSetCmd())
	cmd.AddCommand(focusCheckCmd())
	cmd.AddCommand(focusStatusCmd())
	cmd.AddCommand(focusResetCmd())
	cmd.AddCommand(focusClearCmd())

	return cmd
}

func focusSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [goal]",
		Short: "Set the session goal",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			goal := strings.Join(args, " ")
			guardian := focus.NewGuardian(goal)

			state := &FocusState{
				Goal:       goal,
				Keywords:   guardian.Keywords,
				DriftCount: 0,
				Status:     "On track",
			}

			if err := saveFocusState(state); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error saving focus: " + err.Error()))
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(state, "", "  ")
				fmt.Println(string(data))
				return
			}

			fmt.Println(tui.SuccessStyle.Render("✓ Focus set"))
			fmt.Printf("\n  Goal: %s\n", goal)
			fmt.Printf("  Keywords: %s\n", strings.Join(guardian.Keywords, ", "))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Gamera-kun is watching. Stay on track."))
		},
	}
}

func focusCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [message]",
		Short: "Check if a message drifts from the goal",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			message := strings.Join(args, " ")

			state, err := loadFocusState()
			if err != nil || state.Goal == "" {
				fmt.Println(tui.MutedStyle.Render("No goal set. Use 'clood focus set <goal>' first."))
				return
			}

			guardian := focus.NewGuardian(state.Goal)
			guardian.DriftCount = state.DriftCount

			result := guardian.CheckMessage(message)

			// Update state
			state.DriftCount = guardian.DriftCount
			state.Status = guardian.GetStatus()
			saveFocusState(state)

			if output.IsJSON() {
				data, _ := json.MarshalIndent(map[string]interface{}{
					"is_drift":    result.IsDrift,
					"confidence":  result.Confidence,
					"keywords":    result.Keywords,
					"drift_count": guardian.DriftCount,
					"status":      state.Status,
					"warning":     result.Message,
				}, "", "  ")
				fmt.Println(string(data))
				return
			}

			if result.IsDrift {
				if result.Message != "" {
					// Threshold reached - show Gamera-kun warning
					fmt.Println()
					fmt.Println(tui.WarningStyle.Render("🐢 Gamera-kun stirs..."))
					fmt.Println()
					fmt.Printf("  You seem to be talking about: %s\n", result.Message)
					fmt.Printf("  But your goal was: %s\n", state.Goal)
					fmt.Println()
					fmt.Println(tui.MutedStyle.Render("  Is this still what you want to work on?"))
					fmt.Println(tui.MutedStyle.Render("  Use 'clood focus reset' to acknowledge, or 'clood focus set' to change goals."))
				} else {
					fmt.Printf("⚠️  Drift detected (%d/%d before warning)\n", guardian.DriftCount, guardian.Threshold)
				}
			} else {
				fmt.Println(tui.SuccessStyle.Render("✓ On track"))
				if len(result.Keywords) > 0 {
					fmt.Printf("  Matched: %s\n", strings.Join(result.Keywords, ", "))
				}
			}
		},
	}
}

func focusStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current focus state",
		Run: func(cmd *cobra.Command, args []string) {
			state, err := loadFocusState()
			if err != nil || state.Goal == "" {
				if output.IsJSON() {
					fmt.Println(`{"goal": null, "status": "No goal set"}`)
					return
				}
				fmt.Println(tui.MutedStyle.Render("No goal set."))
				fmt.Println(tui.MutedStyle.Render("Use 'clood focus set <goal>' to start tracking."))
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(state, "", "  ")
				fmt.Println(string(data))
				return
			}

			fmt.Println()
			fmt.Println(tui.RenderHeader("Focus Guardian Status"))
			fmt.Println()

			// Status indicator
			var statusIcon string
			switch state.Status {
			case "On track":
				statusIcon = tui.SuccessStyle.Render("🟢 On track")
			case "Wandering slightly":
				statusIcon = tui.WarningStyle.Render("🟡 Wandering slightly")
			case "Drifting from goal":
				statusIcon = tui.ErrorStyle.Render("🔴 Drifting from goal")
			default:
				statusIcon = state.Status
			}

			fmt.Printf("  Status:     %s\n", statusIcon)
			fmt.Printf("  Goal:       %s\n", state.Goal)
			fmt.Printf("  Keywords:   %s\n", strings.Join(state.Keywords, ", "))
			fmt.Printf("  Drift Count: %d\n", state.DriftCount)
			fmt.Println()
		},
	}
}

func focusResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset drift counter (acknowledge warning)",
		Run: func(cmd *cobra.Command, args []string) {
			state, err := loadFocusState()
			if err != nil || state.Goal == "" {
				fmt.Println(tui.MutedStyle.Render("No goal set."))
				return
			}

			state.DriftCount = 0
			state.Status = "On track"
			saveFocusState(state)

			if output.IsJSON() {
				data, _ := json.MarshalIndent(state, "", "  ")
				fmt.Println(string(data))
				return
			}

			fmt.Println(tui.SuccessStyle.Render("✓ Drift counter reset"))
			fmt.Println(tui.MutedStyle.Render("  Gamera-kun returns to slumber."))
		},
	}
}

func focusClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear goal entirely",
		Run: func(cmd *cobra.Command, args []string) {
			path := getFocusStatePath()
			os.Remove(path)

			if output.IsJSON() {
				fmt.Println(`{"cleared": true}`)
				return
			}

			fmt.Println(tui.SuccessStyle.Render("✓ Focus cleared"))
			fmt.Println(tui.MutedStyle.Render("  No goal is set."))
		},
	}
}

// GetFocusSummary returns a summary for handoff/session
func GetFocusSummary() string {
	state, err := loadFocusState()
	if err != nil || state.Goal == "" {
		return ""
	}

	return fmt.Sprintf("Goal: %s | Status: %s", state.Goal, state.Status)
}

// GetFocusState returns current focus state for external use
func GetFocusState() *FocusState {
	state, err := loadFocusState()
	if err != nil {
		return nil
	}
	return state
}

func getFocusStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clood", "focus.yaml")
}

func loadFocusState() (*FocusState, error) {
	path := getFocusStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state FocusState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func saveFocusState(state *FocusState) error {
	path := getFocusStatePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
