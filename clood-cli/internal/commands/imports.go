package commands

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// ImportInfo represents imports for a single file
type ImportInfo struct {
	File    string   `json:"file"`
	Package string   `json:"package"`
	Imports []string `json:"imports"`
}

// DependencyInfo represents reverse dependency information
type DependencyInfo struct {
	Package   string   `json:"package"`
	ImportedBy []string `json:"imported_by"`
}

func ImportsCmd() *cobra.Command {
	var jsonOutput bool
	var reverseMode bool
	var showAll bool

	cmd := &cobra.Command{
		Use:   "imports [FILE|DIR]",
		Short: "Analyze Go imports and dependencies",
		Long:  "Parse Go files to show imports or find reverse dependencies (who imports a package).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}

			info, err := os.Stat(target)
			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+err.Error()))
				return nil
			}

			if reverseMode {
				// Reverse mode: find files that import the target package
				return findReverseImports(target, jsonOutput)
			}

			var results []ImportInfo

			if info.IsDir() {
				// Process all Go files in directory
				results, err = processDirectory(target, showAll)
			} else {
				// Process single file
				result, parseErr := parseGoImports(target)
				if parseErr != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error parsing file: "+parseErr.Error()))
					return nil
				}
				results = []ImportInfo{*result}
			}

			if err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error: "+err.Error()))
				return nil
			}

			// Output results
			if jsonOutput {
				output, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(output))
			} else {
				printImports(results)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&reverseMode, "reverse", "r", false, "Find files that import the given package path")
	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all files (including those with no imports)")

	return cmd
}

func processDirectory(dir string, showAll bool) ([]ImportInfo, error) {
	var results []ImportInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip unwanted directories
		if info.IsDir() {
			name := info.Name()
			if shouldSkipImportDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files by default
		if strings.HasSuffix(path, "_test.go") && !showAll {
			return nil
		}

		result, err := parseGoImports(path)
		if err != nil {
			return nil // Skip files that fail to parse
		}

		if showAll || len(result.Imports) > 0 {
			results = append(results, *result)
		}

		return nil
	})

	return results, err
}

func parseGoImports(path string) (*ImportInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	result := &ImportInfo{
		File:    path,
		Package: node.Name.Name,
		Imports: make([]string, 0, len(node.Imports)),
	}

	for _, imp := range node.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		result.Imports = append(result.Imports, importPath)
	}

	// Sort imports for consistent output
	sort.Strings(result.Imports)

	return result, nil
}

func findReverseImports(pkgPath string, jsonOutput bool) error {
	// Search from current directory for files that import the given package
	var importedBy []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if shouldSkipImportDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		result, err := parseGoImports(path)
		if err != nil {
			return nil
		}

		// Check if this file imports the target package
		for _, imp := range result.Imports {
			if imp == pkgPath || strings.HasSuffix(imp, "/"+pkgPath) || strings.Contains(imp, pkgPath) {
				importedBy = append(importedBy, path)
				break
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if jsonOutput {
		output, _ := json.MarshalIndent(DependencyInfo{
			Package:   pkgPath,
			ImportedBy: importedBy,
		}, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Println(tui.RenderHeader("Reverse Dependencies"))
		fmt.Println()
		fmt.Printf("  Package: %s\n", tui.HeaderStyle.Render(pkgPath))
		fmt.Printf("  Imported by: %d files\n\n", len(importedBy))

		if len(importedBy) == 0 {
			fmt.Println(tui.MutedStyle.Render("  No files import this package"))
		} else {
			for _, f := range importedBy {
				fmt.Printf("    %s\n", f)
			}
		}
	}

	return nil
}

func printImports(results []ImportInfo) {
	if len(results) == 0 {
		fmt.Println(tui.MutedStyle.Render("No Go files found"))
		return
	}

	fmt.Println(tui.RenderHeader("Imports Analysis"))
	fmt.Println()

	// Group by package for cleaner output
	packages := make(map[string][]ImportInfo)
	for _, r := range results {
		packages[r.Package] = append(packages[r.Package], r)
	}

	// Get sorted package names
	var pkgNames []string
	for pkg := range packages {
		pkgNames = append(pkgNames, pkg)
	}
	sort.Strings(pkgNames)

	for _, pkg := range pkgNames {
		files := packages[pkg]
		fmt.Printf("  %s %s\n", tui.HeaderStyle.Render("package"), pkg)

		for _, f := range files {
			relPath := f.File
			fmt.Printf("    %s\n", tui.MutedStyle.Render(relPath))

			// Categorize imports
			var stdLib, internal, external []string
			for _, imp := range f.Imports {
				if isStdLib(imp) {
					stdLib = append(stdLib, imp)
				} else if strings.Contains(imp, "github.com/dirtybirdnj/clood") {
					internal = append(internal, imp)
				} else {
					external = append(external, imp)
				}
			}

			if len(internal) > 0 {
				fmt.Printf("      %s\n", tui.SuccessStyle.Render("internal:"))
				for _, imp := range internal {
					// Shorten internal imports for readability
					short := strings.TrimPrefix(imp, "github.com/dirtybirdnj/clood/")
					fmt.Printf("        %s\n", short)
				}
			}

			if len(external) > 0 {
				fmt.Printf("      %s\n", tui.TierDeepStyle.Render("external:"))
				for _, imp := range external {
					fmt.Printf("        %s\n", imp)
				}
			}

			if len(stdLib) > 0 {
				fmt.Printf("      %s %s\n", tui.MutedStyle.Render("stdlib:"), tui.MutedStyle.Render(fmt.Sprintf("(%d packages)", len(stdLib))))
			}
		}
		fmt.Println()
	}
}

func shouldSkipImportDir(name string) bool {
	skipDirs := map[string]bool{
		".git":         true,
		".svn":         true,
		"vendor":       true,
		"node_modules": true,
		"testdata":     true,
	}
	return skipDirs[name] || strings.HasPrefix(name, ".")
}

func isStdLib(pkg string) bool {
	// Standard library packages don't contain dots (except for some like "golang.org/x/...")
	return !strings.Contains(pkg, ".")
}
