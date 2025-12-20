// Package router provides intelligent query classification and routing to appropriate
// LLM models based on query complexity and task type.
//
// # Tier System
//
// The router uses a 4-tier classification system to match queries with appropriate models:
//
//   - Tier 1 (Fast): Simple queries, quick lookups, one-liners, format conversions
//   - Tier 2 (Deep): Complex multi-step tasks, code generation, refactoring
//   - Tier 3 (Analysis): Code review, reasoning, security analysis, trade-off evaluation
//   - Tier 4 (Writing): Documentation, prose, tutorials, commit messages
//
// # Classification Algorithm
//
// Queries are classified by scanning for indicator phrases in each category.
// Each matching phrase adds 1 point to that tier's score. Additional heuristics:
//   - Queries > 200 chars get +1 to Deep tier
//   - Multiple sentences (>2 periods or >1 newline) get +1 to Deep tier
//
// The tier with the highest score wins. In case of ties, priority order is:
// Analysis > Writing > Deep > Fast (specialized tiers preferred).
//
// # Routing Flow
//
//  1. Query arrives
//  2. ClassifyQuery() determines tier (or use forceTier to override)
//  3. Config lookup: tier -> model (e.g., Tier 1 -> qwen2.5-coder:3b)
//  4. Host selection: find online host with that model
//  5. Fallback chain if model not found:
//     a. Try tier's fallback model
//     b. Try any online host with fallback model
//     c. Try best available host with fallback model
//
// # Configuration
//
// Model-to-tier mappings are configured in ~/.clood/config.yaml:
//
//	routing:
//	  tier1_model: qwen2.5-coder:3b
//	  tier2_model: qwen2.5-coder:7b
//	  tier3_model: llama3.1:8b
//	  tier4_model: qwen2.5-coder:7b
//	  fallback: true
//
// # Usage
//
//	r := router.NewRouter(cfg)
//	result, err := r.Route("analyze this code for bugs", 0, "")
//	// result.Tier = TierAnalysis
//	// result.Model = "llama3.1:8b"
//	// result.Host = best available host
//	// result.Client = ready-to-use ollama.Client
package router

import (
	"strings"

	"github.com/dirtybirdnj/clood/internal/config"
	"github.com/dirtybirdnj/clood/internal/hosts"
	"github.com/dirtybirdnj/clood/internal/ollama"
)

// Tier constants - use config.Tier* constants for consistency
const (
	TierFast     = 1 // Simple queries -> fast model
	TierDeep     = 2 // Complex queries -> deep model
	TierAnalysis = 3 // Code review/reasoning -> analysis model
	TierWriting  = 4 // Documentation/prose -> writing model
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

// analysisIndicators suggest a query needs reasoning/review capabilities
var analysisIndicators = []string{
	// Code review
	"review",
	"analyze",
	"critique",
	"problems with",
	"what's wrong",
	"bug in",
	"issues in",
	"find bugs",
	"security issues",

	// Reasoning
	"explain why",
	"reason about",
	"think through",
	"edge cases",
	"potential issues",

	// Comparison
	"trade-offs",
	"pros and cons",
	"compare",
	"which is better",
}

// writingIndicators suggest a query needs documentation/prose capabilities
var writingIndicators = []string{
	// Documentation
	"write documentation",
	"document this",
	"write a readme",
	"pr description",
	"commit message for",
	"release notes",

	// Explanation prose
	"explain to",
	"describe in plain",
	"summarize",
	"eli5",
	"tutorial",

	// Technical writing
	"api docs",
	"docstring",
	"jsdoc",
	"godoc",
}

// ClassifyQuery determines which tier should handle a query
func ClassifyQuery(query string) int {
	lower := strings.ToLower(query)

	// Score each tier
	scores := map[int]int{
		TierFast:     0,
		TierDeep:     0,
		TierAnalysis: 0,
		TierWriting:  0,
	}

	// Check for analysis indicators (highest priority for reasoning tasks)
	for _, indicator := range analysisIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierAnalysis]++
		}
	}

	// Check for writing indicators
	for _, indicator := range writingIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierWriting]++
		}
	}

	// Check for complex indicators
	for _, indicator := range complexIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierDeep]++
		}
	}

	// Check for simple indicators
	for _, indicator := range simpleIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierFast]++
		}
	}

	// Length heuristic: very long queries are often complex
	if len(query) > 200 {
		scores[TierDeep]++
	}

	// Multiple sentences suggest complexity
	if strings.Count(query, ".") > 2 || strings.Count(query, "\n") > 1 {
		scores[TierDeep]++
	}

	// Find the tier with the highest score
	maxScore := 0
	bestTier := TierFast

	// Priority order: Analysis > Writing > Deep > Fast
	// This ensures specialized tiers are preferred when scores are equal
	for _, tier := range []int{TierAnalysis, TierWriting, TierDeep, TierFast} {
		if scores[tier] > maxScore {
			maxScore = scores[tier]
			bestTier = tier
		}
	}

	return bestTier
}

