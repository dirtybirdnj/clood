package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

type ProjectSummary struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Entry      string            `json:"entry,omitempty"`
	Files      int               `json:"files"`
	Dirs       int               `json:"dirs"`
	LOC        int               `json:"loc,omitempty"`
	Deps       []string          `json:"deps,omitempty"`
	Structure  map[string]string `json:"structure,omitempty"`
	Indicators []string          `json:"indicators"`
}

func SummaryCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "summary [path]",
		Short: "Generate project structure summary",
		Long:  "Detect project type and generate a JSON summary for LLM context.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			summary, err := generateSummary(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render(err.Error()))
				return
			}

			if jsonOutput {
				output, _ := json.MarshalIndent(summary, "", "  ")
				fmt.Println(string(output))
			} else {
				printSummary(summary)
			}
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func generateSummary(path string) (*ProjectSummary, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	summary := &ProjectSummary{
		Name:       filepath.Base(absPath),
		Type:       "unknown",
		Structure:  make(map[string]string),
		Indicators: []string{},
	}

	// Count files and dirs
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Skip hidden and common ignores
		name := info.Name()
		if name[0] == '.' || name == "node_modules" || name == "vendor" || name == "__pycache__" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			summary.Dirs++
		} else {
			summary.Files++
		}
		return nil
	})

	// Detect project type
	detectProjectType(path, summary)

	return summary, nil
}

func detectProjectType(path string, summary *ProjectSummary) {
	indicators := map[string]struct {
		projectType string
		entry       string
	}{
		"go.mod":         {"go", "main.go"},
		"Cargo.toml":     {"rust", "src/main.rs"},
		"package.json":   {"node", "index.js"},
		"pyproject.toml": {"python", "main.py"},
		"requirements.txt": {"python", "main.py"},
		"Gemfile":        {"ruby", "main.rb"},
		"pom.xml":        {"java", "src/main/java"},
		"build.gradle":   {"java", "src/main/java"},
		"Makefile":       {"make", ""},
		"CMakeLists.txt": {"cmake", ""},
		"docker-compose.yml": {"docker", ""},
		"Dockerfile":     {"docker", ""},
	}

	for file, info := range indicators {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			summary.Indicators = append(summary.Indicators, file)
			if summary.Type == "unknown" {
				summary.Type = info.projectType
				summary.Entry = info.entry
			}
		}
	}

	// Detect structure
	dirs := []string{"cmd", "src", "lib", "pkg", "internal", "test", "tests", "docs", "scripts"}
	for _, dir := range dirs {
		if info, err := os.Stat(filepath.Join(path, dir)); err == nil && info.IsDir() {
			summary.Structure[dir] = describeDir(dir)
		}
	}
}

func describeDir(name string) string {
	descriptions := map[string]string{
		"cmd":      "CLI entry points",
		"src":      "Source code",
		"lib":      "Library code",
		"pkg":      "Public packages",
		"internal": "Private packages",
		"test":     "Test files",
		"tests":    "Test files",
		"docs":     "Documentation",
		"scripts":  "Build/utility scripts",
	}
	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return "Directory"
}

func printSummary(s *ProjectSummary) {
	fmt.Println(tui.RenderHeader("Project Summary"))
	fmt.Println()
	fmt.Printf("  %s: %s\n", tui.HeaderStyle.Render("Name"), s.Name)
	fmt.Printf("  %s: %s\n", tui.HeaderStyle.Render("Type"), s.Type)
	if s.Entry != "" {
		fmt.Printf("  %s: %s\n", tui.HeaderStyle.Render("Entry"), s.Entry)
	}
	fmt.Printf("  %s: %d files, %d directories\n", tui.HeaderStyle.Render("Size"), s.Files, s.Dirs)

	if len(s.Indicators) > 0 {
		fmt.Printf("  %s: %v\n", tui.HeaderStyle.Render("Detected"), s.Indicators)
	}

	if len(s.Structure) > 0 {
		fmt.Printf("\n  %s\n", tui.HeaderStyle.Render("Structure:"))
		for dir, desc := range s.Structure {
			fmt.Printf("    %s/ - %s\n", dir, tui.MutedStyle.Render(desc))
		}
	}
}
