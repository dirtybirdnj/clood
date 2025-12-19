package sd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CivitAIImage represents parsed generation metadata from CivitAI.
type CivitAIImage struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	NSFW      bool      `json:"nsfw"`
	CreatedAt time.Time `json:"createdAt"`

	// Generation metadata (from meta field)
	Meta CivitAIMeta `json:"meta"`

	// Resources used (models, LoRAs)
	Resources []CivitAIResource `json:"-"`
}

// CivitAIMeta contains generation parameters.
type CivitAIMeta struct {
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negativePrompt"`
	Sampler        string  `json:"sampler"`
	CFGScale       float64 `json:"cfgScale"`
	Steps          int     `json:"steps"`
	Seed           int64   `json:"seed"`
	Size           string  `json:"Size"` // "1024x1024" format
	Model          string  `json:"Model"`
	ModelHash      string  `json:"Model hash"`
	VAE            string  `json:"VAE"`
	VAEHash        string  `json:"VAE hash"`
	ClipSkip       int     `json:"Clip skip"`

	// LoRA info often embedded in prompt or resources
	LoRAs []LoRAReference `json:"-"`

	// Raw additional fields we might not have mapped
	Raw map[string]interface{} `json:"-"`
}

// CivitAIResource represents a model/LoRA used in generation.
type CivitAIResource struct {
	Type    string `json:"type"` // "model", "lora", "embedding"
	Name    string `json:"name"`
	ModelID int    `json:"modelId"`
	Hash    string `json:"hash"`
	Weight  float64 `json:"weight,omitempty"` // For LoRAs
}

// LoRAReference parsed from prompt or resources.
type LoRAReference struct {
	Name   string
	Weight float64
	Hash   string
}

// CivitAIParser handles parsing CivitAI URLs and generation parameters.
type CivitAIParser struct {
	HTTPClient *http.Client
	BaseURL    string
}

// NewCivitAIParser creates a parser with default settings.
func NewCivitAIParser() *CivitAIParser {
	return &CivitAIParser{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:    "https://civitai.com/api/v1",
	}
}

// ParseURL extracts image ID from various CivitAI URL formats.
// Supports:
//   - https://civitai.com/images/12345
//   - https://civitai.com/posts/12345 (extracts first image)
//   - https://civitai.com/models/12345?modelVersionId=67890 (model page)
func (p *CivitAIParser) ParseURL(rawURL string) (*URLParseResult, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if !strings.Contains(u.Host, "civitai.com") {
		return nil, fmt.Errorf("not a CivitAI URL: %s", u.Host)
	}

	result := &URLParseResult{
		OriginalURL: rawURL,
	}

	// Match different URL patterns
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")

	if len(pathParts) >= 2 {
		switch pathParts[0] {
		case "images":
			id, err := strconv.Atoi(pathParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid image ID: %s", pathParts[1])
			}
			result.Type = "image"
			result.ImageID = id
		case "posts":
			id, err := strconv.Atoi(pathParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid post ID: %s", pathParts[1])
			}
			result.Type = "post"
			result.PostID = id
		case "models":
			id, err := strconv.Atoi(pathParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid model ID: %s", pathParts[1])
			}
			result.Type = "model"
			result.ModelID = id
			// Check for version ID in query
			if versionStr := u.Query().Get("modelVersionId"); versionStr != "" {
				if versionID, err := strconv.Atoi(versionStr); err == nil {
					result.VersionID = versionID
				}
			}
		}
	}

	if result.Type == "" {
		return nil, fmt.Errorf("unrecognized CivitAI URL format: %s", u.Path)
	}

	return result, nil
}

// URLParseResult holds parsed URL components.
type URLParseResult struct {
	OriginalURL string
	Type        string // "image", "post", "model"
	ImageID     int
	PostID      int
	ModelID     int
	VersionID   int
}

