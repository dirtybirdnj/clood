// Package inception provides LLM-to-LLM sub-stream queries.
// This enables one LLM stream to synchronously query another LLM mid-generation.
//
// Example: A coder LLM can ask a science LLM for calculations, wait for the
// response, and continue generating with the new knowledge.
//
// Architecture:
//
//	Main Stream (Coder) ──> detects <sub-query> ──> PAUSE
//	                               │
//	                               ▼
//	                        Sub-Stream (Expert)
//	                               │
//	                               ▼
//	Main Stream (Coder) <── response injected <── RESUME
package inception

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dirtybirdnj/clood/internal/ollama"
)

// SubQueryPattern matches inception triggers in LLM output.
// Format: <sub-query model="model_name">question here</sub-query>
var SubQueryPattern = regexp.MustCompile(`<sub-query\s+model="([^"]+)">([\s\S]*?)</sub-query>`)

// SubQuery represents a detected sub-query in the stream
type SubQuery struct {
	Model    string // Target model for the sub-query (e.g., "science", "math")
	Query    string // The actual query text
	RawMatch string // The full matched pattern (for removal from buffer)
}

// SubQueryResult contains the response from a sub-stream
type SubQueryResult struct {
	Query    SubQuery
	Response string
	Duration time.Duration
	Error    error
}

// ModelRegistry maps friendly names to actual Ollama model names
type ModelRegistry map[string]string

// DefaultRegistry provides common model aliases
func DefaultRegistry() ModelRegistry {
	return ModelRegistry{
		"science":  "qwen2.5:7b",
		"math":     "qwen2.5:7b",
		"code":     "qwen2.5-coder:7b",
		"coder":    "qwen2.5-coder:7b",
		"fast":     "qwen2.5-coder:3b",
		"creative": "llama3.1:8b",
		"default":  "qwen2.5-coder:3b",
	}
}

// Handler processes streams with inception support
type Handler struct {
	Registry     ModelRegistry
	MaxDepth     int           // Maximum nesting depth (default: 1)
	Timeout      time.Duration // Timeout for sub-queries
	OllamaURL    string        // Ollama API URL
	currentDepth int
	mu           sync.Mutex

	// Callbacks for UI integration
	OnSubQueryStart func(query SubQuery)
	OnSubQueryEnd   func(result SubQueryResult)
	OnDepthExceeded func(query SubQuery)
}

// NewHandler creates a new inception handler
func NewHandler() *Handler {
	return &Handler{
		Registry:  DefaultRegistry(),
		MaxDepth:  1, // One level deep by default
		Timeout:   60 * time.Second,
		OllamaURL: "http://localhost:11434",
	}
}

// DetectSubQuery checks if the buffer contains a sub-query pattern
func DetectSubQuery(buffer string) *SubQuery {
	matches := SubQueryPattern.FindStringSubmatch(buffer)
	if matches == nil {
		return nil
	}

	return &SubQuery{
		Model:    strings.TrimSpace(matches[1]),
		Query:    strings.TrimSpace(matches[2]),
		RawMatch: matches[0],
	}
}

// HasPartialSubQuery checks if buffer might contain an incomplete sub-query
// This helps decide whether to wait for more content before processing
func HasPartialSubQuery(buffer string) bool {
	// Check for opening tag without closing tag
	if strings.Contains(buffer, "<sub-query") && !strings.Contains(buffer, "</sub-query>") {
		return true
	}
	return false
}

