package commands

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

const extractSystemPrompt = `You are a data extraction assistant. Extract structured data from unstructured text.

CRITICAL RULES:
1. Output ONLY valid JSON - no explanations, no markdown
2. Extract all matching entities from the input
3. Use null for missing fields
4. Return an array of objects if multiple entities found
5. Be precise - only extract what's actually in the text

Example output format:
[{"field1": "value1", "field2": "value2"}, ...]`

func ExtractCmd() *cobra.Command {
	var schema string
	var format string
	var outputFile string
	var model string
	var auto bool

	cmd := &cobra.Command{
		Use:   "extract [file]",
		Short: "Extract structured data from unstructured text",
		Long: `Parse unstructured text into JSON, CSV, or YAML.

Reads from file or stdin, extracts data matching the schema.

Examples:
  cat emails.txt | clood extract --schema "name,email,company"
  clood extract invoice.pdf --schema "date,amount,vendor"
  clood extract data.txt --auto --format csv
  clood extract *.txt --schema "timestamp,level,message" -o logs.json`,
		Run: func(cmd *cobra.Command, args []string) {
			// Read input
			var inputText string
			var err error

			if len(args) > 0 {
				// Read from file
				data, err := os.ReadFile(args[0])
				if err != nil {
					fmt.Println(tui.ErrorStyle.Render("Error reading file: " + err.Error()))
					return
				}
				inputText = string(data)
			} else {
				// Read from stdin
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					fmt.Println(tui.ErrorStyle.Render("No input provided. Pipe text or specify a file."))
					return
				}
				reader := bufio.NewReader(os.Stdin)
				var sb strings.Builder
				for {
					line, err := reader.ReadString('\n')
					sb.WriteString(line)
					if err == io.EOF {
						break
					}
					if err != nil {
						fmt.Println(tui.ErrorStyle.Render("Error reading stdin: " + err.Error()))
						return
					}
				}
				inputText = sb.String()
			}

			if strings.TrimSpace(inputText) == "" {
				fmt.Println(tui.ErrorStyle.Render("Input is empty"))
				return
			}

			// Auto-detect schema if requested
			if auto && schema == "" {
				schema = "key,value"
			}

			if schema == "" {
				fmt.Println(tui.ErrorStyle.Render("Schema required. Use --schema \"field1,field2\" or --auto"))
				return
			}

			result, err := extractData(inputText, schema, model)
			if err != nil {
				if output.IsJSON() {
					fmt.Printf(`{"error": %q}`, err.Error())
				} else {
					fmt.Println(tui.ErrorStyle.Render("Extraction failed: " + err.Error()))
				}
				return
			}

			// Format output
			var outputStr string
			switch format {
			case "csv":
				outputStr, err = formatCSV(result)
			case "yaml":
				outputStr, err = formatYAML(result)
			default:
				outputStr, err = formatJSON(result)
			}

			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Format error: " + err.Error()))
				return
			}

			// Write output
			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(outputStr), 0644); err != nil {
					fmt.Println(tui.ErrorStyle.Render("Write error: " + err.Error()))
					return
				}
				fmt.Println(tui.SuccessStyle.Render("✓ Extracted to " + outputFile))
			} else {
				fmt.Println(outputStr)
			}
		},
	}

	cmd.Flags().StringVar(&schema, "schema", "", "Comma-separated field names (e.g., \"name,email,company\")")
	cmd.Flags().StringVar(&format, "format", "json", "Output format: json, csv, yaml")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model to use")
	cmd.Flags().BoolVar(&auto, "auto", false, "Auto-detect schema")

	return cmd
}

func extractData(text, schema, modelOverride string) ([]map[string]interface{}, error) {
	// Parse schema
	fields := strings.Split(schema, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	// Build extraction prompt
	userPrompt := fmt.Sprintf(`Extract data matching this schema: %s

Text to extract from:
%s

Return a JSON array of objects with these exact field names. Include all matching entities found.`,
		strings.Join(fields, ", "), text)

	// Truncate if too long
	if len(userPrompt) > 8000 {
		userPrompt = userPrompt[:8000] + "\n(truncated)"
	}

	// Load config
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

	modelName := modelOverride
	if modelName == "" {
		modelName = cfg.Tiers.Fast.Model
		if modelName == "" && len(targetHost.Models) > 0 {
			modelName = targetHost.Models[0].Name
		}
	}

	// Generate
	client := ollama.NewClient(targetHost.Host.URL, 60*time.Second)
	resp, err := client.GenerateWithSystem(modelName, extractSystemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	// Parse response as JSON
	responseText := strings.TrimSpace(resp.Response)

	// Clean up markdown if present
	responseText = strings.TrimPrefix(responseText, "```json\n")
	responseText = strings.TrimPrefix(responseText, "```\n")
	responseText = strings.TrimSuffix(responseText, "\n```")
	responseText = strings.TrimSuffix(responseText, "```")

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		// Try parsing as single object
		var single map[string]interface{}
		if err2 := json.Unmarshal([]byte(responseText), &single); err2 == nil {
			result = []map[string]interface{}{single}
		} else {
			return nil, fmt.Errorf("invalid JSON response: %w", err)
		}
	}

	return result, nil
}

func formatJSON(data []map[string]interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func formatCSV(data []map[string]interface{}) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	// Get headers from first item
	var headers []string
	for key := range data[0] {
		headers = append(headers, key)
	}

	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Write header
	writer.Write(headers)

	// Write rows
	for _, item := range data {
		row := make([]string, len(headers))
		for i, h := range headers {
			if val, ok := item[h]; ok {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		writer.Write(row)
	}

	writer.Flush()
	return sb.String(), nil
}

func formatYAML(data []map[string]interface{}) (string, error) {
	// Simple YAML output
	var sb strings.Builder
	for i, item := range data {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("- ")
		first := true
		for key, val := range item {
			if !first {
				sb.WriteString("  ")
			}
			sb.WriteString(fmt.Sprintf("%s: %v\n", key, val))
			first = false
		}
	}
	return sb.String(), nil
}
