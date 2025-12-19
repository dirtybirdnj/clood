package sd

import (
	"fmt"
	"strings"
)

// StackLayer represents one layer of the SD generation stack.
type StackLayer int

const (
	LayerHardware StackLayer = iota
	LayerCheckpoint
	LayerLoRA
	LayerVAE
	LayerSampler
	LayerPrompt
	LayerSeed
	LayerPostProcess
)

func (l StackLayer) String() string {
	names := []string{
		"Hardware",
		"Checkpoint",
		"LoRA",
		"VAE",
		"Sampler",
		"Prompt",
		"Seed",
		"PostProcess",
	}
	if int(l) < len(names) {
		return names[l]
	}
	return "Unknown"
}

// MatchLevel indicates how well a local resource matches a required one.
type MatchLevel int

const (
	MatchNone    MatchLevel = iota // No match available
	MatchPartial                   // Same family/type but different version
	MatchSimilar                   // Close enough to substitute
	MatchExact                     // Perfect match
)

func (m MatchLevel) String() string {
	switch m {
	case MatchNone:
		return "None"
	case MatchPartial:
		return "Partial"
	case MatchSimilar:
		return "Similar"
	case MatchExact:
		return "Exact"
	}
	return "Unknown"
}

func (m MatchLevel) Symbol() string {
	switch m {
	case MatchNone:
		return "✗"
	case MatchPartial:
		return "◐"
	case MatchSimilar:
		return "○"
	case MatchExact:
		return "✓"
	}
	return "?"
}

// Recovery percentage based on match level.
func (m MatchLevel) RecoveryRate() float64 {
	switch m {
	case MatchNone:
		return 0.0
	case MatchPartial:
		return 0.4
	case MatchSimilar:
		return 0.75
	case MatchExact:
		return 1.0
	}
	return 0.0
}

// GenerationStack represents a complete SD generation configuration.
type GenerationStack struct {
	// Source metadata
	Source    *GenerationSource
	SourceURL string

	// Stack layers
	Hardware    HardwareSpec
	Checkpoint  CheckpointSpec
	LoRAs       []LoRASpec
	VAE         VAESpec
	Sampler     SamplerSpec
	Prompt      PromptSpec
	Seed        int64
	PostProcess []PostProcessSpec
}

// HardwareSpec defines hardware requirements.
type HardwareSpec struct {
	MinVRAM       int64  // Minimum VRAM in bytes
	RecommendedGPU string // e.g., "RTX 3080" or "Apple M1"
	Backend       string // "cuda", "mps", "rocm", "cpu"
}

// CheckpointSpec defines a checkpoint model requirement.
type CheckpointSpec struct {
	Name       string
	Hash       string
	BaseModel  string   // SD1.5, SDXL, Flux
	Aliases    []string // Alternative names
	SourceURL  string
	FileSize   int64 // For VRAM estimation
}

// LoRASpec defines a LoRA requirement.
type LoRASpec struct {
	Name       string
	Hash       string
	Weight     float64
	BaseModel  string
	SourceURL  string
	Required   bool   // Some LoRAs are critical, others enhance
}

// VAESpec defines VAE requirement.
type VAESpec struct {
	Name      string
	Hash      string
	SourceURL string
}

// SamplerSpec defines sampling configuration.
type SamplerSpec struct {
	Name       string
	Scheduler  string
	Steps      int
	CFGScale   float64
}

// PromptSpec contains prompt information.
type PromptSpec struct {
	Positive   string
	Negative   string
	ClipSkip   int
	StyleTags  []string // Extracted style keywords
}

// PostProcessSpec defines post-processing steps.
type PostProcessSpec struct {
	Type      string // "upscale", "facefix", "detailer"
	Model     string
	Strength  float64
}

// LayerAnalysis is the result of comparing one layer against local inventory.
type LayerAnalysis struct {
	Layer        StackLayer
	Required     interface{} // What the source needs
	Available    interface{} // What we have locally
	Match        MatchLevel
	Recovery     float64     // 0.0 - 1.0
	Workaround   string      // How to compensate for mismatch
	DownloadURL  string      // Where to get missing piece
	DownloadSize int64       // Bytes
}

