package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Message represents a chat message
type Message struct {
	Role      string     `json:"role"` // "system", "user", "assistant", "tool"
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a function call from the model
type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains the function name and arguments
type ToolCallFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Tool defines a tool the model can use
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction defines the function signature
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

// ToolParameters defines the function parameters schema
type ToolParameters struct {
	Type       string                  `json:"type"` // "object"
	Properties map[string]ToolProperty `json:"properties"`
	Required   []string                `json:"required,omitempty"`
}

// ToolProperty defines a single parameter
type ToolProperty struct {
	Type        string `json:"type"` // "string", "integer", "boolean"
	Description string `json:"description"`
}

// ChatRequest is the request body for /api/chat
type ChatRequest struct {
	Model    string                 `json:"model"`
	Messages []Message              `json:"messages"`
	Tools    []Tool                 `json:"tools,omitempty"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ChatResponse is the response from /api/chat
type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`

	// Timing info (only on final response)
	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}

// Chat sends a chat request without tools
func (c *Client) Chat(model string, messages []Message) (*ChatResponse, error) {
	return c.ChatWithTools(model, messages, nil)
}

// ChatWithTools sends a chat request with optional tools
func (c *Client) ChatWithTools(model string, messages []Message, tools []Tool) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// ChatStream sends a chat request and streams the response
func (c *Client) ChatStream(model string, messages []Message, callback func(ChatResponse)) (*ChatResponse, error) {
	return c.ChatStreamWithTools(model, messages, nil, callback)
}

// ChatStreamWithTools sends a chat request with tools and streams the response
func (c *Client) ChatStreamWithTools(model string, messages []Message, tools []Tool, callback func(ChatResponse)) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Use client without timeout for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Post(c.BaseURL+"/api/chat", "application/json", bytes.NewReader(body))
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

	var lastResponse ChatResponse
	for scanner.Scan() {
		var chunk ChatResponse
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

// HasToolCalls checks if the response contains tool calls
func (r *ChatResponse) HasToolCalls() bool {
	return len(r.Message.ToolCalls) > 0
}

// GetToolCalls returns the tool calls from the response
func (r *ChatResponse) GetToolCalls() []ToolCall {
	return r.Message.ToolCalls
}
