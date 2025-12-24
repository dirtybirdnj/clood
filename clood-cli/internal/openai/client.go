// Package openai provides an OpenAI-compatible client for llama.cpp and other
// servers that implement the OpenAI API format.
package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dirtybirdnj/clood/internal/ollama"
)

// Client is an HTTP client for OpenAI-compatible APIs (llama.cpp, vLLM, etc.)
type Client struct {
	BaseURL    string
	APIKey     string // Optional, for cloud providers
	HTTPClient *http.Client
}

// NewClient creates a new OpenAI-compatible client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// NewClientWithKey creates a client with an API key
func NewClientWithKey(baseURL, apiKey string, timeout time.Duration) *Client {
	c := NewClient(baseURL, timeout)
	c.APIKey = apiKey
	return c
}

// ChatMessage represents a message in the chat format
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request body for /v1/chat/completions
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

// ChatChoice represents a single completion choice
type ChatChoice struct {
	Index   int `json:"index"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

// ChatUsage contains token usage statistics
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is the response from /v1/chat/completions
type ChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   ChatUsage    `json:"usage"`
}

// ModelInfo represents a model from /v1/models
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse is the response from /v1/models
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// Generate sends a prompt and returns a response compatible with ollama.GenerateResponse
// This allows the OpenAI client to be used interchangeably with the Ollama client
func (c *Client) Generate(model, prompt string) (*ollama.GenerateResponse, error) {
	req := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	return c.doGenerate(req)
}

// GenerateWithSystem sends a prompt with a system message
func (c *Client) GenerateWithSystem(model, system, prompt string) (*ollama.GenerateResponse, error) {
	req := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	return c.doGenerate(req)
}

// GenerateWithOptions sends a prompt with custom options
func (c *Client) GenerateWithOptions(model, prompt string, opts *ollama.GenerateOptions) (*ollama.GenerateResponse, error) {
	req := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	if opts != nil {
		if opts.NumPredict > 0 {
			req.MaxTokens = opts.NumPredict
		}
		if opts.Temperature > 0 {
			req.Temperature = opts.Temperature
		}
	}

	return c.doGenerate(req)
}

func (c *Client) doGenerate(req ChatRequest) (*ollama.GenerateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	start := time.Now()
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()
	totalDuration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to ollama.GenerateResponse format
	result := &ollama.GenerateResponse{
		Model:         chatResp.Model,
		Done:          true,
		TotalDuration: int64(totalDuration),
	}

	if len(chatResp.Choices) > 0 {
		result.Response = chatResp.Choices[0].Message.Content
	}

	// Map token usage
	result.PromptEvalCount = chatResp.Usage.PromptTokens
	result.EvalCount = chatResp.Usage.CompletionTokens

	// Estimate eval duration from total duration
	// (OpenAI API doesn't provide this granularity)
	if result.EvalCount > 0 {
		result.EvalDuration = int64(totalDuration)
	}

	return result, nil
}

// ListModels returns available models (converted to ollama.Model format)
func (c *Client) ListModels() ([]ollama.Model, error) {
	httpReq, err := http.NewRequest("GET", c.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to ollama.Model format
	models := make([]ollama.Model, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		models[i] = ollama.Model{
			Name:       m.ID,
			ModifiedAt: time.Unix(m.Created, 0),
		}
	}

	return models, nil
}

// Ping checks if the server is reachable
func (c *Client) Ping() (time.Duration, error) {
	start := time.Now()

	httpReq, err := http.NewRequest("GET", c.BaseURL+"/v1/models", nil)
	if err != nil {
		return 0, err
	}

	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return time.Since(start), nil
}

// Health checks basic server health
func (c *Client) Health() error {
	_, err := c.Ping()
	return err
}
