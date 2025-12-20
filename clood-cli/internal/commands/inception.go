package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dirtybirdnj/clood/internal/inception"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Inception message types
type inceptionChunkMsg string
type inceptionInjectionMsg string
type inceptionSubQueryChunkMsg string // Streaming chunk from expert model
type inceptionSubQueryStartMsg struct{ query inception.SubQuery }
type inceptionSubQueryEndMsg struct{ result inception.SubQueryResult }
type inceptionDoneMsg struct{}
type inceptionErrorMsg error

// inceptionModel is the bubbletea model for LLM Inception
type inceptionModel struct {
	viewport   viewport.Model
	content    string
	width      int
	height     int
	ready      bool
	streaming  bool
	following  bool
	modelName  string
	inputBuffer string
	history    []ChatMessage
	currentAssistant string
	promptMode bool

	// Inception-specific
	handler            *inception.Handler
	subQueryActive     bool
	subQueryModel      string
	subQueryCount      int
	subQueryResponses  []string // Track expert responses for continuation
	needsContinuation  bool     // Flag to auto-continue after sub-queries

	// Spinner for loading states
	spinner spinner.Model

	// Channels
	inputChan  chan string
	outputChan chan string
}

// Inception-specific styles
var (
	inceptionHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF00FF")).
		Background(lipgloss.Color("#1a1a2e")).
		Padding(0, 1)

	subQueryStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#2a2a3e"))

	subQueryActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFD700")).
		Blink(true)
)

func InceptionCmd() *cobra.Command {
	var model string
	var expertModel string

	cmd := &cobra.Command{
		Use:   "inception",
		Short: "🌀 LLM Inception - one LLM can query another mid-stream",
		Long: `LLM Inception allows one streaming LLM to synchronously query another.

The main LLM can emit:
  <sub-query model="science">What is orbital velocity?</sub-query>

The stream pauses, queries the expert model, injects the response,
and the main LLM continues with the new knowledge.

Available expert aliases (mapped to installed models):
  science, math, reason  → llama3.1:8b (knowledge/reasoning)
  code, coder, debug     → qwen2.5-coder:3b (programming)
  creative, writer       → mistral:7b (language/creativity)
  fast, quick            → tinyllama/qwen2.5-coder:3b (speed)

Examples:
  clood inception                                    # defaults
  clood inception --model llama3.1:8b                # stronger main model
  clood inception --model qwen2.5-coder:14b          # use your 14b model

Pro tip: Ask something that REQUIRES expert knowledge, like:
  "Write Python code to calculate the ISS orbital position"
  "Explain quantum entanglement and write a simulation"`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create inception handler
			handler := inception.NewHandler()
			if expertModel != "" {
				// Override default expert model
				handler.Registry["science"] = expertModel
				handler.Registry["math"] = expertModel
				handler.Registry["code"] = expertModel
				handler.Registry["default"] = expertModel
			}

			// Initialize spinner
			s := spinner.New()
			s.Spinner = spinner.Dot
			s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))

			m := inceptionModel{
				modelName:  model,
				promptMode: true,
				following:  true,
				history:    []ChatMessage{},
				handler:    handler,
				spinner:    s,
			}

			// Set up handler callbacks
			handler.OnSubQueryStart = func(q inception.SubQuery) {
				// This will be called from the stream processor goroutine
			}
			handler.OnSubQueryEnd = func(r inception.SubQueryResult) {
				// This will be called from the stream processor goroutine
			}

			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
			}
		},
	}

	cmd.Flags().StringVar(&model, "model", "qwen2.5-coder:3b", "Main model for generation")
	cmd.Flags().StringVar(&expertModel, "expert", "", "Expert model for sub-queries (defaults to model registry)")
	cmd.Flags().Bool("demo", false, "Demo mode: inject a sub-query example to show the feature")

	return cmd
}

