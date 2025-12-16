# Last Session - 2025-12-16 (Canaries & Golden Paths)

## Status: DESIGN SESSION COMPLETE

Created `clood-canaries.md` - patterns for catching LLM failure modes and producing predictable output.

---

## This Session's Focus: Anti-Pattern Detection

### The Bug We Caught

User asked to "walk through" a file that existed on remote but not local. Instead of:
1. Checking if remote had changes
2. Suggesting `git pull`
3. Asking to verify the path

The LLM pivoted to "let me design this for you" - completely missing the intent.

**Root cause:** LLMs fill gaps with helpfulness instead of questioning assumptions.

### Canary System Designed

**Intent Detection:**
| Keywords | Expectation | If Not Found |
|----------|-------------|--------------|
| review, walk through, show me | EXISTS | STOP - verify |
| create, make, build | CREATE | Proceed |

**Pre-Flight Checks:**
- Git freshness (is remote ahead?)
- File existence for referenced paths
- Intent classification (review vs create)

### Golden Paths Created

Like civit.ai prompts that produce consistent images, clood recipes produce consistent code:

```yaml
name: "new-cli-command"
prompt: |
  Add command `clood {name}` that {description}.
  Follow patterns from cmd/root.go...
expected_output:
  files_created: ["cmd/{name}.go"]
  files_modified: ["cmd/root.go"]
```

### The Chimbo Pattern

```
ORIENT  → Read before assuming
PLAN    → State minimal change
EXECUTE → Do exactly that
VERIFY  → Prove it worked
```

Anti-pattern (what to avoid):
```
ASSUME → EXPAND → CREATE → HOPE
```

---

## Files Created This Session

- `clood-canaries.md` - Full proposal for canaries, golden paths, recipes, structural guidelines

---

## Key Insight: Context Handoff Workflow

The workflow we're using RIGHT NOW:
1. Work with Claude Code on a task
2. Hit context limits or need fresh session
3. Dump session context to LAST_SESSION.md
4. Commit and push
5. Start new session, pull, continue

**This should be a clood feature.**

---

## Proposed: `clood handoff` Command

```bash
# End of session - dump context for next session
clood handoff --save "Worked on canaries, next: implement pre-flight checks"

# Start of session - load context from last handoff
clood handoff --load

# Show handoff history
clood handoff --history
```

### Implementation Sketch

```go
type Handoff struct {
    Timestamp   time.Time
    Summary     string        // Human-readable what we did
    FilesChanged []string     // What files were touched
    NextSteps   []string      // What to do next
    GitRef      string        // Commit hash at handoff
    Context     string        // Full context blob for LLM consumption
}

// Save creates a handoff checkpoint
func (h *Handoff) Save() error {
    // 1. Capture git status/diff
    // 2. Extract recent file changes
    // 3. Write to ~/.clood/handoffs/{timestamp}.json
    // 4. Optionally commit LAST_SESSION.md
}

// Load retrieves the most recent handoff
func LoadLatestHandoff() (*Handoff, error) {
    // Read from ~/.clood/handoffs/
    // Return most recent
}

// ToPrompt generates LLM-consumable context
func (h *Handoff) ToPrompt() string {
    return fmt.Sprintf(`
## Session Context (from %s)

**Previous work:** %s

**Files changed:**
%s

**Next steps:**
%s

**Git ref:** %s
`, h.Timestamp, h.Summary, strings.Join(h.FilesChanged, "\n"),
   strings.Join(h.NextSteps, "\n"), h.GitRef)
}
```

### Integration with Claude Code

Add to CLAUDE.md:
```markdown
## Session Continuity

At the start of a session, check for handoff context:
1. Run `clood handoff --load` or read LAST_SESSION.md
2. Orient to what was done previously
3. Continue from the stated next steps

Before ending a session with significant work:
1. Offer to create a handoff checkpoint
2. Summarize what was done and what's next
```

---

## Hardware Profiling (Original Ask)

The file that triggered the bug: `clood-cli/docs/HARDWARE_FACTS.md`

Contains:
- One-liner for quick Mac hardware facts collection
- Full `gather-facts.sh` script for comprehensive profiling
- Collected facts for ubuntu25 (benchmark data included)
- Placeholder for mac-mini facts

### Manual Profiling Workflow

```bash
# On target machine, run:
bash gather-facts.sh > $(hostname)-facts.json

# Or quick one-liner for Mac:
echo "{\"hostname\":\"$(hostname)\",\"chip\":\"$(sysctl -n machdep.cpu.brand_string 2>/dev/null || system_profiler SPHardwareDataType | grep 'Chip' | awk -F: '{print $2}' | xargs)\",\"memory_gb\":$(( $(sysctl -n hw.memsize) / 1073741824 )),\"ollama\":$(curl -s http://localhost:11434/api/tags 2>/dev/null || echo 'null')}"

# Paste output into HARDWARE_FACTS.md
```

