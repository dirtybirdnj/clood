package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/ollama"
	"github.com/dirtybirdnj/clood/internal/router"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Message represents a single chat message
type Message struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Tokens    int       `json:"tokens,omitempty"`
}

// Saga represents the ongoing conversation for a project
type Saga struct {
	Name        string    `json:"name"`
	ProjectPath string    `json:"project_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Messages    []Message `json:"messages"`
	Context     string    `json:"context,omitempty"` // Loaded project context
}

// SagaStats contains context usage information
type SagaStats struct {
	HistoryTokens  int
	ContextTokens  int
	TotalTokens    int
	MaxTokens      int
	UsagePercent   float64
	MessageCount   int
}

const (
	maxContextTokens = 16000 // Default max context
	tokensPerChar    = 0.25  // Rough estimate: 4 chars per token
)

func ChatCmd() *cobra.Command {
	var forceTier int
	var forceModel string
	var forceHost string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Start an interactive chat session (The Saga)",
		Long: `Begin or continue The Saga - an ongoing conversation about your project.

The saga:
  - Auto-resumes where you left off
  - Loads project context automatically
  - Shows context health meter
  - Supports slash commands

Slash commands:
  /save FILE   - Save conversation to file
  /clear       - Clear history (keep context)
  /stats       - Show saga statistics
  /context     - Show loaded context
  /quit        - Exit and save saga

Examples:
  clood chat                    # Start/continue saga
  clood chat --tier 4           # Force writing tier`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChat(forceTier, forceModel, forceHost)
		},
	}

	cmd.Flags().IntVarP(&forceTier, "tier", "T", 0, "Force specific tier (1=fast, 2=deep, 3=analysis, 4=writing)")
	cmd.Flags().StringVarP(&forceModel, "model", "m", "", "Force specific model")
	cmd.Flags().StringVarP(&forceHost, "host", "H", "", "Force specific host")

	return cmd
}

func runChat(forceTier int, forceModel, forceHost string) error {
	// Check for .cloodignore
	if _, err := os.Stat(".cloodignore"); err == nil {
		return fmt.Errorf("saga disabled for this directory (.cloodignore found)\nUse --force to override")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Load or create saga
	saga, isNew, err := loadOrCreateSaga()
	if err != nil {
		return fmt.Errorf("loading saga: %w", err)
	}

	// Display header
	if isNew {
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("The Saga of %s begins.", saga.Name)))
		fmt.Println()

		// Load project context for new saga
		context, tokens := loadProjectContext()
		if context != "" {
			saga.Context = context
			fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("Context loaded: %d tokens", tokens)))
		}
	} else {
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("Continuing The Saga of %s...", saga.Name)))
		fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("Last session: %s (%d messages)",
			saga.UpdatedAt.Format("Jan 2 15:04"), len(saga.Messages))))
	}

	// Show health meter
	stats := saga.GetStats()
	renderHealthMeter(stats)
	fmt.Println()

	// Setup router
	r := router.NewRouter(cfg)

	// REPL loop
	reader := bufio.NewReader(os.Stdin)

	for {
		// Prompt
		fmt.Print(tui.MutedStyle.Render("You: "))

		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			shouldQuit, err := handleSlashCommand(input, saga)
			if err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
			}
			if shouldQuit {
				break
			}
			continue
		}

		// Add user message to history
		saga.Messages = append(saga.Messages, Message{
			Role:      "user",
			Content:   input,
			Timestamp: time.Now(),
			Tokens:    estimateTokens(input),
		})

		// Build prompt with context and history
		prompt := buildChatPrompt(saga)

		// Route and execute
		tier := forceTier
		if tier == 0 {
			tier = config.TierDeep // Default to deep for chat
		}

		result, err := r.Route(input, tier, forceModel)
		if err != nil {
			fmt.Println(tui.ErrorStyle.Render("Routing error: " + err.Error()))
			continue
		}

		if result.Client == nil {
			fmt.Println(tui.ErrorStyle.Render("No available host"))
			continue
		}

		// Show what model we're using
		fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("[%s on %s]", result.Model, result.Host.Host.Name)))
		fmt.Println()

		// Stream response
		var responseBuilder strings.Builder
		_, err = result.Client.GenerateStream(result.Model, prompt, func(chunk ollama.GenerateResponse) {
			fmt.Print(chunk.Response)
			responseBuilder.WriteString(chunk.Response)
		})

		if err != nil {
			fmt.Println()
			fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
			continue
		}

		fmt.Println()
		fmt.Println()

		// Add assistant message to history
		response := responseBuilder.String()
		saga.Messages = append(saga.Messages, Message{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now(),
			Tokens:    estimateTokens(response),
		})

		// Save saga after each exchange
		saga.UpdatedAt = time.Now()
		if err := saveSaga(saga); err != nil {
			fmt.Println(tui.ErrorStyle.Render("Warning: couldn't save saga: " + err.Error()))
		}

		// Update health meter
		stats = saga.GetStats()
		if stats.UsagePercent > 80 {
			fmt.Println(tui.ErrorStyle.Render(fmt.Sprintf("⚠️  Context at %.0f%% - consider /clear or compression", stats.UsagePercent)))
		}
	}

	return nil
}

