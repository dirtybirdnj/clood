// Package sd - The Mythical Aquarium
//
// In the vertical garden, the waters run deep.
// Fish are base models. Shrimp are LoRAs.
// Hybrids are merges. The reef is the ecosystem.

package sd

// FishSpecies represents a base model classification.
// Each species has different characteristics and habitat depth.
type FishSpecies string

const (
	// Pure Species - Base Models
	Fugu    FishSpecies = "fugu"    // SDXL - powerful, deadly if misconfigured
	Ayu     FishSpecies = "ayu"     // SD 1.5 - the classic, swims upstream
	Oarfish FishSpecies = "oarfish" // Flux - deep messenger, new paradigm
	Iwana   FishSpecies = "iwana"   // NoobAI - mountain newcomer, cold streams
	Unagi   FishSpecies = "unagi"   // Pony/specialized - slippery, transforms

	// Hybrids - Merged Models
	Splake     FishSpecies = "splake"      // Illustrious × Toon cross
	TigerTrout FishSpecies = "tiger_trout" // Illustrious × Realism
	Cutbow     FishSpecies = "cutbow"      // Illustrious × Niji
)

// ShrimpSpecies represents a LoRA classification.
// Shrimp attach to fish and modify their behavior.
type ShrimpSpecies string

const (
	// Shrimp - LoRAs
	SkunkCleaner ShrimpSpecies = "skunk_cleaner" // Style refinement, trusts the process
	CoralBanded  ShrimpSpecies = "coral_banded"  // High-quality, beautiful but territorial
	Harlequin    ShrimpSpecies = "harlequin"     // Specialized, only works with specific bases
	SexyShrimp   ShrimpSpecies = "sexy_shrimp"   // ...the name writes itself
	Peppermint   ShrimpSpecies = "peppermint"    // Fix/cleanup, eats the aiptasia (bad hands)
	FireShrimp   ShrimpSpecies = "fire_shrimp"   // Dramatic effect, makes everything pop
	GlassShrimp  ShrimpSpecies = "glass_shrimp"  // Basic/common, subtle effect
	PistolShrimp ShrimpSpecies = "pistol_shrimp" // Upscaler/detail, SNAP - massive impact
)

// Depth represents the vertical position in the garden.
type Depth string

const (
	Surface Depth = "surface" // Fast/Turbo models
	Mid     Depth = "mid"     // Standard models
	Deep    Depth = "deep"    // Heavy/Experimental
	Abyss   Depth = "abyss"   // The unknown, bleeding edge
)

// Fish represents a base model in the aquarium.
type Fish struct {
	Name        string      `json:"name" yaml:"name"`
	Species     FishSpecies `json:"species" yaml:"species"`
	Depth       Depth       `json:"depth" yaml:"depth"`
	SizeGB      float64     `json:"size_gb" yaml:"size_gb"`
	Lineage     []string    `json:"lineage,omitempty" yaml:"lineage,omitempty"` // Parent species for hybrids
	Checkpoint  string      `json:"checkpoint" yaml:"checkpoint"`               // Actual filename
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
}

// Shrimp represents a LoRA in the reef.
type Shrimp struct {
	Name          string        `json:"name" yaml:"name"`
	Species       ShrimpSpecies `json:"species" yaml:"species"`
	HostFish      []FishSpecies `json:"host_fish,omitempty" yaml:"host_fish,omitempty"` // Compatible bases
	DefaultWeight float64       `json:"default_weight" yaml:"default_weight"`
	Filename      string        `json:"filename" yaml:"filename"`
	TriggerWords  []string      `json:"trigger_words,omitempty" yaml:"trigger_words,omitempty"`
	Description   string        `json:"description,omitempty" yaml:"description,omitempty"`
}

// Aquarium holds the complete ecosystem.
type Aquarium struct {
	Fish   []Fish   `json:"fish" yaml:"fish"`
	Shrimp []Shrimp `json:"shrimp" yaml:"shrimp"`
}

