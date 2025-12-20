// Package system provides hardware detection and model classification.
package system

import (
	"strings"
)

// ModelCategory represents a model's primary purpose
type ModelCategory string

const (
	CategoryCoding    ModelCategory = "coding"
	CategoryReasoning ModelCategory = "reasoning"
	CategoryVision    ModelCategory = "vision"
	CategoryGeneral   ModelCategory = "general"
	CategoryScience   ModelCategory = "science"
	CategoryChat      ModelCategory = "chat"
)

// CategoryInfo describes a model category
type CategoryInfo struct {
	Name        string
	Description string
	Emoji       string
}

// Categories maps category to info
var Categories = map[ModelCategory]CategoryInfo{
	CategoryCoding:    {Name: "Coding", Description: "Code generation, review, and debugging", Emoji: "💻"},
	CategoryReasoning: {Name: "Reasoning", Description: "Complex analysis and multi-step thinking", Emoji: "🧠"},
	CategoryVision:    {Name: "Vision", Description: "Image analysis and understanding", Emoji: "👁️"},
	CategoryGeneral:   {Name: "General", Description: "General purpose text tasks", Emoji: "📝"},
	CategoryScience:   {Name: "Science", Description: "Scientific and mathematical tasks", Emoji: "🔬"},
	CategoryChat:      {Name: "Chat", Description: "Conversational AI", Emoji: "💬"},
}

// ModelClassification holds model metadata
type ModelClassification struct {
	Category    ModelCategory
	Strengths   []string
	BestFor     string
	MinVRAM     float64 // Minimum VRAM in GB for comfortable use
}

// KnownModels maps model family prefixes to classifications
var KnownModels = map[string]ModelClassification{
	// Coding models
	"qwen2.5-coder":   {Category: CategoryCoding, BestFor: "Code generation & completion", MinVRAM: 2},
	"deepseek-coder":  {Category: CategoryCoding, BestFor: "Code understanding & debugging", MinVRAM: 4},
	"codellama":       {Category: CategoryCoding, BestFor: "Code completion & infilling", MinVRAM: 4},
	"starcoder":       {Category: CategoryCoding, BestFor: "Multi-language code generation", MinVRAM: 4},
	"codegemma":       {Category: CategoryCoding, BestFor: "Fast code completion", MinVRAM: 2},
	"granite-code":    {Category: CategoryCoding, BestFor: "Enterprise code tasks", MinVRAM: 2},
	"yi-coder":        {Category: CategoryCoding, BestFor: "Lightweight code assistance", MinVRAM: 1},
	"codestral":       {Category: CategoryCoding, BestFor: "Advanced code reasoning", MinVRAM: 12},

	// Reasoning models
	"deepseek-r1":     {Category: CategoryReasoning, BestFor: "Complex multi-step reasoning", MinVRAM: 8},
	"phi3":            {Category: CategoryReasoning, BestFor: "Efficient reasoning tasks", MinVRAM: 2},
	"phi":             {Category: CategoryReasoning, BestFor: "Small but capable reasoning", MinVRAM: 2},
	"qwen2.5":         {Category: CategoryReasoning, BestFor: "Balanced reasoning & coding", MinVRAM: 4},

	// Vision models
	"llava":           {Category: CategoryVision, BestFor: "Image understanding & description", MinVRAM: 4},
	"llava-phi3":      {Category: CategoryVision, BestFor: "Efficient image analysis", MinVRAM: 3},
	"moondream":       {Category: CategoryVision, BestFor: "Lightweight vision tasks", MinVRAM: 2},
	"bakllava":        {Category: CategoryVision, BestFor: "Visual question answering", MinVRAM: 4},

	// General models
	"llama3.1":        {Category: CategoryGeneral, BestFor: "General purpose & tool use", MinVRAM: 4},
	"llama3":          {Category: CategoryGeneral, BestFor: "General purpose tasks", MinVRAM: 4},
	"llama2":          {Category: CategoryGeneral, BestFor: "General text generation", MinVRAM: 4},
	"mistral":         {Category: CategoryGeneral, BestFor: "Fast general purpose", MinVRAM: 4},
	"mixtral":         {Category: CategoryGeneral, BestFor: "High-quality MoE inference", MinVRAM: 24},
	"gemma":           {Category: CategoryGeneral, BestFor: "Efficient general tasks", MinVRAM: 2},
	"gemma2":          {Category: CategoryGeneral, BestFor: "Improved general tasks", MinVRAM: 2},
	"falcon":          {Category: CategoryGeneral, BestFor: "Open general purpose", MinVRAM: 1},
	"tinyllama":       {Category: CategoryGeneral, BestFor: "Ultra-fast simple tasks", MinVRAM: 1},
	"stablelm":        {Category: CategoryGeneral, BestFor: "Stable text generation", MinVRAM: 1},

	// Tool use
	"llama3-groq-tool-use": {Category: CategoryGeneral, BestFor: "Function calling & tools", MinVRAM: 4},
}

