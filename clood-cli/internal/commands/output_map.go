package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/agents"
	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// OutputMapCmd creates the output-map command for Strata-compatible output generation
func OutputMapCmd() *cobra.Command {
	var agentList string
	var outputFile string
	var files []string
	var projectName string

	cmd := &cobra.Command{
		Use:   "output-map [task]",
		Short: "Generate structured output map from agent analysis",
		Long: `Run multiple agents and generate a consolidated output map.

This produces Strata-compatible JSON output with structured issues,
action items, and metadata from each agent's analysis.

Examples:
  # Run reviewer agent on a file
  clood output-map --agent reviewer -f src/main.go "Review this code"

  # Run multiple agents
  clood output-map --agents "reviewer,analyzer" -f src/ "Full analysis"

  # Save output map to file
  clood output-map --agent reviewer -f src/ -o results.json "Review"`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			task := strings.Join(args, " ")

			cfg, err := config.Load()
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Config error: " + err.Error()))
				return
			}

			// Determine project name
			if projectName == "" {
				cwd, _ := os.Getwd()
				projectName = filepath.Base(cwd)
			}

			// Parse agent list
			agentNames := strings.Split(agentList, ",")
			for i := range agentNames {
				agentNames[i] = strings.TrimSpace(agentNames[i])
			}

			// Create output map
			outputMap := output.NewOutputMap(projectName)

			// Setup host manager
			mgr := hosts.NewManager()
			mgr.AddHosts(cfg.Hosts)

			// Load agent configurations
			agentCfg := agents.LoadConfigWithFallback()

			// Run each agent
			for _, agentName := range agentNames {
				if agentName == "" {
					continue
				}

				fmt.Printf("%s Running %s agent...\n", tui.MutedStyle.Render("●"), agentName)

				result := runAgentForMap(agentName, task, files, cfg, mgr, agentCfg)
				outputMap.AddResult(*result)

				if result.Error != "" {
					fmt.Printf("  %s\n", tui.ErrorStyle.Render("Error: "+result.Error))
				} else {
					fmt.Printf("  %s %s\n", tui.SuccessStyle.Render("✓"), result.Summary)
				}
			}

			// Generate summary
			outputMap.Summary = generateMapSummary(outputMap)

			// Output
			jsonOutput, _ := outputMap.ToJSON()

			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(jsonOutput), 0644); err != nil {
					fmt.Println(tui.ErrorStyle.Render("Write error: " + err.Error()))
					return
				}
				fmt.Println()
				fmt.Println(tui.SuccessStyle.Render("✓ Output map saved to " + outputFile))
			} else {
				fmt.Println()
				fmt.Println(tui.RenderHeader("Output Map"))
				fmt.Println()
				fmt.Println(jsonOutput)
			}
		},
	}

	cmd.Flags().StringVar(&agentList, "agents", "reviewer", "Comma-separated list of agents to run")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for JSON map")
	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "Files to include as context")
	cmd.Flags().StringVar(&projectName, "project", "", "Project name (default: current directory)")

	// Alias for single agent
	cmd.Flags().StringP("agent", "a", "", "Single agent to run (shorthand for --agents)")
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if a, _ := cmd.Flags().GetString("agent"); a != "" && agentList == "reviewer" {
			agentList = a
		}
	}

	return cmd
}

func runAgentForMap(agentName, task string, files []string, cfg *config.Config, mgr *hosts.Manager, agentCfg *agents.AgentConfig) *output.AgentResult {
	start := time.Now()

	result := &output.AgentResult{
		TaskType:  "review",
		Agent:     agentName,
		Files:     files,
		Timestamp: time.Now(),
	}

	// Get agent config
	agent := agentCfg.GetAgent(agentName)
	if agent == nil {
		result.Error = "Agent not found: " + agentName
		return result
	}

	// Determine task type from agent
	switch agentName {
	case "reviewer", "code-reviewer":
		result.TaskType = "review"
	case "coder", "generator":
		result.TaskType = "generate"
	case "documenter":
		result.TaskType = "document"
	default:
		result.TaskType = "analyze"
	}

	// Find host
	hostName := agent.Host
	if hostName == "" {
		best := mgr.GetBestHost()
		if best == nil {
			result.Error = "No hosts available"
			return result
		}
		hostName = best.Host.Name
	}

	host := mgr.GetHost(hostName)
	if host == nil {
		result.Error = "Host not found: " + hostName
		return result
	}

	status := mgr.CheckHost(host)
	if !status.Online {
		result.Error = "Host offline: " + hostName
		return result
	}

	result.Host = hostName

	// Select model
	model := agent.Model
	if model == "" {
		model = cfg.Tiers.Fast.Model
	}
	if !hostHasModel(status, model) && len(status.Models) > 0 {
		model = status.Models[0].Name
	}
	result.Model = model

	// Build prompt
	prompt := buildDelegatePrompt(task, files)

	// Execute
	client := mgr.GetClient(hostName)
	resp, err := client.GenerateWithSystem(model, agent.System, prompt)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	result.DurationMs = time.Since(start).Milliseconds()
	result.Tokens = resp.EvalCount
	result.RawResponse = resp.Response

	// Parse response into structured format
	parsed := output.ParseAgentResponse(resp.Response, result.TaskType)
	result.Issues = parsed.Issues
	result.ActionItems = parsed.ActionItems
	result.IssueCount = parsed.IssueCount
	result.HighCount = parsed.HighCount
	result.MediumCount = parsed.MediumCount
	result.LowCount = parsed.LowCount
	result.Summary = parsed.Summary
	result.GeneratedCode = parsed.GeneratedCode
	result.Explanation = parsed.Explanation
	result.Sections = parsed.Sections

	if result.Summary == "" {
		result.Summary = "Analysis complete"
	}

	return result
}

func generateMapSummary(om *output.OutputMap) map[string]interface{} {
	summary := make(map[string]interface{})

	totalIssues := 0
	totalHigh := 0
	totalMedium := 0
	totalLow := 0
	totalActions := 0
	totalDuration := int64(0)

	for _, r := range om.Results {
		totalIssues += r.IssueCount
		totalHigh += r.HighCount
		totalMedium += r.MediumCount
		totalLow += r.LowCount
		totalActions += len(r.ActionItems)
		totalDuration += r.DurationMs
	}

	summary["total_agents"] = len(om.Results)
	summary["total_issues"] = totalIssues
	summary["high_severity"] = totalHigh
	summary["medium_severity"] = totalMedium
	summary["low_severity"] = totalLow
	summary["total_action_items"] = totalActions
	summary["total_duration_ms"] = totalDuration

	return summary
}
