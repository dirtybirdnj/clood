package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

// Section represents a cat's output section in the scroll
type Section struct {
	Index    int
	Name     string    // e.g., "Persian (deepseek-coder:6.7b)"
	Model    string
	Status   string    // "done", "generating", "waiting"
	Duration string
	Tokens   string
	TokSec   string
	Content  string
	Line     int       // Line number where this section starts
}

// watchModel is the bubbletea model for the watch TUI
type watchModel struct {
	viewport    viewport.Model
	content     string
	sections    []Section
	following   bool      // Auto-scroll to bottom
	width       int
	height      int
	file        string
	lastSize    int64
	ready       bool
	currentSect int       // Currently highlighted section
}

// Styles for the watch TUI
var (
	watchTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		Background(lipgloss.Color("#1a1a2e")).
		Padding(0, 1)

	watchStatusStyle = lipgloss.NewStyle().
		Foreground(tui.ColorAccent)

	watchSectionHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorAccent)

	watchSectionDoneStyle = lipgloss.NewStyle().
		Foreground(tui.ColorSuccess)

	watchSectionActiveStyle = lipgloss.NewStyle().
		Foreground(tui.ColorPrimary).
		Bold(true)

	watchHelpStyle = lipgloss.NewStyle().
		Foreground(tui.ColorMuted)

	watchInfoStyle = lipgloss.NewStyle().
		Foreground(tui.ColorSecondary)
)

// Messages for bubbletea
type tickMsg time.Time
type fileUpdateMsg string

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m watchModel) Init() tea.Cmd {
	return tea.Batch(tickCmd())
}

func (m watchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "f":
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

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.sections) {
				m.currentSect = idx
				m.gotoSection(idx)
				m.following = false
			}

		case "n":
			// Next section
			if m.currentSect < len(m.sections)-1 {
				m.currentSect++
				m.gotoSection(m.currentSect)
				m.following = false
			}

		case "p":
			// Previous section
			if m.currentSect > 0 {
				m.currentSect--
				m.gotoSection(m.currentSect)
				m.following = false
			}

		case "j", "down":
			m.following = false

		case "k", "up":
			m.following = false
		}

	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 2
		verticalMargins := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargins)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargins
		}
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Check for file updates
		if m.file != "" {
			if info, err := os.Stat(m.file); err == nil {
				if info.Size() != m.lastSize {
					m.lastSize = info.Size()
					content, _ := os.ReadFile(m.file)
					m.content = string(content)
					m.sections = parseSections(m.content)
					m.viewport.SetContent(m.renderContent())
					if m.following {
						m.viewport.GotoBottom()
					}
				}
			}
		}
		cmds = append(cmds, tickCmd())
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *watchModel) gotoSection(idx int) {
	if idx >= 0 && idx < len(m.sections) {
		// Calculate approximate line position
		lines := strings.Split(m.content, "\n")
		targetLine := 0
		sectionCount := 0
		sectionPattern := regexp.MustCompile(`^>>> \[\d+/\d+\]|^═══.*═══$|^### `)

		for i, line := range lines {
			if sectionPattern.MatchString(line) || strings.HasPrefix(line, ">>> [") {
				if sectionCount == idx {
					targetLine = i
					break
				}
				sectionCount++
			}
		}
		m.viewport.SetYOffset(targetLine)
	}
}

