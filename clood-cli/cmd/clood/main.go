package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/dirtybirdnj/clood/internal/commands"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

var version = "0.2.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "clood",
		Short: "Lightning in a Bottle - Local LLM Infrastructure",
		Long:  tui.RenderBanner() + "\n\nInfrastructure layer for local LLM workflows with multi-host routing.",
		Run: func(cmd *cobra.Command, args []string) {
			// If not a TTY (piped/scripted), show help for MCP discovery
			// If TTY (human), show zen greeting
			if !isTerminal() {
				cmd.Help()
				return
			}
			showZenGreeting()
		},
	}

	// Version flag
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(tui.RenderBanner() + "\nVersion: {{.Version}}\n")

	// Global --json flag for machine-readable output (MCP/agent friendly)
	rootCmd.PersistentFlags().BoolVarP(&output.JSONMode, "json", "j", false, "Output in JSON format (for agents/MCP)")

	// Add subcommands - infrastructure focused
	rootCmd.AddCommand(commands.SystemCmd())
	rootCmd.AddCommand(commands.HostsCmd())
	rootCmd.AddCommand(commands.ModelsCmd())
	rootCmd.AddCommand(commands.PullCmd())
	rootCmd.AddCommand(commands.BenchCmd())
	rootCmd.AddCommand(commands.AskCmd())
	rootCmd.AddCommand(commands.CatfightCmd())
	rootCmd.AddCommand(commands.RunCmd())
	rootCmd.AddCommand(commands.DelegateCmd())
	rootCmd.AddCommand(commands.AgentsCmd())
	rootCmd.AddCommand(commands.HealthCmd())
	rootCmd.AddCommand(commands.TuneCmd())
	rootCmd.AddCommand(commands.BonsaiCmd())

	// Code analysis commands
	rootCmd.AddCommand(commands.GrepCmd())
	rootCmd.AddCommand(commands.ImportsCmd())
	rootCmd.AddCommand(commands.AnalyzeCmd())
	rootCmd.AddCommand(commands.TreeCmd())
	rootCmd.AddCommand(commands.SummaryCmd())
	rootCmd.AddCommand(commands.ContextCmd())
	rootCmd.AddCommand(commands.SymbolsCmd())

	// Project management commands
	rootCmd.AddCommand(commands.IssuesCmd())
	rootCmd.AddCommand(commands.ChatCmd())
	rootCmd.AddCommand(commands.HandoffCmd())
	rootCmd.AddCommand(commands.SessionCmd())

	// MCP server command
	rootCmd.AddCommand(commands.ServeCmd())

	// Init command
	rootCmd.AddCommand(initCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render(err.Error()))
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize clood configuration",
		Long:  "Creates the default configuration file at ~/.config/clood/config.yaml",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(tui.MutedStyle.Render("Run 'clood --help' for configuration instructions."))
		},
	}
}

// Haikus for the zen greeting
var zenHaikus = []string{
	`Lightning in bottle—
Local models wait in mist,
One command away.`,

	`Server garden grows,
Silicon leaves catch the prompt,
Tokens bloom like spring.`,

	`The catfight begins,
Models clash in Kitchen Stadium—
Truth emerges whole.`,

	`Ollama whispers,
Across the network it flows,
Wisdom finds its host.`,

	`Context is the key,
Without it models are blind—
Feed them what they need.`,

	`Bonsai patience here,
Prune your prompts with careful thought,
Less becomes much more.`,

	`Lost in the sauce now,
Too strange to live, weird to die—
The peak holds us here.`,

	`Three hosts stand ready,
GPU heat shimmers in wait,
The query takes flight.`,
}

// isTerminal checks if stdout is a terminal (TTY)
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func showZenGreeting() {
	// Seed random
	rand.Seed(time.Now().UnixNano())

	// Pick a random haiku
	haiku := zenHaikus[rand.Intn(len(zenHaikus))]

	// Show bonsai with message (tiny size for zen)
	cbCmd := exec.Command("cbonsai", "-p", "-L", "12", "-M", "2")
	bonsaiOutput, err := cbCmd.CombinedOutput()
	if err == nil {
		fmt.Println()
		fmt.Print(string(bonsaiOutput))
	}

	// Show haiku
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render(haiku))
	fmt.Println()

	// Docs hint
	fmt.Println(tui.MutedStyle.Render("  clood --help    Full documentation"))
	fmt.Println()
}