// StackAnalysis is the complete analysis of a generation stack.
type StackAnalysis struct {
	Source   *GenerationSource
	Layers   []LayerAnalysis
	Overall  OverallAnalysis
	Options  []RemixOption
}

// OverallAnalysis summarizes the stack comparison.
type OverallAnalysis struct {
	TotalLayers     int
	MatchedLayers   int
	PartialLayers   int
	MissingLayers   int
	OverallRecovery float64 // Weighted average of layer recoveries
	CanGenerate     bool    // True if we have enough to attempt generation
	BlockingIssues  []string
	Warnings        []string
}

// RemixOption presents a choice to the user.
type RemixOption struct {
	ID          string
	Label       string
	Description string
	Recovery    float64 // Expected recovery with this option
	Downloads   []DownloadItem
}

// DownloadItem represents something that can be downloaded.
type DownloadItem struct {
	Name     string
	URL      string
	Size     int64
	Type     string // "checkpoint", "lora", "vae"
}

// NewStackFromSource creates a GenerationStack from parsed source data.
func NewStackFromSource(gs *GenerationSource) *GenerationStack {
	stack := &GenerationStack{
		Source:    gs,
		SourceURL: gs.SourceURL,
	}

	// Map checkpoint
	if gs.Checkpoint.Name != "" {
		stack.Checkpoint = CheckpointSpec{
			Name:      gs.Checkpoint.Name,
			Hash:      gs.Checkpoint.Hash,
			BaseModel: gs.Checkpoint.BaseModel,
			SourceURL: gs.Checkpoint.SourceURL,
		}
	}

	// Map LoRAs
	for _, lora := range gs.LoRAs {
		stack.LoRAs = append(stack.LoRAs, LoRASpec{
			Name:      lora.Name,
			Hash:      lora.Hash,
			Weight:    lora.Weight,
			BaseModel: lora.BaseModel,
			SourceURL: lora.SourceURL,
			Required:  lora.Weight > 0.5, // High weight = more critical
		})
	}

	// Map VAE
	if gs.VAE.Name != "" {
		stack.VAE = VAESpec{
			Name:      gs.VAE.Name,
			Hash:      gs.VAE.Hash,
			SourceURL: gs.VAE.SourceURL,
		}
	}

	// Map sampler
	stack.Sampler = SamplerSpec{
		Name:     gs.Sampler.Name,
		Steps:    gs.Sampler.Steps,
		CFGScale: gs.Sampler.CFGScale,
	}

	// Map prompt
	stack.Prompt = PromptSpec{
		Positive:  gs.Prompt,
		Negative:  gs.NegativePrompt,
		StyleTags: extractStyleTags(gs.Prompt),
	}

	// Map seed
	stack.Seed = gs.Seed

	// Estimate hardware requirements
	stack.Hardware = estimateHardwareRequirements(stack)

	return stack
}

// estimateHardwareRequirements guesses VRAM needs based on model type.
func estimateHardwareRequirements(stack *GenerationStack) HardwareSpec {
	var minVRAM int64

	// Base requirement by model type
	switch normalizeBaseModel(stack.Checkpoint.BaseModel) {
	case "SDXL":
		minVRAM = 8 * 1024 * 1024 * 1024 // 8GB
	case "Flux":
		minVRAM = 12 * 1024 * 1024 * 1024 // 12GB for Flux
	case "SD1.5":
		minVRAM = 4 * 1024 * 1024 * 1024 // 4GB
	default:
		minVRAM = 6 * 1024 * 1024 * 1024 // Conservative default
	}

	// Add LoRA overhead (~10% per LoRA)
	loraOverhead := int64(float64(minVRAM) * 0.1 * float64(len(stack.LoRAs)))
	minVRAM += loraOverhead

	return HardwareSpec{
		MinVRAM: minVRAM,
	}
}

