package main

import (
	"fmt"
	"os"

	"github.com/dirtybirdnj/clood/internal/commands"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "clood",
		Short: "Lightning in a Bottle - Local LLM toolkit",
		Long:  tui.RenderBanner() + "\n\nA smart CLI for local LLM workflows with tiered inference.",
		Run: func(cmd *cobra.Command, args []string) {
			// No args = launch TUI
			if len(args) == 0 {
				fmt.Println(tui.RenderBanner())
				fmt.Println()
				fmt.Println(tui.MutedStyle.Render("Use 'clood --help' for available commands"))
				fmt.Println(tui.MutedStyle.Render("Use 'clood ask \"your question\"' to query"))
			}
		},
	}

	// Version flag
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(tui.RenderBanner() + "\nVersion: {{.Version}}\n")

	// Add subcommands
	rootCmd.AddCommand(commands.TreeCmd())
	rootCmd.AddCommand(commands.SummaryCmd())
	rootCmd.AddCommand(commands.ContextCmd())
	rootCmd.AddCommand(commands.AskCmd())
	rootCmd.AddCommand(commands.SymbolsCmd())
	rootCmd.AddCommand(commands.HealthCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render(err.Error()))
		os.Exit(1)
	}
}
