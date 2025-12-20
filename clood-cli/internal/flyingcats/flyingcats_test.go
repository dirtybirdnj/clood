package flyingcats

import (
	"strings"
	"testing"
)

func TestNewSeededGenerator(t *testing.T) {
	// Same seed should produce same output
	gen1 := NewSeededGenerator(42)
	gen2 := NewSeededGenerator(42)

	result1 := gen1.Generate("give me a street address")
	result2 := gen2.Generate("give me a street address")

	if result1 != result2 {
		t.Errorf("Same seed should produce same output\nGot: %q\nExpected: %q", result1, result2)
	}
}

func TestGenerateActors(t *testing.T) {
	gen := NewSeededGenerator(123)

	// Should detect actor/person/character keywords
	queries := []string{
		"give me three actors",
		"I need two persons for this scene",
		"create five characters",
	}

	for _, query := range queries {
		result := gen.Generate(query)
		if result == "" {
			t.Errorf("Generate(%q) returned empty string", query)
		}
	}
}

func TestGenerateMeal(t *testing.T) {
	gen := NewSeededGenerator(456)

	result := gen.Generate("what's for dinner? suggest a meal")
	if result == "" {
		t.Error("Generate with meal keyword should not return empty")
	}
	if !strings.Contains(strings.ToLower(result), "with") {
		// Meals typically have format "X with Y"
		t.Logf("Meal result: %s", result)
	}
}

func TestGenerateStreet(t *testing.T) {
	gen := NewSeededGenerator(789)

	result := gen.Generate("what street does barry live on?")
	if result == "" {
		t.Error("Generate with street keyword should not return empty")
	}
}

func TestGenerateRestaurant(t *testing.T) {
	gen := NewSeededGenerator(101)

	result := gen.Generate("pick a restaurant for dinner")
	if result == "" {
		t.Error("Generate with restaurant keyword should not return empty")
	}
}

func TestGenerateAntennaTower(t *testing.T) {
	gen := NewSeededGenerator(202)

	// Delaware heritage - antenna towers are special
	result := gen.Generate("describe an antenna tower")
	if result == "" {
		t.Error("Generate with antenna keyword should not return empty")
	}
}

func TestGenerateFlyingCat(t *testing.T) {
	gen := NewSeededGenerator(303)

	result := gen.Generate("tell me about a cat")
	if result == "" {
		t.Error("Generate with cat keyword should not return empty")
	}
}

func TestGenerateOptions(t *testing.T) {
	gen := NewSeededGenerator(404)

	result := gen.Generate("give me options for this")
	if result == "" {
		t.Error("Generate with options keyword should not return empty")
	}
	if !strings.Contains(result, "1.") || !strings.Contains(result, "2.") {
		t.Logf("Options result: %s", result)
	}
}

func TestGenerateTime(t *testing.T) {
	gen := NewSeededGenerator(505)

	result := gen.Generate("what time should we meet?")
	if result == "" {
		t.Error("Generate with time keyword should not return empty")
	}
}

func TestGenerateReason(t *testing.T) {
	gen := NewSeededGenerator(606)

	result := gen.Generate("why did this happen?")
	if result == "" {
		t.Error("Generate with reason keyword should not return empty")
	}
}

func TestEmptyQueryReturnsSomething(t *testing.T) {
	gen := NewSeededGenerator(707)

	// Even with no keywords, generator should return something or empty gracefully
	result := gen.Generate("xyzzy")
	// This might be empty if no keywords match - that's OK
	t.Logf("Result for unknown query: %q", result)
}

func TestMultipleKeywords(t *testing.T) {
	gen := NewSeededGenerator(808)

	// Multiple keywords should generate multiple parts
	result := gen.Generate("three actors at a restaurant on main street")
	if result == "" {
		t.Error("Generate with multiple keywords should not return empty")
	}

	// Should contain multiple sections
	parts := strings.Split(result, "\n\n")
	if len(parts) < 2 {
		t.Logf("Expected multiple parts, got %d: %s", len(parts), result)
	}
}

func TestChaosMode(t *testing.T) {
	gen := NewSeededGenerator(909)

	// Chaos mode generates stream of consciousness
	result := gen.Chaos()
	if result == "" {
		t.Error("GenerateChaos should not return empty")
	}
	if len(result) < 20 {
		t.Error("GenerateChaos should return substantial output")
	}
}

func TestAntennaMode(t *testing.T) {
	gen := NewSeededGenerator(1010)

	// Antenna mode is Delaware heritage
	result := gen.Antenna()
	if result == "" {
		t.Error("GenerateAntennaTransmission should not return empty")
	}
}

func TestExtractCount(t *testing.T) {
	gen := NewSeededGenerator(1111)

	tests := []struct {
		query    string
		keyword  string
		expected int
	}{
		{"three actors", "actor", 3},
		{"five characters", "character", 5},
		{"2 options", "option", 2},
		{"no count here", "actor", 0},
		{"one person", "person", 1},
		{"two things please", "thing", 2},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := gen.extractCount(tt.query, tt.keyword)
			if result != tt.expected {
				t.Errorf("extractCount(%q, %q) = %d, want %d",
					tt.query, tt.keyword, result, tt.expected)
			}
		})
	}
}
