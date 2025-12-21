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

// Command group IDs
const (
	GroupStart       = "start"
	GroupInfra       = "infra"
	GroupQuery       = "query"
	GroupCompare     = "compare"
	GroupCodebase    = "codebase"
	GroupAIPowered   = "ai"
	GroupSession     = "session"
	GroupAgents      = "agents"
	GroupMCP         = "mcp"
	GroupMeta        = "meta"
	GroupExperimental = "experimental"
)

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

	// Define command groups (order matters - this is display order)
	rootCmd.AddGroup(
		&cobra.Group{ID: GroupStart, Title: "🚀 Getting Started:"},
		&cobra.Group{ID: GroupInfra, Title: "🔌 Infrastructure & Discovery:"},
		&cobra.Group{ID: GroupQuery, Title: "💬 Querying LLMs:"},
		&cobra.Group{ID: GroupCompare, Title: "⚔️  Model Comparison:"},
		&cobra.Group{ID: GroupCodebase, Title: "🔍 Codebase Analysis (zero network):"},
		&cobra.Group{ID: GroupAIPowered, Title: "🤖 AI-Powered Tools:"},
		&cobra.Group{ID: GroupSession, Title: "📚 Session & Context:"},
		&cobra.Group{ID: GroupAgents, Title: "🕵️  Agents & Delegation:"},
		&cobra.Group{ID: GroupMCP, Title: "🔗 MCP Server:"},
		&cobra.Group{ID: GroupMeta, Title: "🔧 Meta & Development:"},
		&cobra.Group{ID: GroupExperimental, Title: "🧪 Experimental:"},
	)

	// ═══════════════════════════════════════════════════════════════
	// 🚀 GETTING STARTED
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, initCmd(), GroupStart)
	addWithGroup(rootCmd, commands.SetupCmd(), GroupStart)
	addWithGroup(rootCmd, commands.VerifyCmd(), GroupStart)
	addWithGroup(rootCmd, commands.DoctorCmd(), GroupStart)
	addWithGroup(rootCmd, commands.UpdateCmd(), GroupStart)
	addWithGroup(rootCmd, completionCmd(), GroupStart)

	// ═══════════════════════════════════════════════════════════════
	// 🔌 INFRASTRUCTURE & DISCOVERY
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.PreflightCmd(), GroupInfra)
	addWithGroup(rootCmd, commands.SystemCmd(), GroupInfra)
	addWithGroup(rootCmd, commands.HostsCmd(), GroupInfra)
	addWithGroup(rootCmd, commands.ModelsCmd(), GroupInfra)
	addWithGroup(rootCmd, commands.DiscoverCmd(), GroupInfra)
	addWithGroup(rootCmd, commands.HealthCmd(), GroupInfra)
	addWithGroup(rootCmd, commands.TuneCmd(), GroupInfra)
	addWithGroup(rootCmd, commands.PullCmd(), GroupInfra)

	// ═══════════════════════════════════════════════════════════════
	// 💬 QUERYING LLMs
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.AskCmd(), GroupQuery)
	addWithGroup(rootCmd, commands.RunCmd(), GroupQuery)
	addWithGroup(rootCmd, commands.ChatCmd(), GroupQuery)

	// ═══════════════════════════════════════════════════════════════
	// ⚔️  MODEL COMPARISON
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.CatfightCmd(), GroupCompare)
	addWithGroup(rootCmd, commands.CatfightLiveCmd(), GroupCompare)
	addWithGroup(rootCmd, commands.WatchCmd(), GroupCompare)
	addWithGroup(rootCmd, commands.BenchCmd(), GroupCompare)

	// ═══════════════════════════════════════════════════════════════
	// 🔍 CODEBASE ANALYSIS (zero network)
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.GrepCmd(), GroupCodebase)
	addWithGroup(rootCmd, commands.TreeCmd(), GroupCodebase)
	addWithGroup(rootCmd, commands.SymbolsCmd(), GroupCodebase)
	addWithGroup(rootCmd, commands.ImportsCmd(), GroupCodebase)
	addWithGroup(rootCmd, commands.ContextCmd(), GroupCodebase)
	addWithGroup(rootCmd, commands.SummaryCmd(), GroupCodebase)

	// ═══════════════════════════════════════════════════════════════
	// 🤖 AI-POWERED TOOLS
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.AnalyzeCmd(), GroupAIPowered)
	addWithGroup(rootCmd, commands.CommitMsgCmd(), GroupAIPowered)
	addWithGroup(rootCmd, commands.ReviewPRCmd(), GroupAIPowered)
	addWithGroup(rootCmd, commands.GenerateTestsCmd(), GroupAIPowered)
	addWithGroup(rootCmd, commands.ExtractCmd(), GroupAIPowered)

	// ═══════════════════════════════════════════════════════════════
	// 📚 SESSION & CONTEXT
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.SessionCmd(), GroupSession)
	addWithGroup(rootCmd, commands.HandoffCmd(), GroupSession)
	addWithGroup(rootCmd, commands.CheckpointCmd(), GroupSession)
	addWithGroup(rootCmd, commands.FocusCmd(), GroupSession)
	addWithGroup(rootCmd, commands.BeansCmd(), GroupSession)

	// ═══════════════════════════════════════════════════════════════
	// 🕵️  AGENTS & DELEGATION
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.AgentCmd(), GroupAgents)
	addWithGroup(rootCmd, commands.AgentPreflightCmd(), GroupAgents)
	addWithGroup(rootCmd, commands.AgentsCmd(), GroupAgents)
	addWithGroup(rootCmd, commands.DelegateCmd(), GroupAgents)

	// ═══════════════════════════════════════════════════════════════
	// 🔗 MCP SERVER
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.McpCmd(), GroupMCP)
	addWithGroup(rootCmd, commands.ServeCmd(), GroupMCP)

	// ═══════════════════════════════════════════════════════════════
	// 🔧 META & DEVELOPMENT
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.BuildCmd(), GroupMeta)
	addWithGroup(rootCmd, commands.BcbcCmd(), GroupMeta)
	addWithGroup(rootCmd, commands.BuildCheckCmd(), GroupMeta)
	addWithGroup(rootCmd, commands.SettingsAuditCmd(), GroupMeta)
	addWithGroup(rootCmd, commands.IssuesCmd(), GroupMeta)

	// ═══════════════════════════════════════════════════════════════
	// 🧪 EXPERIMENTAL
	// ═══════════════════════════════════════════════════════════════
	addWithGroup(rootCmd, commands.BonsaiCmd(), GroupExperimental)
	addWithGroup(rootCmd, commands.FlyingCatsCmd(), GroupExperimental)
	addWithGroup(rootCmd, commands.InceptionCmd(), GroupExperimental)
	addWithGroup(rootCmd, commands.SnakewayProtoCmd(), GroupExperimental)
	addWithGroup(rootCmd, commands.OutputMapCmd(), GroupExperimental)

	// Set current version for update command
	commands.CurrentVersion = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render(err.Error()))
		os.Exit(1)
	}
}

