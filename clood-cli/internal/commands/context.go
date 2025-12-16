package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func ContextCmd() *cobra.Command {
	var maxTokens int
	var includeTree bool
	var includeReadme bool

	cmd := &cobra.Command{
		Use:   "context [path]",
		Short: "Generate LLM-optimized context",
		Long:  "Generate a context blob optimized for LLM consumption, sized to fit token limits.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			context := generateContext(path, maxTokens, includeTree, includeReadme)
			fmt.Println(context)
		},
	}

	cmd.Flags().IntVarP(&maxTokens, "tokens", "t", 4000, "Target token count (approximate)")
	cmd.Flags().BoolVar(&includeTree, "tree", true, "Include directory tree")
	cmd.Flags().BoolVar(&includeReadme, "readme", true, "Include README content")

	return cmd
}

func generateContext(path string, maxTokens int, includeTree bool, includeReadme bool) string {
	var sb strings.Builder

	absPath, _ := filepath.Abs(path)
	projectName := filepath.Base(absPath)

	sb.WriteString(fmt.Sprintf("# Project: %s\n\n", projectName))

	// Detect project type
	summary, err := generateSummary(path)
	if err == nil {
		sb.WriteString(fmt.Sprintf("**Type:** %s\n", summary.Type))
		sb.WriteString(fmt.Sprintf("**Files:** %d files, %d directories\n", summary.Files, summary.Dirs))
		if len(summary.Indicators) > 0 {
			sb.WriteString(fmt.Sprintf("**Config:** %s\n", strings.Join(summary.Indicators, ", ")))
		}
		sb.WriteString("\n")
	}

	// Include README if present and requested
	if includeReadme {
		readmeContent := findAndReadReadme(path)
		if readmeContent != "" {
			sb.WriteString("## README\n\n")
			// Truncate if too long (rough estimate: 4 chars per token)
			maxReadmeChars := maxTokens * 2 // Leave room for other content
			if len(readmeContent) > maxReadmeChars {
				readmeContent = readmeContent[:maxReadmeChars] + "\n\n... (truncated)"
			}
			sb.WriteString(readmeContent)
			sb.WriteString("\n\n")
		}
	}

	// Include tree if requested
	if includeTree {
		sb.WriteString("## Directory Structure\n\n```\n")
		tree := captureTree(path, 3)
		sb.WriteString(tree)
		sb.WriteString("```\n\n")
	}

	// Include key files summary
	sb.WriteString("## Key Files\n\n")
	keyFiles := findKeyFiles(path)
	for _, kf := range keyFiles {
		sb.WriteString(fmt.Sprintf("- `%s`\n", kf))
	}

	return sb.String()
}

func findAndReadReadme(path string) string {
	readmeNames := []string{"README.md", "README", "readme.md", "Readme.md"}

	for _, name := range readmeNames {
		content, err := os.ReadFile(filepath.Join(path, name))
		if err == nil {
			return string(content)
		}
	}
	return ""
}

func captureTree(path string, depth int) string {
	var sb strings.Builder
	captureTreeRecursive(path, "", depth, 0, &sb)
	return sb.String()
}

func captureTreeRecursive(path string, prefix string, maxDepth int, currentDepth int, sb *strings.Builder) {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	var filtered []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if name == "node_modules" || name == "vendor" || name == "__pycache__" {
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
		if entry.IsDir() {
			name += "/"
		}

		sb.WriteString(prefix + connector + name + "\n")

		if entry.IsDir() {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			captureTreeRecursive(filepath.Join(path, entry.Name()), newPrefix, maxDepth, currentDepth+1, sb)
		}
	}
}

func findKeyFiles(path string) []string {
	keyFilePatterns := []string{
		"main.go", "main.py", "main.rs", "index.js", "index.ts",
		"app.go", "app.py", "app.js", "app.ts",
		"Makefile", "Dockerfile", "docker-compose.yml",
		"go.mod", "package.json", "Cargo.toml", "pyproject.toml",
	}

	var found []string
	for _, pattern := range keyFilePatterns {
		matches, _ := filepath.Glob(filepath.Join(path, pattern))
		for _, m := range matches {
			rel, _ := filepath.Rel(path, m)
			found = append(found, rel)
		}
		// Also check cmd/ for Go projects
		matches, _ = filepath.Glob(filepath.Join(path, "cmd", "*", pattern))
		for _, m := range matches {
			rel, _ := filepath.Rel(path, m)
			found = append(found, rel)
		}
	}

	return found
}

// Placeholder for syntax highlighting
func highlightCode(code string, lang string) string {
	_ = tui.MutedStyle // Use tui package
	return code
}