// FetchImage retrieves image metadata from CivitAI API.
func (p *CivitAIParser) FetchImage(imageID int) (*CivitAIImage, error) {
	url := fmt.Sprintf("%s/images?imageId=%d", p.BaseURL, imageID)

	resp, err := p.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Items []CivitAIImage `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Items) == 0 {
		return nil, fmt.Errorf("image not found: %d", imageID)
	}

	img := &apiResp.Items[0]

	// Parse LoRAs from prompt
	img.Meta.LoRAs = parseLoRAsFromPrompt(img.Meta.Prompt)

	return img, nil
}

// ParseGenerationParams parses pasted generation parameters.
// Supports multiple formats:
//   - A1111 style: "prompt\nNegative prompt: ...\nSteps: 20, Sampler: ..."
//   - Raw JSON from ComfyUI
//   - CivitAI metadata blocks
func (p *CivitAIParser) ParseGenerationParams(text string) (*CivitAIMeta, error) {
	text = strings.TrimSpace(text)

	// Try JSON first (ComfyUI workflow or raw meta)
	if strings.HasPrefix(text, "{") {
		var meta CivitAIMeta
		if err := json.Unmarshal([]byte(text), &meta); err == nil {
			meta.LoRAs = parseLoRAsFromPrompt(meta.Prompt)
			return &meta, nil
		}
		// Might be ComfyUI workflow - try to extract from that
		var workflow map[string]interface{}
		if err := json.Unmarshal([]byte(text), &workflow); err == nil {
			return extractMetaFromWorkflow(workflow)
		}
	}

	// Try A1111 format
	if meta := parseA1111Format(text); meta != nil {
		return meta, nil
	}

	// Last resort: treat entire text as prompt
	return &CivitAIMeta{
		Prompt: text,
		LoRAs:  parseLoRAsFromPrompt(text),
	}, nil
}

// parseLoRAsFromPrompt extracts <lora:name:weight> tags from prompt text.
func parseLoRAsFromPrompt(prompt string) []LoRAReference {
	// Match <lora:name:weight> or <lora:name>
	re := regexp.MustCompile(`<lora:([^:>]+)(?::([0-9.]+))?>`)
	matches := re.FindAllStringSubmatch(prompt, -1)

	var loras []LoRAReference
	for _, match := range matches {
		lora := LoRAReference{
			Name:   match[1],
			Weight: 1.0, // Default weight
		}
		if len(match) > 2 && match[2] != "" {
			if w, err := strconv.ParseFloat(match[2], 64); err == nil {
				lora.Weight = w
			}
		}
		loras = append(loras, lora)
	}

	return loras
}

// parseA1111Format parses Automatic1111 style parameter dumps.
func parseA1111Format(text string) *CivitAIMeta {
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return nil
	}

	meta := &CivitAIMeta{}

	// First line(s) until "Negative prompt:" is the positive prompt
	var promptLines []string
	var negPromptLines []string
	var paramLine string
	inNegative := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Negative prompt:") {
			inNegative = true
			negPromptLines = append(negPromptLines, strings.TrimPrefix(line, "Negative prompt:"))
		} else if strings.HasPrefix(line, "Steps:") {
			paramLine = line
			inNegative = false
		} else if inNegative {
			negPromptLines = append(negPromptLines, line)
		} else if paramLine == "" {
			promptLines = append(promptLines, line)
		}
	}

	meta.Prompt = strings.TrimSpace(strings.Join(promptLines, " "))
	meta.NegativePrompt = strings.TrimSpace(strings.Join(negPromptLines, " "))

	// Parse the parameter line: "Steps: 20, Sampler: DPM++ 2M Karras, CFG scale: 7, ..."
	if paramLine != "" {
		params := parseParamLine(paramLine)

		if v, ok := params["Steps"]; ok {
			meta.Steps, _ = strconv.Atoi(v)
		}
		if v, ok := params["Sampler"]; ok {
			meta.Sampler = v
		}
		if v, ok := params["CFG scale"]; ok {
			meta.CFGScale, _ = strconv.ParseFloat(v, 64)
		}
		if v, ok := params["Seed"]; ok {
			meta.Seed, _ = strconv.ParseInt(v, 10, 64)
		}
		if v, ok := params["Size"]; ok {
			meta.Size = v
		}
		if v, ok := params["Model"]; ok {
			meta.Model = v
		}
		if v, ok := params["Model hash"]; ok {
			meta.ModelHash = v
		}
		if v, ok := params["VAE"]; ok {
			meta.VAE = v
		}
		if v, ok := params["VAE hash"]; ok {
			meta.VAEHash = v
		}
		if v, ok := params["Clip skip"]; ok {
			meta.ClipSkip, _ = strconv.Atoi(v)
		}
	}

	// Extract LoRAs from prompt
	meta.LoRAs = parseLoRAsFromPrompt(meta.Prompt)

	// If we got meaningful data, return it
	if meta.Prompt != "" || meta.Steps > 0 {
		return meta
	}

	return nil
}

// parseParamLine parses "Key1: value1, Key2: value2" format.
func parseParamLine(line string) map[string]string {
	params := make(map[string]string)

	// Handle "Key: value" pairs separated by commas
	// But values might contain commas (like "Size: 1024, 1024")
	re := regexp.MustCompile(`([A-Za-z][A-Za-z0-9 ]*?):\s*([^,]+(?:,\s*\d+)?)(?:,\s*|$)`)
	matches := re.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		key := strings.TrimSpace(match[1])
		value := strings.TrimSpace(match[2])
		params[key] = value
	}

	return params
}

// extractMetaFromWorkflow attempts to extract generation params from ComfyUI workflow JSON.
func extractMetaFromWorkflow(workflow map[string]interface{}) (*CivitAIMeta, error) {
	meta := &CivitAIMeta{}

	// ComfyUI workflows have nodes with class_type
	// Look for KSampler, CLIPTextEncode, CheckpointLoaderSimple, etc.
	for _, node := range workflow {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			continue
		}

		classType, _ := nodeMap["class_type"].(string)
		inputs, _ := nodeMap["inputs"].(map[string]interface{})

		switch classType {
		case "KSampler":
			if steps, ok := inputs["steps"].(float64); ok {
				meta.Steps = int(steps)
			}
			if cfg, ok := inputs["cfg"].(float64); ok {
				meta.CFGScale = cfg
			}
			if seed, ok := inputs["seed"].(float64); ok {
				meta.Seed = int64(seed)
			}
			if sampler, ok := inputs["sampler_name"].(string); ok {
				meta.Sampler = sampler
			}
		case "CLIPTextEncode":
			if text, ok := inputs["text"].(string); ok {
				// First one is usually positive prompt
				if meta.Prompt == "" {
					meta.Prompt = text
				} else if meta.NegativePrompt == "" {
					meta.NegativePrompt = text
				}
			}
		case "CheckpointLoaderSimple":
			if ckpt, ok := inputs["ckpt_name"].(string); ok {
				meta.Model = ckpt
			}
		case "VAELoader":
			if vae, ok := inputs["vae_name"].(string); ok {
				meta.VAE = vae
			}
		case "LoraLoader":
			if loraName, ok := inputs["lora_name"].(string); ok {
				weight := 1.0
				if w, ok := inputs["strength_model"].(float64); ok {
					weight = w
				}
				meta.LoRAs = append(meta.LoRAs, LoRAReference{
					Name:   loraName,
					Weight: weight,
				})
			}
		}
	}

	if meta.Prompt == "" && meta.Steps == 0 {
		return nil, fmt.Errorf("no generation parameters found in workflow")
	}

	return meta, nil
}

// GetDimensions parses the Size field into width and height.
func (m *CivitAIMeta) GetDimensions() (width, height int) {
	if m.Size == "" {
		return 1024, 1024 // Default
	}

	// Handle "1024x1024" or "1024, 1024" formats
	m.Size = strings.ReplaceAll(m.Size, " ", "")
	parts := strings.FieldsFunc(m.Size, func(r rune) bool {
		return r == 'x' || r == 'X' || r == ','
	})

	if len(parts) >= 2 {
		width, _ = strconv.Atoi(parts[0])
		height, _ = strconv.Atoi(parts[1])
	}

	if width == 0 {
		width = 1024
	}
	if height == 0 {
		height = 1024
	}

	return width, height
}

// CleanPrompt removes LoRA tags and cleans up the prompt for display.
func (m *CivitAIMeta) CleanPrompt() string {
	// Remove <lora:...> tags
	re := regexp.MustCompile(`<lora:[^>]+>`)
	clean := re.ReplaceAllString(m.Prompt, "")
	// Clean up extra spaces/commas
	clean = regexp.MustCompile(`\s*,\s*,\s*`).ReplaceAllString(clean, ", ")
	clean = strings.TrimSpace(clean)
	return clean
}
