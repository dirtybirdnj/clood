package tui

import "github.com/charmbracelet/lipgloss"

// ============================================================================
// CLOOD ASCII ART - CUSTOMIZE THIS!
// ============================================================================
//
// CHARACTER PALETTE (safe for all terminals):
//
// BOX DRAWING (Unicode):
//   Single: ─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼
//   Double: ═ ║ ╔ ╗ ╚ ╝ ╠ ╣ ╦ ╩ ╬
//   Rounded: ╭ ╮ ╰ ╯
//   Heavy: ━ ┃ ┏ ┓ ┗ ┛
//
// BLOCK ELEMENTS:
//   Full: █ ▓ ▒ ░
//   Half: ▄ ▀ ▌ ▐
//   Quadrant: ▖ ▗ ▘ ▝ ▞ ▟ ▙ ▛ ▜
//
// GEOMETRIC:
//   ◆ ◇ ○ ● ◐ ◑ ◒ ◓ ◢ ◣ ◤ ◥
//   ▲ ▼ ◀ ▶ △ ▽ ◁ ▷
//
// WEATHER/NATURE (for cloud theme):
//   ☁ ⛅ ⛈ 🌧 🌩 ⚡ ✨ ❄ 💧
//
// MISC SYMBOLS:
//   ★ ☆ ✦ ✧ ⬡ ⬢ ⎔ ⌘ ⏣
//
// ASCII-ONLY FALLBACK (maximum compatibility):
//   Letters: A-Z, a-z
//   Symbols: @ # $ % ^ & * ( ) - _ = + [ ] { } | \ / < > ? ! ~
//   Box-ish: + - | _ = #
//
// ============================================================================

// Logo is the main ASCII art banner for clood.
// Replace this with your custom design!
//
// TIPS:
//   - Use a fixed-width font when editing
//   - Test with: go run ./cmd/clood
//   - Keep width under 60 chars for narrow terminals
//   - Use raw string literals (backticks) to preserve formatting
//
// GENERATORS:
//   - https://patorjk.com/software/taag/ (text to ASCII)
//   - https://www.ascii-art-generator.org/ (image to ASCII)
//   - https://textkool.com/en/ascii-art-generator (many fonts)
//
var Logo = `
   _____ _      ____   ____  _____
  / ____| |    / __ \ / __ \|  __ \
 | |    | |   | |  | | |  | | |  | |
 | |    | |   | |  | | |  | | |  | |
 | |____| |___| |__| | |__| | |__| |
  \_____|______\____/ \____/|_____/
`

// LogoSmall is a compact version for narrow terminals or headers
var LogoSmall = `☁ CLOOD`

// Tagline appears below the logo
var Tagline = "Lightning in a Bottle"

// ============================================================================
// COLOR PALETTE
// ============================================================================
//
// Lipgloss supports:
//   - ANSI 16 colors: lipgloss.Color("1") through lipgloss.Color("15")
//   - ANSI 256 colors: lipgloss.Color("21"), lipgloss.Color("208"), etc.
//   - True color (hex): lipgloss.Color("#FF6600")
//   - Adaptive: lipgloss.AdaptiveColor{Light: "0", Dark: "15"}
//
// ANSI 256 Color Reference:
//   0-7:    Standard colors (black, red, green, yellow, blue, magenta, cyan, white)
//   8-15:   Bright versions
//   16-231: 6x6x6 color cube
//   232-255: Grayscale (dark to light)
//
// CLOOD THEME COLORS (cloud/lightning inspired):
//

var (
	// Primary brand color - electric blue (lightning)
	ColorPrimary = lipgloss.Color("#00D4FF")

	// Secondary - soft cloud gray
	ColorSecondary = lipgloss.Color("#B8C5D0")

	// Accent - lightning gold/yellow
	ColorAccent = lipgloss.Color("#FFD700")

	// Success - green
	ColorSuccess = lipgloss.Color("#00FF88")

	// Warning - orange
	ColorWarning = lipgloss.Color("#FF8C00")

	// Error - red
	ColorError = lipgloss.Color("#FF4444")

	// Muted - dim gray for less important text
	ColorMuted = lipgloss.Color("#666666")

	// Background hints (for adaptive themes)
	ColorBgDark  = lipgloss.Color("#1a1a2e")
	ColorBgLight = lipgloss.Color("#f0f4f8")
)

// ============================================================================
// STYLED COMPONENTS
// ============================================================================

var (
	// LogoStyle applies colors to the main logo
	LogoStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// TaglineStyle for the subtitle
	TaglineStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Italic(true)

	// BoxStyle for content containers
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(1, 2)

	// HeaderStyle for section headers
	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Underline(true)

	// TierFastStyle for Tier 1 (fast/simple) indicator
	TierFastStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	// TierDeepStyle for Tier 2 (deep/complex) indicator
	TierDeepStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// SuccessStyle for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// ErrorStyle for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// MutedStyle for secondary information
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// RenderBanner returns the full styled banner
func RenderBanner() string {
	logo := LogoStyle.Render(Logo)
	tagline := TaglineStyle.Render("  " + Tagline)
	return logo + "\n" + tagline
}

// RenderHeader returns a styled section header
func RenderHeader(title string) string {
	return HeaderStyle.Render(title)
}

// RenderTier returns a styled tier indicator
func RenderTier(tier int) string {
	if tier == 1 {
		return TierFastStyle.Render("⚡ Tier 1: Speed Mode")
	}
	return TierDeepStyle.Render("🧠 Tier 2: Deep Mode")
}