func loadOrCreateSaga() (*Saga, bool, error) {
	sagaPath := getSagaPath()

	// Check if saga exists
	data, err := os.ReadFile(sagaPath)
	if err == nil {
		var saga Saga
		if err := json.Unmarshal(data, &saga); err != nil {
			return nil, false, fmt.Errorf("parsing saga: %w", err)
		}
		return &saga, false, nil
	}

	// Create new saga
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)

	saga := &Saga{
		Name:        projectName,
		ProjectPath: cwd,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Messages:    []Message{},
	}

	// Ensure .clood directory exists
	cloodDir := filepath.Join(cwd, ".clood")
	if err := os.MkdirAll(cloodDir, 0755); err != nil {
		return nil, false, fmt.Errorf("creating .clood dir: %w", err)
	}

	return saga, true, nil
}

func saveSaga(saga *Saga) error {
	sagaPath := getSagaPath()

	data, err := json.MarshalIndent(saga, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sagaPath, data, 0644)
}

func getSagaPath() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".clood", "saga.json")
}

func loadProjectContext() (string, int) {
	var contextBuilder strings.Builder

	// Look for context files
	contextFiles := []string{
		"llm-context/CODEBASE.md",
		"llm-context/API.md",
		"llm-context/ARCHITECTURE.md",
		"CLAUDE.md",
		"README.md",
	}

	for _, file := range contextFiles {
		if content, err := os.ReadFile(file); err == nil {
			contextBuilder.WriteString(fmt.Sprintf("=== %s ===\n", file))
			contextBuilder.Write(content)
			contextBuilder.WriteString("\n\n")
		}
	}

	context := contextBuilder.String()
	// Truncate if too large
	maxContextChars := 8000 // ~2000 tokens
	if len(context) > maxContextChars {
		context = context[:maxContextChars] + "\n... (truncated)"
	}

	return context, estimateTokens(context)
}

func buildChatPrompt(saga *Saga) string {
	var promptBuilder strings.Builder

	// Add project context
	if saga.Context != "" {
		promptBuilder.WriteString("Project Context:\n")
		promptBuilder.WriteString(saga.Context)
		promptBuilder.WriteString("\n---\n\n")
	}

	// Add conversation history (last N messages)
	maxHistory := 20
	startIdx := 0
	if len(saga.Messages) > maxHistory {
		startIdx = len(saga.Messages) - maxHistory
	}

	promptBuilder.WriteString("Conversation:\n")
	for _, msg := range saga.Messages[startIdx:] {
		if msg.Role == "user" {
			promptBuilder.WriteString("User: ")
		} else {
			promptBuilder.WriteString("Assistant: ")
		}
		promptBuilder.WriteString(msg.Content)
		promptBuilder.WriteString("\n\n")
	}

	return promptBuilder.String()
}

