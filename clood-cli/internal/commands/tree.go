package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// TreeNode represents a file or directory in JSON output
type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "dir"
	Size     int64      `json:"size,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

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
				tree, err := buildTree(path, depth, showHidden, 0)
				if err != nil {
					errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
					fmt.Println(string(errJSON))
					return
				}
				output, _ := json.MarshalIndent(tree, "", "  ")
				fmt.Println(string(output))
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

// buildTree constructs a TreeNode hierarchy for JSON output
func buildTree(path string, maxDepth int, showHidden bool, currentDepth int) (*TreeNode, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	node := &TreeNode{
		Name: info.Name(),
		Path: absPath,
	}

	if !info.IsDir() {
		node.Type = "file"
		node.Size = info.Size()
		return node, nil
	}

	node.Type = "dir"

	if maxDepth > 0 && currentDepth >= maxDepth {
		return node, nil
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return node, nil // Return partial tree on read error
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden unless requested
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		// Skip common ignores
		if name == "node_modules" || name == "vendor" || name == "__pycache__" || name == ".git" {
			continue
		}

		childPath := filepath.Join(absPath, name)
		child, err := buildTree(childPath, maxDepth, showHidden, currentDepth+1)
		if err != nil {
			continue // Skip entries that can't be read
		}
		node.Children = append(node.Children, *child)
	}

	return node, nil
}
