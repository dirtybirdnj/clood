package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Cat represents a model in the catfight
type Cat struct {
	Name  string
	Model string
}

// DefaultCats are the proven performers from Kitchen Stadium
var DefaultCats = []Cat{
	{Name: "Persian", Model: "deepseek-coder:6.7b"},
	{Name: "Tabby", Model: "mistral:7b"},
	{Name: "Siamese", Model: "qwen2.5-coder:3b"},
}

// CatfightResult holds the output from one cat
type CatfightResult struct {
	Cat      Cat
	Response string
	Duration time.Duration
	Tokens   int
	TokSec   float64
	Error    error
}

func CatfightCmd() *cobra.Command {
	var promptFile string
	var models string
	var outputDir string
	var host string
	var quiet bool

	cmd := &cobra.Command{
		Use:   "catfight [prompt]",
		Short: "Run multiple models against the same prompt (Kitchen Stadium style)",
		Long: `Release the cats! Run multiple LLMs against the same prompt and compare outputs.

Default cats (proven performers):
  Persian  - deepseek-coder:6.7b (reigning champion)
  Tabby    - mistral:7b
  Siamese  - qwen2.5-coder:3b

Examples:
  clood catfight "Write a hello world in Go"
  clood catfight -f prompt.txt
  clood catfight --models "llama3.1:8b,mistral:7b" "Explain recursion"
  clood catfight -o /tmp/battle3 -f battle3_prompt.txt`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get the prompt
			var prompt string
			if promptFile != "" {
				data, err := os.ReadFile(promptFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error reading prompt file: "+err.Error()))
					return
				}
				prompt = string(data)
			} else if len(args) > 0 {
				prompt = strings.Join(args, " ")
			} else {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("No prompt provided. Use -f <file> or pass prompt as argument."))
				return
			}

			// Determine which cats to use
			cats := DefaultCats
			if models != "" {
				cats = []Cat{}
				for _, m := range strings.Split(models, ",") {
					m = strings.TrimSpace(m)
					cats = append(cats, Cat{Name: modelToName(m), Model: m})
				}
			}

			// Set up output directory
			if outputDir != "" {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error creating output dir: "+err.Error()))
					return
				}
			}

			// Create client
			baseURL := "http://localhost:11434"
			if host != "" {
				baseURL = host
			}
			client := ollama.NewClient(baseURL, 5*time.Minute)

			// ALLEZ CUISINE!
			fmt.Println(tui.RenderHeader("KITCHEN STADIUM - CATFIGHT"))
			fmt.Println()
			fmt.Printf("%s %d cats entering the arena\n", tui.MutedStyle.Render("Contenders:"), len(cats))
			for _, cat := range cats {
				fmt.Printf("  %s %s\n", tui.AccentStyle.Render(cat.Name+":"), cat.Model)
			}
			fmt.Println()

			if !quiet {
				fmt.Println(tui.MutedStyle.Render("Prompt:"))
				displayPrompt := prompt
				if len(displayPrompt) > 200 {
					displayPrompt = displayPrompt[:200] + "..."
				}
				fmt.Println(displayPrompt)
				fmt.Println()
			}

			// Run each cat
			results := []CatfightResult{}
			for i, cat := range cats {
				fmt.Printf("%s [%d/%d] %s (%s)\n",
					tui.AccentStyle.Render(">>>"),
					i+1, len(cats),
					cat.Name,
					cat.Model)

				start := time.Now()
				resp, err := client.Generate(cat.Model, prompt)
				duration := time.Since(start)

				result := CatfightResult{
					Cat:      cat,
					Duration: duration,
				}

				if err != nil {
					result.Error = err
					fmt.Printf("    %s %v\n", tui.ErrorStyle.Render("ERROR:"), err)
				} else {
					result.Response = resp.Response
					result.Tokens = resp.EvalCount
					if resp.EvalDuration > 0 {
						result.TokSec = float64(resp.EvalCount) / (float64(resp.EvalDuration) / 1e9)
					}
					fmt.Printf("    %s %.1fs | %d tokens | %.1f tok/s\n",
						tui.SuccessStyle.Render("DONE"),
						duration.Seconds(),
						result.Tokens,
						result.TokSec)

					// Save output if outputDir specified
					if outputDir != "" {
						filename := filepath.Join(outputDir, sanitizeFilename(cat.Model)+".txt")
						if err := os.WriteFile(filename, []byte(resp.Response), 0644); err != nil {
							fmt.Printf("    %s %v\n", tui.WarningStyle.Render("Save error:"), err)
						} else {
							fmt.Printf("    %s %s\n", tui.MutedStyle.Render("Saved:"), filename)
						}
					}
				}
				results = append(results, result)
				fmt.Println()
			}

			// Summary
			fmt.Println(tui.RenderHeader("RESULTS"))
			fmt.Println()
			fmt.Printf("%-12s %-25s %10s %8s %10s\n", "CAT", "MODEL", "TIME", "TOKENS", "TOK/S")
			fmt.Println(strings.Repeat("-", 70))

			var fastest *CatfightResult
			for _, r := range results {
				status := ""
				if r.Error != nil {
					status = tui.ErrorStyle.Render("FAILED")
				} else {
					status = fmt.Sprintf("%8.1fs %8d %10.1f", r.Duration.Seconds(), r.Tokens, r.TokSec)
					if fastest == nil || (r.Error == nil && r.Duration < fastest.Duration) {
						fastest = &r
					}
				}
				fmt.Printf("%-12s %-25s %s\n", r.Cat.Name, r.Cat.Model, status)
			}

			if fastest != nil {
				fmt.Println()
				fmt.Printf("%s %s wins with %.1fs!\n",
					tui.AccentStyle.Render("WINNER:"),
					fastest.Cat.Name,
					fastest.Duration.Seconds())
			}

			// Show all responses for easy comparison / LLM consumption
			fmt.Println()
			fmt.Println(tui.RenderHeader("RESPONSES"))
			fmt.Println()
			for _, r := range results {
				fmt.Printf("%s %s (%s)\n",
					tui.AccentStyle.Render("###"),
					r.Cat.Name,
					r.Cat.Model)
				fmt.Println(strings.Repeat("-", 60))
				if r.Error != nil {
					fmt.Printf("%s\n", tui.ErrorStyle.Render("ERROR: "+r.Error.Error()))
				} else {
					fmt.Println(r.Response)
				}
				fmt.Println()
			}
		},
	}

	cmd.Flags().StringVarP(&promptFile, "file", "f", "", "Read prompt from file")
	cmd.Flags().StringVarP(&models, "models", "m", "", "Comma-separated list of models (default: persian,tabby,siamese)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Directory to save outputs")
	cmd.Flags().StringVarP(&host, "host", "H", "", "Ollama host URL (default: http://localhost:11434)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Don't show prompt preview")

	return cmd
}

// modelToName creates a friendly name from a model string
func modelToName(model string) string {
	// Check known cats
	knownCats := map[string]string{
		"deepseek-coder:6.7b": "Persian",
		"mistral:7b":          "Tabby",
		"qwen2.5-coder:3b":    "Siamese",
		"llama3.1:8b":         "Caracal",
		"deepseek-r1:14b":     "HouseLion",
		"starcoder2:7b":       "Bengal",
		"tinyllama":           "Kitten",
	}
	if name, ok := knownCats[model]; ok {
		return name
	}
	// Use first part of model name
	parts := strings.Split(model, ":")
	return strings.Title(parts[0])
}

// sanitizeFilename makes a model name safe for filenames
func sanitizeFilename(model string) string {
	return strings.ReplaceAll(strings.ReplaceAll(model, ":", "_"), "/", "_")
}
