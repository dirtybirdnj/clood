package focus

import (
	"strings"
	"unicode"
)

// Guardian implements focus drift detection
// Named after Gamera-kun, the slow tortoise who guards against "stupid faster"
type Guardian struct {
	Goal             string
	Keywords         []string
	DriftCount       int
	Threshold        int
	RecentMessages   []string
	MaxRecentHistory int
}

// DriftResult contains the analysis of a message
type DriftResult struct {
	IsDrift    bool
	Confidence float64
	Keywords   []string // Which goal keywords were found
	Message    string   // The Gamera-kun message
}

// NewGuardian creates a focus guardian with default settings
func NewGuardian(goal string) *Guardian {
	g := &Guardian{
		Goal:             goal,
		Keywords:         extractKeywords(goal),
		DriftCount:       0,
		Threshold:        3,  // Warn after 3 drifted messages
		MaxRecentHistory: 5,  // Track last 5 messages for pattern
		RecentMessages:   []string{},
	}
	return g
}

// CheckMessage analyzes if a message drifts from the goal
func (g *Guardian) CheckMessage(message string) DriftResult {
	if g.Goal == "" {
		return DriftResult{IsDrift: false}
	}

	// Extract keywords from message
	msgKeywords := extractKeywords(message)

	// Find overlapping keywords
	matches := []string{}
	for _, goalKw := range g.Keywords {
		for _, msgKw := range msgKeywords {
			if strings.EqualFold(goalKw, msgKw) ||
			   strings.Contains(strings.ToLower(msgKw), strings.ToLower(goalKw)) ||
			   strings.Contains(strings.ToLower(goalKw), strings.ToLower(msgKw)) {
				matches = append(matches, goalKw)
				break
			}
		}
	}

	// Calculate drift
	confidence := float64(len(matches)) / float64(len(g.Keywords))
	isDrift := len(matches) == 0

	// Track recent messages
	g.RecentMessages = append(g.RecentMessages, message)
	if len(g.RecentMessages) > g.MaxRecentHistory {
		g.RecentMessages = g.RecentMessages[1:]
	}

	// Update drift counter
	if isDrift {
		g.DriftCount++
	} else {
		g.DriftCount = 0 // Reset on relevant message
	}

	result := DriftResult{
		IsDrift:    isDrift,
		Confidence: confidence,
		Keywords:   matches,
	}

	// Only warn if threshold reached
	if isDrift && g.DriftCount >= g.Threshold {
		result.Message = g.buildWarningMessage(message)
	}

	return result
}

// buildWarningMessage creates the Gamera-kun warning
func (g *Guardian) buildWarningMessage(message string) string {
	// Extract what the user seems to be talking about
	newTopic := summarizeTopic(message)

	return newTopic
}

// summarizeTopic extracts a brief description of the message topic
func summarizeTopic(message string) string {
	keywords := extractKeywords(message)
	if len(keywords) == 0 {
		return "something else"
	}
	if len(keywords) > 3 {
		keywords = keywords[:3]
	}
	return strings.Join(keywords, ", ")
}

// Reset clears drift tracking (e.g., when user confirms they want to continue)
func (g *Guardian) Reset() {
	g.DriftCount = 0
	g.RecentMessages = []string{}
}

// UpdateGoal changes the focus goal
func (g *Guardian) UpdateGoal(newGoal string) {
	g.Goal = newGoal
	g.Keywords = extractKeywords(newGoal)
	g.Reset()
}

// extractKeywords pulls meaningful words from text
func extractKeywords(text string) []string {
	// Common words to ignore
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "can": true,
		"i": true, "you": true, "he": true, "she": true, "it": true,
		"we": true, "they": true, "me": true, "him": true, "her": true,
		"us": true, "them": true, "my": true, "your": true, "his": true,
		"its": true, "our": true, "their": true,
		"this": true, "that": true, "these": true, "those": true,
		"what": true, "which": true, "who": true, "whom": true,
		"and": true, "or": true, "but": true, "if": true, "then": true,
		"so": true, "as": true, "of": true, "at": true, "by": true,
		"for": true, "with": true, "about": true, "to": true, "from": true,
		"in": true, "on": true, "up": true, "out": true, "into": true,
		"also": true, "just": true, "too": true, "very": true, "really": true,
		"want": true, "need": true, "like": true, "please": true, "help": true,
		"let": true, "now": true, "how": true, "why": true, "when": true,
		"add": true, "fix": true, "make": true, "get": true, "put": true,
	}

	// Split on non-alphanumeric
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	// Filter and collect keywords
	keywords := []string{}
	seen := map[string]bool{}

	for _, word := range words {
		lower := strings.ToLower(word)
		if len(lower) < 3 {
			continue
		}
		if stopWords[lower] {
			continue
		}
		if seen[lower] {
			continue
		}
		seen[lower] = true
		keywords = append(keywords, lower)
	}

	return keywords
}

// GetStatus returns a summary of the guardian's state
func (g *Guardian) GetStatus() string {
	if g.Goal == "" {
		return "No goal set"
	}
	if g.DriftCount == 0 {
		return "On track"
	}
	if g.DriftCount < g.Threshold {
		return "Wandering slightly"
	}
	return "Drifting from goal"
}
