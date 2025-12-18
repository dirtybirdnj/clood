// Package sd provides tools for orchestrating Stable Diffusion image generation
// through ComfyUI workflows and prompt engineering via LLMs.
package sd

import (
	"fmt"
	"strings"
)

// Prompt represents a complete SD prompt with positive and negative components.
type Prompt struct {
	Positive string   `json:"positive" yaml:"positive"`
	Negative string   `json:"negative" yaml:"negative"`
	LoRAs    []LoRA   `json:"loras,omitempty" yaml:"loras,omitempty"`
	Seed     int64    `json:"seed,omitempty" yaml:"seed,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"` // For organization/search
}

// LoRA represents a LoRA model to apply with its weight.
type LoRA struct {
	Name   string  `json:"name" yaml:"name"`     // e.g., "ghibli_style_offset"
	Weight float64 `json:"weight" yaml:"weight"` // typically 0.0-1.0
}

// DefaultNegative provides common negative prompt elements.
var DefaultNegative = []string{
	"bad anatomy",
	"bad hands",
	"blurry",
	"watermark",
	"signature",
	"text",
	"low quality",
	"worst quality",
}

// NewPrompt creates a prompt with sensible defaults.
func NewPrompt(positive string) *Prompt {
	return &Prompt{
		Positive: positive,
		Negative: strings.Join(DefaultNegative, ", "),
		Seed:     -1, // Random seed
	}
}

// WithNegative sets or appends to the negative prompt.
func (p *Prompt) WithNegative(negative string) *Prompt {
	p.Negative = negative
	return p
}

// WithLoRA adds a LoRA to the prompt.
func (p *Prompt) WithLoRA(name string, weight float64) *Prompt {
	p.LoRAs = append(p.LoRAs, LoRA{Name: name, Weight: weight})
	return p
}

// WithSeed sets a specific seed for reproducibility.
func (p *Prompt) WithSeed(seed int64) *Prompt {
	p.Seed = seed
	return p
}

// FormatPositive returns the positive prompt with LoRA syntax injected.
// Example: "<lora:ghibli_style:0.8> a tortoise with spectacles, ghibli style"
func (p *Prompt) FormatPositive() string {
	var parts []string

	// Add LoRA tags first
	for _, lora := range p.LoRAs {
		parts = append(parts, fmt.Sprintf("<lora:%s:%.2f>", lora.Name, lora.Weight))
	}

	parts = append(parts, p.Positive)
	return strings.Join(parts, " ")
}

// PromptRequest is sent to an LLM to generate/enhance a prompt.
type PromptRequest struct {
	Description string `json:"description"`          // Natural language description
	Style       string `json:"style,omitempty"`      // e.g., "ghibli", "realistic", "anime"
	Subject     string `json:"subject,omitempty"`    // Main subject focus
	Mood        string `json:"mood,omitempty"`       // e.g., "whimsical", "dark", "serene"
	AspectRatio string `json:"aspect_ratio,omitempty"` // e.g., "16:9", "1:1", "9:16"
}

// PromptResponse is returned by an LLM after prompt generation.
type PromptResponse struct {
	Prompt      Prompt   `json:"prompt"`
	Suggestions []string `json:"suggestions,omitempty"` // Alternative approaches
	Reasoning   string   `json:"reasoning,omitempty"`   // Why these choices
}

// GhibliPreset returns a prompt pre-configured for Studio Ghibli style.
func GhibliPreset(subject string) *Prompt {
	return NewPrompt(fmt.Sprintf("%s, studio ghibli style, hayao miyazaki, watercolor, soft lighting, whimsical, detailed background", subject)).
		WithLoRA("ghibli_style_offset", 0.8).
		WithNegative("realistic, photographic, 3d render, cgi, bad anatomy, blurry, watermark")
}

// PulpFictionGhibliPreset returns the crossover preset - the briefcase moment.
func PulpFictionGhibliPreset(character string, scene string) *Prompt {
	positive := fmt.Sprintf("%s from pulp fiction, %s, studio ghibli style, hayao miyazaki, "+
		"watercolor background, soft dramatic lighting, 1990s aesthetic reimagined as anime, "+
		"detailed character design, expressive faces", character, scene)

	return NewPrompt(positive).
		WithLoRA("ghibli_style_offset", 0.85).
		WithSeed(42069). // The sacred seed
		WithNegative("realistic, photographic, 3d render, bad anatomy, blurry, watermark, text, western cartoon style")
}