// ExampleAquarium returns a sample ecosystem based on known models.
func ExampleAquarium() *Aquarium {
	return &Aquarium{
		Fish: []Fish{
			{
				Name:        "Illustrious XL",
				Species:     Fugu,
				Depth:       Mid,
				SizeGB:      6.5,
				Checkpoint:  "illustriousXL_v01.safetensors",
				Description: "The deadly beautiful one. Handle with care.",
			},
			{
				Name:        "NoobAI XL",
				Species:     Iwana,
				Depth:       Mid,
				SizeGB:      6.6,
				Checkpoint:  "noobaiXLNAIXL_vPred10Version.safetensors",
				Description: "Mountain newcomer, thrives in cold streams.",
			},
			{
				Name:        "SD 1.5 Base",
				Species:     Ayu,
				Depth:       Surface,
				SizeGB:      4.0,
				Checkpoint:  "v1-5-pruned-emaonly.safetensors",
				Description: "The classic. Swims upstream against the current.",
			},
			{
				Name:        "Flux Dev",
				Species:     Oarfish,
				Depth:       Deep,
				SizeGB:      23.0,
				Checkpoint:  "realDream_fluxDevV4.safetensors",
				Description: "Messenger from the Dragon Palace. The deep one.",
			},
			{
				Name:        "IllustriousToonMix",
				Species:     Splake,
				Depth:       Mid,
				SizeGB:      6.5,
				Lineage:     []string{"illustrious", "toon"},
				Checkpoint:  "illustrioustoonMIX_v40.safetensors",
				Description: "Hybrid: Illustrious × Toon. Best of both waters.",
			},
			{
				Name:        "zImage Turbo",
				Species:     Ayu,
				Depth:       Surface,
				SizeGB:      0.08,
				Checkpoint:  "zImage_turbo.safetensors",
				Description: "Fast little swimmer. Speed over complexity.",
			},
		},
		Shrimp: []Shrimp{
			{
				Name:          "Ghibli Style",
				Species:       SkunkCleaner,
				HostFish:      []FishSpecies{Fugu, Ayu},
				DefaultWeight: 0.8,
				Filename:      "ghibli_style_offset.safetensors",
				TriggerWords:  []string{"ghibli style", "studio ghibli"},
				Description:   "Cleans the output into Miyazaki dreams.",
			},
			{
				Name:          "Detail Tweaker",
				Species:       PistolShrimp,
				DefaultWeight: 0.5,
				Filename:      "detail_tweaker.safetensors",
				Description:   "SNAP. Adds massive detail impact.",
			},
			{
				Name:          "Bad Hands Fix",
				Species:       Peppermint,
				DefaultWeight: 0.3,
				Filename:      "bad_hands_fix.safetensors",
				Description:   "Eats the aiptasia. Fixes the cursed hands.",
			},
		},
	}
}

// GetFishByCheckpoint finds a fish by its checkpoint filename.
func (a *Aquarium) GetFishByCheckpoint(checkpoint string) *Fish {
	for i := range a.Fish {
		if a.Fish[i].Checkpoint == checkpoint {
			return &a.Fish[i]
		}
	}
	return nil
}

// GetShrimpByFilename finds a shrimp by its filename.
func (a *Aquarium) GetShrimpByFilename(filename string) *Shrimp {
	for i := range a.Shrimp {
		if a.Shrimp[i].Filename == filename {
			return &a.Shrimp[i]
		}
	}
	return nil
}

// CompatibleShrimp returns all shrimp that can attach to a fish species.
func (a *Aquarium) CompatibleShrimp(species FishSpecies) []Shrimp {
	var compatible []Shrimp
	for _, s := range a.Shrimp {
		if len(s.HostFish) == 0 {
			// Universal shrimp
			compatible = append(compatible, s)
			continue
		}
		for _, host := range s.HostFish {
			if host == species {
				compatible = append(compatible, s)
				break
			}
		}
	}
	return compatible
}