func handleSlashCommand(input string, saga *Saga) (bool, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false, nil
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/quit", "/q", "/exit":
		fmt.Println(tui.MutedStyle.Render("Saga saved. See you next time."))
		return true, saveSaga(saga)

	case "/save":
		if len(parts) < 2 {
			return false, fmt.Errorf("usage: /save FILENAME")
		}
		return false, saveConversationToFile(saga, parts[1])

	case "/clear":
		saga.Messages = []Message{}
		fmt.Println(tui.SuccessStyle.Render("History cleared."))
		return false, saveSaga(saga)

	case "/stats":
		stats := saga.GetStats()
		fmt.Println()
		fmt.Println(tui.RenderHeader("Saga Statistics"))
		fmt.Printf("  Messages:      %d\n", stats.MessageCount)
		fmt.Printf("  History:       %d tokens\n", stats.HistoryTokens)
		fmt.Printf("  Context:       %d tokens\n", stats.ContextTokens)
		fmt.Printf("  Total:         %d / %d tokens (%.0f%%)\n", stats.TotalTokens, stats.MaxTokens, stats.UsagePercent)
		fmt.Println()
		renderHealthMeter(stats)
		fmt.Println()
		return false, nil

	case "/context":
		if saga.Context == "" {
			fmt.Println(tui.MutedStyle.Render("No context loaded."))
		} else {
			fmt.Println(tui.RenderHeader("Loaded Context"))
			// Show first 500 chars
			preview := saga.Context
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			fmt.Println(preview)
		}
		return false, nil

	case "/help":
		fmt.Println()
		fmt.Println(tui.RenderHeader("Slash Commands"))
		fmt.Println("  /save FILE   Save conversation to file")
		fmt.Println("  /clear       Clear history (keep context)")
		fmt.Println("  /stats       Show saga statistics")
		fmt.Println("  /context     Show loaded context")
		fmt.Println("  /help        Show this help")
		fmt.Println("  /quit        Exit and save saga")
		fmt.Println()
		return false, nil

	default:
		return false, fmt.Errorf("unknown command: %s (try /help)", cmd)
	}
}

func saveConversationToFile(saga *Saga, filename string) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# The Saga of %s\n\n", saga.Name))
	content.WriteString(fmt.Sprintf("Exported: %s\n\n", time.Now().Format("2006-01-02 15:04")))
	content.WriteString("---\n\n")

	for _, msg := range saga.Messages {
		if msg.Role == "user" {
			content.WriteString("**You:**\n")
		} else {
			content.WriteString("**Assistant:**\n")
		}
		content.WriteString(msg.Content)
		content.WriteString("\n\n")
	}

	if err := os.WriteFile(filename, []byte(content.String()), 0644); err != nil {
		return err
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("Saved to %s", filename)))
	return nil
}

func (s *Saga) GetStats() SagaStats {
	historyTokens := 0
	for _, msg := range s.Messages {
		historyTokens += msg.Tokens
		if msg.Tokens == 0 {
			historyTokens += estimateTokens(msg.Content)
		}
	}

	contextTokens := estimateTokens(s.Context)
	totalTokens := historyTokens + contextTokens

	return SagaStats{
		HistoryTokens:  historyTokens,
		ContextTokens:  contextTokens,
		TotalTokens:    totalTokens,
		MaxTokens:      maxContextTokens,
		UsagePercent:   float64(totalTokens) / float64(maxContextTokens) * 100,
		MessageCount:   len(s.Messages),
	}
}

func estimateTokens(text string) int {
	return int(float64(len(text)) * tokensPerChar)
}

func renderHealthMeter(stats SagaStats) {
	// Build progress bar
	barWidth := 30
	filled := int(stats.UsagePercent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Color based on usage
	var meterStr string
	if stats.UsagePercent < 50 {
		meterStr = tui.SuccessStyle.Render(fmt.Sprintf("Context: %s %.0f%%", bar, stats.UsagePercent))
	} else if stats.UsagePercent < 80 {
		meterStr = tui.TierFastStyle.Render(fmt.Sprintf("Context: %s %.0f%% ⚠️", bar, stats.UsagePercent))
	} else {
		meterStr = tui.ErrorStyle.Render(fmt.Sprintf("Context: %s %.0f%% 🔴", bar, stats.UsagePercent))
	}

	fmt.Println(meterStr)
}
