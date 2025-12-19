package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Question represents a detected question in the AI response
type Question struct {
	Index    int
	Text     string // The question text
	Context  string // Surrounding explanation
	Response string // User's answer (empty until answered)
	State    string // awaiting, answered, skipped, ignored, avoided
	Line     int    // Line number where this question starts
}

// Message types for streaming
type streamChunkMsg string
type streamDoneMsg struct{}
type streamErrorMsg error

// snakewayModel is the bubbletea model for Snake Way prototype
type snakewayModel struct {
	viewport    viewport.Model
	content     string
	questions   []Question
	currentQ    int
	width       int
	height      int
	ready       bool
	turn        int    // Current conversation turn
	streaming   bool   // Whether we're in streaming mode
	following   bool   // Auto-follow new content
	streamChan  chan string
	modelName   string // Model being used for generation
	inputBuffer string // Always-visible input buffer at bottom
}

// Styles
var (
	swHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1)

	swQuestionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00BFFF"))

	swQuestionActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00FF00")).
				Background(lipgloss.Color("#333333"))

	swStateAwaitingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	swStateAnsweredStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))

	swHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	swTurnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B"))
)

func SnakewayProtoCmd() *cobra.Command {
	var live bool
	var stream bool
	var model string

	cmd := &cobra.Command{
		Use:   "snakeway-proto",
		Short: "Snake Way prototype - viewport navigation test",
		Long: `Phase 1 prototype for Snake Way: just the scrolling part.

Tests:
- Viewport with conversation content
- Section navigation (questions)
- j/k scroll, n/p next/prev question, 1-9 jump

Modes:
- Default: Static test content (instant)
- --live: Generate from ollama (waits for completion)
- --stream: REALTIME streaming (scroll while generating!)

No input zones yet - just navigation.`,
		Run: func(cmd *cobra.Command, args []string) {
			var content string
			var questions []Question
			var streamChan chan string

			if stream {
				// Streaming mode - start with minimal content, update live
				content = "═══════════════════════════════════════════════════════════════════\n"
				content += "STREAMING MODE - Content loading...\n"
				content += "═══════════════════════════════════════════════════════════════════\n\n"
				streamChan = make(chan string, 100)

				// Start streaming in background
				go streamFromOllama(model, streamChan)
			} else if live {
				// Generate real content from ollama (blocking)
				fmt.Println(tui.MutedStyle.Render("Generating live content from ollama..."))
				content, questions = generateLiveConversation(model)
			} else {
				// Use fake test content
				content, questions = generateTestConversation()
			}

			m := snakewayModel{
				content:    content,
				questions:  questions,
				currentQ:   0,
				turn:       3,
				streaming:  stream,
				following:  stream, // Auto-follow in stream mode
				streamChan: streamChan,
				modelName:  model,
			}

			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
			}
		},
	}

	cmd.Flags().BoolVar(&live, "live", false, "Generate live content from ollama (blocking)")
	cmd.Flags().BoolVar(&stream, "stream", false, "REALTIME streaming - scroll while generating")
	cmd.Flags().StringVar(&model, "model", "qwen2.5-coder:3b", "Model to use for generation")

	return cmd
}

func (m snakewayModel) Init() tea.Cmd {
	if m.streaming && m.streamChan != nil {
		return waitForStream(m.streamChan)
	}
	return nil
}

// waitForStream returns a command that waits for the next stream chunk
func waitForStream(ch chan string) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		return streamChunkMsg(chunk)
	}
}

