package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

const testGenSystemPrompt = `You are a test generator for software projects. Generate comprehensive unit tests for the provided source code.

Rules:
1. Use the appropriate testing framework for the language (Go: testing package, Python: pytest, JS: jest)
2. Include table-driven tests where appropriate
3. Cover happy path, edge cases, and error cases
4. Use descriptive test names
5. Include setup and teardown if needed
6. Add comments explaining what each test verifies

Output ONLY the test code, ready to be saved to a file. No explanations outside the code.`

const testGenGoPrompt = `Generate Go unit tests using the standard testing package.
Use table-driven tests with this structure:
tests := []struct {
    name     string
    input    ...
    expected ...
}{...}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {...})
}`

const testGenPythonPrompt = `Generate Python tests using pytest.
Use parametrize where appropriate:
@pytest.mark.parametrize("input,expected", [...])
def test_function(input, expected):
    assert function(input) == expected`

// TestGenResult holds the generation output
type TestGenResult struct {
	SourceFile string  `json:"source_file"`
	TestFile   string  `json:"test_file,omitempty"`
	Model      string  `json:"model"`
	Host       string  `json:"host"`
	Duration   float64 `json:"duration_seconds"`
	Tests      string  `json:"tests"`
	Tokens     int     `json:"tokens"`
}

func GenerateTestsCmd() *cobra.Command {
	var outputFile string
	var function string
	var style string
	var model string

	cmd := &cobra.Command{
		Use:   "generate-tests <source-file>",
		Short: "Generate unit tests using local LLMs",
		Long: `Generate unit tests from source code using local models.

Saves Claude tokens by using local models for test scaffolding.

Examples:
  clood generate-tests src/auth/login.go
  clood generate-tests src/router.go --function HandleRequest
  clood generate-tests src/api.go -o src/api_test.go
  clood generate-tests src/utils.py --style pytest`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceFile := args[0]
			result, err := generateTests(sourceFile, function, style, model, outputFile)
			if err != nil {
				if output.IsJSON() {
					fmt.Printf(`{"error": %q}`, err.Error())
				} else {
					fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
				}
				return
			}

			if output.IsJSON() {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			printTestGenResult(result)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for tests")
	cmd.Flags().StringVar(&function, "function", "", "Generate tests for specific function only")
	cmd.Flags().StringVar(&style, "style", "", "Test style (table-driven, pytest, jest)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model to use (default: tier1 fast)")

	return cmd
}

func generateTests(sourceFile, function, style, modelOverride, outputFile string) (*TestGenResult, error) {
	// Read source file
	sourceCode, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read source file: %w", err)
	}

	// Detect language
	ext := strings.ToLower(filepath.Ext(sourceFile))
	lang := detectLanguage(ext)

	// Build prompt
	systemPrompt := testGenSystemPrompt
	switch lang {
	case "go":
		systemPrompt += "\n\n" + testGenGoPrompt
	case "python":
		systemPrompt += "\n\n" + testGenPythonPrompt
	}

	if style != "" {
		systemPrompt += fmt.Sprintf("\n\nUse %s test style.", style)
	}

	userPrompt := fmt.Sprintf("Generate tests for this %s file:\n\n```%s\n%s\n```", lang, lang, string(sourceCode))

	if function != "" {
		userPrompt += fmt.Sprintf("\n\nFocus on testing the function: %s", function)
	}

	// Truncate if too long
	if len(userPrompt) > 12000 {
		userPrompt = userPrompt[:12000] + "\n```\n(truncated)"
	}

	// Load config and find host
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	mgr := hosts.NewManager()
	mgr.AddHosts(cfg.Hosts)

	var targetHost *hosts.HostStatus
	statuses := mgr.CheckAllHosts()
	for _, s := range statuses {
		if s.Online && len(s.Models) > 0 {
			targetHost = s
			break
		}
	}

	if targetHost == nil {
		return nil, fmt.Errorf("no Ollama hosts available")
	}

	// Select model
	modelName := modelOverride
	if modelName == "" {
		modelName = cfg.Tiers.Fast.Model
		// Prefer coding model if available
		allModels := mgr.GetAllModels()
		for name := range allModels {
			if strings.Contains(name, "coder") || strings.Contains(name, "code") {
				modelName = name
				break
			}
		}
	}

	// Generate
	fmt.Printf("%s Generating tests with %s...\n", tui.MutedStyle.Render("●"), modelName)

	client := ollama.NewClient(targetHost.Host.URL, 120*time.Second)
	start := time.Now()

	resp, err := client.GenerateWithSystem(modelName, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	duration := time.Since(start)
	tests := strings.TrimSpace(resp.Response)

	// Clean up markdown code blocks if present
	tests = cleanCodeBlocks(tests, lang)

	result := &TestGenResult{
		SourceFile: sourceFile,
		Model:      modelName,
		Host:       targetHost.Host.Name,
		Duration:   duration.Seconds(),
		Tests:      tests,
		Tokens:     resp.EvalCount,
	}

	// Write to file if requested
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(tests), 0644); err != nil {
			return nil, fmt.Errorf("cannot write output file: %w", err)
		}
		result.TestFile = outputFile
	}

	return result, nil
}

func detectLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".ts", ".jsx", ".tsx":
		return "javascript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	default:
		return "code"
	}
}

func cleanCodeBlocks(code, lang string) string {
	// Remove markdown code block wrappers
	code = strings.TrimPrefix(code, "```"+lang+"\n")
	code = strings.TrimPrefix(code, "```\n")
	code = strings.TrimSuffix(code, "\n```")
	code = strings.TrimSuffix(code, "```")
	return strings.TrimSpace(code)
}

func printTestGenResult(result *TestGenResult) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("Generated Tests"))
	fmt.Println()

	fmt.Printf("  Source: %s\n", result.SourceFile)
	fmt.Printf("  Model:  %s (%s)\n", result.Model, result.Host)
	fmt.Printf("  Time:   %.1fs (%d tokens)\n", result.Duration, result.Tokens)

	if result.TestFile != "" {
		fmt.Printf("  Output: %s\n", result.TestFile)
		fmt.Println()
		fmt.Println(tui.SuccessStyle.Render("✓ Tests written to file"))
	} else {
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("─────────────────────────────────────"))
		fmt.Println()
		fmt.Println(result.Tests)
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("─────────────────────────────────────"))
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("Use -o <file> to save to a file"))
	}
}
