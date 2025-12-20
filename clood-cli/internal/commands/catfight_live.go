package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// StreamingCat represents a cat in the live arena
type StreamingCat struct {
	Name      string
	Model     string
	Buffer    strings.Builder
	Status    string // "streaming", "done", "error"
	StartTime time.Time
	EndTime   time.Time
	Tokens    int
}

// Message types for the TUI
type catChunkMsg struct {
	catIndex int
	chunk    string
}
type catDoneMsg struct {
	catIndex int
}
type catErrorMsg struct {
	catIndex int
	err      error
}
type arenaTickMsg time.Time
type quitTimerMsg struct{}
type battleSummary struct {
	winner      string
	winnerModel string
	winnerTime  float64
	totalTime   float64
	totalTokens int
}

// liveArenaModel is the BubbleTea model for streaming catfight
type liveArenaModel struct {
	viewport     viewport.Model
	cats         []StreamingCat
	prompt       string
	width        int
	height       int
	ready        bool
	allDone      bool
	showingSummary bool
	summary      *battleSummary
	startTime    time.Time
	ollamaURL    string
	layout       string // "interleaved" or "split"
	following    bool
	streamChans  []chan string
	mu           sync.Mutex
}

// Styles for live arena
var (
	catNameStyles = []lipgloss.Style{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6B6B")), // Persian - red
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4ECDC4")), // Tabby - teal
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFE66D")), // Siamese - yellow
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#95E1D3")), // Extra - mint
	}
	streamingIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Blink(true)
	doneIndicator      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	separatorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

func CatfightLiveCmd() *cobra.Command {
	var models string
	var ollamaURL string
	var layout string

	cmd := &cobra.Command{
		Use:   "catfight-live [prompt]",
		Short: "🏟️ LIVE streaming catfight - watch all cats battle in realtime",
		Long: `Release the cats! Watch multiple LLMs stream simultaneously.

Unlike regular catfight which runs sequentially, this shows all cats
generating in parallel with live streaming output. Battle ends with
a winner announcement and auto-exits after 3 seconds.

Interactive mode (default):
  clood catfight-live "Write a hello world in Go"

  Controls:
    q/Esc    Quit early
    f        Toggle auto-follow
    g/G      Go to top/bottom
    Any key  Exit after battle ends

Non-interactive / Piping:
  For scripts, logging, or if you don't want to watch the stream
  (👀 guy peeking through bushes meme), use regular 'catfight':

  # Sequential battle with full results
  clood catfight "your prompt" > battle.txt

  # JSON output for parsing
  clood catfight --json "your prompt" > battle.json

  # Quick single-model test
  clood ask "your prompt" --model qwen2.5-coder:3b

Examples:
  clood catfight-live "Write a hello world in Go"
  clood catfight-live --models "qwen2.5-coder:3b,llama3.1:8b" "Explain recursion"
  clood catfight-live --layout split "Write a haiku about coding"`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			prompt := strings.Join(args, " ")

			// Parse models
			var cats []StreamingCat
			if models != "" {
				for _, m := range strings.Split(models, ",") {
					m = strings.TrimSpace(m)
					cats = append(cats, StreamingCat{
						Name:   modelToName(m),
						Model:  m,
						Status: "streaming",
					})
				}
			} else {
				// Default cats
				cats = []StreamingCat{
					{Name: "Siamese", Model: "qwen2.5-coder:3b", Status: "streaming"},
					{Name: "Caracal", Model: "llama3.1:8b", Status: "streaming"},
				}
			}

			// Create channels for each cat
			streamChans := make([]chan string, len(cats))
			for i := range streamChans {
				streamChans[i] = make(chan string, 100)
			}

			m := liveArenaModel{
				cats:        cats,
				prompt:      prompt,
				ollamaURL:   ollamaURL,
				layout:      layout,
				following:   true,
				startTime:   time.Now(),
				streamChans: streamChans,
			}

			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

			// Start streaming goroutines BEFORE running the program
			for i := range cats {
				go streamCatFight(ollamaURL, cats[i].Model, prompt, i, p)
			}

			if _, err := p.Run(); err != nil {
				fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
			}
		},
	}

	cmd.Flags().StringVarP(&models, "models", "m", "", "Comma-separated models (default: qwen2.5-coder:3b,llama3.1:8b)")
	cmd.Flags().StringVar(&ollamaURL, "url", "http://localhost:11434", "Ollama URL")
	cmd.Flags().StringVar(&layout, "layout", "interleaved", "Display layout: interleaved or split")

	return cmd
}