func (m snakewayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case streamChunkMsg:
		// Append new content from stream
		m.content += string(msg)
		// Re-wrap content for display
		m.viewport.SetContent(m.renderContent())
		if m.following {
			m.viewport.GotoBottom()
		}
		// Keep listening for more chunks
		if m.streamChan != nil {
			cmds = append(cmds, waitForStream(m.streamChan))
		}

	case streamDoneMsg:
		m.streaming = false
		m.content += "\n───────────────────────────────────────────────────────────────────\n"
		m.content += "STREAMING COMPLETE\n"
		m.content += "───────────────────────────────────────────────────────────────────\n"
		m.viewport.SetContent(m.renderContent())
		// Re-detect questions now that content is complete
		m.questions = detectQuestions(m.content)

	case streamErrorMsg:
		m.content += fmt.Sprintf("\nERROR: %v\n", msg)
		m.viewport.SetContent(m.renderContent())

	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// Clear input or quit
			if m.inputBuffer != "" {
				m.inputBuffer = ""
			} else {
				return m, tea.Quit
			}

		case "enter":
			// Submit response if we have input and not streaming
			if m.inputBuffer != "" && !m.streaming {
				m.submitResponse()
				// Start listening for stream
				if m.streamChan != nil {
					cmds = append(cmds, waitForStream(m.streamChan))
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

		case "ctrl+e":
			m.viewport.GotoBottom()
			m.following = true

		case "up", "down":
			m.following = false
			// Let viewport handle scroll

		default:
			// If it's a printable character, add to input buffer
			if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
				m.inputBuffer += key
			} else if key == "space" {
				m.inputBuffer += " "
			}
		}

	case tea.MouseMsg:
		// Mouse scroll disables auto-follow
		if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
			m.following = false
		}

	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 2
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

func (m *snakewayModel) gotoQuestion(idx int) {
	if idx >= 0 && idx < len(m.questions) {
		m.viewport.SetYOffset(m.questions[idx].Line)
	}
}

func (m *snakewayModel) submitResponse() {
	if m.currentQ >= len(m.questions) {
		return
	}

	// Mark question as answered
	m.questions[m.currentQ].Response = m.inputBuffer
	m.questions[m.currentQ].State = "answered"

	// Add user response to content
	m.content += "\n═══════════════════════════════════════════════════════════════════\n"
	m.content += fmt.Sprintf("USER RESPONSE TO Q%d\n", m.currentQ+1)
	m.content += "═══════════════════════════════════════════════════════════════════\n\n"
	m.content += m.inputBuffer + "\n\n"

	// Clear input buffer
	response := m.inputBuffer
	m.inputBuffer = ""

	// Start streaming follow-up
	m.content += "═══════════════════════════════════════════════════════════════════\n"
	m.content += fmt.Sprintf("FOLLOW-UP FROM [%s]\n", m.modelName)
	m.content += "═══════════════════════════════════════════════════════════════════\n\n"

	m.streaming = true
	m.following = true
	m.streamChan = make(chan string, 100)

	// Start follow-up stream in background
	question := m.questions[m.currentQ].Text
	go streamFollowUp(m.modelName, question, response, m.streamChan)

	m.viewport.SetContent(m.renderContent())
	m.viewport.GotoBottom()
}

// streamFollowUp sends the user's response and streams the LLM's follow-up
func streamFollowUp(model, question, userResponse string, ch chan string) {
	defer close(ch)

	prompt := fmt.Sprintf(`The user was asked: "%s"

They responded: "%s"

Please acknowledge their response and ask a relevant follow-up question to continue the conversation. Be conversational and helpful. Include at least one follow-up question (ending with ?).`, question, userResponse)

	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": true,
		"options": map[string]interface{}{
			"num_predict": 500,
			"temperature": 0.7,
		},
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		ch <- fmt.Sprintf("\nError: %v\n", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if chunk.Response != "" {
			ch <- chunk.Response
		}
		if chunk.Done {
			break
		}
	}
}

