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

type Symbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Exported bool   `json:"exported"`
}

func SymbolsCmd() *cobra.Command {
	var jsonOutput bool
	var exportedOnly bool
	var kindFilter string

	cmd := &cobra.Command{
		Use:   "symbols [path]",
		Short: "Extract symbols (functions, types, etc.)",
		Long:  "Parse source files and extract function/type/class definitions.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			symbols := extractSymbols(path, exportedOnly, kindFilter)

			if jsonOutput {
				output, _ := json.MarshalIndent(symbols, "", "  ")
				fmt.Println(string(output))
			} else {
				printSymbols(symbols)
			}
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&exportedOnly, "exported", "e", false, "Only show exported symbols")
	cmd.Flags().StringVarP(&kindFilter, "type", "t", "", "Filter by kind (func, type, const, var)")

	return cmd
}

func extractSymbols(path string, exportedOnly bool, kindFilter string) []Symbol {
	var symbols []Symbol

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip hidden and vendor
		if strings.Contains(p, "/.") || strings.Contains(p, "/vendor/") || strings.Contains(p, "/node_modules/") {
			return nil
		}

		ext := filepath.Ext(p)
		switch ext {
		case ".go":
			symbols = append(symbols, parseGoFile(p, exportedOnly, kindFilter)...)
		case ".py":
			symbols = append(symbols, parsePythonFile(p, kindFilter)...)
		case ".js", ".ts", ".jsx", ".tsx":
			symbols = append(symbols, parseJSFile(p, exportedOnly, kindFilter)...)
		}

		return nil
	})

	return symbols
}

func parseGoFile(path string, exportedOnly bool, kindFilter string) []Symbol {
	var symbols []Symbol

	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	// Regex patterns for Go
	funcPattern := regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)`)
	typePattern := regexp.MustCompile(`^type\s+(\w+)`)
	constPattern := regexp.MustCompile(`^const\s+(\w+)`)
	varPattern := regexp.MustCompile(`^var\s+(\w+)`)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		var name, kind string

		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "func"
		} else if matches := typePattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "type"
		} else if matches := constPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "const"
		} else if matches := varPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "var"
		}

		if name != "" {
			exported := name[0] >= 'A' && name[0] <= 'Z'

			if exportedOnly && !exported {
				continue
			}
			if kindFilter != "" && kind != kindFilter {
				continue
			}

			symbols = append(symbols, Symbol{
				Name:     name,
				Kind:     kind,
				File:     path,
				Line:     lineNum,
				Exported: exported,
			})
		}
	}

	return symbols
}

func parsePythonFile(path string, kindFilter string) []Symbol {
	var symbols []Symbol

	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	funcPattern := regexp.MustCompile(`^def\s+(\w+)`)
	classPattern := regexp.MustCompile(`^class\s+(\w+)`)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var name, kind string

		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "func"
		} else if matches := classPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "class"
		}

		if name != "" {
			if kindFilter != "" && kind != kindFilter {
				continue
			}

			exported := !strings.HasPrefix(name, "_")

			symbols = append(symbols, Symbol{
				Name:     name,
				Kind:     kind,
				File:     path,
				Line:     lineNum,
				Exported: exported,
			})
		}
	}

	return symbols
}

func parseJSFile(path string, exportedOnly bool, kindFilter string) []Symbol {
	var symbols []Symbol

	file, err := os.Open(path)
	if err != nil {
		return symbols
	}
	defer file.Close()

	// Simplified patterns - a real implementation would use a proper parser
	funcPattern := regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+(\w+)`)
	classPattern := regexp.MustCompile(`(?:export\s+)?class\s+(\w+)`)
	constPattern := regexp.MustCompile(`(?:export\s+)?const\s+(\w+)`)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var name, kind string
		exported := strings.Contains(line, "export")

		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "func"
		} else if matches := classPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "class"
		} else if matches := constPattern.FindStringSubmatch(line); matches != nil {
			name, kind = matches[1], "const"
		}

		if name != "" {
			if exportedOnly && !exported {
				continue
			}
			if kindFilter != "" && kind != kindFilter {
				continue
			}

			symbols = append(symbols, Symbol{
				Name:     name,
				Kind:     kind,
				File:     path,
				Line:     lineNum,
				Exported: exported,
			})
		}
	}

	return symbols
}

func printSymbols(symbols []Symbol) {
	if len(symbols) == 0 {
		fmt.Println(tui.MutedStyle.Render("No symbols found"))
		return
	}

	fmt.Println(tui.RenderHeader("Symbols"))
	fmt.Println()

	currentFile := ""
	for _, s := range symbols {
		if s.File != currentFile {
			currentFile = s.File
			fmt.Printf("\n%s\n", tui.HeaderStyle.Render(currentFile))
		}

		exportMark := " "
		if s.Exported {
			exportMark = tui.SuccessStyle.Render("*")
		}

		fmt.Printf("  %s %s %s %s\n",
			exportMark,
			tui.MutedStyle.Render(fmt.Sprintf("L%-4d", s.Line)),
			kindStyle(s.Kind),
			s.Name,
		)
	}
}

func kindStyle(kind string) string {
	switch kind {
	case "func":
		return tui.TierFastStyle.Render("[func]")
	case "type", "class":
		return tui.TierDeepStyle.Render("[" + kind + "]")
	default:
		return tui.MutedStyle.Render("[" + kind + "]")
	}
}