// extractStyleTags pulls common style keywords from prompt.
func extractStyleTags(prompt string) []string {
	styles := []string{
		"anime", "photorealistic", "oil painting", "watercolor",
		"concept art", "digital art", "3d render", "pixel art",
		"sketch", "illustration", "cinematic", "portrait",
		"landscape", "ghibli", "cyberpunk", "fantasy",
		"sci-fi", "horror", "cute", "dark", "bright",
	}

	prompt = strings.ToLower(prompt)
	var found []string
	for _, style := range styles {
		if strings.Contains(prompt, style) {
			found = append(found, style)
		}
	}

	return found
}

// normalizeBaseModel standardizes base model names.
func normalizeBaseModel(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "xl"):
		return "SDXL"
	case strings.Contains(name, "flux"):
		return "Flux"
	case strings.Contains(name, "1.5") || strings.Contains(name, "1_5"):
		return "SD1.5"
	case strings.Contains(name, "2."):
		return "SD2.x"
	default:
		return name
	}
}

// AnalyzeStack compares a generation stack against local inventory.
func AnalyzeStack(stack *GenerationStack, inventory *LocalInventory) *StackAnalysis {
	analysis := &StackAnalysis{
		Source: stack.Source,
	}

	// Analyze each layer
	analysis.Layers = append(analysis.Layers, analyzeHardware(stack.Hardware, inventory))
	analysis.Layers = append(analysis.Layers, analyzeCheckpoint(stack.Checkpoint, inventory))

	for i, lora := range stack.LoRAs {
		loraAnalysis := analyzeLoRA(lora, inventory)
		loraAnalysis.Layer = LayerLoRA // Use base layer type
		// Annotate which LoRA this is
		if loraAnalysis.Workaround == "" && loraAnalysis.Match != MatchExact {
			loraAnalysis.Workaround = fmt.Sprintf("LoRA %d/%d", i+1, len(stack.LoRAs))
		}
		analysis.Layers = append(analysis.Layers, loraAnalysis)
	}

	analysis.Layers = append(analysis.Layers, analyzeVAE(stack.VAE, inventory))
	analysis.Layers = append(analysis.Layers, analyzeSampler(stack.Sampler, inventory))
	analysis.Layers = append(analysis.Layers, analyzePrompt(stack.Prompt))
	analysis.Layers = append(analysis.Layers, analyzeSeed(stack.Seed))

	// Calculate overall metrics
	analysis.Overall = calculateOverall(analysis.Layers, stack)

	// Generate remix options
	analysis.Options = generateOptions(analysis, stack, inventory)

	return analysis
}

func analyzeHardware(req HardwareSpec, inv *LocalInventory) LayerAnalysis {
	analysis := LayerAnalysis{
		Layer:    LayerHardware,
		Required: req,
	}

	if inv == nil || inv.Hardware.TotalVRAM == 0 {
		analysis.Match = MatchPartial
		analysis.Recovery = 0.5
		analysis.Workaround = "Hardware info not available - will attempt generation"
		return analysis
	}

	analysis.Available = inv.Hardware

	if inv.Hardware.TotalVRAM >= req.MinVRAM {
		analysis.Match = MatchExact
		analysis.Recovery = 1.0
	} else if inv.Hardware.TotalVRAM >= req.MinVRAM/2 {
		analysis.Match = MatchPartial
		analysis.Recovery = 0.5
		analysis.Workaround = "May need reduced resolution or steps"
	} else {
		analysis.Match = MatchNone
		analysis.Recovery = 0.1
		analysis.Workaround = "Insufficient VRAM - consider cloud generation"
	}

	return analysis
}