func (m snakewayModel) renderContent() string {
	var sb strings.Builder

	// Calculate content width and padding
	contentWidth := 70
	if m.width > 20 && m.width < contentWidth+20 {
		contentWidth = m.width - 10
	}
	leftPad := ""
	if m.width > contentWidth+10 {
		leftPad = strings.Repeat(" ", (m.width-contentWidth)/2)
	}

	// First, wrap the raw content to our content width
	// Split by existing newlines, wrap each paragraph
	paragraphs := strings.Split(m.content, "\n")
	var wrappedLines []string
	for _, para := range paragraphs {
		// Don't wrap formatting lines
		if strings.HasPrefix(para, "═══") || strings.HasPrefix(para, "───") ||
			strings.HasPrefix(para, "TURN") || strings.HasPrefix(para, "STREAMING") ||
			strings.HasPrefix(para, "FLYING") || strings.HasPrefix(para, "ADDITIONAL") ||
			strings.HasPrefix(para, "END OF") || para == "" {
			wrappedLines = append(wrappedLines, para)
		} else {
			// Wrap this paragraph
			wrapped := wordWrap(para, contentWidth)
			for _, wl := range strings.Split(wrapped, "\n") {
				wrappedLines = append(wrappedLines, wl)
			}
		}
	}

	for i, line := range wrappedLines {
		// Check if this line is a question marker
		isQuestion := false
		questionIdx := -1
		for qi, q := range m.questions {
			if q.Line == i {
				isQuestion = true
				questionIdx = qi
				break
			}
		}

		if isQuestion {
			// Render question with state indicator
			state := "○" // awaiting
			stateStyle := swStateAwaitingStyle
			if m.questions[questionIdx].State == "answered" {
				state = "●"
				stateStyle = swStateAnsweredStyle
			}

			if questionIdx == m.currentQ {
				sb.WriteString(leftPad)
				sb.WriteString(swQuestionActiveStyle.Render(fmt.Sprintf("► Q%d %s ", questionIdx+1, state)))
				sb.WriteString(swQuestionActiveStyle.Render(line))
			} else {
				sb.WriteString(leftPad)
				sb.WriteString(swQuestionStyle.Render(fmt.Sprintf("  Q%d ", questionIdx+1)))
				sb.WriteString(stateStyle.Render(state + " "))
				sb.WriteString(line)
			}
			sb.WriteString("\n")
		} else if strings.HasPrefix(line, "═══") || strings.HasPrefix(line, "───") {
			sb.WriteString(leftPad)
			sb.WriteString(swTurnStyle.Render(line))
			sb.WriteString("\n")
		} else if strings.HasPrefix(line, "TURN") || strings.HasPrefix(line, "STREAMING") ||
			strings.HasPrefix(line, "FLYING") || strings.HasPrefix(line, "ADDITIONAL") ||
			strings.HasPrefix(line, "END OF") {
			sb.WriteString(leftPad)
			sb.WriteString(swTurnStyle.Render(line))
			sb.WriteString("\n")
		} else {
			sb.WriteString(leftPad)
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (m snakewayModel) View() string {
	if !m.ready {
		return "\n  Initializing Snake Way..."
	}

	// Header
	title := swHeaderStyle.Render("🐍 SNAKE WAY")

	// Model indicator
	modelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	modelInfo := modelStyle.Render(fmt.Sprintf(" [%s]", m.modelName))

	// Status indicators
	statusParts := []string{}
	if m.streaming {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(" ● STREAMING"))
	}
	if m.following {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(" [Following ▼]"))
	}
	if len(m.questions) > 0 {
		statusParts = append(statusParts, swQuestionStyle.Render(fmt.Sprintf(" Q%d/%d", m.currentQ+1, len(m.questions))))
	}

	header := fmt.Sprintf("%s%s%s\n%s\n",
		title, modelInfo, strings.Join(statusParts, ""),
		strings.Repeat("─", m.width))

	// Footer - input field appears when typing, otherwise help text
	var footer string
	if m.inputBuffer != "" {
		// Show input field when user is typing
		inputStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1)
		promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))

		footer = fmt.Sprintf("\n%s %s█  %s",
			promptStyle.Render("→"),
			inputStyle.Render(m.inputBuffer),
			swHelpStyle.Render("[enter]send [esc]clear"))
	} else {
		help := "[esc]quit [pgup/dn]scroll [ctrl+g/e]top/btm  Just start typing to respond..."
		if m.streaming {
			help = "[esc]quit [scroll while streaming!] Type anytime..."
		}
		helpStyled := swHelpStyle.Render(help)
		scrollPct := swHelpStyle.Render(fmt.Sprintf(" %d%%", int(m.viewport.ScrollPercent()*100)))
		footer = fmt.Sprintf("\n%s%s", helpStyled, scrollPct)
	}

	return header + m.viewport.View() + footer
}

