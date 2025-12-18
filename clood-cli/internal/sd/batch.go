package sd

import (
	"fmt"
	"path/filepath"
	"time"
)

// BatchConfig defines a set of variations to test against a single prompt.
// This is the catfight arena configuration.
type BatchConfig struct {
	Name        string        `json:"name" yaml:"name"`
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
	BasePrompt  *Prompt       `json:"prompt" yaml:"prompt"`
	Variations  []Variation   `json:"variations" yaml:"variations"`
	OutputDir   string        `json:"output_dir" yaml:"output_dir"`
	Timeout     time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Variation represents one contestant in the catfight.
type Variation struct {
	Name       string   `json:"name" yaml:"name"`             // e.g., "ghibli_high_weight"
	Checkpoint string   `json:"checkpoint" yaml:"checkpoint"` // Override checkpoint
	LoRAs      []LoRA   `json:"loras,omitempty" yaml:"loras,omitempty"`
	LoRAWeight *float64 `json:"lora_weight,omitempty" yaml:"lora_weight,omitempty"` // Override base LoRA weight
	Steps      *int     `json:"steps,omitempty" yaml:"steps,omitempty"`
	CFGScale   *float64 `json:"cfg_scale,omitempty" yaml:"cfg_scale,omitempty"`
	Sampler    string   `json:"sampler,omitempty" yaml:"sampler,omitempty"`
	Width      *int     `json:"width,omitempty" yaml:"width,omitempty"`
	Height     *int     `json:"height,omitempty" yaml:"height,omitempty"`
}

// BatchResult holds the output from a batch run.
type BatchResult struct {
	Config     *BatchConfig     `json:"config"`
	StartTime  time.Time        `json:"start_time"`
	EndTime    time.Time        `json:"end_time"`
	Results    []VariationResult `json:"results"`
	TotalTime  time.Duration    `json:"total_time"`
}

// VariationResult holds the output from one variation.
type VariationResult struct {
	Variation   Variation     `json:"variation"`
	OutputPath  string        `json:"output_path"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	GenerateTime time.Duration `json:"generate_time"`
	Metadata    ImageMetadata `json:"metadata"`
}

// ImageMetadata stores info about the generated image.
type ImageMetadata struct {
	Seed       int64   `json:"seed"`
	Steps      int     `json:"steps"`
	CFGScale   float64 `json:"cfg_scale"`
	Sampler    string  `json:"sampler"`
	Checkpoint string  `json:"checkpoint"`
	LoRAs      []LoRA  `json:"loras,omitempty"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Prompt     string  `json:"prompt"`
	Negative   string  `json:"negative"`
}

// NewBatchConfig creates a batch configuration for catfighting.
func NewBatchConfig(name string, prompt *Prompt) *BatchConfig {
	return &BatchConfig{
		Name:       name,
		BasePrompt: prompt,
		OutputDir:  filepath.Join("outputs", "batches", name),
		Timeout:    5 * time.Minute,
	}
}

// AddVariation adds a contestant to the catfight.
func (b *BatchConfig) AddVariation(v Variation) *BatchConfig {
	b.Variations = append(b.Variations, v)
	return b
}

// LoRAWeightSweep creates variations testing different LoRA weights.
// Perfect for finding the sweet spot.
func LoRAWeightSweep(loraName string, checkpoint string, weights []float64) []Variation {
	var variations []Variation
	for _, w := range weights {
		weight := w // capture for pointer
		variations = append(variations, Variation{
			Name:       fmt.Sprintf("%s_w%.1f", loraName, w),
			Checkpoint: checkpoint,
			LoRAs:      []LoRA{{Name: loraName, Weight: w}},
			LoRAWeight: &weight,
		})
	}
	return variations
}

// CheckpointSweep creates variations testing different checkpoints with same LoRA.
func CheckpointSweep(checkpoints []string, lora *LoRA) []Variation {
	var variations []Variation
	for _, ckpt := range checkpoints {
		v := Variation{
			Name:       ckpt,
			Checkpoint: ckpt,
		}
		if lora != nil {
			v.LoRAs = []LoRA{*lora}
		}
		variations = append(variations, v)
	}
	return variations
}

// GhibliCatfight returns a pre-configured batch for testing Ghibli styles.
func GhibliCatfight(subject string) *BatchConfig {
	prompt := GhibliPreset(subject)
	prompt.WithSeed(42069) // Fixed seed for fair comparison

	batch := NewBatchConfig("ghibli_catfight", prompt)
	batch.Description = fmt.Sprintf("Ghibli style catfight: %s", subject)

	// Weight sweep
	for _, w := range []float64{0.5, 0.7, 0.85, 1.0} {
		weight := w
		batch.AddVariation(Variation{
			Name:       fmt.Sprintf("ghibli_w%.2f", w),
			Checkpoint: "sd_xl_base_1.0.safetensors",
			LoRAs:      []LoRA{{Name: "ghibli_style_offset", Weight: w}},
			LoRAWeight: &weight,
		})
	}

	return batch
}

// PulpFictionGhibliCatfight - THE BRIEFCASE MOMENT
func PulpFictionGhibliCatfight() *BatchConfig {
	prompt := PulpFictionGhibliPreset(
		"jules winnfield and vincent vega",
		"opening a glowing briefcase in a diner booth, supernatural golden light illuminating their faces",
	)

	batch := NewBatchConfig("pulp_fiction_ghibli", prompt)
	batch.Description = "Say 'what' one more time - Miyazaki edition"

	// Different style approaches
	batch.AddVariation(Variation{
		Name:       "sdxl_ghibli_full",
		Checkpoint: "sd_xl_base_1.0.safetensors",
		LoRAs:      []LoRA{{Name: "ghibli_style_offset", Weight: 1.0}},
	})
	batch.AddVariation(Variation{
		Name:       "sdxl_ghibli_subtle",
		Checkpoint: "sd_xl_base_1.0.safetensors",
		LoRAs:      []LoRA{{Name: "ghibli_style_offset", Weight: 0.6}},
	})
	batch.AddVariation(Variation{
		Name:       "sd15_darksushi_ghibli",
		Checkpoint: "darkSushiMix_225D.safetensors",
		LoRAs:      []LoRA{{Name: "ghibli_style_offset", Weight: 0.8}},
	})
	batch.AddVariation(Variation{
		Name:       "sd15_miyazaki_lora",
		Checkpoint: "darkSushiMix_225D.safetensors",
		LoRAs:      []LoRA{{Name: "hayao_miyazaki_style", Weight: 0.9}},
	})

	return batch
}
