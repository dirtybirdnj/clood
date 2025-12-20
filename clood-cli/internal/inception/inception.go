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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
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
// These should map to models commonly available on local Ollama installs
func DefaultRegistry() ModelRegistry {
	return ModelRegistry{
		// Knowledge/reasoning experts
		"science":  "llama3.1:8b",
		"math":     "llama3.1:8b",
		"reason":   "llama3.1:8b",
		"expert":   "llama3.1:8b",
		// Code specialists
		"code":     "qwen2.5-coder:3b",
		"coder":    "qwen2.5-coder:3b",
		"debug":    "qwen2.5-coder:3b",
		// Creative/language
		"creative": "mistral:7b",
		"writer":   "mistral:7b",
		// Speed tier
		"fast":     "qwen2.5-coder:3b",
		"quick":    "tinyllama:latest",
		"default":  "llama3.1:8b",
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
	OnSubQueryChunk func(chunk string)      // Called for each streaming chunk
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
	// Check for partial opening tag at the end of buffer (last 15 chars)
	// The tag might be split across chunks: "<sub-" then "query..."
	suffix := buffer
	if len(suffix) > 15 {
		suffix = buffer[len(buffer)-15:]
	}
	partials := []string{"<s", "<su", "<sub", "<sub-", "<sub-q", "<sub-qu", "<sub-que", "<sub-quer", "<sub-query"}
	for _, partial := range partials {
		if strings.Contains(suffix, partial) && !strings.Contains(buffer, "</sub-query>") {
			return true
		}
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
		"stream": true, // Stream for visual feedback
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

	// Stream the response and accumulate
	var responseBuilder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		if chunk.Response != "" {
			responseBuilder.WriteString(chunk.Response)
			// Notify UI of streaming chunk
			if h.OnSubQueryChunk != nil {
				h.OnSubQueryChunk(chunk.Response)
			}
		}
		if chunk.Done {
			break
		}
	}

	result.Response = responseBuilder.String()
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

	// Execute the sub-query (streams via OnSubQueryChunk if set)
	result := h.ExecuteSubQuery(ctx, *query)

	// Remove the sub-query tag from buffer
	cleanedBuffer := strings.Replace(buffer, query.RawMatch, "", 1)

	// Format the injection - if OnSubQueryChunk is set, we already streamed the content
	// so just include the footer. Otherwise include full response.
	var injection string
	if result.Error != nil {
		injection = fmt.Sprintf("\n[Sub-query failed: %v]\n", result.Error)
	} else if h.OnSubQueryChunk != nil {
		// Content was streamed, just add footer
		injection = fmt.Sprintf("\n───── END EXPERT [%.1fs] ─────\n\n", result.Duration.Seconds())
	} else {
		// No streaming, include full response
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
				_, injection, _ := p.Handler.ProcessBuffer(p.ctx, bufferStr)

				// Output everything before the sub-query tag position
				preQuery := strings.Split(bufferStr, "<sub-query")[0]
				if preQuery != "" {
					p.OutputChan <- preQuery
				}

				// Output the injection (sub-query response)
				p.OutputChan <- injection

				// Reset buffer and keep anything AFTER the closing tag
				p.buffer.Reset()
				// Find content after </sub-query> in the ORIGINAL buffer
				afterParts := strings.SplitN(bufferStr, "</sub-query>", 2)
				if len(afterParts) > 1 && afterParts[1] != "" {
					p.buffer.WriteString(afterParts[1])
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
const InceptionSystemPrompt = `You are an AI with a special power: you can query expert AI models mid-response.

IMPORTANT: You SHOULD use sub-queries when the task involves:
- Scientific facts, formulas, or calculations → ask "science" or "math"
- Code review, debugging, or best practices → ask "code"
- Creative ideas or writing assistance → ask "creative"

FORMAT (you must use this exact XML format):
<sub-query model="MODEL_NAME">Your specific question here</sub-query>

AVAILABLE EXPERTS:
- science: Physics, chemistry, biology, scientific facts
- math: Calculations, formulas, proofs
- code: Programming, debugging, code review
- creative: Writing, brainstorming, ideas
- reason: Logic, analysis, problem decomposition

EXAMPLE - Here's exactly how to use it:
"To calculate orbital velocity, I need the physics formula.
<sub-query model="science">What is the formula for orbital velocity and what is the orbital velocity at 400km altitude above Earth?</sub-query>
Now using that formula, here's the Python implementation..."

RULES:
1. Use sub-queries proactively - don't guess when you can ask an expert
2. One sub-query at a time - wait for the response before continuing
3. The response will appear inline - continue naturally after receiving it
4. Be specific in your questions to get useful answers`