// generateTestConversation creates fake conversation data with questions
func generateTestConversation() (string, []Question) {
	var sb strings.Builder
	var questions []Question
	lineNum := 0

	// Turn 1: User prompt
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("TURN 1 - User\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("\n")
	sb.WriteString("I want to build a REST API for a task management system.\n")
	sb.WriteString("It should support multiple users and have authentication.\n")
	sb.WriteString("\n")
	lineNum = 7

	// Turn 2: AI response with questions (Flying Cats style chaos)
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("TURN 2 - Assistant\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("\n")
	sb.WriteString("I'd be happy to help you build a task management REST API! Before we\n")
	sb.WriteString("dive into the implementation, I have some clarifying questions:\n")
	sb.WriteString("\n")
	lineNum += 7

	// Question 1
	questions = append(questions, Question{
		Index: 0,
		Text:  "What authentication method do you prefer?",
		State: "awaiting",
		Line:  lineNum,
	})
	sb.WriteString("What authentication method do you prefer?\n")
	sb.WriteString("   • JWT tokens (stateless, scalable)\n")
	sb.WriteString("   • Session-based auth (traditional, server-side)\n")
	sb.WriteString("   • OAuth2 (for third-party integration)\n")
	sb.WriteString("\n")
	lineNum += 5

	// Question 2
	questions = append(questions, Question{
		Index: 1,
		Text:  "Do you need real-time updates for task changes?",
		State: "awaiting",
		Line:  lineNum,
	})
	sb.WriteString("Do you need real-time updates for task changes?\n")
	sb.WriteString("   This would affect whether we use WebSockets or polling.\n")
	sb.WriteString("\n")
	lineNum += 3

	// Question 3
	questions = append(questions, Question{
		Index: 2,
		Text:  "What's your expected scale?",
		State: "awaiting",
		Line:  lineNum,
	})
	sb.WriteString("What's your expected scale?\n")
	sb.WriteString("   • Small team (< 50 users)\n")
	sb.WriteString("   • Medium organization (50-500 users)\n")
	sb.WriteString("   • Enterprise (500+ users)\n")
	sb.WriteString("\n")
	lineNum += 5

	// Some rambling context (the "hot garbage" / flying cats content)
	sb.WriteString("───────────────────────────────────────────────────────────────────\n")
	sb.WriteString("ADDITIONAL CONTEXT\n")
	sb.WriteString("───────────────────────────────────────────────────────────────────\n")
	sb.WriteString("\n")
	sb.WriteString("Based on common patterns, here's what I'm thinking:\n")
	sb.WriteString("\n")
	sb.WriteString("For a task management system, you'll typically want:\n")
	sb.WriteString("- Users table with roles (admin, manager, member)\n")
	sb.WriteString("- Tasks table with status, priority, assignee\n")
	sb.WriteString("- Projects/Boards to organize tasks\n")
	sb.WriteString("- Comments and activity log\n")
	sb.WriteString("\n")
	sb.WriteString("The authentication choice significantly impacts your architecture.\n")
	sb.WriteString("JWT is great for microservices, session-based is simpler for monoliths.\n")
	sb.WriteString("\n")
	sb.WriteString("Speaking of flying cats, did you know that the terminal velocity of a\n")
	sb.WriteString("housecat is about 60 mph? This is completely irrelevant to your API\n")
	sb.WriteString("but demonstrates the kind of 'hot garbage' that LLMs sometimes produce.\n")
	sb.WriteString("The key is having good navigation so you can skip past it efficiently.\n")
	sb.WriteString("\n")
	lineNum += 18

	// Turn 3: More questions
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("TURN 3 - Assistant (Follow-up)\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("\n")
	sb.WriteString("A few more questions to refine the approach:\n")
	sb.WriteString("\n")
	lineNum += 6

	// Question 4
	questions = append(questions, Question{
		Index: 3,
		Text:  "Do you need file attachments on tasks?",
		State: "awaiting",
		Line:  lineNum,
	})
	sb.WriteString("Do you need file attachments on tasks?\n")
	sb.WriteString("   This affects storage architecture (S3, local, etc.)\n")
	sb.WriteString("\n")
	lineNum += 3

	// Question 5
	questions = append(questions, Question{
		Index: 4,
		Text:  "Should tasks have due dates and reminders?",
		State: "awaiting",
		Line:  lineNum,
	})
	sb.WriteString("Should tasks have due dates and reminders?\n")
	sb.WriteString("   If yes, we'll need a notification system.\n")
	sb.WriteString("\n")
	lineNum += 3

	// Question 6
	questions = append(questions, Question{
		Index: 5,
		Text:  "What database are you planning to use?",
		State: "awaiting",
		Line:  lineNum,
	})
	sb.WriteString("What database are you planning to use?\n")
	sb.WriteString("   • PostgreSQL (recommended for relational data)\n")
	sb.WriteString("   • MongoDB (if you need flexible schemas)\n")
	sb.WriteString("   • SQLite (for simpler deployments)\n")
	sb.WriteString("\n")
	lineNum += 5

	// More rambling
	sb.WriteString("───────────────────────────────────────────────────────────────────\n")
	sb.WriteString("FLYING CATS DIGRESSION\n")
	sb.WriteString("───────────────────────────────────────────────────────────────────\n")
	sb.WriteString("\n")
	sb.WriteString("While we wait for your answers, let me tell you about the time I\n")
	sb.WriteString("tried to implement a task manager using only regex and hopes. It\n")
	sb.WriteString("did not go well. The tasks escaped. They formed a union. They\n")
	sb.WriteString("demanded better working conditions. I had to negotiate with a\n")
	sb.WriteString("TODO item named Gerald who insisted on being called 'High Priority\n")
	sb.WriteString("Gerald' even though he was clearly just 'check email' in disguise.\n")
	sb.WriteString("\n")
	sb.WriteString("This is the kind of content you want to be able to quickly scroll\n")
	sb.WriteString("past. Hence, Snake Way navigation. Press 'n' to jump to the next\n")
	sb.WriteString("question, or use '1-6' to jump directly.\n")
	sb.WriteString("\n")
	sb.WriteString("───────────────────────────────────────────────────────────────────\n")
	sb.WriteString("END OF CONVERSATION\n")
	sb.WriteString("───────────────────────────────────────────────────────────────────\n")

	return sb.String(), questions
}

// generateLiveConversation calls ollama to generate real content
func generateLiveConversation(model string) (string, []Question) {
	var sb strings.Builder
	var questions []Question
	lineNum := 0

	prompts := []struct {
		role   string
		prompt string
	}{
		{"User", `Write a very long, rambling response about flying cats. Include:
- A detailed history of cats learning to fly (make it up, be creative)
- Technical specifications for cat aviation gear
- At least 5 numbered questions about cat flight patterns
- Random tangents about unrelated topics
- Make it at least 500 words. Be verbose. Ramble. Go off on tangents.`},
		{"User", `Continue the flying cats saga. Now discuss:
- The great cat-bird war of 2024
- Regulations for feline airspace
- Ask 5 more questions about cat aviation safety
- Include a recipe for something completely unrelated
- Tell a story about a cat named Gerald who refuses to fly
- Make it LONG. At least 600 words.`},
		{"User", `Final chapter of the flying cats epic:
- The prophecy of the Nimbus Cat
- Snake Way and how cats navigate it
- 5 questions about the future of cat transportation
- A dramatic monologue from a cat pilot
- Random technical jargon mixed with nonsense
- End with a haiku
- BE EXTREMELY VERBOSE. 700+ words.`},
	}

	for i, p := range prompts {
		// User turn
		sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
		sb.WriteString(fmt.Sprintf("TURN %d - %s\n", i*2+1, p.role))
		sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
		sb.WriteString("\n")
		sb.WriteString(p.prompt + "\n")
		sb.WriteString("\n")
		lineNum += 6

		// AI response
		sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
		sb.WriteString(fmt.Sprintf("TURN %d - Assistant\n", i*2+2))
		sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
		sb.WriteString("\n")
		lineNum += 4

		fmt.Printf("  Generating turn %d...\n", i+1)
		response := callOllama(model, p.prompt)

		// Parse response for questions and add to content
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			// Simple question detection: ends with ?
			trimmed := strings.TrimSpace(line)
			if strings.HasSuffix(trimmed, "?") && len(trimmed) > 10 {
				questions = append(questions, Question{
					Index: len(questions),
					Text:  trimmed,
					State: "awaiting",
					Line:  lineNum,
				})
			}
			sb.WriteString(line + "\n")
			lineNum++
		}
		sb.WriteString("\n")
		lineNum++
	}

	sb.WriteString("───────────────────────────────────────────────────────────────────\n")
	sb.WriteString("END OF CONVERSATION\n")
	sb.WriteString("───────────────────────────────────────────────────────────────────\n")

	return sb.String(), questions
}

// callOllama makes a request to ollama API
func callOllama(model, prompt string) string {
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": 2000, // Generate lots of tokens
			"temperature": 0.9,  // More creative/chaotic
		},
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 180 * time.Second} // 3 min for long generations
	resp, err := client.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Sprintf("Error calling ollama: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Sprintf("Error decoding response: %v", err)
	}

	return result.Response
}