func (m liveArenaModel) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return arenaTickMsg(t)
		}),
	)
}

func (m liveArenaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case catChunkMsg:
		m.mu.Lock()
		if msg.catIndex < len(m.cats) {
			m.cats[msg.catIndex].Buffer.WriteString(msg.chunk)
			m.cats[msg.catIndex].Tokens++
		}
		m.mu.Unlock()
		m.viewport.SetContent(m.renderContent())
		if m.following {
			m.viewport.GotoBottom()
		}

	case catDoneMsg:
		m.mu.Lock()
		if msg.catIndex < len(m.cats) {
			m.cats[msg.catIndex].Status = "done"
			m.cats[msg.catIndex].EndTime = time.Now()
		}
		// Check if all done
		allDone := true
		for _, cat := range m.cats {
			if cat.Status == "streaming" {
				allDone = false
				break
			}
		}
		if allDone && !m.allDone {
			m.allDone = true
			m.showingSummary = true
			m.summary = m.computeSummary()
			// Start quit timer - auto quit after 3 seconds
			cmds = append(cmds, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return quitTimerMsg{}
			}))
		}
		m.mu.Unlock()
		m.viewport.SetContent(m.renderContent())

	case quitTimerMsg:
		// Auto-quit after battle ends
		return m, tea.Quit

	case catErrorMsg:
		m.mu.Lock()
		if msg.catIndex < len(m.cats) {
			m.cats[msg.catIndex].Status = "error"
			m.cats[msg.catIndex].Buffer.WriteString(fmt.Sprintf("\n[ERROR: %v]", msg.err))
		}
		m.mu.Unlock()
		m.viewport.SetContent(m.renderContent())

	case arenaTickMsg:
		// Refresh display periodically
		m.viewport.SetContent(m.renderContent())
		if !m.allDone {
			cmds = append(cmds, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return arenaTickMsg(t)
			}))
		}

	case tea.KeyMsg:
		// If battle is done, any key quits
		if m.showingSummary {
			return m, tea.Quit
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "f", "F":
			m.following = !m.following
			if m.following {
				m.viewport.GotoBottom()
			}
		case "g":
			m.viewport.GotoTop()
			m.following = false
		case "G":
			m.viewport.GotoBottom()
			m.following = true
		}

	case tea.WindowSizeMsg:
		headerHeight := 4
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

