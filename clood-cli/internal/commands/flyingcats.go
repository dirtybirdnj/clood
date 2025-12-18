package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dirtybirdnj/clood/internal/flyingcats"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// FlyingCatsOutput is the JSON output structure
type FlyingCatsOutput struct {
	Query  string `json:"query,omitempty"`
	Mode   string `json:"mode"`
	Output string `json:"output"`
	Seed   int64  `json:"seed,omitempty"`
}

func FlyingCatsCmd() *cobra.Command {
	var chaos bool
	var antenna bool
	var seed int64
	var jsonOutput bool
	var count int

	cmd := &cobra.Command{
		Use:   "flying-cats [query]",
		Short: "Generate structured nonsense for prompt scaffolding",
		Long: `Summon the Flying Cats of Delaware!

Named for the legendary felines who nest in radio towers and antenna structures
along the highway. Their stories were first told on long car trips, featuring
the protagonist Pooparoo (a Toonces-like cat of questionable driving ability).

Flying cats sit atop the catfight architecture, generating structured randomness
for prompts without requiring LLM inference. This is the special sauce that helps
the system reason about what it's doing.

The generator parses your query for structural elements:
  - actors/people/characters (with counts: "three actors")
  - meals, food, restaurants
  - streets, addresses, cities
  - antenna towers, radio structures (Delaware heritage)
  - options, choices, alternatives
  - times, reasons, names

Examples:
  clood flying-cats "give me options for three actors at a restaurant"
  clood flying-cats "what street does barry live on?"
  clood flying-cats "a meal at a restaurant with two characters"
  clood flying-cats --antenna              # Delaware tower transmission mode
  clood flying-cats --chaos                # Pure stream of consciousness
  clood flying-cats --seed 42 "options"    # Reproducible randomness
  clood flying-cats -n 3 "what time?"      # Generate 3 variations`,
		Run: func(cmd *cobra.Command, args []string) {
			var gen *flyingcats.Generator
			if seed != 0 {
				gen = flyingcats.NewSeededGenerator(seed)
			} else {
				gen = flyingcats.NewGenerator()
			}

			var output string
			var query string
			var mode string

			if antenna {
				mode = "antenna"
				if count > 1 {
					var outputs []string
					for i := 0; i < count; i++ {
						outputs = append(outputs, fmt.Sprintf("--- Transmission %d ---\n%s", i+1, gen.Antenna()))
					}
					output = strings.Join(outputs, "\n\n")
				} else {
					output = gen.Antenna()
				}
			} else if chaos {
				mode = "chaos"
				if count > 1 {
					var outputs []string
					for i := 0; i < count; i++ {
						outputs = append(outputs, fmt.Sprintf("--- Chaos Stream %d ---\n%s", i+1, gen.Chaos()))
					}
					output = strings.Join(outputs, "\n\n")
				} else {
					output = gen.Chaos()
				}
			} else if len(args) > 0 {
				mode = "query"
				query = strings.Join(args, " ")
				if count > 1 {
					var outputs []string
					for i := 0; i < count; i++ {
						outputs = append(outputs, fmt.Sprintf("--- Variation %d ---\n%s", i+1, gen.Generate(query)))
					}
					output = strings.Join(outputs, "\n\n")
				} else {
					output = gen.Generate(query)
				}
			} else {
				// No args and no mode - show a helpful cat
				mode = "greeting"
				output = gen.Antenna()
			}

			if jsonOutput {
				result := FlyingCatsOutput{
					Query:  query,
					Mode:   mode,
					Output: output,
				}
				if seed != 0 {
					result.Seed = seed
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return
			}

			// Pretty terminal output
			if !antenna {
				fmt.Println(tui.RenderHeader("FLYING CATS OF DELAWARE"))
				fmt.Println()
			}
			fmt.Println(output)
			fmt.Println()
		},
	}

	cmd.Flags().BoolVar(&chaos, "chaos", false, "Pure stream of consciousness mode")
	cmd.Flags().BoolVar(&antenna, "antenna", false, "Delaware radio tower transmission mode")
	cmd.Flags().Int64Var(&seed, "seed", 0, "Random seed for reproducible output")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().IntVarP(&count, "count", "n", 1, "Generate multiple variations")

	return cmd
}

// Pooparoo is an easter egg - the legendary protagonist
func Pooparoo() string {
	gen := flyingcats.NewGenerator()
	lines := []string{
		tui.AccentStyle.Render("=== THE LEGEND OF POOPAROO ==="),
		"",
		"In the antenna towers along I-95, there lived a cat of legend.",
		"Pooparoo, descendant of Toonces, keeper of the broadcast frequencies.",
		"",
		gen.Generate("a flying cat at an antenna tower"),
		"",
		"The cats still remember. The towers still hum.",
		"And somewhere on the highway, Pooparoo watches.",
	}
	return strings.Join(lines, "\n")
}

func init() {
	// Ensure the tui package is imported for styling
	_ = os.Stderr
}
