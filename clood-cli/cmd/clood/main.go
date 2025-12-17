package main

import (
	"fmt"
	"os"

	"github.com/dirtybirdnj/clood/internal/commands"
	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
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
			// No args = show banner and quick status
			fmt.Println(tui.RenderBanner())
			fmt.Println()

			// Quick status check
			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.MutedStyle.Render("Config: not loaded"))
			} else {
				fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("Config: %s", config.ConfigPath())))

				// Quick host check
				mgr := hosts.NewManager()
				mgr.AddHosts(cfg.Hosts)

				online := 0
				for _, h := range cfg.Hosts {
					status := mgr.CheckHost(h)
					if status.Online {
						online++
					}
				}

				if online > 0 {
					fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("Hosts: %d/%d online", online, len(cfg.Hosts))))
				} else {
					fmt.Println(tui.ErrorStyle.Render(fmt.Sprintf("Hosts: 0/%d online", len(cfg.Hosts))))
				}

				fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("Tiers: fast=%s, deep=%s",
					cfg.Tiers.Fast.Model,
					cfg.Tiers.Deep.Model)))
			}

			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("Commands:"))
			fmt.Println(tui.MutedStyle.Render("  clood system     Hardware analysis"))
			fmt.Println(tui.MutedStyle.Render("  clood hosts      List Ollama hosts"))
			fmt.Println(tui.MutedStyle.Render("  clood models     List available models"))
			fmt.Println(tui.MutedStyle.Render("  clood bench      Benchmark a model"))
			fmt.Println(tui.MutedStyle.Render("  clood ask        Query with auto-routing"))
			fmt.Println(tui.MutedStyle.Render("  clood health     Full health check"))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("Use 'clood --help' for more information"))
		},
	}

	// Version flag
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(tui.RenderBanner() + "\nVersion: {{.Version}}\n")

	// Add subcommands - infrastructure focused
	rootCmd.AddCommand(commands.SystemCmd())
	rootCmd.AddCommand(commands.HostsCmd())
	rootCmd.AddCommand(commands.ModelsCmd())
	rootCmd.AddCommand(commands.BenchCmd())
	rootCmd.AddCommand(commands.AskCmd())
	rootCmd.AddCommand(commands.HealthCmd())
	rootCmd.AddCommand(commands.TuneCmd())

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
			path := config.ConfigPath()

			if config.Exists() {
				fmt.Printf("Config already exists at %s\n", path)
				fmt.Println(tui.MutedStyle.Render("Use 'clood config' to edit"))
				return
			}

			if err := config.Init(); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error creating config: " + err.Error()))
				return
			}

			fmt.Println(tui.SuccessStyle.Render("Created config at " + path))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("Edit this file to configure your Ollama hosts and model tiers."))
			fmt.Println(tui.MutedStyle.Render("Run 'clood hosts' to verify connectivity."))
		},
	}
}
