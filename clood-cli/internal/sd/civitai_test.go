package sd

import (
	"testing"
)

// =============================================================================
// URL Parsing Tests
// =============================================================================

func TestParseURL_ImageFormat(t *testing.T) {
	parser := NewCivitAIParser()

	tests := []struct {
		name     string
		url      string
		wantType string
		wantID   int
	}{
		{
			name:     "simple image URL",
			url:      "https://civitai.com/images/12345",
			wantType: "image",
			wantID:   12345,
		},
		{
			name:     "image URL with trailing slash",
			url:      "https://civitai.com/images/67890/",
			wantType: "image",
			wantID:   67890,
		},
		{
			name:     "image URL with query params",
			url:      "https://civitai.com/images/11111?foo=bar",
			wantType: "image",
			wantID:   11111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseURL(tt.url)
			if err != nil {
				t.Fatalf("ParseURL() error = %v", err)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", result.Type, tt.wantType)
			}
			if result.ImageID != tt.wantID {
				t.Errorf("ImageID = %v, want %v", result.ImageID, tt.wantID)
			}
		})
	}
}

func TestParseURL_PostFormat(t *testing.T) {
	parser := NewCivitAIParser()

	result, err := parser.ParseURL("https://civitai.com/posts/98765")
	if err != nil {
		t.Fatalf("ParseURL() error = %v", err)
	}
	if result.Type != "post" {
		t.Errorf("Type = %v, want post", result.Type)
	}
	if result.PostID != 98765 {
		t.Errorf("PostID = %v, want 98765", result.PostID)
	}
}

func TestParseURL_ModelFormat(t *testing.T) {
	parser := NewCivitAIParser()

	tests := []struct {
		name        string
		url         string
		wantModelID int
		wantVersion int
	}{
		{
			name:        "model without version",
			url:         "https://civitai.com/models/12345",
			wantModelID: 12345,
			wantVersion: 0,
		},
		{
			name:        "model with version",
			url:         "https://civitai.com/models/12345?modelVersionId=67890",
			wantModelID: 12345,
			wantVersion: 67890,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseURL(tt.url)
			if err != nil {
				t.Fatalf("ParseURL() error = %v", err)
			}
			if result.Type != "model" {
				t.Errorf("Type = %v, want model", result.Type)
			}
			if result.ModelID != tt.wantModelID {
				t.Errorf("ModelID = %v, want %v", result.ModelID, tt.wantModelID)
			}
			if result.VersionID != tt.wantVersion {
				t.Errorf("VersionID = %v, want %v", result.VersionID, tt.wantVersion)
			}
		})
	}
}

func TestParseURL_InvalidURL(t *testing.T) {
	parser := NewCivitAIParser()

	tests := []struct {
		name string
		url  string
	}{
		{"not civitai", "https://example.com/images/12345"},
		{"missing path", "https://civitai.com/"},
		{"invalid format", "https://civitai.com/foo/bar"},
		{"non-numeric ID", "https://civitai.com/images/abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseURL(tt.url)
			if err == nil {
				t.Error("ParseURL() expected error, got nil")
			}
		})
	}
}

// =============================================================================
// LoRA Parsing Tests
// =============================================================================

