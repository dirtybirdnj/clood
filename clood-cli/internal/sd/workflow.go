package sd

import (
	"encoding/json"
	"fmt"
)

// ComfyWorkflow represents a ComfyUI workflow that can be executed via API.
// This is a simplified representation - ComfyUI workflows are node graphs.
type ComfyWorkflow struct {
	Nodes map[string]ComfyNode `json:"nodes"`
}

// ComfyNode represents a single node in the ComfyUI graph.
type ComfyNode struct {
	ClassType string                 `json:"class_type"`
	Inputs    map[string]interface{} `json:"inputs"`
}

// WorkflowConfig contains all parameters needed to generate a workflow.
type WorkflowConfig struct {
	Prompt       *Prompt  `json:"prompt"`
	Checkpoint   string   `json:"checkpoint"`    // e.g., "sd_xl_base_1.0.safetensors"
	Width        int      `json:"width"`
	Height       int      `json:"height"`
	Steps        int      `json:"steps"`
	CFGScale     float64  `json:"cfg_scale"`
	Sampler      string   `json:"sampler"`       // e.g., "euler_ancestral", "dpmpp_2m"
	Scheduler    string   `json:"scheduler"`     // e.g., "normal", "karras"
	BatchSize    int      `json:"batch_size"`
	OutputPrefix string   `json:"output_prefix"`
}

// DefaultWorkflowConfig returns sensible defaults for SDXL.
func DefaultWorkflowConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Width:        1024,
		Height:       1024,
		Steps:        25,
		CFGScale:     7.0,
		Sampler:      "euler_ancestral",
		Scheduler:    "normal",
		BatchSize:    1,
		OutputPrefix: "clood",
	}
}

// BuildBasicWorkflow creates a simple txt2img workflow for ComfyUI.
// This generates the JSON that ComfyUI's API accepts.
func BuildBasicWorkflow(cfg *WorkflowConfig) (*ComfyWorkflow, error) {
	if cfg.Prompt == nil {
		return nil, fmt.Errorf("prompt is required")
	}
	if cfg.Checkpoint == "" {
		return nil, fmt.Errorf("checkpoint is required")
	}

	seed := cfg.Prompt.Seed
	if seed < 0 {
		seed = 0 // ComfyUI uses 0 for random
	}

	workflow := &ComfyWorkflow{
		Nodes: map[string]ComfyNode{
			"3": {
				ClassType: "KSampler",
				Inputs: map[string]interface{}{
					"seed":         seed,
					"steps":        cfg.Steps,
					"cfg":          cfg.CFGScale,
					"sampler_name": cfg.Sampler,
					"scheduler":    cfg.Scheduler,
					"denoise":      1.0,
					"model":        []interface{}{"4", 0},
					"positive":     []interface{}{"6", 0},
					"negative":     []interface{}{"7", 0},
					"latent_image": []interface{}{"5", 0},
				},
			},
			"4": {
				ClassType: "CheckpointLoaderSimple",
				Inputs: map[string]interface{}{
					"ckpt_name": cfg.Checkpoint,
				},
			},
			"5": {
				ClassType: "EmptyLatentImage",
				Inputs: map[string]interface{}{
					"width":      cfg.Width,
					"height":     cfg.Height,
					"batch_size": cfg.BatchSize,
				},
			},
			"6": {
				ClassType: "CLIPTextEncode",
				Inputs: map[string]interface{}{
					"text": cfg.Prompt.FormatPositive(),
					"clip": []interface{}{"4", 1},
				},
			},
			"7": {
				ClassType: "CLIPTextEncode",
				Inputs: map[string]interface{}{
					"text": cfg.Prompt.Negative,
					"clip": []interface{}{"4", 1},
				},
			},
			"8": {
				ClassType: "VAEDecode",
				Inputs: map[string]interface{}{
					"samples": []interface{}{"3", 0},
					"vae":     []interface{}{"4", 2},
				},
			},
			"9": {
				ClassType: "SaveImage",
				Inputs: map[string]interface{}{
					"filename_prefix": cfg.OutputPrefix,
					"images":          []interface{}{"8", 0},
				},
			},
		},
	}

	return workflow, nil
}

// ToJSON serializes the workflow to JSON for the ComfyUI API.
func (w *ComfyWorkflow) ToJSON() ([]byte, error) {
	return json.MarshalIndent(w, "", "  ")
}

// APIPayload wraps the workflow in the format ComfyUI's /prompt endpoint expects.
type APIPayload struct {
	Prompt   map[string]ComfyNode `json:"prompt"`
	ClientID string               `json:"client_id,omitempty"`
}

// ToAPIPayload converts the workflow to the API submission format.
func (w *ComfyWorkflow) ToAPIPayload(clientID string) *APIPayload {
	return &APIPayload{
		Prompt:   w.Nodes,
		ClientID: clientID,
	}
}

// LoRALoaderNode creates a LoRA loader node to insert into the workflow.
// This modifies the model pipeline to apply LoRA weights.
func LoRALoaderNode(loraName string, strength float64, modelInput, clipInput []interface{}) ComfyNode {
	return ComfyNode{
		ClassType: "LoraLoader",
		Inputs: map[string]interface{}{
			"lora_name":       loraName + ".safetensors",
			"strength_model":  strength,
			"strength_clip":   strength,
			"model":           modelInput,
			"clip":            clipInput,
		},
	}
}
