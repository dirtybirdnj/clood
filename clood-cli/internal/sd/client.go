package sd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client handles communication with a ComfyUI server.
type Client struct {
	BaseURL   string
	ClientID  string
	Timeout   time.Duration
	OutputDir string
}

// NewClient creates a ComfyUI client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:   baseURL,
		ClientID:  uuid.New().String(),
		Timeout:   10 * time.Minute,
		OutputDir: filepath.Join(os.Getenv("HOME"), ".clood", "gallery"),
	}
}

// QueueResponse is returned when queueing a prompt to ComfyUI.
type QueueResponse struct {
	PromptID string `json:"prompt_id"`
	Number   int    `json:"number"`
}

// QueuePrompt sends a workflow to ComfyUI for execution.
func (c *Client) QueuePrompt(workflow *ComfyWorkflow) (*QueueResponse, error) {
	payload := workflow.ToAPIPayload(c.ClientID)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := http.Post(c.BaseURL+"/prompt", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("post prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("queue prompt failed (%d): %s", resp.StatusCode, string(body))
	}

	var result QueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// HistoryEntry represents a completed prompt in ComfyUI history.
type HistoryEntry struct {
	Outputs map[string]struct {
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
	} `json:"outputs"`
	Status struct {
		Completed bool `json:"completed"`
	} `json:"status"`
}

// GetHistory retrieves the history for a prompt ID.
func (c *Client) GetHistory(promptID string) (*HistoryEntry, error) {
	resp, err := http.Get(fmt.Sprintf("%s/history/%s", c.BaseURL, promptID))
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get history failed (%d): %s", resp.StatusCode, string(body))
	}

	var history map[string]HistoryEntry
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, fmt.Errorf("decode history: %w", err)
	}

	entry, ok := history[promptID]
	if !ok {
		return nil, fmt.Errorf("prompt not found in history")
	}

	return &entry, nil
}

// DownloadImage fetches an image from ComfyUI and saves it locally.
func (c *Client) DownloadImage(filename, subfolder, imgType string) (string, error) {
	url := fmt.Sprintf("%s/view?filename=%s&subfolder=%s&type=%s",
		c.BaseURL, filename, subfolder, imgType)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	// Save to local file
	localPath := filepath.Join(c.OutputDir, filename)
	f, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return localPath, nil
}

// GenerateResult holds the output from a generation.
type GenerateResult struct {
	PromptID   string
	ImagePaths []string
	Duration   time.Duration
	Workflow   *WorkflowConfig
}

// Generate runs a complete generation workflow and waits for completion.
func (c *Client) Generate(ctx context.Context, cfg *WorkflowConfig) (*GenerateResult, error) {
	start := time.Now()

	// Build workflow
	workflow, err := BuildBasicWorkflow(cfg)
	if err != nil {
		return nil, fmt.Errorf("build workflow: %w", err)
	}

	// Queue prompt
	promptResp, err := c.QueuePrompt(workflow)
	if err != nil {
		return nil, fmt.Errorf("queue prompt: %w", err)
	}

	// Wait for completion via WebSocket
	images, err := c.waitForCompletion(ctx, promptResp.PromptID)
	if err != nil {
		return nil, fmt.Errorf("wait for completion: %w", err)
	}

	return &GenerateResult{
		PromptID:   promptResp.PromptID,
		ImagePaths: images,
		Duration:   time.Since(start),
		Workflow:   cfg,
	}, nil
}

