// Package inception - parallel.go
// Multi-host parallel sub-query execution for LLM Inception
//
// This enables the main LLM to query multiple expert models simultaneously
// across different hosts, then compare/synthesize the results.
//
// Example syntax:
//   <parallel-query>
//     <ask model="science" host="localhost">What is orbital velocity?</ask>
//     <ask model="science" host="macmini">What is orbital velocity?</ask>
//     <ask model="code">Write the formula in Python</ask>
//   </parallel-query>

package inception

import (
	"context"
	"regexp"
	"sync"
	"time"
)

// ParallelQueryPattern matches parallel query blocks
// Format: <parallel-query>...<ask> tags...</parallel-query>
var ParallelQueryPattern = regexp.MustCompile(`<parallel-query>([\s\S]*?)</parallel-query>`)

// AskPattern matches individual ask tags within a parallel block
// Format: <ask model="model" host="host">query</ask>
// Host is optional - defaults to localhost
var AskPattern = regexp.MustCompile(`<ask\s+model="([^"]+)"(?:\s+host="([^"]+)")?>([\s\S]*?)</ask>`)

// ParallelQuery represents a single query within a parallel block
type ParallelQuery struct {
	Model    string // Model name or alias (e.g., "science", "llama3.1:8b")
	Host     string // Target host (e.g., "localhost", "macmini", "192.168.1.100:11434")
	Query    string // The actual question
	Index    int    // Position in the parallel block (for ordering results)
}

// ParallelResult holds the result from one parallel query
type ParallelResult struct {
	Query    ParallelQuery
	Response string
	Duration time.Duration
	Error    error
}

// ParallelBlock represents a complete parallel query block
type ParallelBlock struct {
	Queries  []ParallelQuery
	RawMatch string // Full matched pattern for removal
}

// ParallelHandler manages parallel query execution
type ParallelHandler struct {
	Handler       *Handler              // Parent inception handler
	DefaultHost   string                // Default host if not specified
	HostRegistry  map[string]string     // Friendly name -> URL mapping
	Timeout       time.Duration         // Per-query timeout

	// Callbacks for UI integration
	OnQueryStart  func(query ParallelQuery)
	OnQueryChunk  func(query ParallelQuery, chunk string)
	OnQueryEnd    func(result ParallelResult)
	OnAllComplete func(results []ParallelResult)
}

// NewParallelHandler creates a handler for parallel queries
func NewParallelHandler(parent *Handler) *ParallelHandler {
	return &ParallelHandler{
		Handler:     parent,
		DefaultHost: "localhost:11434",
		HostRegistry: map[string]string{
			"localhost": "http://localhost:11434",
			// TODO: Load from config or discovery
		},
		Timeout: 60 * time.Second,
	}
}

// DetectParallelQuery checks if buffer contains a parallel query block
func DetectParallelQuery(buffer string) *ParallelBlock {
	matches := ParallelQueryPattern.FindStringSubmatch(buffer)
	if matches == nil {
		return nil
	}

	block := &ParallelBlock{
		RawMatch: matches[0],
	}

	// Parse individual <ask> tags
	askMatches := AskPattern.FindAllStringSubmatch(matches[1], -1)
	for i, ask := range askMatches {
		host := ask[2]
		if host == "" {
			host = "localhost"
		}
		block.Queries = append(block.Queries, ParallelQuery{
			Model: ask[1],
			Host:  host,
			Query: ask[3],
			Index: i,
		})
	}

	return block
}

// HasPartialParallelQuery checks for incomplete parallel query blocks
func HasPartialParallelQuery(buffer string) bool {
	// Check for opening tag without closing
	if regexp.MustCompile(`<parallel-query`).MatchString(buffer) &&
		!regexp.MustCompile(`</parallel-query>`).MatchString(buffer) {
		return true
	}
	// Check for partial tags at end
	partials := []string{"<p", "<pa", "<par", "<para", "<paral", "<parall", "<paralle", "<parallel", "<parallel-", "<parallel-q"}
	for _, p := range partials {
		if len(buffer) >= len(p) && buffer[len(buffer)-len(p):] == p {
			return true
		}
	}
	return false
}

