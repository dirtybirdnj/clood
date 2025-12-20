# Parallel Multi-Host Inception Plan

## Overview

Extend LLM Inception to support parallel queries across multiple hosts/models,
allowing the main LLM to gather diverse expert opinions and synthesize them.

## New Syntax

```xml
<parallel-query>
  <ask model="science" host="localhost">What is orbital velocity?</ask>
  <ask model="science" host="macmini">What is orbital velocity?</ask>
  <ask model="code" host="localhost">Write the formula in Python</ask>
</parallel-query>
```

- `model` - Required. Model name or alias from registry
- `host` - Optional. Defaults to "localhost". Can be friendly name or IP:port

## Files Modified/Created

### New Files
- `internal/inception/parallel.go` - Core parallel execution logic (CREATED - needs implementation)

### Files to Modify
- `internal/inception/inception.go` - Add parallel detection to StreamProcessor
- `internal/commands/inception.go` - Wire up parallel callbacks and display

## Implementation Phases

### Phase 1: Wire Up Detection (~15 min)

In `inception.go` StreamProcessor.Process():
```go
// After checking for regular sub-query, check for parallel
if block := DetectParallelQuery(bufferStr); block != nil {
    // Process parallel block
}
```

### Phase 2: Implement executeOne (~45 min)

In `parallel.go`, fill in executeOne():
```go
func (h *ParallelHandler) executeOne(ctx context.Context, query ParallelQuery) ParallelResult {
    // 1. Build request body (similar to ExecuteSubQuery)
    reqBody := map[string]interface{}{
        "model":  modelName,
        "prompt": query.Query,
        "stream": true,
        // ...
    }

    // 2. POST to hostURL/api/generate

    // 3. Stream response, calling OnQueryChunk

    // 4. Return accumulated result
}
```

### Phase 3: Host Discovery Integration (~20 min)

Wire into existing infrastructure:
```go
// In NewParallelHandler or setup
hosts, _ := router.GetOnlineHosts()
for _, host := range hosts {
    h.HostRegistry[host.Name] = host.URL
}
```

### Phase 4: UI Integration (~30 min)

In `commands/inception.go`:
```go
// New message type
type inceptionParallelStartMsg struct{ block *inception.ParallelBlock }
type inceptionParallelResultMsg struct{ result inception.ParallelResult }
type inceptionParallelDoneMsg struct{ results []inception.ParallelResult }

// Handle in Update()
case inceptionParallelStartMsg:
    m.content += fmt.Sprintf("\n───── PARALLEL QUERY (%d experts) ─────\n", len(msg.block.Queries))
    // Show which hosts/models are being queried

case inceptionParallelResultMsg:
    // Show individual result as it completes
    m.content += fmt.Sprintf("\n[%s@%s] (%.1fs)\n%s\n",
        msg.result.Query.Model,
        msg.result.Query.Host,
        msg.result.Duration.Seconds(),
        msg.result.Response)

case inceptionParallelDoneMsg:
    m.content += "\n───── END PARALLEL ─────\n"
    // Trigger synthesis continuation
```

### Phase 5: Synthesis Continuation (~15 min)

After parallel results gathered, auto-continue like single sub-queries:
```go
if len(parallelResults) > 0 {
    synthesisPrompt := inception.BuildSynthesisPrompt(parallelResults)
    m.history = append(m.history, ChatMessage{Role: "user", Content: synthesisPrompt})
    // Start new stream
}
```

## System Prompt Update

Add to InceptionSystemPrompt:
```
You can also query MULTIPLE experts in parallel:

<parallel-query>
  <ask model="MODEL" host="HOST">Question</ask>
  <ask model="MODEL" host="HOST">Question</ask>
</parallel-query>

Available hosts: localhost, macmini (or any configured host)

Use parallel queries when you want to:
- Compare answers from different models
- Get multiple perspectives on a problem
- Verify facts across sources
```

## Testing Commands

```bash
# Basic parallel test
./clood inception --model llama3.1:8b

# Prompt to trigger parallel:
"Compare how different models explain orbital velocity.
Query both science experts on localhost and macmini."
```

## Success Criteria

1. Parallel queries execute concurrently (not sequentially)
2. Results stream in as they complete (not all at once)
3. Each result shows host/model/timing
4. Main model receives all results and synthesizes
5. Works across actual different hosts (not just localhost)

## Future Enhancements

- Timeout handling per host (fast fail on slow hosts)
- Automatic retry on host failure
- Load balancing across hosts
- Result caching for identical queries
- Confidence scoring on conflicting answers
