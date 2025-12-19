package sd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Source represents a model/image repository.
type Source string

const (
	SourceCivitAI    Source = "civitai"
	SourceHuggingFace Source = "huggingface"
	SourceTensorArt  Source = "tensorart"
	SourceOpenArt    Source = "openart"
	SourceLocal      Source = "local"
	SourceUnknown    Source = "unknown"
)

// GenerationSource represents parsed generation metadata from any source.
// This is the unified format all source-specific parsers convert to.
type GenerationSource struct {
	Source      Source
	SourceURL   string
	SourceID    string
	FetchedAt   time.Time

	// Core generation params (unified)
	Prompt         string
	NegativePrompt string
	Checkpoint     CheckpointRef
	LoRAs          []LoRARef
	VAE            VAERef
	Sampler        SamplerRef
	Seed           int64
	Dimensions     Dimensions

	// Raw data for debugging
	RawMeta map[string]interface{}
}

// CheckpointRef references a checkpoint model.
type CheckpointRef struct {
	Name      string
	Hash      string
	BaseModel string // SD1.5, SDXL, Flux, etc.
	SourceURL string // Where to download
}

// LoRARef references a LoRA adapter.
type LoRARef struct {
	Name      string
	Hash      string
	Weight    float64
	BaseModel string
	SourceURL string
}

// VAERef references a VAE model.
type VAERef struct {
	Name      string
	Hash      string
	SourceURL string
}

// SamplerRef defines sampling configuration.
type SamplerRef struct {
	Name      string
	Steps     int
	CFGScale  float64
	Scheduler string // If applicable
}

// Dimensions for the output image.
type Dimensions struct {
	Width  int
	Height int
}

// SourceParser is the interface all source parsers implement.
type SourceParser interface {
	// CanParse returns true if this parser handles the given URL/text
	CanParse(input string) bool
	// Parse extracts generation metadata from URL or text
	Parse(input string) (*GenerationSource, error)
	// Source returns the source type
	Source() Source
}

// MultiSourceParser tries multiple parsers to handle any input.
type MultiSourceParser struct {
	Parsers    []SourceParser
	HTTPClient *http.Client
}

// NewMultiSourceParser creates a parser that handles all supported sources.
func NewMultiSourceParser() *MultiSourceParser {
	client := &http.Client{Timeout: 30 * time.Second}

	return &MultiSourceParser{
		HTTPClient: client,
		Parsers: []SourceParser{
			NewCivitAISourceParser(client),
			NewHuggingFaceParser(client),
			NewTensorArtParser(client),
			NewOpenArtParser(client),
			NewRawTextParser(), // Fallback
		},
	}
}

// Parse attempts to parse input using any matching parser.
func (m *MultiSourceParser) Parse(input string) (*GenerationSource, error) {
	input = strings.TrimSpace(input)

	for _, parser := range m.Parsers {
		if parser.CanParse(input) {
			return parser.Parse(input)
		}
	}

	return nil, fmt.Errorf("no parser found for input")
}

// DetectSource attempts to identify the source without fully parsing.
func (m *MultiSourceParser) DetectSource(input string) Source {
	input = strings.TrimSpace(input)

	for _, parser := range m.Parsers {
		if parser.CanParse(input) {
			return parser.Source()
		}
	}

	return SourceUnknown
}

// === CivitAI Source Parser ===

type CivitAISourceParser struct {
	client *http.Client
	parser *CivitAIParser
}

func NewCivitAISourceParser(client *http.Client) *CivitAISourceParser {
	p := NewCivitAIParser()
	p.HTTPClient = client
	return &CivitAISourceParser{client: client, parser: p}
}

func (p *CivitAISourceParser) Source() Source { return SourceCivitAI }

func (p *CivitAISourceParser) CanParse(input string) bool {
	return strings.Contains(input, "civitai.com")
}

