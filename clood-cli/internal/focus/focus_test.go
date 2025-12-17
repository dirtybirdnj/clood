package focus

import (
	"testing"
)

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "fix the authentication bug",
			expected: []string{"authentication", "bug"},
		},
		{
			input:    "add a dark mode toggle",
			expected: []string{"dark", "mode", "toggle"},
		},
		{
			input:    "the quick brown fox",
			expected: []string{"quick", "brown", "fox"},
		},
		{
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractKeywords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("extractKeywords(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, kw := range result {
				if kw != tt.expected[i] {
					t.Errorf("extractKeywords(%q)[%d] = %q, want %q", tt.input, i, kw, tt.expected[i])
				}
			}
		})
	}
}

func TestGuardianDriftDetection(t *testing.T) {
	g := NewGuardian("fix authentication bug")

	// Related message - should not drift
	result := g.CheckMessage("I found the authentication error in login.go")
	if result.IsDrift {
		t.Error("Expected no drift for related message")
	}

	// Reset for clean test
	g.Reset()

	// Unrelated messages - should eventually trigger drift
	unrelateds := []string{
		"What about adding a dark mode?",
		"Can we also add a settings page?",
		"Let's implement the export feature",
	}

	for i, msg := range unrelateds {
		result = g.CheckMessage(msg)
		if !result.IsDrift {
			t.Errorf("Message %d should be detected as drift: %q", i, msg)
		}
	}

	// After 3 drifted messages, should have a warning
	if result.Message == "" {
		t.Error("Expected warning message after threshold reached")
	}
}

func TestGuardianReset(t *testing.T) {
	g := NewGuardian("fix auth bug")

	// Drift a bit
	g.CheckMessage("unrelated message one")
	g.CheckMessage("unrelated message two")

	if g.DriftCount != 2 {
		t.Errorf("Expected drift count 2, got %d", g.DriftCount)
	}

	g.Reset()

	if g.DriftCount != 0 {
		t.Errorf("Expected drift count 0 after reset, got %d", g.DriftCount)
	}
}

func TestGuardianUpdateGoal(t *testing.T) {
	g := NewGuardian("fix auth bug")

	// Check initial keywords
	if len(g.Keywords) != 2 {
		t.Errorf("Expected 2 keywords, got %d: %v", len(g.Keywords), g.Keywords)
	}

	g.UpdateGoal("implement dark mode feature")

	// Should have new keywords
	if len(g.Keywords) != 4 {
		t.Errorf("Expected 4 keywords after update, got %d: %v", len(g.Keywords), g.Keywords)
	}

	// Drift count should reset
	if g.DriftCount != 0 {
		t.Errorf("Expected drift count 0 after goal update, got %d", g.DriftCount)
	}
}

func TestNoGoalNoDrift(t *testing.T) {
	g := NewGuardian("")

	// With no goal, nothing should be detected as drift
	result := g.CheckMessage("absolutely anything")
	if result.IsDrift {
		t.Error("Expected no drift when goal is empty")
	}
}

func TestPartialKeywordMatch(t *testing.T) {
	g := NewGuardian("fix authentication bug")

	// "auth" should match "authentication" via substring
	result := g.CheckMessage("the auth module needs work")
	if result.IsDrift {
		t.Error("Expected partial keyword match for 'auth' -> 'authentication'")
	}
}

func TestGetStatus(t *testing.T) {
	g := NewGuardian("fix something")

	status := g.GetStatus()
	if status != "On track" {
		t.Errorf("Expected 'On track', got %q", status)
	}

	g.CheckMessage("unrelated")
	status = g.GetStatus()
	if status != "Wandering slightly" {
		t.Errorf("Expected 'Wandering slightly', got %q", status)
	}

	g.CheckMessage("still unrelated")
	g.CheckMessage("even more unrelated")
	status = g.GetStatus()
	if status != "Drifting from goal" {
		t.Errorf("Expected 'Drifting from goal', got %q", status)
	}
}
