package bonsai

import (
	"os/exec"
	"strings"
	"testing"
)

func TestLoadHersheyFont(t *testing.T) {
	font, err := LoadHersheyFont("HersheySans1")
	if err != nil {
		t.Fatalf("Failed to load font: %v", err)
	}

	if font.Name != "HersheySans1" {
		t.Errorf("Expected font name HersheySans1, got %s", font.Name)
	}

	// Should have basic ASCII characters
	if _, ok := font.Glyphs['A']; !ok {
		t.Error("Font should contain 'A' glyph")
	}
	if _, ok := font.Glyphs['&']; !ok {
		t.Error("Font should contain '&' glyph (used by bonsai)")
	}

	t.Logf("Loaded font with %d glyphs", len(font.Glyphs))
}

func TestAvailableFonts(t *testing.T) {
	fonts := AvailableFonts()
	if len(fonts) == 0 {
		t.Error("Expected at least one font")
	}
	t.Logf("Available fonts: %v", fonts)
}

func TestSVGGenerator_GenerateSVG(t *testing.T) {
	gen, err := NewSVGGenerator("HersheySans1")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create a simple test result
	result := BonsaiResult{
		ASCII:  "AB\nCD",
		Width:  10,
		Height: 2,
		Layers: []ColorLayer{
			{
				Color:     "32",
				ColorName: "green",
				Characters: []ColoredChar{
					{Char: 'A', Color: "32", Row: 0, Col: 0},
					{Char: 'B', Color: "32", Row: 0, Col: 1},
				},
			},
			{
				Color:     "90",
				ColorName: "bright-black",
				Characters: []ColoredChar{
					{Char: 'C', Color: "90", Row: 1, Col: 0},
					{Char: 'D', Color: "90", Row: 1, Col: 1},
				},
			},
		},
	}

	svg := gen.GenerateSVG(result)

	// Check SVG structure
	if !strings.Contains(svg, "<?xml") {
		t.Error("SVG should have XML declaration")
	}
	if !strings.Contains(svg, "<svg") {
		t.Error("SVG should have svg element")
	}
	if !strings.Contains(svg, `id="layer-green"`) {
		t.Error("SVG should have green layer")
	}
	if !strings.Contains(svg, `id="layer-bright-black"`) {
		t.Error("SVG should have bright-black layer")
	}
	if !strings.Contains(svg, "<path") {
		t.Error("SVG should have path elements")
	}

	t.Logf("Generated SVG length: %d bytes", len(svg))
}

func TestSVGGenerator_CbonsaiIntegration(t *testing.T) {
	// Skip if cbonsai not installed
	if _, err := exec.LookPath("cbonsai"); err != nil {
		t.Skip("cbonsai not installed")
	}

	// Run cbonsai
	cmd := exec.Command("cbonsai", "-p", "-L", "16", "-s", "42")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cbonsai failed: %v", err)
	}

	// Parse with colors
	result := ParseANSIWithColors(string(output))

	// Generate SVG
	gen, err := NewSVGGenerator("HersheySans1")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	svg := gen.GenerateSVG(result)

	// Verify SVG is valid
	if !strings.Contains(svg, "<svg") {
		t.Error("Expected valid SVG")
	}

	// Count layers
	layerCount := strings.Count(svg, `<g id="layer-`)
	if layerCount < 2 {
		t.Errorf("Expected multiple color layers, got %d", layerCount)
	}

	t.Logf("Generated bonsai SVG: %d bytes, %d layers", len(svg), layerCount)
}