func (m inceptionModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m inceptionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case inceptionChunkMsg:
		chunk := string(msg)
		m.content += chunk
		m.currentAssistant += chunk
		m.viewport.SetContent(m.renderContent())
		if m.following {
			m.viewport.GotoBottom()
		}
		// Keep listening
		if m.outputChan != nil {
			cmds = append(cmds, waitForInceptionOutput(m.outputChan))
		}

	case inceptionInjectionMsg:
		// Sub-query response injection
		injection := string(msg)
		m.content += injection
		m.subQueryActive = false
		m.needsContinuation = true // Mark that we need to continue after main stream
		m.subQueryResponses = append(m.subQueryResponses, injection)
		m.viewport.SetContent(m.renderContent())
		if m.following {
			m.viewport.GotoBottom()
		}
		// Keep listening
		if m.outputChan != nil {
			cmds = append(cmds, waitForInceptionOutput(m.outputChan))
		}

	case inceptionSubQueryChunkMsg:
		// Streaming chunk from expert model - append to display
		m.content += string(msg)
		m.viewport.SetContent(m.renderContent())
		if m.following {
			m.viewport.GotoBottom()
		}
		// Keep listening for more chunks
		if m.outputChan != nil {
			cmds = append(cmds, waitForInceptionOutput(m.outputChan))
		}

	case inceptionSubQueryStartMsg:
		m.subQueryActive = true
		m.subQueryModel = msg.query.Model
		m.subQueryCount++
		// Add visual indicator - start of expert response section
		m.content += fmt.Sprintf("\n───── ⚡ EXPERT [%s] ─────\n", msg.query.Model)
		m.viewport.SetContent(m.renderContent())
		if m.following {
			m.viewport.GotoBottom()
		}
		// Keep listening for streaming chunks
		if m.outputChan != nil {
			cmds = append(cmds, waitForInceptionOutput(m.outputChan))
		}

	case inceptionSubQueryEndMsg:
		m.subQueryActive = false

	case spinner.TickMsg:
		// Update spinner animation
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)

	case inceptionDoneMsg:
		m.streaming = false
		if m.currentAssistant != "" {
			m.history = append(m.history, ChatMessage{Role: "assistant", Content: m.currentAssistant})
			m.currentAssistant = ""
		}

		// Auto-continue if we had sub-queries - the main model needs to incorporate expert responses
		if m.needsContinuation && len(m.subQueryResponses) > 0 {
			m.needsContinuation = false
			m.content += "\n───── 🔄 CONTINUING WITH EXPERT KNOWLEDGE ─────\n\n"
			m.viewport.SetContent(m.renderContent())

			// Build continuation prompt with expert responses
			var expertContext strings.Builder
			expertContext.WriteString("The expert models have provided the following information:\n\n")
			for _, resp := range m.subQueryResponses {
				expertContext.WriteString(resp)
				expertContext.WriteString("\n")
			}
			expertContext.WriteString("\nPlease continue your response, incorporating this expert knowledge. Do not use any more sub-queries - provide the final answer directly.")

			// Add to history and continue
			m.history = append(m.history, ChatMessage{Role: "user", Content: expertContext.String()})
			m.subQueryResponses = nil // Clear for next round

			// Start new stream for continuation
			m.streaming = true
			m.following = true
			rawChan := make(chan string, 100)
			ctx := context.Background()
			processor := inception.NewStreamProcessor(ctx, m.handler, rawChan)
			m.outputChan = make(chan string, 100)

			processedChan := processor.Start()
			go func() {
				for chunk := range processedChan {
					m.outputChan <- chunk
				}
				close(m.outputChan)
			}()

			historyCopy := make([]ChatMessage, len(m.history))
			copy(historyCopy, m.history)
			go streamInceptionChat(m.modelName, historyCopy, rawChan)

			cmds = append(cmds, waitForInceptionOutput(m.outputChan))
		} else {
			m.content += "\n═══════════════════════════════════════════════════════════════════\n"
		}
		m.viewport.SetContent(m.renderContent())

	case inceptionErrorMsg:
		m.content += fmt.Sprintf("\n❌ ERROR: %v\n", msg)
		m.viewport.SetContent(m.renderContent())

	case tea.KeyMsg:
		key := msg.String()

		// Handle paste first (bracketed paste sends all chars in Runes with Paste=true)
		if msg.Paste {
			for _, r := range msg.Runes {
				m.inputBuffer += string(r)
			}
			return m, nil
		}

		switch key {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.inputBuffer != "" {
				m.inputBuffer = ""
			} else {
				return m, tea.Quit
			}

		case "enter":
			if m.inputBuffer != "" && !m.streaming {
				if m.promptMode {
					m.startInception()
					m.viewport.SetContent(m.renderContent())
				} else {
					m.submitFollowUp()
				}
				// Start listening for processed output
				if m.outputChan != nil {
					cmds = append(cmds, waitForInceptionOutput(m.outputChan))
				}
			}

		case "backspace":
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}

		case "ctrl+f", "pgdown":
			m.following = false
			m.viewport.ViewDown()

		case "ctrl+b", "pgup":
			m.following = false
			m.viewport.ViewUp()

		case "ctrl+g":
			m.viewport.GotoTop()
			m.following = false

		case "ctrl+e", "F":
			m.viewport.GotoBottom()
			m.following = true

		default:
			// Handle pasted text (msg.Runes contains all pasted characters)
			if len(msg.Runes) > 0 {
				for _, r := range msg.Runes {
					if r >= 32 && r <= 126 {
						m.inputBuffer += string(r)
					}
				}
			} else if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
				m.inputBuffer += key
			} else if key == "space" {
				m.inputBuffer += " "
			}
		}

	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 5
		verticalMargins := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargins)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.renderContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargins
		}
		m.width = msg.Width
		m.height = msg.Height
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func waitForInceptionOutput(ch chan string) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return inceptionDoneMsg{}
		}
		// Check if this is an injection (sub-query response footer)
		if strings.HasPrefix(chunk, "\n───── END EXPERT") {
			return inceptionInjectionMsg(chunk)
		}
		// Check if this is a sub-query start marker
		if strings.HasPrefix(chunk, "⚡START⚡") {
			model := strings.TrimPrefix(chunk, "⚡START⚡")
			return inceptionSubQueryStartMsg{query: inception.SubQuery{Model: model}}
		}
		// Check if this is a streaming sub-query chunk
		if strings.HasPrefix(chunk, "⚡") {
			return inceptionSubQueryChunkMsg(strings.TrimPrefix(chunk, "⚡"))
		}
		return inceptionChunkMsg(chunk)
	}
}

