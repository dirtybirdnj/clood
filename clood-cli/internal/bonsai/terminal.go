// Package bonsai provides utilities for generating and rendering ASCII bonsai trees.
// It includes an ANSI terminal parser that can extract clean ASCII from ncurses output.
package bonsai

import (
	"fmt"
	"regexp"
	"strings"
)

// ColoredChar represents a character with its color information
type ColoredChar struct {
	Char  rune
	Color string // ANSI color code (e.g., "32" for green, "92" for bright green)
	Row   int
	Col   int
}

// ColorLayer represents all characters of a specific color
type ColorLayer struct {
	Color      string
	ColorName  string
	Characters []ColoredChar
}

// ANSIColorToName converts an ANSI color code to a human-readable name
func ANSIColorToName(code string) string {
	names := map[string]string{
		"30": "black", "31": "red", "32": "green", "33": "yellow",
		"34": "blue", "35": "magenta", "36": "cyan", "37": "white",
		"90": "bright-black", "91": "bright-red", "92": "bright-green",
		"93": "bright-yellow", "94": "bright-blue", "95": "bright-magenta",
		"96": "bright-cyan", "97": "bright-white",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return fmt.Sprintf("color-%s", code)
}

// ANSIColorToHex converts an ANSI color code to a hex color value
func ANSIColorToHex(code string) string {
	// cbonsai uses these colors:
	// 32 = green (trunk/leaves), 92 = bright green (leaves)
	// 33 = yellow (leaves), 93 = bright yellow (leaves)
	// 90 = gray (pot)
	colors := map[string]string{
		"30": "#000000", "31": "#aa0000", "32": "#00aa00", "33": "#aa5500",
		"34": "#0000aa", "35": "#aa00aa", "36": "#00aaaa", "37": "#aaaaaa",
		"90": "#555555", "91": "#ff5555", "92": "#55ff55", "93": "#ffff55",
		"94": "#5555ff", "95": "#ff55ff", "96": "#55ffff", "97": "#ffffff",
	}
	if hex, ok := colors[code]; ok {
		return hex
	}
	return "#000000"
}

// VirtualScreen represents a 2D character grid for rendering terminal output
type VirtualScreen struct {
	Width       int
	Height      int
	Buffer      [][]rune
	ColorBuffer [][]string // Color code for each cell
	CursorRow   int
	CursorCol   int
	CurrentColor string    // Current active color code
}

// NewVirtualScreen creates a new virtual screen with the given dimensions
func NewVirtualScreen(width, height int) *VirtualScreen {
	buffer := make([][]rune, height)
	colorBuffer := make([][]string, height)
	for i := range buffer {
		buffer[i] = make([]rune, width)
		colorBuffer[i] = make([]string, width)
		for j := range buffer[i] {
			buffer[i][j] = ' '
			colorBuffer[i][j] = ""
		}
	}
	return &VirtualScreen{
		Width:       width,
		Height:      height,
		Buffer:      buffer,
		ColorBuffer: colorBuffer,
		CursorRow:   0,
		CursorCol:   0,
		CurrentColor: "",
	}
}

// WriteChar writes a character at the current cursor position with current color
func (vs *VirtualScreen) WriteChar(r rune) {
	if vs.CursorRow >= 0 && vs.CursorRow < vs.Height &&
		vs.CursorCol >= 0 && vs.CursorCol < vs.Width {
		vs.Buffer[vs.CursorRow][vs.CursorCol] = r
		vs.ColorBuffer[vs.CursorRow][vs.CursorCol] = vs.CurrentColor
	}
	vs.CursorCol++
	if vs.CursorCol >= vs.Width {
		vs.CursorCol = 0
		vs.CursorRow++
	}
}

// SetColor sets the current drawing color
func (vs *VirtualScreen) SetColor(color string) {
	vs.CurrentColor = color
}

// MoveCursor moves the cursor to the specified position (1-indexed, like ANSI)
func (vs *VirtualScreen) MoveCursor(row, col int) {
	// ANSI is 1-indexed, convert to 0-indexed
	vs.CursorRow = row - 1
	vs.CursorCol = col - 1
}

// Home moves cursor to top-left
func (vs *VirtualScreen) Home() {
	vs.CursorRow = 0
	vs.CursorCol = 0
}

// Clear clears the screen and color buffer
func (vs *VirtualScreen) Clear() {
	for i := range vs.Buffer {
		for j := range vs.Buffer[i] {
			vs.Buffer[i][j] = ' '
			vs.ColorBuffer[i][j] = ""
		}
	}
}

// GetColorLayers extracts characters grouped by color
func (vs *VirtualScreen) GetColorLayers() []ColorLayer {
	colorMap := make(map[string][]ColoredChar)

	for row := 0; row < vs.Height; row++ {
		for col := 0; col < vs.Width; col++ {
			char := vs.Buffer[row][col]
			color := vs.ColorBuffer[row][col]
			if char != ' ' {
				colorMap[color] = append(colorMap[color], ColoredChar{
					Char:  char,
					Color: color,
					Row:   row,
					Col:   col,
				})
			}
		}
	}

	var layers []ColorLayer
	for color, chars := range colorMap {
		layers = append(layers, ColorLayer{
			Color:      color,
			ColorName:  ANSIColorToName(color),
			Characters: chars,
		})
	}
	return layers
}

// Render returns the screen as a string, trimming trailing empty lines
func (vs *VirtualScreen) Render() string {
	var lines []string
	lastNonEmpty := -1

	for i, row := range vs.Buffer {
		line := string(row)
		// Trim trailing spaces from each line
		line = strings.TrimRight(line, " ")
		lines = append(lines, line)
		if len(line) > 0 {
			lastNonEmpty = i
		}
	}

	// Only include lines up to the last non-empty line
	if lastNonEmpty >= 0 {
		lines = lines[:lastNonEmpty+1]
	} else {
		return ""
	}

	return strings.Join(lines, "\n")
}

// ParseANSI parses ncurses/ANSI output and returns clean ASCII
// It handles cursor positioning, color codes, and other escape sequences.
// For ncurses programs like cbonsai that use alternate screen buffer,
// it captures the content before the screen is cleared on exit.
func ParseANSI(input string) string {
	// Default terminal size (cbonsai uses 80x24 by default)
	screen := NewVirtualScreen(80, 24)

	// Track clear events - cbonsai clears once at start, once before exit
	clearCount := 0
	var savedScreen string

	// Regular expressions for ANSI escape sequences
	// CSI (Control Sequence Introducer) pattern: ESC [ ... letter
	csiPattern := regexp.MustCompile(`\x1b\[([0-9;?]*)([A-Za-z])`)
	// OSC (Operating System Command) pattern: ESC ] ... BEL or ESC \
	oscPattern := regexp.MustCompile(`\x1b\].*?(?:\x07|\x1b\\)`)
	// Character set selection: ESC ( or ESC )
	charsetPattern := regexp.MustCompile(`\x1b[()][\x20-\x2f]*[\x30-\x7e]`)
	// Other escape sequences: ESC followed by single character
	escPattern := regexp.MustCompile(`\x1b[^\[\]()]`)

	// Remove OSC sequences first
	input = oscPattern.ReplaceAllString(input, "")
	// Remove charset selection sequences
	input = charsetPattern.ReplaceAllString(input, "")
	// Remove simple escape sequences
	input = escPattern.ReplaceAllString(input, "")

	// Process CSI sequences
	pos := 0
	for pos < len(input) {
		// Find next CSI sequence
		loc := csiPattern.FindStringSubmatchIndex(input[pos:])
		if loc == nil {
			// No more CSI sequences, write remaining text
			for _, r := range input[pos:] {
				if r >= 32 && r < 127 { // Printable ASCII only
					screen.WriteChar(r)
				}
			}
			break
		}

		// Write text before the sequence
		for _, r := range input[pos : pos+loc[0]] {
			if r >= 32 && r < 127 { // Printable ASCII only
				screen.WriteChar(r)
			}
		}

		// Parse the CSI sequence
		params := input[pos+loc[2] : pos+loc[3]]
		cmd := input[pos+loc[4] : pos+loc[5]]

		switch cmd {
		case "H", "f": // Cursor position
			row, col := 1, 1
			if params != "" {
				parts := strings.Split(params, ";")
				if len(parts) >= 1 && parts[0] != "" {
					row = parseNum(parts[0])
				}
				if len(parts) >= 2 && parts[1] != "" {
					col = parseNum(parts[1])
				}
			}
			screen.MoveCursor(row, col)
		case "J": // Erase in display
			if params == "2" || params == "" {
				clearCount++
				// Save screen before the SECOND clear (the exit clear)
				if clearCount == 2 {
					savedScreen = screen.Render()
				}
				screen.Clear()
			}
		case "A": // Cursor up
			n := parseNumDefault(params, 1)
			screen.CursorRow -= n
		case "B": // Cursor down
			n := parseNumDefault(params, 1)
			screen.CursorRow += n
		case "C": // Cursor forward
			n := parseNumDefault(params, 1)
			screen.CursorCol += n
		case "D": // Cursor back
			n := parseNumDefault(params, 1)
			screen.CursorCol -= n
		case "d": // Cursor to row (VPA)
			row := parseNumDefault(params, 1)
			screen.CursorRow = row - 1
		case "G": // Cursor to column (CHA)
			col := parseNumDefault(params, 1)
			screen.CursorCol = col - 1
		case "h": // Set mode - check for alternate screen
			// ?1049h enters alternate screen, we can ignore
		case "l": // Reset mode - check for alternate screen exit
			if strings.Contains(params, "1049") {
				// Exiting alternate screen - if we saved content, return it
				if savedScreen != "" {
					return savedScreen
				}
			}
		case "m": // SGR - Select Graphic Rendition (colors and attributes)
			// Parse the color code - look for foreground colors (30-37, 90-97)
			parts := strings.Split(params, ";")
			for _, part := range parts {
				if part == "0" || part == "" {
					// Reset all attributes
					screen.SetColor("")
				} else if (part >= "30" && part <= "37") || (part >= "90" && part <= "97") {
					// Foreground color
					screen.SetColor(part)
				}
			}
		// Ignore other codes: r (set scrolling region), etc.
		}

		pos += loc[1]
	}

	// If we saved screen content, use that; otherwise use current screen
	if savedScreen != "" {
		return savedScreen
	}
	return screen.Render()
}

func parseNum(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	if n == 0 {
		return 1
	}
	return n
}

// BonsaiResult contains the parsed bonsai tree with color information
type BonsaiResult struct {
	ASCII  string       // Clean ASCII representation
	Layers []ColorLayer // Characters grouped by color
	Width  int          // Screen width
	Height int          // Actual height (trimmed)
}

// ParseANSIWithColors parses ncurses/ANSI output and returns both clean ASCII and color layers
func ParseANSIWithColors(input string) BonsaiResult {
	screen := NewVirtualScreen(80, 24)
	clearCount := 0
	var savedScreen *VirtualScreen

	csiPattern := regexp.MustCompile(`\x1b\[([0-9;?]*)([A-Za-z])`)
	oscPattern := regexp.MustCompile(`\x1b\].*?(?:\x07|\x1b\\)`)
	charsetPattern := regexp.MustCompile(`\x1b[()][\x20-\x2f]*[\x30-\x7e]`)
	escPattern := regexp.MustCompile(`\x1b[^\[\]()]`)

	input = oscPattern.ReplaceAllString(input, "")
	input = charsetPattern.ReplaceAllString(input, "")
	input = escPattern.ReplaceAllString(input, "")

	pos := 0
	for pos < len(input) {
		loc := csiPattern.FindStringSubmatchIndex(input[pos:])
		if loc == nil {
			for _, r := range input[pos:] {
				if r >= 32 && r < 127 {
					screen.WriteChar(r)
				}
			}
			break
		}

		for _, r := range input[pos : pos+loc[0]] {
			if r >= 32 && r < 127 {
				screen.WriteChar(r)
			}
		}

		params := input[pos+loc[2] : pos+loc[3]]
		cmd := input[pos+loc[4] : pos+loc[5]]

		switch cmd {
		case "H", "f":
			row, col := 1, 1
			if params != "" {
				parts := strings.Split(params, ";")
				if len(parts) >= 1 && parts[0] != "" {
					row = parseNum(parts[0])
				}
				if len(parts) >= 2 && parts[1] != "" {
					col = parseNum(parts[1])
				}
			}
			screen.MoveCursor(row, col)
		case "J":
			if params == "2" || params == "" {
				clearCount++
				if clearCount == 2 {
					// Deep copy the screen before clearing
					savedScreen = copyScreen(screen)
				}
				screen.Clear()
			}
		case "A":
			screen.CursorRow -= parseNumDefault(params, 1)
		case "B":
			screen.CursorRow += parseNumDefault(params, 1)
		case "C":
			screen.CursorCol += parseNumDefault(params, 1)
		case "D":
			screen.CursorCol -= parseNumDefault(params, 1)
		case "d":
			screen.CursorRow = parseNumDefault(params, 1) - 1
		case "G":
			screen.CursorCol = parseNumDefault(params, 1) - 1
		case "l":
			if strings.Contains(params, "1049") && savedScreen != nil {
				return BonsaiResult{
					ASCII:  savedScreen.Render(),
					Layers: savedScreen.GetColorLayers(),
					Width:  savedScreen.Width,
					Height: countNonEmptyLines(savedScreen),
				}
			}
		case "m":
			parts := strings.Split(params, ";")
			for _, part := range parts {
				if part == "0" || part == "" {
					screen.SetColor("")
				} else if (part >= "30" && part <= "37") || (part >= "90" && part <= "97") {
					screen.SetColor(part)
				}
			}
		}
		pos += loc[1]
	}

	resultScreen := screen
	if savedScreen != nil {
		resultScreen = savedScreen
	}
	return BonsaiResult{
		ASCII:  resultScreen.Render(),
		Layers: resultScreen.GetColorLayers(),
		Width:  resultScreen.Width,
		Height: countNonEmptyLines(resultScreen),
	}
}

// copyScreen creates a deep copy of a VirtualScreen
func copyScreen(src *VirtualScreen) *VirtualScreen {
	dst := NewVirtualScreen(src.Width, src.Height)
	dst.CursorRow = src.CursorRow
	dst.CursorCol = src.CursorCol
	dst.CurrentColor = src.CurrentColor
	for i := range src.Buffer {
		copy(dst.Buffer[i], src.Buffer[i])
		copy(dst.ColorBuffer[i], src.ColorBuffer[i])
	}
	return dst
}

// countNonEmptyLines returns the number of non-empty lines in the screen
func countNonEmptyLines(screen *VirtualScreen) int {
	count := 0
	for i := range screen.Buffer {
		for j := range screen.Buffer[i] {
			if screen.Buffer[i][j] != ' ' {
				count = i + 1
				break
			}
		}
	}
	return count
}

func parseNumDefault(s string, def int) int {
	if s == "" {
		return def
	}
	return parseNum(s)
}