func analyzeCheckpoint(req CheckpointSpec, inv *LocalInventory) LayerAnalysis {
	analysis := LayerAnalysis{
		Layer:    LayerCheckpoint,
		Required: req,
	}

	if inv == nil {
		analysis.Match = MatchNone
		analysis.Recovery = 0.0
		analysis.DownloadURL = req.SourceURL
		return analysis
	}

	// Try to find matching checkpoint
	match := inv.FindCheckpoint(req)
	if match != nil {
		analysis.Available = match.Checkpoint
		analysis.Match = match.Level
		analysis.Recovery = match.Level.RecoveryRate()
		if match.Level != MatchExact {
			analysis.Workaround = fmt.Sprintf("Using %s instead of %s", match.Checkpoint.Name, req.Name)
		}
	} else {
		analysis.Match = MatchNone
		analysis.Recovery = 0.0
		analysis.DownloadURL = req.SourceURL

		// Suggest alternative if we have something in the same family
		if req.BaseModel != "" {
			for _, ckpt := range inv.Checkpoints {
				if normalizeBaseModel(ckpt.BaseModel) == normalizeBaseModel(req.BaseModel) {
					analysis.Workaround = fmt.Sprintf("No exact match, but %s is same model family", ckpt.Name)
					analysis.Match = MatchPartial
					analysis.Recovery = 0.4
					analysis.Available = ckpt
					break
				}
			}
		}
	}

	return analysis
}

func analyzeLoRA(req LoRASpec, inv *LocalInventory) LayerAnalysis {
	analysis := LayerAnalysis{
		Layer:    LayerLoRA,
		Required: req,
	}

	if inv == nil {
		analysis.Match = MatchNone
		analysis.Recovery = 0.0
		analysis.DownloadURL = req.SourceURL
		return analysis
	}

	match := inv.FindLoRA(req)
	if match != nil {
		analysis.Available = match.LoRA
		analysis.Match = match.Level
		analysis.Recovery = match.Level.RecoveryRate()
	} else {
		// LoRA missing - but we can compensate with prompt
		analysis.Match = MatchNone
		analysis.Recovery = 0.0
		analysis.DownloadURL = req.SourceURL

		// Check if we can compensate
		if req.Weight < 0.5 {
			analysis.Workaround = "Low-weight LoRA - style can be approximated in prompt"
			analysis.Recovery = 0.3
		} else {
			analysis.Workaround = "Critical LoRA missing - results will differ significantly"
		}
	}

	return analysis
}

func analyzeVAE(req VAESpec, inv *LocalInventory) LayerAnalysis {
	analysis := LayerAnalysis{
		Layer:    LayerVAE,
		Required: req,
	}

	if req.Name == "" {
		// No specific VAE required - checkpoint default is fine
		analysis.Match = MatchExact
		analysis.Recovery = 1.0
		analysis.Workaround = "Using checkpoint default VAE"
		return analysis
	}

	if inv == nil {
		analysis.Match = MatchPartial
		analysis.Recovery = 0.8 // VAE differences are usually subtle
		analysis.Workaround = "VAE info unavailable - using defaults"
		return analysis
	}

	match := inv.FindVAE(req)
	if match != nil {
		analysis.Available = match.VAE
		analysis.Match = match.Level
		analysis.Recovery = match.Level.RecoveryRate()
	} else {
		analysis.Match = MatchPartial
		analysis.Recovery = 0.8 // VAE mismatches usually don't ruin images
		analysis.Workaround = "Specific VAE not available - colors may differ slightly"
		analysis.DownloadURL = req.SourceURL
	}

	return analysis
}

