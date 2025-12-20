package router

import (
	"testing"
)

func TestClassifyQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected int
	}{
		// Simple/Fast tier queries
		{"simple what is", "what is a goroutine?", TierFast},
		{"simple how to", "how to format a string in Go", TierFast},
		{"simple syntax", "syntax for slice append", TierFast},
		{"regex request", "regex for email validation", TierFast},
		{"one-liner request", "one-liner to reverse a string", TierFast},

		// Complex/Deep tier queries
		{"refactoring task", "refactor this function to use interfaces", TierDeep},
		{"implement feature", "implement a cache with TTL expiration", TierDeep},
		{"code generation", "generate a REST API handler for users", TierDeep},
		{"multi-step task", "build a CLI tool with cobra and test it", TierDeep},
		{"codebase task", "add a feature in this codebase", TierDeep},

		// Analysis tier queries
		{"code review", "review this code for bugs", TierAnalysis},
		{"security analysis", "find security issues in this function", TierAnalysis},
		{"explain why", "explain why this approach is better", TierAnalysis},
		{"trade-offs", "what are the trade-offs of this design", TierAnalysis},
		{"bug analysis", "what's wrong with this code", TierAnalysis},

		// Writing tier queries
		{"documentation", "write documentation for this module", TierWriting},
		{"readme", "write a readme for this project", TierWriting},
		{"commit message", "write a commit message for these changes", TierWriting},
		{"tutorial", "write a tutorial on using this library", TierWriting},
		{"api docs", "generate api docs for this package", TierWriting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyQuery(tt.query)
			if result != tt.expected {
				t.Errorf("ClassifyQuery(%q) = %d (%s), want %d (%s)",
					tt.query, result, TierName(result), tt.expected, TierName(tt.expected))
			}
		})
	}
}

func TestClassifyWithConfidence(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedTier   int
		minConfidence  float64
	}{
		{"clear analysis", "review this code and find all bugs", TierAnalysis, 0.3},
		{"clear writing", "write documentation for the api", TierWriting, 0.3},
		{"ambiguous", "hello", TierFast, 0.0}, // No indicators, defaults to fast
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, confidence := ClassifyWithConfidence(tt.query)
			if tier != tt.expectedTier {
				t.Errorf("ClassifyWithConfidence(%q) tier = %d, want %d",
					tt.query, tier, tt.expectedTier)
			}
			if confidence < tt.minConfidence {
				t.Errorf("ClassifyWithConfidence(%q) confidence = %.2f, want >= %.2f",
					tt.query, confidence, tt.minConfidence)
			}
		})
	}
}

func TestTierName(t *testing.T) {
	tests := []struct {
		tier     int
		expected string
	}{
		{TierFast, "Fast"},
		{TierDeep, "Deep"},
		{TierAnalysis, "Analysis"},
		{TierWriting, "Writing"},
		{99, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := TierName(tt.tier)
			if result != tt.expected {
				t.Errorf("TierName(%d) = %q, want %q", tt.tier, result, tt.expected)
			}
		})
	}
}

func TestLengthHeuristics(t *testing.T) {
	// Very long queries should boost Deep tier
	shortQuery := "what is a pointer"
	longQuery := "I have a complex system with multiple services " +
		"that need to communicate over gRPC, and I want to implement " +
		"a custom load balancer that takes into account the current " +
		"health of each service, the latency of recent requests, and " +
		"the capacity of each server. Additionally, I need to handle " +
		"graceful degradation when services become unavailable."

	shortTier := ClassifyQuery(shortQuery)
	longTier := ClassifyQuery(longQuery)

	if shortTier != TierFast {
		t.Errorf("Short query should classify as Fast, got %s", TierName(shortTier))
	}
	if longTier != TierDeep {
		t.Errorf("Long query should classify as Deep, got %s", TierName(longTier))
	}
}

func TestMultipleSentenceHeuristics(t *testing.T) {
	// Multiple sentences should boost Deep tier
	singleSentence := "what is dependency injection"
	multipleSentences := "I want to refactor this code. First we need to extract the interface. Then implement the concrete types. Finally add tests."

	singleTier := ClassifyQuery(singleSentence)
	multipleTier := ClassifyQuery(multipleSentences)

	if singleTier != TierFast {
		t.Errorf("Single sentence should classify as Fast, got %s", TierName(singleTier))
	}
	if multipleTier != TierDeep {
		t.Errorf("Multiple sentences should classify as Deep, got %s", TierName(multipleTier))
	}
}