// ExecuteParallel runs all queries concurrently and gathers results
func (h *ParallelHandler) ExecuteParallel(ctx context.Context, block *ParallelBlock) []ParallelResult {
	results := make([]ParallelResult, len(block.Queries))
	var wg sync.WaitGroup

	for i, query := range block.Queries {
		wg.Add(1)
		go func(idx int, q ParallelQuery) {
			defer wg.Done()
			results[idx] = h.executeOne(ctx, q)
		}(i, query)
	}

	wg.Wait()

	if h.OnAllComplete != nil {
		h.OnAllComplete(results)
	}

	return results
}

// executeOne runs a single query on the specified host
func (h *ParallelHandler) executeOne(ctx context.Context, query ParallelQuery) ParallelResult {
	start := time.Now()
	result := ParallelResult{Query: query}

	if h.OnQueryStart != nil {
		h.OnQueryStart(query)
	}

	// TODO: Implement actual execution
	// 1. Resolve host URL from HostRegistry
	// 2. Resolve model name from Handler.Registry
	// 3. Make HTTP request to host's Ollama API
	// 4. Stream response, calling OnQueryChunk for each chunk
	// 5. Accumulate full response

	hostURL := h.resolveHost(query.Host)
	modelName := h.Handler.resolveModel(query.Model)

	_ = hostURL   // TODO: Use for HTTP request
	_ = modelName // TODO: Use for model parameter

	// STUB: Return placeholder
	result.Response = "TODO: Implement parallel execution"
	result.Duration = time.Since(start)

	if h.OnQueryEnd != nil {
		h.OnQueryEnd(result)
	}

	return result
}

// resolveHost converts a friendly host name to a full URL
func (h *ParallelHandler) resolveHost(name string) string {
	if url, ok := h.HostRegistry[name]; ok {
		return url
	}
	// Assume it's already a URL or host:port
	if regexp.MustCompile(`^https?://`).MatchString(name) {
		return name
	}
	return "http://" + name + ":11434"
}

// FormatComparison creates a display string comparing parallel results
func FormatComparison(results []ParallelResult) string {
	// TODO: Implement nice formatting
	// Something like:
	//
	// ───── PARALLEL QUERY (3 experts) ─────
	//
	// [science@localhost] (2.3s)
	// Response text here...
	//
	// [science@macmini] (3.1s)
	// Response text here...
	//
	// ───── END PARALLEL [3.1s total] ─────

	return "TODO: Format comparison"
}

// BuildSynthesisPrompt creates a prompt asking the main model to compare results
func BuildSynthesisPrompt(results []ParallelResult) string {
	// TODO: Build a prompt that includes all results and asks for synthesis
	// Example:
	//
	// "The expert models provided these responses:
	//
	// [Expert 1 - science@localhost]:
	// <response>
	//
	// [Expert 2 - science@macmini]:
	// <response>
	//
	// Compare these responses. Note any differences or areas of consensus.
	// Then provide your final answer incorporating the best information from each expert."

	return "TODO: Build synthesis prompt"
}

// ParallelSystemPromptAddition is appended to InceptionSystemPrompt when parallel is enabled
const ParallelSystemPromptAddition = `

PARALLEL QUERIES:
You can also query MULTIPLE experts simultaneously using parallel queries:

<parallel-query>
  <ask model="MODEL_NAME" host="HOST_NAME">Your question here</ask>
  <ask model="MODEL_NAME" host="HOST_NAME">Another question</ask>
</parallel-query>

- host is optional (defaults to localhost)
- All queries run in parallel for speed
- Results are gathered and presented together
- Use this to compare perspectives or verify information

Example - comparing expert opinions:
<parallel-query>
  <ask model="science" host="localhost">What causes auroras?</ask>
  <ask model="science" host="macmini">What causes auroras?</ask>
</parallel-query>

After receiving parallel results, synthesize the information and note any differences.`