func analyzeSampler(req SamplerSpec, inv *LocalInventory) LayerAnalysis {
	analysis := LayerAnalysis{
		Layer:    LayerSampler,
		Required: req,
	}

	// Samplers are almost always available in ComfyUI
	if req.Name == "" {
		req.Name = "euler"
		req.Steps = 20
		req.CFGScale = 7.0
	}

	// Known samplers in ComfyUI
	knownSamplers := map[string]bool{
		"euler": true, "euler_ancestral": true, "heun": true,
		"dpm_2": true, "dpm_2_ancestral": true,
		"dpmpp_2s_ancestral": true, "dpmpp_sde": true,
		"dpmpp_2m": true, "dpmpp_2m_sde": true, "dpmpp_3m_sde": true,
		"lms": true, "lcm": true, "ddim": true, "ddpm": true,
		"uni_pc": true, "uni_pc_bh2": true,
	}

	samplerKey := strings.ToLower(strings.ReplaceAll(req.Name, " ", "_"))
	samplerKey = strings.ReplaceAll(samplerKey, "++", "pp")

	if knownSamplers[samplerKey] {
		analysis.Match = MatchExact
		analysis.Recovery = 1.0
		analysis.Available = req
	} else {
		// Find closest match
		analysis.Match = MatchSimilar
		analysis.Recovery = 0.85
		analysis.Workaround = fmt.Sprintf("Sampler '%s' substituted with similar algorithm", req.Name)
		analysis.Available = SamplerSpec{
			Name:     "dpmpp_2m",
			Steps:    req.Steps,
			CFGScale: req.CFGScale,
		}
	}

	return analysis
}

func analyzePrompt(req PromptSpec) LayerAnalysis {
	// Prompts are always available
	return LayerAnalysis{
		Layer:     LayerPrompt,
		Required:  req,
		Available: req,
		Match:     MatchExact,
		Recovery:  1.0,
	}
}

func analyzeSeed(seed int64) LayerAnalysis {
	// Seeds are always reproducible
	return LayerAnalysis{
		Layer:     LayerSeed,
		Required:  seed,
		Available: seed,
		Match:     MatchExact,
		Recovery:  1.0,
	}
}

func calculateOverall(layers []LayerAnalysis, stack *GenerationStack) OverallAnalysis {
	overall := OverallAnalysis{
		TotalLayers: len(layers),
	}

	// Weight layers by importance
	weights := map[StackLayer]float64{
		LayerHardware:    0.05, // Can usually work around
		LayerCheckpoint:  0.40, // Most critical
		LayerLoRA:        0.25, // Important for style
		LayerVAE:         0.05, // Minor impact
		LayerSampler:     0.10, // Some impact
		LayerPrompt:      0.10, // Always available
		LayerSeed:        0.05, // Always available
		LayerPostProcess: 0.00, // Optional
	}

	var totalWeight float64
	var weightedRecovery float64

	for _, layer := range layers {
		weight := weights[layer.Layer]
		totalWeight += weight
		weightedRecovery += layer.Recovery * weight

		switch layer.Match {
		case MatchExact:
			overall.MatchedLayers++
		case MatchSimilar, MatchPartial:
			overall.PartialLayers++
			if layer.Workaround != "" {
				overall.Warnings = append(overall.Warnings, layer.Workaround)
			}
		case MatchNone:
			overall.MissingLayers++
			if layer.Layer == LayerCheckpoint {
				overall.BlockingIssues = append(overall.BlockingIssues,
					fmt.Sprintf("Missing checkpoint: %v", layer.Required))
			}
		}
	}

	if totalWeight > 0 {
		overall.OverallRecovery = weightedRecovery / totalWeight
	}

	// Can we generate?
	overall.CanGenerate = len(overall.BlockingIssues) == 0 && overall.OverallRecovery >= 0.3

	return overall
}

func generateOptions(analysis *StackAnalysis, stack *GenerationStack, inv *LocalInventory) []RemixOption {
	var options []RemixOption

	// Option 1: Generate now with what we have
	if analysis.Overall.CanGenerate {
		options = append(options, RemixOption{
			ID:          "generate_now",
			Label:       "Generate Now (Best Effort)",
			Description: fmt.Sprintf("Generate with %.0f%% fidelity using available resources", analysis.Overall.OverallRecovery*100),
			Recovery:    analysis.Overall.OverallRecovery,
		})
	}

	// Option 2: Download missing pieces
	var downloads []DownloadItem
	for _, layer := range analysis.Layers {
		if layer.DownloadURL != "" && layer.Match == MatchNone {
			downloads = append(downloads, DownloadItem{
				URL:  layer.DownloadURL,
				Type: layer.Layer.String(),
				Size: layer.DownloadSize,
			})
		}
	}

	if len(downloads) > 0 {
		options = append(options, RemixOption{
			ID:          "download_missing",
			Label:       "Download Missing Resources",
			Description: fmt.Sprintf("Download %d missing items for better reproduction", len(downloads)),
			Recovery:    0.95, // Estimated after downloads
			Downloads:   downloads,
		})
	}

	// Option 3: Rebuild from scratch
	options = append(options, RemixOption{
		ID:          "rebuild",
		Label:       "Rebuild from Preferences",
		Description: "Start fresh using your preferred models and style",
		Recovery:    0.0, // User-determined
	})

	return options
}