// ExecuteSubQuery runs a synchronous query to another model
func (h *Handler) ExecuteSubQuery(ctx context.Context, query SubQuery) SubQueryResult {
	start := time.Now()
	result := SubQueryResult{Query: query}

	// Check depth limit
	h.mu.Lock()
	if h.currentDepth >= h.MaxDepth {
		h.mu.Unlock()
		result.Error = fmt.Errorf("inception depth limit (%d) exceeded", h.MaxDepth)
		result.Duration = time.Since(start)
		if h.OnDepthExceeded != nil {
			h.OnDepthExceeded(query)
		}
		return result
	}
	h.currentDepth++
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.currentDepth--
		h.mu.Unlock()
	}()

	// Resolve model name
	modelName := h.resolveModel(query.Model)

	// Notify start
	if h.OnSubQueryStart != nil {
		h.OnSubQueryStart(query)
	}

	// Build the request with a system prompt that prevents further inception
	systemPrompt := `You are an expert assistant responding to a sub-query from another AI.
Be concise and direct. Answer the question precisely.
IMPORTANT: Do NOT use <sub-query> tags in your response. Answer directly.`

	reqBody := map[string]interface{}{
		"model":  modelName,
		"system": systemPrompt,
		"prompt": query.Query,
		"stream": false, // Synchronous - we wait for full response
		"options": map[string]interface{}{
			"num_predict": 500,         // Keep responses concise
			"temperature": 0.3,         // More deterministic for factual queries
			"num_ctx":     4096,        // Reasonable context
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		result.Error = fmt.Errorf("marshal request: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// Create request with context for timeout
	ctx, cancel := context.WithTimeout(ctx, h.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", h.OllamaURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("create request: %w", err)
		result.Duration = time.Since(start)
		return result
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("execute request: %w", err)
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	var ollamaResp ollama.GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		result.Error = fmt.Errorf("decode response: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	result.Response = ollamaResp.Response
	result.Duration = time.Since(start)

	// Notify end
	if h.OnSubQueryEnd != nil {
		h.OnSubQueryEnd(result)
	}

	return result
}

// resolveModel converts a friendly name to an actual model name
func (h *Handler) resolveModel(name string) string {
	if resolved, ok := h.Registry[strings.ToLower(name)]; ok {
		return resolved
	}
	// If not in registry, assume it's already a model name
	return name
}

// ProcessBuffer checks a buffer for sub-queries and handles them
// Returns: cleaned buffer (with sub-query removed), injection text, whether processing occurred
func (h *Handler) ProcessBuffer(ctx context.Context, buffer string) (string, string, bool) {
	query := DetectSubQuery(buffer)
	if query == nil {
		return buffer, "", false
	}

	// Execute the sub-query synchronously
	result := h.ExecuteSubQuery(ctx, *query)

	// Remove the sub-query tag from buffer
	cleanedBuffer := strings.Replace(buffer, query.RawMatch, "", 1)

	// Format the injection
	var injection string
	if result.Error != nil {
		injection = fmt.Sprintf("\n[Sub-query failed: %v]\n", result.Error)
	} else {
		injection = fmt.Sprintf("\n───── 🌀 SUB-QUERY RESPONSE [%s] ─────\n%s\n───── END SUB-QUERY [%.1fs] ─────\n\n",
			query.Model, result.Response, result.Duration.Seconds())
	}

	return cleanedBuffer, injection, true
}

// StreamProcessor wraps a stream channel with inception support
type StreamProcessor struct {
	Handler    *Handler
	InputChan  <-chan string
	OutputChan chan string
	buffer     strings.Builder
	ctx        context.Context
}

// NewStreamProcessor creates a processor that wraps an input channel
func NewStreamProcessor(ctx context.Context, handler *Handler, input <-chan string) *StreamProcessor {
	return &StreamProcessor{
		Handler:    handler,
		InputChan:  input,
		OutputChan: make(chan string, 100),
		ctx:        ctx,
	}
}

// Process reads from input, handles inception, writes to output
func (p *StreamProcessor) Process() {
	defer close(p.OutputChan)

	for {
		select {
		case <-p.ctx.Done():
			return
		case chunk, ok := <-p.InputChan:
			if !ok {
				// Input closed, flush remaining buffer
				if p.buffer.Len() > 0 {
					p.OutputChan <- p.buffer.String()
				}
				return
			}

			p.buffer.WriteString(chunk)
			bufferStr := p.buffer.String()

			// Check for complete sub-query
			if query := DetectSubQuery(bufferStr); query != nil {
				// Found complete sub-query - process it
				cleaned, injection, _ := p.Handler.ProcessBuffer(p.ctx, bufferStr)

				// Output everything before the sub-query tag position
				preQuery := strings.Split(bufferStr, "<sub-query")[0]
				if preQuery != "" {
					p.OutputChan <- preQuery
				}

				// Output the injection (sub-query response)
				p.OutputChan <- injection

				// Reset buffer with cleaned content (after sub-query)
				p.buffer.Reset()
				// Keep anything after the sub-query in the buffer
				afterQuery := strings.SplitN(cleaned, query.RawMatch, 2)
				if len(afterQuery) > 1 {
					p.buffer.WriteString(afterQuery[1])
				}

			} else if HasPartialSubQuery(bufferStr) {
				// Might be incomplete sub-query, hold in buffer
				continue
			} else {
				// No sub-query, output and clear buffer
				p.OutputChan <- bufferStr
				p.buffer.Reset()
			}
		}
	}
}

// Start begins processing in a goroutine and returns the output channel
func (p *StreamProcessor) Start() <-chan string {
	go p.Process()
	return p.OutputChan
}

// InceptionSystemPrompt is added to the main LLM to teach it about sub-queries
const InceptionSystemPrompt = `You have the ability to query expert AI models during your response.

To ask another AI for help, use this format:
<sub-query model="MODEL_NAME">Your question here</sub-query>

Available models:
- science: For scientific facts, physics, chemistry, biology
- math: For calculations and mathematical proofs
- code: For code review and programming help
- creative: For creative writing and brainstorming

Example usage:
"I need to calculate the orbital velocity...
<sub-query model="science">What is the orbital velocity formula and the velocity at 400km altitude?</sub-query>
Using that information, the simulation should..."

The sub-query will be executed and the response injected into your context.
Continue your response naturally after receiving the sub-query result.
Only use ONE sub-query at a time. Wait for the response before asking another.`
