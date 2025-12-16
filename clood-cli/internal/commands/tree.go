package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func TreeCmd() *cobra.Command {
	var depth int
	var showHidden bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "tree [path]",
		Short: "Display directory tree (respects .gitignore)",
		Long:  "Smart directory tree that respects .gitignore and shows file metadata.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			if jsonOutput {
				// TODO: JSON output
				fmt.Println(`{"error": "JSON output not yet implemented"}`)
				return
			}

			fmt.Println(tui.RenderHeader("Directory Tree: " + path))
			fmt.Println()

			err := printTree(path, "", depth, showHidden, 0)
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render(err.Error()))
			}
		},
	}

	cmd.Flags().IntVarP(&depth, "depth", "d", 3, "Maximum depth to traverse")
	cmd.Flags().BoolVarP(&showHidden, "all", "a", false, "Show hidden files")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func printTree(path string, prefix string, maxDepth int, showHidden bool, currentDepth int) error {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Filter entries
	var filtered []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden unless requested
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		// Skip common ignores (simplified - TODO: parse .gitignore)
		if name == "node_modules" || name == "vendor" || name == "__pycache__" || name == ".git" {
			continue
		}

		filtered = append(filtered, entry)
	}

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		name := entry.Name()
		displayName := name

		if entry.IsDir() {
			displayName = tui.HeaderStyle.Render(name + "/")
		} else {
			// Show file size
			info, err := entry.Info()
			if err == nil {
				size := formatSize(info.Size())
				displayName = name + " " + tui.MutedStyle.Render("("+size+")")
			}
		}

		fmt.Println(prefix + connector + displayName)

		if entry.IsDir() {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			printTree(filepath.Join(path, name), newPrefix, maxDepth, showHidden, currentDepth+1)
		}
	}

	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