// ClassifyModel returns the classification for a model name
func ClassifyModel(modelName string) ModelClassification {
	// Normalize name (remove tag like :7b, :latest)
	baseName := modelName
	if idx := strings.Index(modelName, ":"); idx > 0 {
		baseName = modelName[:idx]
	}
	baseName = strings.ToLower(baseName)

	// Try exact match first
	if class, ok := KnownModels[baseName]; ok {
		return class
	}

	// Try prefix matching
	for prefix, class := range KnownModels {
		if strings.HasPrefix(baseName, prefix) {
			return class
		}
	}

	// Default to general
	return ModelClassification{
		Category: CategoryGeneral,
		BestFor:  "General purpose",
		MinVRAM:  2,
	}
}

// GetCategoryInfo returns info about a category
func GetCategoryInfo(cat ModelCategory) CategoryInfo {
	if info, ok := Categories[cat]; ok {
		return info
	}
	return CategoryInfo{Name: "Unknown", Description: "Unknown category", Emoji: "❓"}
}

// RecommendedByCategory returns the best model for each category given available VRAM
func RecommendedByCategory(availableVRAM float64) map[ModelCategory][]string {
	recommendations := make(map[ModelCategory][]string)

	// Coding recommendations by VRAM
	if availableVRAM >= 12 {
		recommendations[CategoryCoding] = []string{"codestral:22b", "qwen2.5-coder:14b"}
	} else if availableVRAM >= 6 {
		recommendations[CategoryCoding] = []string{"qwen2.5-coder:7b", "deepseek-coder:6.7b"}
	} else if availableVRAM >= 2 {
		recommendations[CategoryCoding] = []string{"qwen2.5-coder:3b", "yi-coder:1.5b"}
	} else {
		recommendations[CategoryCoding] = []string{"tinyllama"}
	}

	// Reasoning recommendations
	if availableVRAM >= 8 {
		recommendations[CategoryReasoning] = []string{"deepseek-r1:14b", "phi3:14b"}
	} else if availableVRAM >= 4 {
		recommendations[CategoryReasoning] = []string{"phi3:3.8b", "qwen2.5:7b"}
	} else {
		recommendations[CategoryReasoning] = []string{"phi3:mini", "tinyllama"}
	}

	// Vision recommendations
	if availableVRAM >= 4 {
		recommendations[CategoryVision] = []string{"llava:13b", "llava-phi3"}
	} else if availableVRAM >= 2 {
		recommendations[CategoryVision] = []string{"moondream", "llava-phi3"}
	}

	// General recommendations
	if availableVRAM >= 8 {
		recommendations[CategoryGeneral] = []string{"llama3.1:8b", "mistral:7b"}
	} else if availableVRAM >= 4 {
		recommendations[CategoryGeneral] = []string{"llama3.1:8b", "gemma2:2b"}
	} else {
		recommendations[CategoryGeneral] = []string{"tinyllama", "stablelm2:1.6b"}
	}

	return recommendations
}
