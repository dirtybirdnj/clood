package tui

import (
	"fmt"
	"sync"
	"time"
)

// SpinnerFrames are the animation frames for the loading spinner
// Using braille patterns for smooth animation
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerFramesDots uses simple dots
var SpinnerFramesDots = []string{".", "..", "...", "....", ".....", "......"}

// SpinnerFramesCloud uses cloud-themed animation
var SpinnerFramesCloud = []string{"☁ ", "☁.", "☁..", "☁...", "☁ ..", "☁  .", "☁   "}

// Spinner provides an animated loading indicator with streaming updates
type Spinner struct {
	message    string
	frames     []string
	frameIdx   int
	interval   time.Duration
	done       chan struct{}
	stopped    bool
	mu         sync.Mutex
	updates    []string
	maxUpdates int
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:    message,
		frames:     SpinnerFrames,
		interval:   80 * time.Millisecond,
		done:       make(chan struct{}),
		maxUpdates: 10,
	}
}

// SetFrames allows customizing the spinner animation frames
func (s *Spinner) SetFrames(frames []string) *Spinner {
	s.frames = frames
	return s
}

// SetInterval sets the animation speed
func (s *Spinner) SetInterval(d time.Duration) *Spinner {
	s.interval = d
	return s
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	go s.run()
}

// Update adds a streaming update line below the spinner
func (s *Spinner) Update(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.updates = append(s.updates, line)
	// Keep only the last maxUpdates lines
	if len(s.updates) > s.maxUpdates {
		s.updates = s.updates[len(s.updates)-s.maxUpdates:]
	}
}

// UpdateMessage changes the main spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Stop halts the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()
	close(s.done)
	// Clear the spinner line
	s.clearLines(1 + len(s.updates))
}

// StopWithMessage stops and prints a final message
func (s *Spinner) StopWithMessage(message string) {
	s.mu.Lock()
	updateCount := len(s.updates)
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()
	close(s.done)
	// Clear spinner and update lines
	s.clearLines(1 + updateCount)
	fmt.Println(message)
}

func (s *Spinner) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.render()
		}
	}
}

func (s *Spinner) render() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	// Calculate how many lines to clear (spinner + updates)
	totalLines := 1 + len(s.updates)
	s.clearLines(totalLines)

	// Render spinner with current frame
	frame := s.frames[s.frameIdx]
	s.frameIdx = (s.frameIdx + 1) % len(s.frames)

	fmt.Printf("\r%s %s %s\n",
		AccentStyle.Render(frame),
		s.message,
		MutedStyle.Render(s.getElapsedIndicator()))

	// Render updates
	for _, update := range s.updates {
		fmt.Printf("  %s\n", update)
	}
}

func (s *Spinner) clearLines(n int) {
	// Move cursor up n lines and clear each
	for i := 0; i < n; i++ {
		fmt.Print("\033[A") // Move up
		fmt.Print("\033[K") // Clear line
	}
}

func (s *Spinner) getElapsedIndicator() string {
	return ""
}

// ProgressSpinner shows progress with a count
type ProgressSpinner struct {
	*Spinner
	current int
	total   int
}

// NewProgressSpinner creates a spinner that shows progress
func NewProgressSpinner(message string, total int) *ProgressSpinner {
	return &ProgressSpinner{
		Spinner: NewSpinner(message),
		total:   total,
	}
}

// Increment increases the progress counter and adds an update
func (p *ProgressSpinner) Increment(updateText string) {
	p.current++
	p.UpdateMessage(fmt.Sprintf("%s (%d/%d)", p.Spinner.message, p.current, p.total))
	if updateText != "" {
		p.Update(updateText)
	}
}

// SimpleSpinner provides a minimal spinning indicator for quick operations
// It just prints frames in place without the full Spinner machinery
func SimpleSpinner(message string, work func()) {
	done := make(chan struct{})
	frames := SpinnerFrames

	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Print("\r\033[K") // Clear the line
				return
			default:
				fmt.Printf("\r%s %s",
					AccentStyle.Render(frames[i%len(frames)]),
					message)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	work()
	close(done)
	time.Sleep(100 * time.Millisecond) // Let the goroutine clean up
}

// StreamingLoader shows a message while work happens, updating with results
type StreamingLoader struct {
	title      string
	items      []LoaderItem
	mu         sync.Mutex
	done       chan struct{}
	started    bool
	frameIdx   int
	startTime  time.Time
}

// LoaderItem represents an item being loaded with its status
type LoaderItem struct {
	Name    string
	Status  string // "pending", "checking", "online", "offline", "error"
	Details string
}

// NewStreamingLoader creates a loader that shows items as they complete
func NewStreamingLoader(title string) *StreamingLoader {
	return &StreamingLoader{
		title: title,
		done:  make(chan struct{}),
	}
}

// AddItem adds an item to track
func (l *StreamingLoader) AddItem(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, LoaderItem{Name: name, Status: "pending"})
}

// UpdateItem updates an item's status
func (l *StreamingLoader) UpdateItem(name, status, details string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.items {
		if l.items[i].Name == name {
			l.items[i].Status = status
			l.items[i].Details = details
			break
		}
	}
}

// Start begins the animated display
func (l *StreamingLoader) Start() {
	l.mu.Lock()
	if l.started {
		l.mu.Unlock()
		return
	}
	l.started = true
	l.startTime = time.Now()
	l.mu.Unlock()

	go l.run()
}

// Stop ends the display
func (l *StreamingLoader) Stop() {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()
	close(l.done)
	time.Sleep(50 * time.Millisecond)
}

func (l *StreamingLoader) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Initial render
	l.render()

	for {
		select {
		case <-l.done:
			l.renderFinal()
			return
		case <-ticker.C:
			l.render()
		}
	}
}

func (l *StreamingLoader) render() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Clear previous render
	lineCount := 2 + len(l.items) // title + blank + items
	for i := 0; i < lineCount; i++ {
		fmt.Print("\033[A\033[K")
	}

	// Spinner frame
	frame := SpinnerFrames[l.frameIdx%len(SpinnerFrames)]
	l.frameIdx++

	// Title with spinner
	elapsed := time.Since(l.startTime).Round(100 * time.Millisecond)
	fmt.Printf("%s %s %s\n\n",
		AccentStyle.Render(frame),
		l.title,
		MutedStyle.Render(fmt.Sprintf("(%s)", elapsed)))

	// Items
	for _, item := range l.items {
		l.renderItem(item)
	}
}

func (l *StreamingLoader) renderFinal() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Clear previous render
	lineCount := 2 + len(l.items)
	for i := 0; i < lineCount; i++ {
		fmt.Print("\033[A\033[K")
	}
}

func (l *StreamingLoader) renderItem(item LoaderItem) {
	var indicator, statusText string

	switch item.Status {
	case "pending":
		indicator = MutedStyle.Render("○")
		statusText = MutedStyle.Render("waiting...")
	case "checking":
		indicator = AccentStyle.Render("◐")
		statusText = AccentStyle.Render("checking...")
	case "online":
		indicator = SuccessStyle.Render("●")
		statusText = SuccessStyle.Render("online")
		if item.Details != "" {
			statusText += " " + MutedStyle.Render(item.Details)
		}
	case "offline":
		indicator = ErrorStyle.Render("○")
		statusText = ErrorStyle.Render("offline")
	case "error":
		indicator = ErrorStyle.Render("✗")
		statusText = ErrorStyle.Render("error")
		if item.Details != "" {
			statusText += " " + MutedStyle.Render(item.Details)
		}
	}

	fmt.Printf("  %s %s %s\n", indicator, item.Name, statusText)
}
