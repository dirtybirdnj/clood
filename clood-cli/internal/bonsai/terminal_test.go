package bonsai

import (
	"os/exec"
	"strings"
	"testing"
)

func TestVirtualScreen(t *testing.T) {
	screen := NewVirtualScreen(10, 5)

	// Test writing at position
	screen.MoveCursor(1, 1)
	screen.WriteChar('H')
	screen.WriteChar('i')

	result := screen.Render()
	if !strings.HasPrefix(result, "Hi") {
		t.Errorf("Expected 'Hi', got: %q", result)
	}

	// Test cursor positioning
	screen.MoveCursor(3, 5)
	screen.WriteChar('X')

	result = screen.Render()
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines, got %d", len(lines))
	}
	if len(lines) >= 3 && !strings.Contains(lines[2], "X") {
		t.Errorf("Expected 'X' on line 3, got: %q", lines[2])
	}
}

func TestParseANSI_Simple(t *testing.T) {
	// Simple cursor positioning and text
	input := "\x1b[2;3HTest"
	result := ParseANSI(input)

	// Should have empty first line, then "  Test" starting at column 3
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("Expected 2 lines, got %d: %q", len(lines), result)
		return
	}
	if !strings.Contains(lines[1], "Test") {
		t.Errorf("Expected 'Test' on line 2, got: %q", lines[1])
	}
}

func TestParseANSI_ColorStrip(t *testing.T) {
	// Text with color codes should strip them
	input := "\x1b[32mGreen\x1b[0m Normal"
	result := ParseANSI(input)

	if !strings.Contains(result, "Green") {
		t.Errorf("Expected 'Green' in output, got: %q", result)
	}
	if !strings.Contains(result, "Normal") {
		t.Errorf("Expected 'Normal' in output, got: %q", result)
	}
	// Should not contain escape characters
	if strings.Contains(result, "\x1b") {
		t.Errorf("Output should not contain escape sequences: %q", result)
	}
}

func TestParseANSI_CbonsaiIntegration(t *testing.T) {
	// Skip if cbonsai not installed
	if _, err := exec.LookPath("cbonsai"); err != nil {
		t.Skip("cbonsai not installed")
	}

	// Run cbonsai with a fixed seed for reproducibility
	cmd := exec.Command("cbonsai", "-p", "-L", "16", "-s", "42")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cbonsai failed: %v", err)
	}

	// Parse the output
	result := ParseANSI(string(output))

	// Should have some content
	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}

	// Should contain tree characters
	if !strings.Contains(result, "&") && !strings.Contains(result, "|") {
		t.Errorf("Expected tree characters (& or |) in output:\n%s", result)
	}

	// Should contain pot base
	if !strings.Contains(result, "___") {
		t.Errorf("Expected pot base (___) in output:\n%s", result)
	}

	// Should not contain escape sequences
	if strings.Contains(result, "\x1b") {
		t.Error("Output should not contain escape sequences")
	}

	t.Logf("Parsed cbonsai output:\n%s", result)
}

func TestParseANSIWithColors_ColorExtraction(t *testing.T) {
	// Skip if cbonsai not installed
	if _, err := exec.LookPath("cbonsai"); err != nil {
		t.Skip("cbonsai not installed")
	}

	cmd := exec.Command("cbonsai", "-p", "-L", "16", "-s", "42")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cbonsai failed: %v", err)
	}

	result := ParseANSIWithColors(string(output))

	// Should have ASCII output
	if len(result.ASCII) == 0 {
		t.Error("Expected non-empty ASCII result")
	}

	// Should have color layers
	if len(result.Layers) == 0 {
		t.Error("Expected color layers")
	}

	// Log the layers found
	t.Logf("Found %d color layers:", len(result.Layers))
	for _, layer := range result.Layers {
		t.Logf("  %s (%s): %d characters", layer.ColorName, layer.Color, len(layer.Characters))
	}

	// cbonsai typically uses green (32), bright green (92), yellow (33), and gray (90)
	var hasGreen, hasPot bool
	for _, layer := range result.Layers {
		if layer.Color == "32" || layer.Color == "92" {
			hasGreen = true
		}
		if layer.Color == "90" {
			hasPot = true
		}
	}

	if !hasGreen {
		t.Error("Expected green color layer for tree")
	}
	if !hasPot {
		t.Error("Expected gray color layer for pot")
	}
}