func (m *inceptionModel) startInception() {
	prompt := m.inputBuffer
	m.inputBuffer = ""
	m.promptMode = false

	// Add inception system prompt to teach the model about sub-queries
	systemPrompt := inception.InceptionSystemPrompt

	// Add to history with system context
	m.history = append(m.history, ChatMessage{Role: "system", Content: systemPrompt})
	m.history = append(m.history, ChatMessage{Role: "user", Content: prompt})

	// Add to display
	m.content = "═══════════════════════════════════════════════════════════════════\n"
	m.content += "🌀 INCEPTION MODE\n"
	m.content += "═══════════════════════════════════════════════════════════════════\n\n"
	m.content += "USER PROMPT:\n" + prompt + "\n\n"
	m.content += "═══════════════════════════════════════════════════════════════════\n"
	m.content += fmt.Sprintf("ASSISTANT [%s] (inception-enabled)\n", m.modelName)
	m.content += "═══════════════════════════════════════════════════════════════════\n\n"

	// Start streaming with inception processing
	m.streaming = true
	m.following = true

	// Create raw stream channel from Ollama
	rawChan := make(chan string, 100)

	// Create inception processor
	ctx := context.Background()
	processor := inception.NewStreamProcessor(ctx, m.handler, rawChan)
	m.outputChan = make(chan string, 100)

	// Set up callbacks for streaming sub-query
	m.handler.OnSubQueryStart = func(q inception.SubQuery) {
		// Send header with special prefix
		m.outputChan <- fmt.Sprintf("⚡START⚡%s", q.Model)
	}
	m.handler.OnSubQueryChunk = func(chunk string) {
		// Send chunk with prefix
		m.outputChan <- "⚡" + chunk
	}

	// Start the processor
	processedChan := processor.Start()

	// Forward processed output to our output channel
	go func() {
		for chunk := range processedChan {
			m.outputChan <- chunk
		}
		close(m.outputChan)
	}()

	// Start Ollama stream
	historyCopy := make([]ChatMessage, len(m.history))
	copy(historyCopy, m.history)
	go streamInceptionChat(m.modelName, historyCopy, rawChan)
}

