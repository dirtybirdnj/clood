package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dirtybirdnj/clood/internal/agents"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func AgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "List and manage agent roles",
		Long: `View and manage configured agent roles for LLM delegation.

Agents are defined in .clood/agents.yaml (project) or ~/.config/clood/agents.yaml (global).
Each agent specifies a model, host, system prompt, and other settings.

Use agents with: clood run --agent <name> "prompt"`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := agents.LoadConfigWithFallback()

			// Check if config file exists
			path, exists := agents.ConfigExists()

			fmt.Println(tui.RenderHeader("Agent Roles"))
			fmt.Println()

			if exists {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Config:"), path)
			} else {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Config:"), "using defaults (no config file found)")
			}
			fmt.Println()

			// Sort agents by name
			names := cfg.ListAgents()
			sort.Strings(names)

			for _, name := range names {
				agent := cfg.GetAgent(name)
				printAgent(agent)
			}

			fmt.Println()
			fmt.Printf("  %s %d agents available\n",
				tui.MutedStyle.Render("Total:"), len(names))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("  Usage: clood run --agent <name> \"prompt\""))
		},
	}

	cmd.AddCommand(agentsShowCmd())
	cmd.AddCommand(agentsInitCmd())

	return cmd
}

func agentsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show detailed agent configuration",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			cfg := agents.LoadConfigWithFallback()
			agent := cfg.GetAgent(name)

			if agent == nil {
				fmt.Println(tui.ErrorStyle.Render("Agent not found: " + name))
				return
			}

			fmt.Println(tui.RenderHeader(fmt.Sprintf("Agent: %s", name)))
			fmt.Println()

			if agent.Description != "" {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Description:"), agent.Description)
			}
			if agent.Model != "" {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Model:"), agent.Model)
			}
			if agent.Host != "" {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Host:"), agent.Host)
			}
			fmt.Printf("  %s %.1f\n", tui.MutedStyle.Render("Temperature:"), agent.GetEffectiveTemperature())
			if agent.MaxTokens > 0 {
				fmt.Printf("  %s %d\n", tui.MutedStyle.Render("Max Tokens:"), agent.MaxTokens)
			}
			if agent.Timeout != "" {
				fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Timeout:"), agent.Timeout)
			}

			if agent.System != "" {
				fmt.Println()
				fmt.Printf("  %s\n", tui.MutedStyle.Render("System Prompt:"))
				// Indent and wrap system prompt
				lines := strings.Split(agent.System, "\n")
				for _, line := range lines {
					fmt.Printf("    %s\n", line)
				}
			}
		},
	}
}

func agentsInitCmd() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create an example agents.yaml config",
		Run: func(cmd *cobra.Command, args []string) {
			var path string
			if global {
				// Global config
				homeDir, err := getHomeDir()
				if err != nil {
					fmt.Println(tui.ErrorStyle.Render("Cannot determine home directory"))
					return
				}
				path = homeDir + "/.config/clood/agents.yaml"
			} else {
				// Project config
				path = ".clood/agents.yaml"
			}

			// Check if exists
			if fileExists(path) {
				fmt.Printf("%s already exists at %s\n",
					tui.MutedStyle.Render("Config"),
					path)
				return
			}

			// Create directory if needed
			if err := ensureDir(path); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error creating directory: " + err.Error()))
				return
			}

			// Write example config
			if err := writeFile(path, exampleAgentsYAML); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error writing config: " + err.Error()))
				return
			}

			fmt.Println(tui.SuccessStyle.Render("Created " + path))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("Edit this file to customize your agent roles."))
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Create global config instead of project-level")

	return cmd
}

func printAgent(agent *agents.Agent) {
	fmt.Printf("  %s %s\n",
		tui.SuccessStyle.Render("●"),
		agent.Name)

	if agent.Description != "" {
		fmt.Printf("    %s\n", tui.MutedStyle.Render(agent.Description))
	}

	// Show model and host if specified
	var details []string
	if agent.Model != "" {
		details = append(details, fmt.Sprintf("model=%s", agent.Model))
	}
	if agent.Host != "" {
		details = append(details, fmt.Sprintf("host=%s", agent.Host))
	}
	details = append(details, fmt.Sprintf("temp=%.1f", agent.GetEffectiveTemperature()))

	if len(details) > 0 {
		fmt.Printf("    %s\n", tui.MutedStyle.Render(strings.Join(details, ", ")))
	}

	fmt.Println()
}

const exampleAgentsYAML = `# Agent Role Configuration
# Project-level: .clood/agents.yaml
# Global: ~/.config/clood/agents.yaml

agents:
  reviewer:
    description: "Code review specialist"
    model: llama3.1:8b
    host: ubuntu25
    system: |
      You are a code reviewer. Analyze code for:
      - Bugs and edge cases
      - Security vulnerabilities
      - Performance issues
      - Style and readability
      Return findings as structured bullet points.
    temperature: 0.3

  coder:
    description: "Code generation specialist"
    model: qwen2.5-coder:7b
    host: ubuntu25
    system: |
      You are a coding assistant. Write clean, well-documented code.
      Follow existing patterns in the codebase.
      Include error handling.
    temperature: 0.7

  documenter:
    description: "Documentation writer"
    model: llama3.1:8b
    host: ubuntu25
    system: |
      You write clear, concise documentation.
      Use examples where helpful.
      Match the project's documentation style.
    temperature: 0.5

  analyst:
    description: "Code analysis and reasoning"
    model: llama3.1:8b
    host: ubuntu25
    system: |
      You analyze code structure and architecture.
      Explain how components interact.
      Identify potential improvements.
    temperature: 0.4

defaults:
  timeout: 120s
  max_tokens: 4096
`

// Helper functions
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func getHomeDir() (string, error) {
	return os.UserHomeDir()
}