---

## Next Steps

1. [ ] Implement `clood handoff` command
2. [ ] Add pre-flight checks to clood-cli
3. [ ] Create `docs/recipes/` directory with initial recipes
4. [ ] Add canary rules to CLAUDE.md
5. [ ] Test hardware profiling on mac-mini
6. [ ] Build recipe loader/executor

---

## Previous Session Context (preserved below)

---

# Previous: 2025-12-16 (Night Build)

## Status: BUILD COMPLETE

Binary built and ready for testing: `clood-cli/clood` (13MB arm64)

---

## Vision: clood as Infrastructure Layer

**Not another coding assistant.** clood is the infrastructure layer that answers:
- "What can my hardware run?"
- "Which of my Ollama instances should handle this?"
- "What models should I have installed?"

```
┌─────────────────────────────────────────────────────────┐
│  aider / crush / mods / your-app                        │  ← Apps
├─────────────────────────────────────────────────────────┤
│  clood                                                  │  ← Orchestration
│  - hardware discovery                                   │
│  - model recommendations                                │
│  - multi-host routing                                   │
│  - benchmarking                                         │
├─────────────────────────────────────────────────────────┤
│  Ollama instances (mbp, ubuntu25, mac-mini, etc.)       │  ← Execution
└─────────────────────────────────────────────────────────┘
```

---

## Commands Implemented

| Command | Status | Description |
|---------|--------|-------------|
| `clood` | ✓ | Show banner + quick status |
| `clood system` | ✓ | Hardware analysis (GPU, RAM, disk) |
| `clood hosts` | ✓ | List Ollama endpoints, health check |
| `clood models` | ✓ | What's installed where |
| `clood bench [model]` | ✓ | Benchmark tok/s |
| `clood ask "query"` | ✓ | Route to best host/model and execute |
| `clood health` | ✓ | Full service health check |
| `clood init` | ✓ | Create default config |

---

## Files Created

### Core Infrastructure
```
internal/ollama/client.go     # Ollama HTTP client (generate, stream, list, bench)
internal/system/hardware.go   # Hardware detection (macOS/Linux)
internal/hosts/hosts.go       # Multi-host management
internal/config/config.go     # YAML config loading
internal/router/router.go     # Query classification + host routing (refactored)
```

### Commands
```
internal/commands/system.go   # Hardware analysis
internal/commands/hosts.go    # Host management
internal/commands/models.go   # Model listing
internal/commands/bench.go    # Benchmarking
internal/commands/ask.go      # Query execution (refactored)
internal/commands/health.go   # Health checks (refactored)
```

### LLM Context (for testing with local LLMs)
```
llm-context/README.md         # How to use context files
llm-context/CODEBASE.md       # Project overview
llm-context/API.md            # Types and functions
llm-context/ARCHITECTURE.md   # System design
```

---

## SSH Verification

ubuntu25 is online and ready:
- Hostname: ubuntu25
- Ollama: Running with 7 models
- Models available: qwen2.5-coder:7b, qwen2.5-coder:3b, tinyllama, mistral:7b, llama3.2:3b, deepseek-coder:6.7b, llama3-groq-tool-use:8b

---

## Morning Test Plan

```bash
# 1. Navigate to clood
cd ~/Code/clood/clood-cli

# 2. Basic verification
./clood --version
./clood --help

# 3. Hardware check
./clood system

# 4. Host discovery
./clood hosts

# 5. Model inventory
./clood models

# 6. Health check
./clood health

# 7. Initialize config (if needed)
./clood init

# 8. Test query routing (dry run)
./clood ask "what is a goroutine" --show-route
./clood ask "refactor this function to use channels" --show-route

# 9. Test actual query (requires Ollama)
./clood ask "what is a pointer in Go"

# 10. Benchmark
./clood bench qwen2.5-coder:3b
```

---

## Testing with Strata/Chimborazo

After verifying clood works, test with strata project:

```bash
cd ~/Code/strata
~/Code/clood/clood-cli/clood ask "explain the maury pipeline"
~/Code/clood/clood-cli/clood ask "how do I add a new data source to thoreau"
```

The strata project has pre-generated `llm-context/` files that can be passed to clood.

---

## Config Location

Default config: `~/.config/clood/config.yaml`

If not present, clood uses built-in defaults:
- Hosts: localhost:11434, localhost:11435, ubuntu25, mac-mini
- Fast tier: qwen2.5-coder:3b
- Deep tier: qwen2.5-coder:7b

Run `clood init` to create the config file.

---

## Known Limitations

1. **macOS only tested** - Linux hardware detection written but not tested
2. **No project-level config yet** - Global config only for now
3. **No TUI mode** - Commands only
4. **No conversation history** - Each query is independent

