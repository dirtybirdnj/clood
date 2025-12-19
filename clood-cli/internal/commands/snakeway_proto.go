package commands

import (
	"fmt"
	"strings"

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

// snakewayModel is the bubbletea model for Snake Way prototype
type snakewayModel struct {
	viewport   viewport.Model
	content    string
	questions  []Question
	currentQ   int
	width      int
	height     int
	ready      bool
	turn       int // Current conversation turn
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
	cmd := &cobra.Command{
		Use:   "snakeway-proto",
		Short: "Snake Way prototype - viewport navigation test",
		Long: `Phase 1 prototype for Snake Way: just the scrolling part.

Tests:
- Viewport with conversation content
- Section navigation (questions)
- j/k scroll, n/p next/prev question, 1-9 jump

No input zones yet - just navigation.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Generate fake conversation with questions
			content, questions := generateTestConversation()

			m := snakewayModel{
				content:   content,
				questions: questions,
				currentQ:  0,
				turn:      3, // We have 3 turns of conversation
			}

			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
			}
		},
	}

	return cmd
}

func (m snakewayModel) Init() tea.Cmd {
	return nil
}

func (m snakewayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "g":
			m.viewport.GotoTop()

		case "G":
			m.viewport.GotoBottom()

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.questions) {
				m.currentQ = idx
				m.gotoQuestion(idx)
			}

		case "n", "tab":
			if m.currentQ < len(m.questions)-1 {
				m.currentQ++
				m.gotoQuestion(m.currentQ)
			}

		case "p", "shift+tab":
			if m.currentQ > 0 {
				m.currentQ--
				m.gotoQuestion(m.currentQ)
			}

		case "enter":
			// Future: enter response mode
			// For now, just flash a message
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
	return m, cmd
}

func (m *snakewayModel) gotoQuestion(idx int) {
	if idx >= 0 && idx < len(m.questions) {
		m.viewport.SetYOffset(m.questions[idx].Line)
	}
}

func (m snakewayModel) renderContent() string {
	var sb strings.Builder
	lines := strings.Split(m.content, "\n")

	for i, line := range lines {
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
				sb.WriteString(swQuestionActiveStyle.Render(fmt.Sprintf("► Q%d %s ", questionIdx+1, state)))
				sb.WriteString(swQuestionActiveStyle.Render(line))
			} else {
				sb.WriteString(swQuestionStyle.Render(fmt.Sprintf("  Q%d ", questionIdx+1)))
				sb.WriteString(stateStyle.Render(state + " "))
				sb.WriteString(line)
			}
			sb.WriteString("\n")
		} else if strings.HasPrefix(line, "═══") || strings.HasPrefix(line, "───") {
			sb.WriteString(swTurnStyle.Render(line))
			sb.WriteString("\n")
		} else if strings.HasPrefix(line, "TURN") {
			sb.WriteString(swTurnStyle.Render(line))
			sb.WriteString("\n")
		} else {
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
	turnInfo := swTurnStyle.Render(fmt.Sprintf(" Turn %d", m.turn))
	questionInfo := swQuestionStyle.Render(fmt.Sprintf(" Q%d/%d", m.currentQ+1, len(m.questions)))

	header := fmt.Sprintf("%s%s%s\n%s\n",
		title, turnInfo, questionInfo,
		strings.Repeat("─", m.width))

	// Footer
	help := swHelpStyle.Render("[q]uit [j/k]scroll [g/G]top/btm [n/p]next/prev [1-9]jump [Enter]respond")
	scrollPct := swHelpStyle.Render(fmt.Sprintf(" %d%%", int(m.viewport.ScrollPercent()*100)))
	footer := fmt.Sprintf("\n%s%s", help, scrollPct)

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