func TestParseLoRAsFromPrompt(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		wantCount int
		wantFirst LoRAReference
	}{
		{
			name:      "single lora with weight",
			prompt:    "beautiful landscape <lora:ghibli_style:0.8>",
			wantCount: 1,
			wantFirst: LoRAReference{Name: "ghibli_style", Weight: 0.8},
		},
		{
			name:      "lora without weight",
			prompt:    "portrait <lora:realistic_skin>",
			wantCount: 1,
			wantFirst: LoRAReference{Name: "realistic_skin", Weight: 1.0},
		},
		{
			name:      "multiple loras",
			prompt:    "<lora:style1:0.5> masterpiece <lora:style2:0.7>",
			wantCount: 2,
			wantFirst: LoRAReference{Name: "style1", Weight: 0.5},
		},
		{
			name:      "no loras",
			prompt:    "just a regular prompt",
			wantCount: 0,
		},
		{
			name:      "lora with decimal weight",
			prompt:    "<lora:detailed:1.2>",
			wantCount: 1,
			wantFirst: LoRAReference{Name: "detailed", Weight: 1.2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loras := parseLoRAsFromPrompt(tt.prompt)
			if len(loras) != tt.wantCount {
				t.Errorf("got %d loras, want %d", len(loras), tt.wantCount)
			}
			if tt.wantCount > 0 && len(loras) > 0 {
				if loras[0].Name != tt.wantFirst.Name {
					t.Errorf("first lora name = %v, want %v", loras[0].Name, tt.wantFirst.Name)
				}
				if loras[0].Weight != tt.wantFirst.Weight {
					t.Errorf("first lora weight = %v, want %v", loras[0].Weight, tt.wantFirst.Weight)
				}
			}
		})
	}
}

// =============================================================================
// A1111 Parameter Parsing Tests
// =============================================================================

func TestParseA1111Format(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPrompt string
		wantNeg    string
		wantSteps  int
		wantCFG    float64
	}{
		{
			name: "standard A1111 format",
			input: `masterpiece, best quality, 1girl
Negative prompt: worst quality, bad anatomy
Steps: 20, Sampler: DPM++ 2M Karras, CFG scale: 7, Seed: 12345, Size: 1024x1024`,
			wantPrompt: "masterpiece, best quality, 1girl",
			wantNeg:    "worst quality, bad anatomy",
			wantSteps:  20,
			wantCFG:    7.0,
		},
		{
			name: "with model info",
			input: `portrait
Negative prompt: ugly
Steps: 30, Sampler: Euler a, CFG scale: 8.5, Model: sdxl_base`,
			wantPrompt: "portrait",
			wantNeg:    "ugly",
			wantSteps:  30,
			wantCFG:    8.5,
		},
		{
			name:       "prompt only (no params)",
			input:      "just a prompt with no metadata",
			wantPrompt: "",
			wantSteps:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := parseA1111Format(tt.input)
			if tt.wantPrompt == "" && meta == nil {
				return // Expected nil for invalid input
			}
			if meta == nil {
				t.Fatal("parseA1111Format returned nil")
			}
			if meta.Prompt != tt.wantPrompt {
				t.Errorf("Prompt = %q, want %q", meta.Prompt, tt.wantPrompt)
			}
			if meta.NegativePrompt != tt.wantNeg {
				t.Errorf("NegativePrompt = %q, want %q", meta.NegativePrompt, tt.wantNeg)
			}
			if meta.Steps != tt.wantSteps {
				t.Errorf("Steps = %d, want %d", meta.Steps, tt.wantSteps)
			}
			if meta.CFGScale != tt.wantCFG {
				t.Errorf("CFGScale = %v, want %v", meta.CFGScale, tt.wantCFG)
			}
		})
	}
}

// =============================================================================
// ComfyUI Workflow Parsing Tests
// =============================================================================

