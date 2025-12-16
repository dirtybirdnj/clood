package router

import (
	"strings"
)

// Tier constants
const (
	TierFast = 1 // Simple queries -> mods
	TierDeep = 2 // Complex queries -> crush
)

// complexIndicators are phrases that suggest a query needs deep reasoning
var complexIndicators = []string{
	// Multi-step tasks
	"refactor",
	"implement",
	"create a",
	"build a",
	"write a function",
	"write a class",
	"add a feature",
	"fix the bug",
	"debug",

	// File operations
	"read the file",
	"modify the",
	"update the",
	"change the",
	"edit",
	"in this codebase",
	"in this project",

	// Analysis tasks
	"explain how",
	"analyze",
	"review",
	"compare",

	// Research tasks
	"search for",
	"find information",
	"look up",

	// Code generation
	"generate",
	"scaffold",
	"boilerplate",

	// Planning
	"plan",
	"design",
	"architect",
}

// simpleIndicators suggest quick, simple queries
var simpleIndicators = []string{
	// Direct questions
	"what is",
	"what's",
	"how do i",
	"how to",
	"why does",
	"when should",

	// Quick tasks
	"convert",
	"translate",
	"format",
	"one-liner",
	"command for",
	"syntax for",

	// Simple generation
	"git commit message",
	"regex for",
	"example of",
}

// ClassifyQuery determines which tier should handle a query
func ClassifyQuery(query string) int {
	lower := strings.ToLower(query)

	// Check for complex indicators first (they take priority)
	complexScore := 0
	for _, indicator := range complexIndicators {
		if strings.Contains(lower, indicator) {
			complexScore++
		}
	}

	// Check for simple indicators
	simpleScore := 0
	for _, indicator := range simpleIndicators {
		if strings.Contains(lower, indicator) {
			simpleScore++
		}
	}

	// Length heuristic: very long queries are often complex
	if len(query) > 200 {
		complexScore++
	}

	// Multiple sentences suggest complexity
	if strings.Count(query, ".") > 2 || strings.Count(query, "\n") > 1 {
		complexScore++
	}

	// Decision
	if complexScore > simpleScore {
		return TierDeep
	}

	// Default to fast tier for simple/ambiguous queries
	return TierFast
}

// ClassifyWithConfidence returns the tier and a confidence score (0-1)
func ClassifyWithConfidence(query string) (int, float64) {
	lower := strings.ToLower(query)

	complexScore := 0
	for _, indicator := range complexIndicators {
		if strings.Contains(lower, indicator) {
			complexScore++
		}
	}

	simpleScore := 0
	for _, indicator := range simpleIndicators {
		if strings.Contains(lower, indicator) {
			simpleScore++
		}
	}

	total := complexScore + simpleScore
	if total == 0 {
		return TierFast, 0.5 // Ambiguous
	}

	if complexScore > simpleScore {
		confidence := float64(complexScore) / float64(total)
		return TierDeep, confidence
	}

	confidence := float64(simpleScore) / float64(total)
	return TierFast, confidence
}
