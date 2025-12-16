package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// GrepMatch represents a single match result
type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// GrepResult contains all matches for a file
type GrepResult struct {
	File    string `json:"file"`
	Matches int    `json:"matches"`
	Lines   []int  `json:"lines,omitempty"`
}

func GrepCmd() *cobra.Command {
	var jsonOutput bool
	var caseInsensitive bool
	var showLineNumbers bool
	var contextBefore int
	var contextAfter int
	var contextBoth int
	var fileTypes []string
	var filesOnly bool
	var countOnly bool

	cmd := &cobra.Command{
		Use:   "grep PATTERN [PATH]",
		Short: "Search codebase content using regex",
		Long:  "Search files recursively for a regex pattern, similar to grep but optimized for codebases.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]
			searchPath := "."
			if len(args) > 1 {
				searchPath = args[1]
			}

			// Handle context flags
			if contextBoth > 0 {
				contextBefore = contextBoth
				contextAfter = contextBoth
			}

			// Compile regex
			flags := ""
			if caseInsensitive {
				flags = "(?i)"
			}
			re, err := regexp.Compile(flags + pattern)
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Invalid regex: "+err.Error()))
				return nil
			}

			// Track results
			var allMatches []GrepMatch
			var fileResults []GrepResult
			totalMatches := 0

			// Walk the directory
			err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip files we can't access
				}

				// Skip directories we don't want
				if info.IsDir() {
					name := info.Name()
					if shouldSkipDir(name) {
						return filepath.SkipDir
					}
					return nil
				}

				// Skip hidden files
				if strings.HasPrefix(info.Name(), ".") {
					return nil
				}

				// Filter by file type if specified
				if len(fileTypes) > 0 && !matchesFileType(path, fileTypes) {
					return nil
				}

				// Search the file
				matches, err := searchFile(path, re, showLineNumbers, contextBefore, contextAfter)
				if err != nil {
					return nil // Skip files we can't read
				}

				if len(matches) > 0 {
					totalMatches += len(matches)

					if filesOnly {
						// Just track file names
						lines := make([]int, len(matches))
						for i, m := range matches {
							lines[i] = m.Line
						}
						fileResults = append(fileResults, GrepResult{
							File:    path,
							Matches: len(matches),
							Lines:   lines,
						})
					} else if countOnly {
						fileResults = append(fileResults, GrepResult{
							File:    path,
							Matches: len(matches),
						})
					} else {
						allMatches = append(allMatches, matches...)
					}
				}

				return nil
			})

			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error walking directory: "+err.Error()))
				return nil
			}

			// Output results
			if jsonOutput {
				if filesOnly || countOnly {
					output, _ := json.MarshalIndent(fileResults, "", "  ")
					fmt.Println(string(output))
				} else {
					output, _ := json.MarshalIndent(allMatches, "", "  ")
					fmt.Println(string(output))
				}
			} else {
				if filesOnly {
					for _, r := range fileResults {
						fmt.Printf("%s %s\n",
							tui.HeaderStyle.Render(r.File),
							tui.MutedStyle.Render(fmt.Sprintf("(%d matches)", r.Matches)))
					}
				} else if countOnly {
					for _, r := range fileResults {
						fmt.Printf("%s: %d\n", r.File, r.Matches)
					}
					fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("\nTotal: %d matches in %d files", totalMatches, len(fileResults))))
				} else {
					printMatches(allMatches, re)
				}
			}

			if !jsonOutput && totalMatches == 0 {
				fmt.Println(tui.MutedStyle.Render("No matches found"))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&caseInsensitive, "ignore-case", "i", false, "Case insensitive search")
	cmd.Flags().BoolVarP(&showLineNumbers, "line-number", "n", true, "Show line numbers")
	cmd.Flags().IntVarP(&contextBefore, "before-context", "B", 0, "Lines before match")
	cmd.Flags().IntVarP(&contextAfter, "after-context", "A", 0, "Lines after match")
	cmd.Flags().IntVarP(&contextBoth, "context", "C", 0, "Lines before and after match")
	cmd.Flags().StringSliceVarP(&fileTypes, "type", "t", nil, "Filter by file type (go, py, js, ts, rs)")
	cmd.Flags().BoolVar(&filesOnly, "files-only", false, "Only list files with matches")
	cmd.Flags().BoolVar(&countOnly, "count", false, "Only show match counts")

	return cmd
}

func shouldSkipDir(name string) bool {
	skipDirs := map[string]bool{
		".git":         true,
		".svn":         true,
		".hg":          true,
		"node_modules": true,
		"vendor":       true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
		"dist":         true,
		"build":        true,
		".idea":        true,
		".vscode":      true,
	}
	return skipDirs[name] || strings.HasPrefix(name, ".")
}

func matchesFileType(path string, types []string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")

	// Map common extensions
	extMap := map[string]string{
		"go":   "go",
		"py":   "py",
		"js":   "js",
		"jsx":  "js",
		"ts":   "ts",
		"tsx":  "ts",
		"rs":   "rs",
		"rb":   "rb",
		"java": "java",
		"c":    "c",
		"cpp":  "cpp",
		"h":    "c",
		"hpp":  "cpp",
		"md":   "md",
		"yaml": "yaml",
		"yml":  "yaml",
		"json": "json",
		"toml": "toml",
	}

	mappedExt := extMap[ext]
	if mappedExt == "" {
		mappedExt = ext
	}

	for _, t := range types {
		if mappedExt == t || ext == t {
			return true
		}
	}
	return false
}

func searchFile(path string, re *regexp.Regexp, showLineNumbers bool, contextBefore, contextAfter int) ([]GrepMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []GrepMatch
	var lines []string
	var matchingLines []int

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// First pass: collect all lines and find matches
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		if re.MatchString(line) {
			matchingLines = append(matchingLines, lineNum-1) // 0-indexed
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Second pass: build matches with context
	printed := make(map[int]bool)

	for _, matchIdx := range matchingLines {
		startLine := matchIdx - contextBefore
		if startLine < 0 {
			startLine = 0
		}
		endLine := matchIdx + contextAfter
		if endLine >= len(lines) {
			endLine = len(lines) - 1
		}

		for i := startLine; i <= endLine; i++ {
			if printed[i] {
				continue
			}
			printed[i] = true

			matches = append(matches, GrepMatch{
				File:    path,
				Line:    i + 1, // 1-indexed for display
				Content: lines[i],
			})
		}
	}

	return matches, nil
}

func printMatches(matches []GrepMatch, re *regexp.Regexp) {
	currentFile := ""

	for _, m := range matches {
		if m.File != currentFile {
			if currentFile != "" {
				fmt.Println() // Blank line between files
			}
			currentFile = m.File
			fmt.Println(tui.HeaderStyle.Render(m.File))
		}

		lineNum := tui.MutedStyle.Render(fmt.Sprintf("%4d", m.Line))

		// Highlight matches in the content
		highlighted := highlightMatches(m.Content, re)

		fmt.Printf("  %s │ %s\n", lineNum, highlighted)
	}
}

func highlightMatches(line string, re *regexp.Regexp) string {
	// Find all matches and highlight them
	matches := re.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}

	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		// Add text before match
		result.WriteString(line[lastEnd:match[0]])
		// Add highlighted match
		result.WriteString(tui.SuccessStyle.Render(line[match[0]:match[1]]))
		lastEnd = match[1]
	}
	// Add remaining text
	result.WriteString(line[lastEnd:])

	return result.String()
}
