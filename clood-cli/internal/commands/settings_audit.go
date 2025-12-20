package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// ClaudeSettings represents Claude Code's settings.local.json structure
type ClaudeSettings struct {
	Permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny,omitempty"`
	} `json:"permissions"`
	EnableAllProjectMcpServers bool     `json:"enableAllProjectMcpServers,omitempty"`
	EnabledMcpjsonServers      []string `json:"enabledMcpjsonServers,omitempty"`
}

// Patterns that indicate corrupted permission entries
var corruptPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^Bash\(#`),                    // Comments learned as commands
	regexp.MustCompile(`^Bash\(__NEW_LINE__`),         // Internal token leak
	regexp.MustCompile(`^Bash\(done\)$`),              // Shell keyword
	regexp.MustCompile(`^Bash\(do\)$`),                // Shell keyword
	regexp.MustCompile(`^Bash\(do [^:]+[^*]\)$`),      // Partial loop command without :*
	regexp.MustCompile(`^Bash\(for\)$`),               // Shell keyword alone
	regexp.MustCompile(`^Bash\(then\)$`),              // Shell keyword
	regexp.MustCompile(`^Bash\(else\)$`),              // Shell keyword
	regexp.MustCompile(`^Bash\(fi\)$`),                // Shell keyword
	regexp.MustCompile(`^Bash\(esac\)$`),              // Shell keyword
}

func SettingsAuditCmd() *cobra.Command {
	var fix bool
	var path string

	cmd := &cobra.Command{
		Use:   "settings-audit",
		Short: "Audit Claude Code settings for corrupted permissions",
		Long: `Detects and optionally fixes corrupted permission entries in Claude Code's
settings.local.json files.

Claude Code's permission learning system can accidentally capture:
- Shell keywords (do, done, for, then, else, fi)
- Comments starting with #
- Internal tokens like __NEW_LINE__
- Partial commands from for loops

This command finds and removes these corrupted entries.

Examples:
  clood settings-audit                    # Audit current directory
  clood settings-audit --path ~/Code/foo  # Audit specific project
  clood settings-audit --fix              # Fix corrupted entries`,
		Run: func(cmd *cobra.Command, args []string) {
			runSettingsAudit(path, fix)
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Remove corrupted entries (otherwise just report)")
	cmd.Flags().StringVar(&path, "path", ".", "Path to project directory")

	return cmd
}

func runSettingsAudit(projectPath string, fix bool) {
	// Find settings file
	settingsPath := filepath.Join(projectPath, ".claude", "settings.local.json")

	// Check if file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		fmt.Println(tui.MutedStyle.Render("No .claude/settings.local.json found in " + projectPath))
		return
	}

	// Read file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		fmt.Println(tui.ErrorStyle.Render("Error reading settings: " + err.Error()))
		return
	}

	// Parse JSON
	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Println(tui.ErrorStyle.Render("Error parsing settings JSON: " + err.Error()))
		return
	}

	// Find corrupted entries
	var corrupted []string
	var clean []string

	for _, perm := range settings.Permissions.Allow {
		isCorrupt := false
		for _, pattern := range corruptPatterns {
			if pattern.MatchString(perm) {
				isCorrupt = true
				corrupted = append(corrupted, perm)
				break
			}
		}
		if !isCorrupt {
			clean = append(clean, perm)
		}
	}

	// Report
	fmt.Println(tui.RenderHeader("Claude Settings Audit"))
	fmt.Println()
	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("File:"), settingsPath)
	fmt.Printf("  %s %d\n", tui.MutedStyle.Render("Total permissions:"), len(settings.Permissions.Allow))
	fmt.Println()

	if len(corrupted) == 0 {
		fmt.Println(tui.SuccessStyle.Render("  ✓ No corrupted entries found"))
		return
	}

	fmt.Println(tui.WarningStyle.Render(fmt.Sprintf("  ⚠ Found %d corrupted entries:", len(corrupted))))
	fmt.Println()

	for _, entry := range corrupted {
		// Truncate long entries
		display := entry
		if len(display) > 60 {
			display = display[:57] + "..."
		}
		fmt.Printf("    %s %s\n", tui.ErrorStyle.Render("✗"), display)
	}
	fmt.Println()

	if !fix {
		fmt.Println(tui.MutedStyle.Render("  Run with --fix to remove these entries"))
		return
	}

	// Fix: write clean permissions back
	settings.Permissions.Allow = clean

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		fmt.Println(tui.ErrorStyle.Render("Error encoding settings: " + err.Error()))
		return
	}

	// Add newline at end
	newData = append(newData, '\n')

	if err := os.WriteFile(settingsPath, newData, 0644); err != nil {
		fmt.Println(tui.ErrorStyle.Render("Error writing settings: " + err.Error()))
		return
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("  ✓ Removed %d corrupted entries", len(corrupted))))
	fmt.Printf("  %s %d clean permissions remain\n", tui.MutedStyle.Render("Remaining:"), len(clean))
}

// Also check for common issues
func checkForDuplicates(perms []string) []string {
	seen := make(map[string]bool)
	var dups []string
	for _, p := range perms {
		if seen[p] {
			dups = append(dups, p)
		}
		seen[p] = true
	}
	return dups
}

// Check for patterns that could be consolidated
func suggestConsolidations(perms []string) []string {
	var suggestions []string

	// Look for patterns like Bash(foo --bar) that could be Bash(foo:*)
	exactCommands := make(map[string][]string)
	for _, p := range perms {
		if strings.HasPrefix(p, "Bash(") && !strings.HasSuffix(p, ":*)") {
			// Extract the command base
			inner := strings.TrimPrefix(p, "Bash(")
			inner = strings.TrimSuffix(inner, ")")
			parts := strings.Fields(inner)
			if len(parts) > 0 {
				base := parts[0]
				exactCommands[base] = append(exactCommands[base], p)
			}
		}
	}

	for base, commands := range exactCommands {
		if len(commands) > 2 {
			suggestions = append(suggestions,
				fmt.Sprintf("Consider: Bash(%s:*) instead of %d specific commands", base, len(commands)))
		}
	}

	return suggestions
}