---

## Next Steps After Testing

1. Add TUI mode (Bubble Tea interactive interface)
2. Add conversation history (SQLite)
3. Add project-level config override
4. Add `clood pull` to auto-download recommended models
5. Add RAG context injection from llm-context files

---

---

## LLM Code Generation Experiment Results

Tested hybrid workflow: Local LLMs generate code, Claude reviews and fixes.

### Tasks Tested

| Task | Model | Result | Notes |
|------|-------|--------|-------|
| Cache Path Helper | qwen2.5-coder:3b | Partial Success | Generated working structure, had redundant hex encoding bug |
| Cache Path Helper Tests | qwen2.5-coder:3b | Failed | Re-implemented function with MD5 instead of SHA256, made up test values |
| SVG Writer | qwen2.5-coder:7b | Success | Clean, working code on first attempt |
| SVG Writer Tests | Human-written | N/A | Wrote tests manually after cache helper test failure |
| PR Description | tinyllama | Failed | Hallucinated (claimed code "wraps io package" - false) |

### PRs Created

- **PR #5**: Cache Path Helper (`internal/sources/cache.go`)
  - LLM-generated base code with human fix
  - Tests written by human after LLM tests were unusable

- **PR #6**: SVG Writer (`internal/output/svg.go`)
  - Clean LLM-generated code
  - Human-written tests

### Model Strengths/Weaknesses

| Model | Good For | Struggles With |
|-------|----------|----------------|
| qwen2.5-coder:3b | Simple functions, basic Go syntax | Test generation, edge cases, consistent hashing |
| qwen2.5-coder:7b | More complex tasks, cleaner output | N/A - performed well |
| tinyllama | Quick completions | Technical descriptions, factual accuracy |

### Key Findings

1. **Coder models for code, not tests**: LLM-generated tests often re-implement the function incorrectly or make up expected values
2. **Larger models = cleaner output**: 7b produced cleaner code than 3b
3. **Text models hallucinate on technical content**: tinyllama invented features that don't exist
4. **Hybrid workflow works**: LLMs can scaffold code, humans refine

### Recommended Models by Task Type

| Task Type | Recommended Model | Rationale |
|-----------|------------------|-----------|
| Simple functions | qwen2.5-coder:3b | Fast, adequate for basic tasks |
| Complex logic | qwen2.5-coder:7b | Better reasoning, cleaner output |
| Technical docs | Llama 3.1 8B or DeepSeek R1 | Better factual grounding |
| Code tests | Human-written | LLMs struggle with test correctness |

### Prompt Engineering Notes

For future LLM prompting:
- Be explicit about NOT re-implementing the function in tests
- Provide exact expected values in test prompts
- Include "Do not hallucinate features" warning for documentation
- Consider multi-shot prompting with examples

---

## Chimborazo Repository Status

GitHub: `git@github.com:dirtybirdnj/chimborazo.git`

### Issues Created

| Issue | Title | Status |
|-------|-------|--------|
| #1 | HTTP Fetcher with Caching | Open |
| #2 | Recipe Parser | Open |
| #3 | Cache Path Helper | Closed (PR #5) |
| #4 | SVG Writer | Closed (PR #6) |

### Next Implementation Task

Issue #1: HTTP Fetcher with Caching
- See `specs/001-http-fetcher.md` for detailed spec
- More complex task, good test for 7b model capabilities

---

---

## Comprehensive LLM Performance Report

### Session Summary

Created 4 PRs for Chimborazo using hybrid LLM workflow:

| PR | Task | LLM Used | Human Intervention |
|----|------|----------|-------------------|
| #5 | Cache Path Helper | qwen2.5-coder:3b | Fixed redundant hex encoding bug |
| #6 | SVG Writer | qwen2.5-coder:7b | Wrote all tests |
| #7 | HTTP Fetcher | Human-written | N/A (critical infrastructure) |
| #8 | Recipe Parser | qwen2.5-coder:7b | Fixed types, wrote tests |

### Detailed Intervention Log

#### PR #5 - Cache Path Helper
- **LLM Output**: Generated function structure correctly
- **Bug**: Double hex encoding (`hex.EncodeToString(hash.Sum(nil))` when hash was already a string)
- **Root Cause**: LLM conflated two hashing patterns
- **Fix**: Removed redundant encoding, used `hasher.Sum(nil)` directly
- **Test Issue**: LLM re-implemented function with MD5 instead of SHA256
- **Recommendation**: Provide explicit "DO NOT re-implement the function" in test prompts

#### PR #6 - SVG Writer
- **LLM Output**: Clean, working code on first attempt
- **Human Work**: Wrote all tests manually
- **Notes**: 7b model performed significantly better than 3b
- **Recommendation**: Use 7b for any non-trivial code generation