func (m watchModel) renderContent() string {
	var sb strings.Builder
	lines := strings.Split(m.content, "\n")

	sectionPattern := regexp.MustCompile(`^>>> \[(\d+)/(\d+)\] (\w+) \(([^)]+)\)`)
	donePattern := regexp.MustCompile(`^\s+DONE (\d+\.\d+)s \| (\d+) tokens \| (\d+\.\d+) tok/s`)
	headerPattern := regexp.MustCompile(`^═+.*═+$|^-{40,}$`)
	responseHeaderPattern := regexp.MustCompile(`^### (\w+) \(([^)]+)\)`)

	for _, line := range lines {
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			// Format: >>> [1/3] Persian (deepseek-coder:6.7b)
			sb.WriteString(watchSectionActiveStyle.Render(line))
			sb.WriteString("\n")
		} else if matches := donePattern.FindStringSubmatch(line); matches != nil {
			// Format:     DONE 24.3s | 512 tokens | 24.1 tok/s
			sb.WriteString(watchSectionDoneStyle.Render(line))
			sb.WriteString("\n")
		} else if headerPattern.MatchString(line) {
			sb.WriteString(watchSectionHeaderStyle.Render(line))
			sb.WriteString("\n")
		} else if matches := responseHeaderPattern.FindStringSubmatch(line); matches != nil {
			sb.WriteString(watchSectionHeaderStyle.Render(line))
			sb.WriteString("\n")
		} else {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (m watchModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Header
	title := watchTitleStyle.Render("KITCHEN STADIUM - WATCH")
	followStatus := ""
	if m.following {
		followStatus = watchStatusStyle.Render(" [Following ▼]")
	} else {
		followStatus = watchInfoStyle.Render(" [Paused]")
	}

	sectionInfo := ""
	if len(m.sections) > 0 {
		sectionInfo = watchInfoStyle.Render(fmt.Sprintf(" %d sections", len(m.sections)))
	}

	header := fmt.Sprintf("%s%s%s\n%s\n",
		title, followStatus, sectionInfo,
		strings.Repeat("─", m.width))

	// Footer
	help := watchHelpStyle.Render("[q]uit [f]ollow [g]top [G]bottom [n]ext [p]rev [1-9]jump")
	scrollInfo := watchInfoStyle.Render(fmt.Sprintf(" %d%%", int(m.viewport.ScrollPercent()*100)))
	footer := fmt.Sprintf("\n%s%s", help, scrollInfo)

	return header + m.viewport.View() + footer
}

// parseSections extracts section info from catfight output
func parseSections(content string) []Section {
	var sections []Section
	lines := strings.Split(content, "\n")

	sectionPattern := regexp.MustCompile(`^>>> \[(\d+)/(\d+)\] (\w+) \(([^)]+)\)`)
	donePattern := regexp.MustCompile(`^\s+DONE (\d+\.\d+)s \| (\d+) tokens \| (\d+\.\d+) tok/s`)

	var currentSection *Section
	for i, line := range lines {
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}
			idx := len(sections)
			currentSection = &Section{
				Index:  idx,
				Name:   matches[3],
				Model:  matches[4],
				Status: "generating",
				Line:   i,
			}
		} else if matches := donePattern.FindStringSubmatch(line); matches != nil && currentSection != nil {
			currentSection.Duration = matches[1]
			currentSection.Tokens = matches[2]
			currentSection.TokSec = matches[3]
			currentSection.Status = "done"
		}
	}
	if currentSection != nil {
		sections = append(sections, *currentSection)
	}

	return sections
}

func WatchCmd() *cobra.Command {
	var file string
	var dir string

	cmd := &cobra.Command{
		Use:   "watch [file]",
		Short: "Watch catfight output in real-time with scrollable TUI",
		Long: `Watch catfight battles unfold in a scrollable, navigable interface.

The watch command provides a "long scroll with moving frame" experience:
- Output accumulates as each model completes
- Auto-follows the latest output by default
- Navigate to review completed sections
- Jump directly to any model's output

Examples:
  # Watch a specific output file
  clood watch /tmp/catfight-results.txt

  # Watch the latest catfight in a directory
  clood watch --dir /tmp/catfight-triage-macmini-*/

  # Pipe catfight output directly (coming soon)
  clood catfight "prompt" | clood watch -

Navigation:
  f        Toggle follow mode (auto-scroll)
  g        Go to top
  G        Go to bottom
  j/k      Scroll down/up
  n/p      Next/previous section
  1-9      Jump to section by number
  q/Esc    Quit`,
		Run: func(cmd *cobra.Command, args []string) {
			var targetFile string

			if len(args) > 0 {
				targetFile = args[0]
			} else if dir != "" {
				// Find the most recent results file in the directory
				matches, _ := filepath.Glob(filepath.Join(dir, "*-results.txt"))
				if len(matches) > 0 {
					// Get most recent
					var newest string
					var newestTime time.Time
					for _, m := range matches {
						if info, err := os.Stat(m); err == nil {
							if info.ModTime().After(newestTime) {
								newestTime = info.ModTime()
								newest = m
							}
						}
					}
					targetFile = newest
				}
			} else if file != "" {
				targetFile = file
			}

			if targetFile == "" {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("No file specified. Use: clood watch <file> or --dir <directory>"))
				return
			}

			// Check if file exists, create if not (for watching new catfights)
			if _, err := os.Stat(targetFile); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, tui.WarningStyle.Render("File not found, waiting for it to be created: "+targetFile))
			}

			// Read initial content
			var initialContent string
			var initialSize int64
			if content, err := os.ReadFile(targetFile); err == nil {
				initialContent = string(content)
				if info, err := os.Stat(targetFile); err == nil {
					initialSize = info.Size()
				}
			}

			m := watchModel{
				content:   initialContent,
				sections:  parseSections(initialContent),
				following: true,
				file:      targetFile,
				lastSize:  initialSize,
			}

			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				fmt.Fprintln(os.Stderr, tui.ErrorStyle.Render("Error running watch: "+err.Error()))
			}
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "File to watch")
	cmd.Flags().StringVarP(&dir, "dir", "d", "", "Directory to find latest results file")

	return cmd
}

// WatchReader creates a watch model from an io.Reader (for piped input)
func WatchReader(r io.Reader) error {
	var content strings.Builder
	scanner := bufio.NewScanner(r)

	m := watchModel{
		following: true,
	}

	// Start reading in background
	go func() {
		for scanner.Scan() {
			content.WriteString(scanner.Text())
			content.WriteString("\n")
			m.content = content.String()
			m.sections = parseSections(m.content)
		}
	}()

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
