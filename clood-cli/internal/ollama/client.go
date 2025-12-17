package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Ollama API
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Ollama client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// DefaultClient returns a client for localhost:11434 with 30s timeout
func DefaultClient() *Client {
	return NewClient("http://localhost:11434", 30*time.Second)
}

// GenerateRequest is the request body for /api/generate
type GenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	System  string                 `json:"system,omitempty"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// GenerateResponse is a single response chunk from /api/generate
type GenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// Model represents a model from /api/tags
type Model struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// TagsResponse is the response from /api/tags
type TagsResponse struct {
	Models []Model `json:"models"`
}

// VersionResponse is the response from /api/version
type VersionResponse struct {
	Version string `json:"version"`
}

// GenerateOptions contains optional parameters for generation
type GenerateOptions struct {
	NumCtx     int     `json:"num_ctx,omitempty"`     // Context window size
	NumPredict int     `json:"num_predict,omitempty"` // Max tokens to generate
	Temperature float64 `json:"temperature,omitempty"` // Sampling temperature
}

// DefaultAnalysisOptions returns optimized options for code analysis
func DefaultAnalysisOptions() *GenerateOptions {
	return &GenerateOptions{
		NumCtx:     16384,
		NumPredict: 4096,
		Temperature: 0.2,
	}
}

// Generate sends a prompt and returns the full response (non-streaming)
func (c *Client) Generate(model, prompt string) (*GenerateResponse, error) {
	return c.GenerateWithOptions(model, prompt, nil)
}

// GenerateWithOptions sends a prompt with custom options
func (c *Client) GenerateWithOptions(model, prompt string, opts *GenerateOptions) (*GenerateResponse, error) {
	req := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	// Apply options if provided
	if opts != nil {
		req.Options = make(map[string]interface{})
		if opts.NumCtx > 0 {
			req.Options["num_ctx"] = opts.NumCtx
		}
		if opts.NumPredict > 0 {
			req.Options["num_predict"] = opts.NumPredict
		}
		if opts.Temperature > 0 {
			req.Options["temperature"] = opts.Temperature
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GenerateWithSystem sends a prompt with a system prompt and returns the full response
func (c *Client) GenerateWithSystem(model, system, prompt string) (*GenerateResponse, error) {
	req := GenerateRequest{
		Model:  model,
		System: system,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GenerateStream sends a prompt and calls the callback for each chunk
func (c *Client) GenerateStream(model, prompt string, callback func(chunk GenerateResponse)) (*GenerateResponse, error) {
	req := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Use a client without timeout for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer for large responses
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lastResponse GenerateResponse
	for scanner.Scan() {
		var chunk GenerateResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue // Skip malformed lines
		}
		if callback != nil {
			callback(chunk)
		}
		if chunk.Done {
			lastResponse = chunk
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}

	return &lastResponse, nil
}

// GenerateStreamWithSystem sends a prompt with system prompt and streams the response
func (c *Client) GenerateStreamWithSystem(model, system, prompt string, callback func(chunk GenerateResponse)) (*GenerateResponse, error) {
	req := GenerateRequest{
		Model:  model,
		System: system,
		Prompt: prompt,
		Stream: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Use a client without timeout for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lastResponse GenerateResponse
	for scanner.Scan() {
		var chunk GenerateResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if callback != nil {
			callback(chunk)
		}
		if chunk.Done {
			lastResponse = chunk
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}

	return &lastResponse, nil
}

// ListModels returns all models available on this Ollama instance
func (c *Client) ListModels() ([]Model, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Models, nil
}

// Version returns the Ollama version
func (c *Client) Version() (string, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/version")
	if err != nil {
		return "", fmt.Errorf("get version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned %d", resp.StatusCode)
	}

	var result VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Version, nil
}

// Ping checks if the Ollama server is reachable
func (c *Client) Ping() (time.Duration, error) {
	start := time.Now()

	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/tags")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("ollama returned %d", resp.StatusCode)
	}

	return time.Since(start), nil
}

// HasModel checks if a specific model is available
func (c *Client) HasModel(name string) (bool, error) {
	models, err := c.ListModels()
	if err != nil {
		return false, err
	}

	for _, m := range models {
		if m.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// PullRequest is the request body for /api/pull
type PullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

// PullResponse is a response chunk from /api/pull
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// Pull downloads a model, calling the callback for progress updates
func (c *Client) Pull(model string, callback func(status string, completed, total int64)) error {
	req := PullRequest{
		Name:   model,
		Stream: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Use client without timeout for potentially long downloads
	pullClient := &http.Client{}
	resp, err := pullClient.Post(c.BaseURL+"/api/pull", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var chunk PullResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if callback != nil {
			callback(chunk.Status, chunk.Completed, chunk.Total)
		}
	}

	return scanner.Err()
}

// BenchmarkResult contains the results of a benchmark run
type BenchmarkResult struct {
	Model            string
	TotalDuration    time.Duration
	LoadDuration     time.Duration
	PromptTokens     int
	GeneratedTokens  int
	PromptTokPerSec  float64
	GenerateTokPerSec float64
}

// Benchmark runs a simple benchmark on the given model
func (c *Client) Benchmark(model string, prompt string) (*BenchmarkResult, error) {
	if prompt == "" {
		prompt = "Write a haiku about programming."
	}

	resp, err := c.Generate(model, prompt)
	if err != nil {
		return nil, err
	}

	result := &BenchmarkResult{
		Model:           model,
		TotalDuration:   time.Duration(resp.TotalDuration),
		LoadDuration:    time.Duration(resp.LoadDuration),
		PromptTokens:    resp.PromptEvalCount,
		GeneratedTokens: resp.EvalCount,
	}

	// Calculate tokens per second
	if resp.PromptEvalDuration > 0 {
		result.PromptTokPerSec = float64(resp.PromptEvalCount) / (float64(resp.PromptEvalDuration) / 1e9)
	}
	if resp.EvalDuration > 0 {
		result.GenerateTokPerSec = float64(resp.EvalCount) / (float64(resp.EvalDuration) / 1e9)
	}

	return result, nil
}