// FormatReport generates a human-readable analysis report.
func (a *StackAnalysis) FormatReport() string {
	var sb strings.Builder

	sb.WriteString("╭─────────────────────────────────────────────────────────────╮\n")
	sb.WriteString("│              STACK DECONSTRUCTION REPORT                    │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")

	// Source info
	if a.Source != nil && a.Source.SourceURL != "" {
		sb.WriteString(fmt.Sprintf("│ Source: %-51s │\n", truncate(a.Source.SourceURL, 51)))
	}

	sb.WriteString("├──────────────┬────────────────────┬────────────┬────────────┤\n")
	sb.WriteString("│ Layer        │ Required           │ Match      │ Recovery   │\n")
	sb.WriteString("├──────────────┼────────────────────┼────────────┼────────────┤\n")

	for _, layer := range a.Layers {
		required := formatRequired(layer.Required)
		sb.WriteString(fmt.Sprintf("│ %-12s │ %-18s │ %s %-8s │ %6.0f%%    │\n",
			layer.Layer.String(),
			truncate(required, 18),
			layer.Match.Symbol(),
			layer.Match.String(),
			layer.Recovery*100,
		))
	}

	sb.WriteString("├──────────────┴────────────────────┴────────────┴────────────┤\n")
	sb.WriteString(fmt.Sprintf("│ OVERALL RECOVERY: %-6.0f%%                                   │\n", a.Overall.OverallRecovery*100))

	if a.Overall.CanGenerate {
		sb.WriteString("│ Status: ✓ Can generate with available resources             │\n")
	} else {
		sb.WriteString("│ Status: ✗ Missing critical resources                        │\n")
	}

	// Warnings
	if len(a.Overall.Warnings) > 0 {
		sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
		sb.WriteString("│ Warnings:                                                   │\n")
		for _, w := range a.Overall.Warnings[:min(3, len(a.Overall.Warnings))] {
			sb.WriteString(fmt.Sprintf("│ • %-57s │\n", truncate(w, 57)))
		}
	}

	// Options
	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
	sb.WriteString("│ Options:                                                    │\n")
	for i, opt := range a.Options {
		sb.WriteString(fmt.Sprintf("│ [%d] %-55s │\n", i+1, truncate(opt.Label, 55)))
	}

	sb.WriteString("╰─────────────────────────────────────────────────────────────╯\n")

	return sb.String()
}

func formatRequired(v interface{}) string {
	switch val := v.(type) {
	case CheckpointSpec:
		if val.Name != "" {
			return val.Name
		}
		return "(default)"
	case LoRASpec:
		return fmt.Sprintf("%s (%.1f)", val.Name, val.Weight)
	case VAESpec:
		if val.Name != "" {
			return val.Name
		}
		return "(default)"
	case SamplerSpec:
		return fmt.Sprintf("%s %d@%.1f", val.Name, val.Steps, val.CFGScale)
	case PromptSpec:
		if len(val.Positive) > 15 {
			return val.Positive[:15] + "..."
		}
		return val.Positive
	case int64:
		return fmt.Sprintf("seed:%d", val)
	case HardwareSpec:
		if val.MinVRAM > 0 {
			return fmt.Sprintf("%.0fGB VRAM", float64(val.MinVRAM)/(1024*1024*1024))
		}
		return "any"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