#### PR #8 - Recipe Parser
- **LLM Output Issues**:
  1. Re-defined types in loader.go (multi-file context problem)
  2. Changed `float64` to `int` for Width/Height
  3. Renamed `OutputConfig` to `Output`
  4. Left `Layer` struct empty with comment
- **Root Cause**: LLM lacks cross-file awareness
- **Good Parts**: Core LoadRecipe() and ValidateRecipe() logic was correct
- **Recommendation**: Include full type definitions in prompts when needed elsewhere

### Model Performance Matrix

| Model | Code Gen | Test Gen | Multi-File | Type Preservation | Text Gen |
|-------|----------|----------|------------|-------------------|----------|
| qwen2.5-coder:3b | ⚠️ | ❌ | ❌ | ⚠️ | N/A |
| qwen2.5-coder:7b | ✅ | ❌ | ❌ | ⚠️ | N/A |
| tinyllama | N/A | N/A | N/A | N/A | ❌ |

Legend: ✅ = Good, ⚠️ = Needs fixes, ❌ = Failed/Unusable

### Key Findings

1. **Test Generation is Unreliable**: LLMs consistently fail at writing correct tests
   - Re-implement functions instead of testing them
   - Make up expected values
   - Miss edge cases

2. **Multi-File Context is Missing**: LLMs don't understand project structure
   - Re-define types that exist elsewhere
   - Don't follow existing naming conventions

3. **Larger Models = Better Quality**: 7b consistently outperformed 3b
   - Cleaner code structure
   - Better error handling patterns
   - More idiomatic Go

4. **Documentation Generation Hallucinates**: tinyllama invented non-existent features

5. **Core Logic is Usually Correct**: Even when fixes are needed, the algorithmic approach is sound

### Recommendations for clood Improvements

#### 1. Context Injection Commands
```bash
# New command to inject file context
clood context --files internal/config/*.go "Write tests for..."

# Automatically include relevant types
clood ask --with-types "Implement LoadRecipe"
```

#### 2. Structured Output Modes
```bash
# Code-only output (no markdown, no explanation)
clood ask --format=code "Write a function..."

# Get just the function body
clood ask --format=code-body "..."
```

#### 3. Validation Pipeline
```bash
# Auto-validate generated Go code
clood ask --validate=go "Write a function..."
# Runs go build, go vet, returns errors to LLM for self-correction
```

#### 4. Test Generation Safeguards
```bash
# Include warning in system prompt for test generation
clood ask --mode=test "Write tests for CachePathForURL"
# Adds: "DO NOT re-implement. Use the ACTUAL function. Verify expected values."
```

#### 5. Multi-Model Workflow
```bash
# Use different models for different tasks
clood generate --code-model=qwen2.5-coder:7b --doc-model=llama3.1:8b
```

### How Claude Code Could Use clood

**Current State**: Claude Code doesn't use clood - it uses its own tools (Glob, Grep, Read).

**Potential Integration Points**:

1. **Pre-Flight Context Gathering**
   ```bash
   # Before starting a task, Claude could run:
   clood summary ./  # Get project overview
   clood symbols internal/config  # Get type definitions
   clood tree --depth=2  # Get structure
   ```

2. **Delegating Simple Tasks**
   ```bash
   # Claude could delegate boilerplate to local LLM:
   clood ask "Generate a Go struct from this JSON schema: {...}"
   clood ask "Add yaml tags to this struct: {...}"
   ```

3. **Token Savings**
   - Use clood for initial exploration (cheaper)
   - Only send relevant context to Claude
   - Local LLMs for repetitive transformations

4. **Command Additions Needed**:
   ```bash
   clood deps         # Show dependency graph
   clood types FILE   # Extract type definitions
   clood funcs FILE   # List function signatures
   clood imports FILE # Show import relationships
   ```

### Experimental: llama3.1:8b for Text Generation

Currently downloading llama3.1:8b on ubuntu25. Hypothesis: better instruction following will improve:
- PR descriptions
- Technical documentation
- Error messages

Will test in next session.

### Files Changed This Session

**Chimborazo PRs**:
- PR #5: `internal/sources/cache.go`, `cache_test.go`
- PR #6: `internal/output/svg.go`, `svg_test.go`
- PR #7: `internal/sources/fetcher.go`, `fetcher_test.go`
- PR #8: `internal/config/recipe.go`, `loader.go`, `loader_test.go`

**Total Lines of Code**: ~1,200 lines across 8 files

### Morning Review Checklist

1. [ ] Review and merge PRs #5-8
2. [ ] Test clood commands locally
3. [ ] Consider implementing clood improvements
4. [ ] Test llama3.1:8b for documentation generation

---

*Comprehensive report complete. Happy hacking!*