func (m *inceptionModel) submitFollowUp() {
	userMsg := m.inputBuffer
	m.history = append(m.history, ChatMessage{Role: "user", Content: userMsg})

	m.content += "\n═══════════════════════════════════════════════════════════════════\n"
	m.content += fmt.Sprintf("USER [turn %d]\n", len(m.history))
	m.content += "═══════════════════════════════════════════════════════════════════\n\n"
	m.content += m.inputBuffer + "\n\n"

	m.inputBuffer = ""

	m.content += "═══════════════════════════════════════════════════════════════════\n"
	m.content += fmt.Sprintf("ASSISTANT [%s]\n", m.modelName)
	m.content += "═══════════════════════════════════════════════════════════════════\n\n"

	m.streaming = true
	m.following = true

	// Same inception pipeline
	rawChan := make(chan string, 100)
	ctx := context.Background()
	processor := inception.NewStreamProcessor(ctx, m.handler, rawChan)
	m.outputChan = make(chan string, 100)

	// Set up callbacks for streaming sub-query
	m.handler.OnSubQueryStart = func(q inception.SubQuery) {
		m.outputChan <- fmt.Sprintf("⚡START⚡%s", q.Model)
	}
	m.handler.OnSubQueryChunk = func(chunk string) {
		m.outputChan <- "⚡" + chunk
	}

	processedChan := processor.Start()
	go func() {
		for chunk := range processedChan {
			m.outputChan <- chunk
		}
		close(m.outputChan)
	}()

	historyCopy := make([]ChatMessage, len(m.history))
	copy(historyCopy, m.history)
	go streamInceptionChat(m.modelName, historyCopy, rawChan)

	m.viewport.SetContent(m.renderContent())
	m.viewport.GotoBottom()
}