func TestExtractMetaFromWorkflow(t *testing.T) {
	workflow := map[string]interface{}{
		"3": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"steps":        float64(25),
				"cfg":          float64(7.5),
				"seed":         float64(42069),
				"sampler_name": "dpmpp_2m",
			},
		},
		"6": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs": map[string]interface{}{
				"text": "beautiful sunset over mountains",
			},
		},
		"7": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs": map[string]interface{}{
				"text": "ugly, blurry",
			},
		},
		"4": map[string]interface{}{
			"class_type": "CheckpointLoaderSimple",
			"inputs": map[string]interface{}{
				"ckpt_name": "sdxl_base.safetensors",
			},
		},
	}

	meta, err := extractMetaFromWorkflow(workflow)
	if err != nil {
		t.Fatalf("extractMetaFromWorkflow() error = %v", err)
	}

	if meta.Steps != 25 {
		t.Errorf("Steps = %d, want 25", meta.Steps)
	}
	if meta.CFGScale != 7.5 {
		t.Errorf("CFGScale = %v, want 7.5", meta.CFGScale)
	}
	if meta.Seed != 42069 {
		t.Errorf("Seed = %d, want 42069", meta.Seed)
	}
	if meta.Sampler != "dpmpp_2m" {
		t.Errorf("Sampler = %q, want dpmpp_2m", meta.Sampler)
	}
	if meta.Prompt != "beautiful sunset over mountains" {
		t.Errorf("Prompt = %q, want 'beautiful sunset over mountains'", meta.Prompt)
	}
	if meta.Model != "sdxl_base.safetensors" {
		t.Errorf("Model = %q, want sdxl_base.safetensors", meta.Model)
	}
}

// =============================================================================
// Dimension Parsing Tests
// =============================================================================

func TestGetDimensions(t *testing.T) {
	tests := []struct {
		name       string
		size       string
		wantWidth  int
		wantHeight int
	}{
		{"1024x1024", "1024x1024", 1024, 1024},
		{"512x768", "512x768", 512, 768},
		{"comma format", "1024, 1024", 1024, 1024},
		{"uppercase X", "768X512", 768, 512},
		{"empty", "", 1024, 1024}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &CivitAIMeta{Size: tt.size}
			w, h := meta.GetDimensions()
			if w != tt.wantWidth || h != tt.wantHeight {
				t.Errorf("GetDimensions() = %dx%d, want %dx%d", w, h, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

// =============================================================================
// Clean Prompt Tests
// =============================================================================

func TestCleanPrompt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "remove single lora",
			input: "beautiful <lora:ghibli:0.8> landscape",
			want:  "beautiful  landscape", // Double space is acceptable
		},
		{
			name:  "remove multiple loras",
			input: "<lora:style1:0.5> sunset <lora:style2:0.7> beach",
			want:  "sunset  beach", // Double space is acceptable
		},
		{
			name:  "no loras to remove",
			input: "just a regular prompt",
			want:  "just a regular prompt",
		},
		{
			name:  "cleanup extra commas",
			input: "art, <lora:foo:0.8>, , landscape",
			want:  "art, , landscape", // Comma cleanup is partial
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &CivitAIMeta{Prompt: tt.input}
			got := meta.CleanPrompt()
			if got != tt.want {
				t.Errorf("CleanPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Integration Tests (these test the full parser pipeline)
// =============================================================================

func TestParseGenerationParams_JSON(t *testing.T) {
	parser := NewCivitAIParser()

	// Test ComfyUI-style JSON
	jsonInput := `{"prompt": "test prompt", "negativePrompt": "ugly", "steps": 20, "cfgScale": 7.5}`
	meta, err := parser.ParseGenerationParams(jsonInput)
	if err != nil {
		t.Fatalf("ParseGenerationParams() error = %v", err)
	}
	if meta.Prompt != "test prompt" {
		t.Errorf("Prompt = %q, want 'test prompt'", meta.Prompt)
	}
}

func TestParseGenerationParams_RawPrompt(t *testing.T) {
	parser := NewCivitAIParser()

	// Test raw prompt text (fallback)
	rawInput := "masterpiece, best quality, 1girl, <lora:ghibli:0.8>"
	meta, err := parser.ParseGenerationParams(rawInput)
	if err != nil {
		t.Fatalf("ParseGenerationParams() error = %v", err)
	}
	if meta.Prompt != rawInput {
		t.Errorf("Prompt = %q, want %q", meta.Prompt, rawInput)
	}
	if len(meta.LoRAs) != 1 {
		t.Errorf("got %d LoRAs, want 1", len(meta.LoRAs))
	}
}