func (p *CivitAISourceParser) Parse(input string) (*GenerationSource, error) {
	urlResult, err := p.parser.ParseURL(input)
	if err != nil {
		return nil, err
	}

	if urlResult.Type != "image" {
		return nil, fmt.Errorf("only image URLs supported, got: %s", urlResult.Type)
	}

	img, err := p.parser.FetchImage(urlResult.ImageID)
	if err != nil {
		return nil, err
	}

	w, h := img.Meta.GetDimensions()

	// Convert to unified format
	gs := &GenerationSource{
		Source:         SourceCivitAI,
		SourceURL:      input,
		SourceID:       fmt.Sprintf("%d", img.ID),
		FetchedAt:      time.Now(),
		Prompt:         img.Meta.Prompt,
		NegativePrompt: img.Meta.NegativePrompt,
		Checkpoint: CheckpointRef{
			Name:      img.Meta.Model,
			Hash:      img.Meta.ModelHash,
		},
		VAE: VAERef{
			Name: img.Meta.VAE,
			Hash: img.Meta.VAEHash,
		},
		Sampler: SamplerRef{
			Name:     img.Meta.Sampler,
			Steps:    img.Meta.Steps,
			CFGScale: img.Meta.CFGScale,
		},
		Seed:       img.Meta.Seed,
		Dimensions: Dimensions{Width: w, Height: h},
	}

	// Convert LoRAs
	for _, lora := range img.Meta.LoRAs {
		gs.LoRAs = append(gs.LoRAs, LoRARef{
			Name:   lora.Name,
			Hash:   lora.Hash,
			Weight: lora.Weight,
		})
	}

	return gs, nil
}

// === Hugging Face Parser ===

type HuggingFaceParser struct {
	client *http.Client
}

func NewHuggingFaceParser(client *http.Client) *HuggingFaceParser {
	return &HuggingFaceParser{client: client}
}

func (p *HuggingFaceParser) Source() Source { return SourceHuggingFace }

func (p *HuggingFaceParser) CanParse(input string) bool {
	return strings.Contains(input, "huggingface.co")
}