// ClassifyWithConfidence returns the tier and a confidence score (0-1)
func ClassifyWithConfidence(query string) (int, float64) {
	lower := strings.ToLower(query)

	// Score each tier
	scores := map[int]int{
		TierFast:     0,
		TierDeep:     0,
		TierAnalysis: 0,
		TierWriting:  0,
	}

	for _, indicator := range analysisIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierAnalysis]++
		}
	}

	for _, indicator := range writingIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierWriting]++
		}
	}

	for _, indicator := range complexIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierDeep]++
		}
	}

	for _, indicator := range simpleIndicators {
		if strings.Contains(lower, indicator) {
			scores[TierFast]++
		}
	}

	// Length heuristics
	if len(query) > 200 {
		scores[TierDeep]++
	}
	if strings.Count(query, ".") > 2 || strings.Count(query, "\n") > 1 {
		scores[TierDeep]++
	}

	// Find best tier and calculate confidence
	total := 0
	maxScore := 0
	bestTier := TierFast

	for _, tier := range []int{TierAnalysis, TierWriting, TierDeep, TierFast} {
		total += scores[tier]
		if scores[tier] > maxScore {
			maxScore = scores[tier]
			bestTier = tier
		}
	}

	if total == 0 {
		return TierFast, 0.5 // Ambiguous
	}

	confidence := float64(maxScore) / float64(total)
	return bestTier, confidence
}

// TierName returns a human-readable name for a tier
func TierName(tier int) string {
	switch tier {
	case TierFast:
		return "Fast"
	case TierDeep:
		return "Deep"
	case TierAnalysis:
		return "Analysis"
	case TierWriting:
		return "Writing"
	default:
		return "Unknown"
	}
}

// RouteResult contains the routing decision
type RouteResult struct {
	Tier       int
	Confidence float64
	Model      string
	Host       *hosts.HostStatus
	Client     *ollama.Client
}

// Router handles query routing to the appropriate host and model
type Router struct {
	config  *config.Config
	manager *hosts.Manager
}

// NewRouter creates a new router with the given config
func NewRouter(cfg *config.Config) *Router {
	mgr := hosts.NewManager()
	mgr.AddHosts(cfg.Hosts)

	return &Router{
		config:  cfg,
		manager: mgr,
	}
}

// Route determines the best host and model for a query
func (r *Router) Route(query string, forceTier int, forceModel string) (*RouteResult, error) {
	result := &RouteResult{}

	// Determine tier
	if forceTier > 0 {
		result.Tier = forceTier
		result.Confidence = 1.0
	} else {
		result.Tier, result.Confidence = ClassifyWithConfidence(query)
	}

	// Determine model
	if forceModel != "" {
		result.Model = forceModel
	} else {
		result.Model = r.config.GetTierModel(result.Tier)
	}

	// Find the best host with this model
	hostStatus := r.manager.GetHostWithModel(result.Model)
	if hostStatus != nil {
		result.Host = hostStatus
		result.Client = r.manager.GetClient(hostStatus.Host.Name)
		return result, nil
	}

	// Try tier fallback model if primary not found
	if r.config.Routing.Fallback {
		fallbackModel := r.config.GetTierFallback(result.Tier)
		if fallbackModel != "" {
			hostStatus = r.manager.GetHostWithModel(fallbackModel)
			if hostStatus != nil {
				result.Model = fallbackModel // Switch to fallback model
				result.Host = hostStatus
				result.Client = r.manager.GetClient(hostStatus.Host.Name)
				return result, nil
			}
		}

		// Last resort: try to find any online host with the fallback model
		bestHost := r.manager.GetBestHost()
		if bestHost != nil {
			result.Host = bestHost
			result.Client = r.manager.GetClient(bestHost.Host.Name)
			// Use fallback model if available, otherwise stick with original
			if fallbackModel != "" {
				result.Model = fallbackModel
			}
			return result, nil
		}
	}

	return result, nil
}

// GetManager returns the host manager
func (r *Router) GetManager() *hosts.Manager {
	return r.manager
}