// streamInceptionChat streams from Ollama to the raw channel
func streamInceptionChat(model string, history []ChatMessage, ch chan string) {
	defer close(ch)

	messages := make([]map[string]string, len(history))
	for i, msg := range history {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true,
		"options": map[string]interface{}{
			"num_predict": 2000,
			"temperature": 0.7,
		},
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		ch <- fmt.Sprintf("\nError: %v\n", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if chunk.Message.Content != "" {
			ch <- chunk.Message.Content
		}
		if chunk.Done {
			break
		}
	}
}

// wrapText wraps a line to the specified width, preserving words
func wrapText(text string, width int) []string {
	if width <= 0 || len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var currentLine strings.Builder
	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= width {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}
	return lines
}

func (m inceptionModel) renderContent() string {
	var sb strings.Builder

	contentWidth := 80
	if m.width > 20 {
		contentWidth = m.width - 10
	}
	if contentWidth < 40 {
		contentWidth = 40
	}
	leftPad := ""
	if m.width > contentWidth+10 {
		leftPad = strings.Repeat(" ", (m.width-contentWidth)/2)
	}

	lines := strings.Split(m.content, "\n")
	for _, line := range lines {
		// Don't wrap separator lines or empty lines
		if strings.HasPrefix(line, "═══") || strings.HasPrefix(line, "───") || line == "" {
			sb.WriteString(leftPad)
			sb.WriteString(swTurnStyle.Render(line))
			sb.WriteString("\n")
			continue
		}

		// Wrap long lines
		wrappedLines := wrapText(line, contentWidth)
		for _, wrappedLine := range wrappedLines {
			// Highlight inception markers
			if strings.Contains(wrappedLine, "🌀") || strings.Contains(wrappedLine, "SUB-QUERY") {
				sb.WriteString(leftPad)
				sb.WriteString(subQueryStyle.Render(wrappedLine))
				sb.WriteString("\n")
			} else if strings.Contains(wrappedLine, "⏳") {
				sb.WriteString(leftPad)
				sb.WriteString(subQueryActiveStyle.Render(wrappedLine))
				sb.WriteString("\n")
			} else {
				sb.WriteString(leftPad)
				sb.WriteString(wrappedLine)
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

func (m inceptionModel) renderPromptMode() string {
	var sb strings.Builder

	boxWidth := 70
	if m.width < boxWidth+4 {
		boxWidth = m.width - 4
	}

	leftPad := (m.width - boxWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	padding := strings.Repeat(" ", leftPad)

	topPad := (m.height - 16) / 2
	if topPad < 0 {
		topPad = 0
	}
	sb.WriteString(strings.Repeat("\n", topPad))

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF00FF"))

	sb.WriteString(padding)
	sb.WriteString(headerStyle.Render("🌀 LLM INCEPTION"))
	sb.WriteString("\n")
	sb.WriteString(padding)
	sb.WriteString(strings.Repeat("─", boxWidth))
	sb.WriteString("\n\n")

	// Info
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	sb.WriteString(padding)
	sb.WriteString(infoStyle.Render(fmt.Sprintf("Main model: %s", m.modelName)))
	sb.WriteString("\n")
	sb.WriteString(padding)
	sb.WriteString(infoStyle.Render("Experts: science, math, code, creative, reason"))
	sb.WriteString("\n\n")

	// Instructions
	sb.WriteString(padding)
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Render("Enter a prompt that might benefit from expert knowledge:"))
	sb.WriteString("\n")
	sb.WriteString(padding)
	sb.WriteString(infoStyle.Render("Example: 'Write code to calculate ISS orbital position'"))
	sb.WriteString("\n\n")

	// Input (word wrap to box width)
	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Background(lipgloss.Color("#1a1a2e")).
		Padding(0, 1)

	maxInputLen := boxWidth - 4 // Leave room for padding and cursor
	if maxInputLen < 20 {
		maxInputLen = 20
	}

	// Wrap input for display
	var inputLines []string
	if m.inputBuffer == "" {
		inputLines = []string{""}
	} else {
		inputLines = wrapText(m.inputBuffer, maxInputLen)
	}

	for _, line := range inputLines {
		sb.WriteString(padding)
		sb.WriteString(inputStyle.Render(line))
		sb.WriteString("\n")
	}
	// Add cursor on last line
	sb.WriteString(padding)
	sb.WriteString(inputStyle.Render("█"))
	sb.WriteString("\n\n")

	// Help
	sb.WriteString(padding)
	sb.WriteString(strings.Repeat("─", boxWidth))
	sb.WriteString("\n")
	sb.WriteString(padding)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	sb.WriteString(helpStyle.Render("[Enter] start inception  [Esc] quit"))
	sb.WriteString("\n")

	return sb.String()
}

func (m inceptionModel) View() string {
	if !m.ready {
		return "\n  Initializing Inception..."
	}

	if m.promptMode {
		return m.renderPromptMode()
	}

	// Header
	title := inceptionHeaderStyle.Render("🌀 INCEPTION")

	modelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	modelInfo := modelStyle.Render(fmt.Sprintf(" [%s]", m.modelName))

	statusParts := []string{}
	if m.streaming {
		if m.subQueryActive {
			// Show spinner while waiting for sub-query
			statusParts = append(statusParts, subQueryActiveStyle.Render(fmt.Sprintf(" %s SUB-QUERY [%s]", m.spinner.View(), m.subQueryModel)))
		} else {
			// Show spinner while streaming
			statusParts = append(statusParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(fmt.Sprintf(" %s STREAMING", m.spinner.View())))
		}
	}
	if m.following {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(" [Following ▼]"))
	}
	if m.subQueryCount > 0 {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render(fmt.Sprintf(" 🌀×%d", m.subQueryCount)))
	}

	header := fmt.Sprintf("%s%s%s\n%s\n",
		title, modelInfo, strings.Join(statusParts, ""),
		strings.Repeat("─", m.width))

	// Footer
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true)

	// Calculate max input width (leave room for prompt emoji and cursor)
	maxInputWidth := m.width - 6
	if maxInputWidth < 20 {
		maxInputWidth = 20
	}

	// Word wrap input for display
	var inputLines []string
	if m.inputBuffer == "" {
		inputLines = []string{lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Type here...")}
	} else {
		inputLines = wrapText(m.inputBuffer, maxInputWidth)
	}

	var helpText string
	if m.streaming {
		if m.subQueryActive {
			helpText = "⏳ Waiting for sub-query response..."
		} else {
			helpText = "[F]ollow tail  Sub-queries auto-detected"
		}
	} else if m.inputBuffer != "" {
		helpText = "[enter]send [esc]clear"
	} else {
		helpText = "Ask follow-up questions  [F]ollow [esc]quit"
	}

	// Build multi-line input display
	var inputDisplay strings.Builder
	for i, line := range inputLines {
		if i == 0 {
			inputDisplay.WriteString(promptStyle.Render("🌀 "))
			inputDisplay.WriteString(inputStyle.Render(line))
		} else {
			inputDisplay.WriteString("\n   ") // Indent continuation lines
			inputDisplay.WriteString(inputStyle.Render(line))
		}
	}
	inputDisplay.WriteString("█") // Cursor at end

	footer := fmt.Sprintf("\n%s\n%s\n%s",
		strings.Repeat("─", m.width),
		inputDisplay.String(),
		swHelpStyle.Render(helpText))

	return header + m.viewport.View() + footer
}