// waitForCompletion connects to ComfyUI WebSocket and waits for the prompt to complete.
func (c *Client) waitForCompletion(ctx context.Context, promptID string) ([]string, error) {
	// WebSocket URL
	wsURL := fmt.Sprintf("ws://%s/ws?clientId=%s",
		c.BaseURL[len("http://"):], c.ClientID)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		// Fallback to polling if WebSocket fails
		return c.pollForCompletion(ctx, promptID)
	}
	defer conn.Close()

	// Set read deadline based on timeout
	conn.SetReadDeadline(time.Now().Add(c.Timeout))

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			// Connection closed or error - try polling
			return c.pollForCompletion(ctx, promptID)
		}

		var msg struct {
			Type string `json:"type"`
			Data struct {
				PromptID string `json:"prompt_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "executing" && msg.Data.PromptID == promptID {
			// Check if this is the "done" signal (data.node is null)
			var execMsg struct {
				Type string `json:"type"`
				Data struct {
					PromptID string `json:"prompt_id"`
					Node     *string `json:"node"`
				} `json:"data"`
			}
			if err := json.Unmarshal(message, &execMsg); err == nil {
				if execMsg.Data.Node == nil {
					// Generation complete
					break
				}
			}
		}
	}

	// Fetch images from history
	return c.fetchImages(promptID)
}

// pollForCompletion falls back to HTTP polling when WebSocket fails.
func (c *Client) pollForCompletion(ctx context.Context, promptID string) ([]string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(c.Timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for generation")
		case <-ticker.C:
			history, err := c.GetHistory(promptID)
			if err != nil {
				continue // Not ready yet
			}
			if history.Status.Completed {
				return c.fetchImages(promptID)
			}
		}
	}
}

// fetchImages downloads all images for a completed prompt.
func (c *Client) fetchImages(promptID string) ([]string, error) {
	history, err := c.GetHistory(promptID)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, output := range history.Outputs {
		for _, img := range output.Images {
			path, err := c.DownloadImage(img.Filename, img.Subfolder, img.Type)
			if err != nil {
				return nil, fmt.Errorf("download %s: %w", img.Filename, err)
			}
			paths = append(paths, path)
		}
	}

	return paths, nil
}

// SystemStats holds ComfyUI system information.
type SystemStats struct {
	System struct {
		OS             string `json:"os"`
		PythonVersion  string `json:"python_version"`
		EmbeddedPython bool   `json:"embedded_python"`
	} `json:"system"`
	Devices []struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Index      int    `json:"index"`
		VRAM       int64  `json:"vram_total"`
		VRAMFree   int64  `json:"vram_free"`
		TorchVRAM  int64  `json:"torch_vram_total"`
	} `json:"devices"`
}

// GetSystemStats returns system information from ComfyUI.
func (c *Client) GetSystemStats() (*SystemStats, error) {
	resp, err := http.Get(c.BaseURL + "/system_stats")
	if err != nil {
		return nil, fmt.Errorf("get system stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("system stats failed: %d", resp.StatusCode)
	}

	var stats SystemStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode stats: %w", err)
	}

	return &stats, nil
}

// ObjectInfo holds available node types from ComfyUI.
type ObjectInfo map[string]struct {
	Input    map[string]interface{} `json:"input"`
	Output   []string               `json:"output"`
	Category string                 `json:"category"`
}

// GetObjectInfo returns available nodes/checkpoints.
func (c *Client) GetObjectInfo() (ObjectInfo, error) {
	resp, err := http.Get(c.BaseURL + "/object_info")
	if err != nil {
		return nil, fmt.Errorf("get object info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("object info failed: %d", resp.StatusCode)
	}

	var info ObjectInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode info: %w", err)
	}

	return info, nil
}

// GetCheckpoints returns available checkpoint models.
func (c *Client) GetCheckpoints() ([]string, error) {
	info, err := c.GetObjectInfo()
	if err != nil {
		return nil, err
	}

	loader, ok := info["CheckpointLoaderSimple"]
	if !ok {
		return nil, fmt.Errorf("CheckpointLoaderSimple not found")
	}

	// Extract checkpoints from the input schema
	required, ok := loader.Input["required"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no required inputs")
	}

	ckptInfo, ok := required["ckpt_name"].([]interface{})
	if !ok || len(ckptInfo) == 0 {
		return nil, fmt.Errorf("no checkpoint info")
	}

	ckptList, ok := ckptInfo[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid checkpoint list format")
	}

	var checkpoints []string
	for _, c := range ckptList {
		if s, ok := c.(string); ok {
			checkpoints = append(checkpoints, s)
		}
	}

	return checkpoints, nil
}

// Ping checks if the ComfyUI server is reachable.
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/system_stats", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}
