package context

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// GatherOptions controls context gathering behavior
type GatherOptions struct {
	MaxTokens   int      // Maximum tokens to include
	Scope       string   // Limit to specific directory
	Include     []string // Specific files to include
	ExcludeTest bool     // Exclude test files
}

// DefaultOptions returns sensible defaults
func DefaultOptions() GatherOptions {
	return GatherOptions{
		MaxTokens:   4000,
		ExcludeTest: true,
	}
}

// GatherContext extracts relevant codebase context for a query
func GatherContext(query string, opts GatherOptions) string {
	var sections []string

	// 1. Extract keywords from query
	keywords := extractKeywords(query)

	// 2. Find files mentioning keywords
	relevantFiles := findRelevantFiles(keywords, opts)

	// 3. Get symbols from relevant files
	symbols := getSymbols(relevantFiles, keywords)

	// 4. Read file snippets
	snippets := getFileSnippets(relevantFiles, keywords, opts)

	// 5. Build context within token budget
	budget := opts.MaxTokens

	// Add symbols section (high priority)
	if len(symbols) > 0 {
		symbolSection := "## Relevant Types and Functions\n\n" + strings.Join(symbols, "\n")
		if estimateTokens(symbolSection) < budget/3 {
			sections = append(sections, symbolSection)
			budget -= estimateTokens(symbolSection)
		}
	}

	// Add code snippets (medium priority)
	for _, snippet := range snippets {
		if estimateTokens(snippet) < budget {
			sections = append(sections, snippet)
			budget -= estimateTokens(snippet)
		}
		if budget < 500 {
			break
		}
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n---\n\n")
}

// extractKeywords pulls identifiers and important terms from query
func extractKeywords(query string) []string {
	var keywords []string

	// Match CamelCase identifiers
	camelCase := regexp.MustCompile(`[A-Z][a-z]+[A-Z][a-zA-Z]*`)
	keywords = append(keywords, camelCase.FindAllString(query, -1)...)

	// Match snake_case identifiers
	snakeCase := regexp.MustCompile(`[a-z]+_[a-z_]+`)
	keywords = append(keywords, snakeCase.FindAllString(query, -1)...)

	// Match common programming terms
	terms := regexp.MustCompile(`\b(cache|router|config|server|client|handler|model|api|auth|user|data|file|error|response|request)\b`)
	for _, m := range terms.FindAllString(strings.ToLower(query), -1) {
		keywords = append(keywords, m)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, kw := range keywords {
		lower := strings.ToLower(kw)
		if !seen[lower] && len(kw) > 2 {
			seen[lower] = true
			unique = append(unique, kw)
		}
	}

	return unique
}

// findRelevantFiles searches for files containing keywords
func findRelevantFiles(keywords []string, opts GatherOptions) []string {
	var allFiles []string
	fileScores := make(map[string]int)

	searchDir := "."
	if opts.Scope != "" {
		searchDir = opts.Scope
	}

	// Include explicitly specified files
	for _, f := range opts.Include {
		matches, _ := filepath.Glob(f)
		allFiles = append(allFiles, matches...)
	}

	// Search for each keyword
	for _, kw := range keywords {
		out, err := exec.Command("grep", "-rl", "--include=*.go", "--include=*.py", "--include=*.js", "--include=*.ts", kw, searchDir).Output()
		if err != nil {
			continue
		}
		files := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, f := range files {
			if f == "" {
				continue
			}
			if opts.ExcludeTest && strings.Contains(f, "_test.go") {
				continue
			}
			fileScores[f]++
		}
	}

	// Sort by score (most keyword matches first)
	for f := range fileScores {
		allFiles = append(allFiles, f)
	}
	sort.Slice(allFiles, func(i, j int) bool {
		return fileScores[allFiles[i]] > fileScores[allFiles[j]]
	})

	// Limit to top files
	if len(allFiles) > 10 {
		allFiles = allFiles[:10]
	}

	return allFiles
}

// getSymbols extracts type/function definitions from files
func getSymbols(files []string, keywords []string) []string {
	var symbols []string

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// Match function definitions
			if strings.HasPrefix(strings.TrimSpace(line), "func ") {
				for _, kw := range keywords {
					if strings.Contains(line, kw) {
						symbols = append(symbols, formatSymbol(file, i+1, line))
						break
					}
				}
			}
			// Match type definitions
			if strings.HasPrefix(strings.TrimSpace(line), "type ") {
				for _, kw := range keywords {
					if strings.Contains(line, kw) {
						// Include type and next few lines
						end := min(i+5, len(lines))
						symbols = append(symbols, formatSymbol(file, i+1, strings.Join(lines[i:end], "\n")))
						break
					}
				}
			}
		}
	}

	return symbols
}

// getFileSnippets reads relevant portions of files
func getFileSnippets(files []string, keywords []string, opts GatherOptions) []string {
	var snippets []string

	for _, file := range files[:min(3, len(files))] {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")

		// Find lines with keywords and include context
		for i, line := range lines {
			for _, kw := range keywords {
				if strings.Contains(line, kw) {
					start := max(0, i-2)
					end := min(len(lines), i+5)
					snippet := fmt.Sprintf("## %s (lines %d-%d)\n```\n%s\n```",
						file, start+1, end,
						strings.Join(lines[start:end], "\n"))
					snippets = append(snippets, snippet)
					break
				}
			}
		}
	}

	return snippets
}

func formatSymbol(file string, line int, content string) string {
	return fmt.Sprintf("// %s:%d\n%s", file, line, strings.TrimSpace(content))
}

func estimateTokens(s string) int {
	// Rough estimate: ~4 chars per token
	return len(s) / 4
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