func (p *HuggingFaceParser) Parse(input string) (*GenerationSource, error) {
	// Parse model URL: https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0
	u, err := url.Parse(input)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid HuggingFace URL")
	}

	owner := parts[0]
	repo := parts[1]
	modelID := owner + "/" + repo

	// Fetch model card
	apiURL := fmt.Sprintf("https://huggingface.co/api/models/%s", modelID)
	resp, err := p.client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HuggingFace API error: %d", resp.StatusCode)
	}

	var modelInfo struct {
		ID       string   `json:"id"`
		Tags     []string `json:"tags"`
		CardData struct {
			BaseModel string `json:"base_model"`
		} `json:"cardData"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
		return nil, err
	}

	// Determine model type from tags
	baseModel := inferBaseModel(modelInfo.Tags)
	if modelInfo.CardData.BaseModel != "" {
		baseModel = modelInfo.CardData.BaseModel
	}

	// Check if it's a LoRA
	isLora := false
	for _, tag := range modelInfo.Tags {
		if strings.Contains(strings.ToLower(tag), "lora") {
			isLora = true
			break
		}
	}

	gs := &GenerationSource{
		Source:    SourceHuggingFace,
		SourceURL: input,
		SourceID:  modelID,
		FetchedAt: time.Now(),
	}

	if isLora {
		gs.LoRAs = []LoRARef{{
			Name:      repo,
			BaseModel: baseModel,
			SourceURL: input,
		}}
	} else {
		gs.Checkpoint = CheckpointRef{
			Name:      repo,
			BaseModel: baseModel,
			SourceURL: input,
		}
	}

	return gs, nil
}

func inferBaseModel(tags []string) string {
	for _, tag := range tags {
		lower := strings.ToLower(tag)
		switch {
		case strings.Contains(lower, "sdxl"):
			return "SDXL"
		case strings.Contains(lower, "sd-1.5") || strings.Contains(lower, "sd1.5"):
			return "SD1.5"
		case strings.Contains(lower, "flux"):
			return "Flux"
		case strings.Contains(lower, "sd-2") || strings.Contains(lower, "sd2"):
			return "SD2.x"
		}
	}
	return "unknown"
}

// === Tensor.Art Parser ===

type TensorArtParser struct {
	client *http.Client
}

func NewTensorArtParser(client *http.Client) *TensorArtParser {
	return &TensorArtParser{client: client}
}

func (p *TensorArtParser) Source() Source { return SourceTensorArt }

func (p *TensorArtParser) CanParse(input string) bool {
	return strings.Contains(input, "tensor.art")
}

func (p *TensorArtParser) Parse(input string) (*GenerationSource, error) {
	// Tensor.Art uses different URL formats
	// Posts: https://tensor.art/posts/12345
	// Models: https://tensor.art/models/12345

	u, err := url.Parse(input)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid Tensor.Art URL")
	}

	// For now, return basic info - full API integration would need their API
	gs := &GenerationSource{
		Source:    SourceTensorArt,
		SourceURL: input,
		SourceID:  parts[1],
		FetchedAt: time.Now(),
	}

	// TODO: Implement full Tensor.Art API when available
	return gs, fmt.Errorf("Tensor.Art parsing not fully implemented - URL noted for manual inspection")
}

// === OpenArt Parser ===

type OpenArtParser struct {
	client *http.Client
}

func NewOpenArtParser(client *http.Client) *OpenArtParser {
	return &OpenArtParser{client: client}
}

func (p *OpenArtParser) Source() Source { return SourceOpenArt }

func (p *OpenArtParser) CanParse(input string) bool {
	return strings.Contains(input, "openart.ai")
}

func (p *OpenArtParser) Parse(input string) (*GenerationSource, error) {
	// OpenArt workflow URLs: https://openart.ai/workflows/...

	gs := &GenerationSource{
		Source:    SourceOpenArt,
		SourceURL: input,
		FetchedAt: time.Now(),
	}

	// TODO: Implement OpenArt workflow fetching
	return gs, fmt.Errorf("OpenArt parsing not fully implemented - URL noted")
}

// === Raw Text Parser (Fallback) ===

type RawTextParser struct {
	civitai *CivitAIParser
}

func NewRawTextParser() *RawTextParser {
	return &RawTextParser{civitai: NewCivitAIParser()}
}

func (p *RawTextParser) Source() Source { return SourceLocal }

func (p *RawTextParser) CanParse(input string) bool {
	// Always matches as fallback
	return true
}

func (p *RawTextParser) Parse(input string) (*GenerationSource, error) {
	// Try to parse as generation parameters
	meta, err := p.civitai.ParseGenerationParams(input)
	if err != nil {
		return nil, err
	}

	w, h := meta.GetDimensions()

	gs := &GenerationSource{
		Source:         SourceLocal,
		FetchedAt:      time.Now(),
		Prompt:         meta.Prompt,
		NegativePrompt: meta.NegativePrompt,
		Checkpoint: CheckpointRef{
			Name: meta.Model,
			Hash: meta.ModelHash,
		},
		VAE: VAERef{
			Name: meta.VAE,
			Hash: meta.VAEHash,
		},
		Sampler: SamplerRef{
			Name:     meta.Sampler,
			Steps:    meta.Steps,
			CFGScale: meta.CFGScale,
		},
		Seed:       meta.Seed,
		Dimensions: Dimensions{Width: w, Height: h},
	}

	for _, lora := range meta.LoRAs {
		gs.LoRAs = append(gs.LoRAs, LoRARef{
			Name:   lora.Name,
			Hash:   lora.Hash,
			Weight: lora.Weight,
		})
	}

	return gs, nil
}

// === Utility Functions ===

// FetchModelInfo attempts to get model info from any source.
func FetchModelInfo(modelName string) (*CheckpointRef, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Try CivitAI search
	searchURL := fmt.Sprintf("https://civitai.com/api/v1/models?query=%s&limit=1", url.QueryEscape(modelName))
	resp, err := client.Get(searchURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var result struct {
				Items []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"items"`
			}
			if json.NewDecoder(resp.Body).Decode(&result) == nil && len(result.Items) > 0 {
				return &CheckpointRef{
					Name:      result.Items[0].Name,
					SourceURL: fmt.Sprintf("https://civitai.com/models/%d", result.Items[0].ID),
				}, nil
			}
		}
	}

	// Try HuggingFace search
	hfURL := fmt.Sprintf("https://huggingface.co/api/models?search=%s&limit=1", url.QueryEscape(modelName))
	resp, err = client.Get(hfURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var results []struct {
				ID string `json:"modelId"`
			}
			if json.NewDecoder(resp.Body).Decode(&results) == nil && len(results) > 0 {
				return &CheckpointRef{
					Name:      results[0].ID,
					SourceURL: fmt.Sprintf("https://huggingface.co/%s", results[0].ID),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("model not found: %s", modelName)
}

// IsValidURL checks if the input looks like a URL.
func IsValidURL(input string) bool {
	re := regexp.MustCompile(`^https?://`)
	return re.MatchString(input)
}