// streamFromOllama streams responses from ollama and sends chunks to the channel
func streamFromOllama(model string, ch chan string) {
	defer close(ch)

	prompt := `You are about to embark on an EPIC journey. Write an incredibly long,
rambling stream of consciousness about FLYING CATS. Include:

- The ancient prophecy of the Nimbus Cat
- Technical specifications for feline aviation (make them absurd)
- At least 10 questions scattered throughout (end them with ?)
- Random tangents about completely unrelated topics
- A recipe for something ridiculous
- A dramatic monologue from a cat pilot named Captain Whiskers
- References to Snake Way (the infinite scroll)
- The great cat-bird war and its consequences
- Gerald the cat who refuses to fly
- At least 5 haikus scattered throughout
- Random technical jargon that makes no sense
- A subplot about a mouse rebellion

BE EXTREMELY VERBOSE. Write at least 2000 words. RAMBLE. Go on tangents.
This is a test of streaming, so the more text the better. Don't stop early.
Keep writing until you've covered everything. Then write more.`

	ch <- "═══════════════════════════════════════════════════════════════════\n"
	ch <- fmt.Sprintf("STREAMING FROM OLLAMA [%s]\n", model)
	ch <- "═══════════════════════════════════════════════════════════════════\n\n"

	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": true, // STREAMING MODE
		"options": map[string]interface{}{
			"num_predict": 4000, // LOTS of tokens
			"temperature": 0.95, // Maximum chaos
		},
	}
	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 300 * time.Second} // 5 min timeout
	resp, err := client.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		ch <- fmt.Sprintf("\nError: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read NDJSON stream line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if chunk.Response != "" {
			ch <- chunk.Response
		}
		if chunk.Done {
			break
		}
	}
}

// wordWrap wraps text to a given width
func wordWrap(text string, width int) string {
	if width <= 0 {
		width = 70
	}
	var result strings.Builder
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	lineLen := 0
	for i, word := range words {
		wordLen := len(word)
		if i == 0 {
			result.WriteString(word)
			lineLen = wordLen
		} else if lineLen+1+wordLen > width {
			result.WriteString("\n")
			result.WriteString(word)
			lineLen = wordLen
		} else {
			result.WriteString(" ")
			result.WriteString(word)
			lineLen += 1 + wordLen
		}
	}
	return result.String()
}

// detectQuestions finds questions in content (lines ending with ?)
func detectQuestions(content string) []Question {
	var questions []Question
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, "?") && len(trimmed) > 15 {
			questions = append(questions, Question{
				Index: len(questions),
				Text:  trimmed,
				State: "awaiting",
				Line:  i,
			})
		}
	}
	return questions
}