func (m liveArenaModel) renderContent() string {
	var sb strings.Builder

	m.mu.Lock()
	defer m.mu.Unlock()

	// Use terminal width for separators, default to 70 if not set
	sepWidth := m.width
	if sepWidth < 40 {
		sepWidth = 70
	}

	doubleLine := strings.Repeat("═", sepWidth)
	singleLine := strings.Repeat("─", sepWidth)

	// Header with prompt
	sb.WriteString(separatorStyle.Render(doubleLine) + "\n")
	sb.WriteString(fmt.Sprintf("PROMPT: %s\n", m.prompt))
	sb.WriteString(separatorStyle.Render(doubleLine) + "\n\n")

	for i, cat := range m.cats {
		style := catNameStyles[i%len(catNameStyles)]

		// Status indicator
		var status string
		switch cat.Status {
		case "streaming":
			status = streamingIndicator.Render("● STREAMING")
		case "done":
			duration := cat.EndTime.Sub(m.startTime).Seconds()
			tokSec := float64(cat.Tokens) / duration
			status = doneIndicator.Render(fmt.Sprintf("✓ DONE (%.1fs, %d tok, %.0f t/s)", duration, cat.Tokens, tokSec))
		case "error":
			status = tui.ErrorStyle.Render("✗ ERROR")
		}

		sb.WriteString(separatorStyle.Render(singleLine) + "\n")
		sb.WriteString(fmt.Sprintf("%s [%s] %s\n", style.Render("🐱 "+cat.Name), cat.Model, status))
		sb.WriteString(separatorStyle.Render(singleLine) + "\n")

		content := cat.Buffer.String()
		if content == "" && cat.Status == "streaming" {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  waiting for first token...") + "\n")
		} else {
			// Indent the content, being careful with trailing newlines
			content = strings.TrimRight(content, "\n")
			if content != "" {
				lines := strings.Split(content, "\n")
				for _, line := range lines {
					sb.WriteString("  " + line + "\n")
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m liveArenaModel) computeSummary() *battleSummary {
	var winner *StreamingCat
	var fastestTime float64 = 9999
	totalTokens := 0

	for i := range m.cats {
		cat := &m.cats[i]
		if cat.Status == "done" {
			duration := cat.EndTime.Sub(m.startTime).Seconds()
			totalTokens += cat.Tokens
			if duration < fastestTime {
				fastestTime = duration
				winner = cat
			}
		}
	}

	if winner == nil {
		return nil
	}

	return &battleSummary{
		winner:      winner.Name,
		winnerModel: winner.Model,
		winnerTime:  fastestTime,
		totalTime:   time.Since(m.startTime).Seconds(),
		totalTokens: totalTokens,
	}
}

func (m liveArenaModel) View() string {
	if !m.ready {
		return "\n  Initializing arena..."
	}

	// Header
	elapsed := time.Since(m.startTime).Seconds()
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")).Render("🏟️ KITCHEN STADIUM - LIVE BATTLE")

	streaming := 0
	done := 0
	for _, cat := range m.cats {
		if cat.Status == "streaming" {
			streaming++
		} else if cat.Status == "done" {
			done++
		}
	}

	status := fmt.Sprintf(" [%d streaming, %d done] %.1fs", streaming, done, elapsed)
	if m.allDone {
		status = doneIndicator.Render(" BATTLE COMPLETE!") + fmt.Sprintf(" %.1fs", elapsed)
	}

	followStatus := ""
	if m.following && !m.allDone {
		followStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(" [Following ▼]")
	}

	header := fmt.Sprintf("%s%s%s\n%s\n",
		title, status, followStatus,
		strings.Repeat("─", m.width))

	// Footer
	var footer string
	if m.showingSummary && m.summary != nil {
		// Show battle summary
		summaryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700"))
		winnerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF00"))
		mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

		footer = fmt.Sprintf("\n%s\n%s\n%s\n%s",
			strings.Repeat("═", m.width),
			summaryStyle.Render(fmt.Sprintf("🏆 WINNER: %s", winnerStyle.Render(m.summary.winner+" ("+m.summary.winnerModel+")"))),
			mutedStyle.Render(fmt.Sprintf("   Fastest: %.1fs | Total tokens: %d | Battle duration: %.1fs",
				m.summary.winnerTime, m.summary.totalTokens, m.summary.totalTime)),
			mutedStyle.Render("   Exiting in 3s... (press any key to exit now)"),
		)
	} else {
		footer = fmt.Sprintf("\n%s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("[q]uit [f]ollow [g]top [G]bottom"))
	}

	return header + m.viewport.View() + footer
}

// streamCatFight streams from Ollama and sends messages to the TUI
func streamCatFight(baseURL, model, prompt string, catIndex int, p *tea.Program) {
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

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Post(baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		p.Send(catErrorMsg{catIndex: catIndex, err: err})
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
			p.Send(catChunkMsg{catIndex: catIndex, chunk: chunk.Response})
		}
		if chunk.Done {
			break
		}
	}

	p.Send(catDoneMsg{catIndex: catIndex})
}