// addWithGroup adds a command to the root with a group assignment
func addWithGroup(root *cobra.Command, cmd *cobra.Command, groupID string) {
	cmd.GroupID = groupID
	root.AddCommand(cmd)
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

	`Flying cats take roost,
Antenna towers hum with life—
Pooparoo watches.`,
}

// isTerminal checks if stdout is a terminal (TTY)
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for clood.

To load completions:

Bash:
  $ source <(clood completion bash)
  # Or add to ~/.bashrc:
  $ clood completion bash > /etc/bash_completion.d/clood

Zsh:
  $ source <(clood completion zsh)
  # Or add to ~/.zshrc:
  $ clood completion zsh > "${fpath[1]}/_clood"

Fish:
  $ clood completion fish | source
  # Or persist:
  $ clood completion fish > ~/.config/fish/completions/clood.fish

PowerShell:
  PS> clood completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}

func showZenGreeting() {
	// Seed random
	rand.Seed(time.Now().UnixNano())

	// Pick a random haiku
	haiku := zenHaikus[rand.Intn(len(zenHaikus))]

	// Show bonsai with message (tiny size for zen)
	if _, err := exec.LookPath("cbonsai"); err == nil {
		cbCmd := exec.Command("cbonsai", "-p", "-L", "12", "-M", "2")
		bonsaiOutput, err := cbCmd.CombinedOutput()
		if err == nil {
			fmt.Println()
			fmt.Print(string(bonsaiOutput))
			// Reset terminal attributes after cbonsai (it may leave escape codes)
			fmt.Print("\033[0m")
		}
	}

	// Show haiku
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render(haiku))
	fmt.Println()

	// Docs hint
	fmt.Println(tui.MutedStyle.Render("  clood --help    Full documentation"))
	fmt.Println()
}
