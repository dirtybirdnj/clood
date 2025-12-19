package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

func PreflightCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Pre-session check: see what's available locally",
		Long: `Shows what local tools and models are available before starting work.

Run this at the START of every coding session to:
- See available local discovery tools (grep, tree, symbols, imports, context)
- Check Ollama host status and available models
- Get the recommended workflow for local-first operations

This helps avoid unnecessary network requests by showing what can be done locally.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get working directory
			cwd, _ := os.Getwd()

			// Load config and check hosts
			cfg, _ := config.Load()

			var onlineHosts []hostInfo
			var allModels []string
			modelSeen := make(map[string]bool)

			if cfg != nil {
				mgr := hosts.NewManager()
				mgr.AddHosts(cfg.Hosts)

				statuses := mgr.CheckAllHosts()
				for _, st := range statuses {
					if st.Online {
						hi := hostInfo{
							Name:    st.Host.Name,
							Latency: st.Latency.Milliseconds(),
						}
						for _, m := range st.Models {
							hi.Models = append(hi.Models, m.Name)
							if !modelSeen[m.Name] {
								modelSeen[m.Name] = true
								allModels = append(allModels, m.Name)
							}
						}
						onlineHosts = append(onlineHosts, hi)
					}
				}
			}

			ollamaOnline := len(onlineHosts) > 0

			// Check for JSON output
			if jsonOutput || output.IsJSON() {
				printPreflightJSON(cwd, onlineHosts, allModels, ollamaOnline)
				return
			}

			// Human-readable output
			printPreflightHuman(cwd, onlineHosts, allModels, ollamaOnline)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

type hostInfo struct {
	Name    string
	Latency int64
	Models  []string
}

func printPreflightJSON(cwd string, hosts []hostInfo, models []string, ollamaOnline bool) {
	result := map[string]interface{}{
		"working_directory": cwd,
		"local_tools": []string{
			"clood grep PATTERN - Search codebase (replaces web search for 'where is X')",
			"clood tree [PATH] - Directory structure (replaces web search for 'project structure')",
			"clood symbols PATH - Extract functions/types (replaces web search for 'what functions...')",
			"clood imports PATH - Dependency analysis (replaces web search for 'what does X import')",
			"clood context [PATH] - Generate project summary",
			"clood system - Hardware info",
		},
		"ollama_online": ollamaOnline,
		"workflow": []string{
			"1. clood tree - Understand structure",
			"2. clood grep - Find relevant code",
			"3. clood symbols - Know the API",
			"4. clood ask - Query local LLM",
			"5. WebSearch - ONLY if above fail",
		},
	}

	if ollamaOnline {
		hostList := make([]map[string]interface{}, 0, len(hosts))
		for _, h := range hosts {
			hostList = append(hostList, map[string]interface{}{
				"name":       h.Name,
				"latency_ms": h.Latency,
				"models":     h.Models,
			})
		}
		result["hosts"] = hostList
		result["models_available"] = models
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func printPreflightHuman(cwd string, hosts []hostInfo, models []string, ollamaOnline bool) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("CLOOD PREFLIGHT CHECK"))
	fmt.Println()

	// Working directory
	fmt.Printf("  %s %s\n\n", tui.MutedStyle.Render("Working Directory:"), cwd)

	// Local discovery tools
	fmt.Println(tui.AccentStyle.Render("  LOCAL DISCOVERY TOOLS (instant, 0 tokens)"))
	fmt.Println()
	tools := []struct{ cmd, desc, replaces string }{
		{"clood grep PATTERN", "Search codebase", "WebSearch for 'where is X'"},
		{"clood tree [PATH]", "Project structure", "WebSearch for 'project layout'"},
		{"clood symbols PATH", "Extract functions/types", "WebSearch for 'what functions...'"},
		{"clood imports PATH", "Dependency analysis", "WebSearch for 'what imports...'"},
		{"clood context [PATH]", "Generate project summary", "Reading many files manually"},
	}
	for _, t := range tools {
		fmt.Printf("    %s\n", tui.SuccessStyle.Render(t.cmd))
		fmt.Printf("      %s\n", tui.MutedStyle.Render(t.desc+" - replaces "+t.replaces))
	}
	fmt.Println()

	// Ollama status
	if ollamaOnline {
		fmt.Println(tui.SuccessStyle.Render("  OLLAMA: ONLINE"))
		for _, h := range hosts {
			fmt.Printf("    %s %s (%dms)\n",
				tui.SuccessStyle.Render("●"),
				h.Name,
				h.Latency)
		}
		fmt.Println()

		// Show models (limit to 5)
		if len(models) > 0 {
			fmt.Printf("    %s ", tui.MutedStyle.Render("Models:"))
			if len(models) > 5 {
				fmt.Printf("%s + %d more\n", strings.Join(models[:5], ", "), len(models)-5)
			} else {
				fmt.Printf("%s\n", strings.Join(models, ", "))
			}
			fmt.Println()
		}

		fmt.Printf("    %s Use %s for local LLM queries (no cloud API needed)\n",
			tui.SuccessStyle.Render("→"),
			tui.AccentStyle.Render("clood ask"))
	} else {
		fmt.Println(tui.ErrorStyle.Render("  OLLAMA: OFFLINE"))
		fmt.Println(tui.MutedStyle.Render("    No Ollama hosts available. Cloud LLM may be needed."))
	}
	fmt.Println()

	// Recommended workflow
	fmt.Println(tui.AccentStyle.Render("  RECOMMENDED WORKFLOW"))
	fmt.Println()
	fmt.Println("    1. " + tui.SuccessStyle.Render("clood tree") + " → Understand project structure")
	fmt.Println("    2. " + tui.SuccessStyle.Render("clood grep") + " → Find relevant code")
	fmt.Println("    3. " + tui.SuccessStyle.Render("clood symbols") + " → Know the API surface")
	if ollamaOnline {
		fmt.Println("    4. " + tui.SuccessStyle.Render("clood ask") + " → Query local LLM if needed")
		fmt.Println("    5. " + tui.ErrorStyle.Render("WebSearch") + " → ONLY if above tools can't help")
	} else {
		fmt.Println("    4. " + tui.ErrorStyle.Render("WebSearch") + " → Only when local tools can't help")
	}
	fmt.Println()

	// Reminder
	fmt.Println(tui.MutedStyle.Render("  Remember: Local tools are instant and free. Network is slow and costly."))
	fmt.Println()
}
