# Model Evaluation Guide

How to use clood to evaluate LLMs and run "Summit of Chimborazo" experiments.

---

## What is a Chimborazo Experiment?

Named after the peak in Ecuador that's the farthest point from Earth's center, a "Chimborazo" experiment asks: **Can local LLMs build something real?**

The format:
1. Define a concrete coding task (implement a module, recreate a tool)
2. Run multiple models against the same prompt
3. Compare outputs for correctness, style, and speed
4. Synthesize the best approaches into the final implementation
5. Document lessons learned

This isn't just benchmarking—it's **adversarial collaboration**. The models compete, but you synthesize their best ideas.

---

## Quick Start: Your First Catfight

```bash
# 1. See what models you have
clood models

# 2. Run a simple catfight
clood catfight "Write a Go function that reverses a string"

# 3. Watch it live
clood catfight-live "Implement a stack data structure in Python"

# 4. Save results for analysis
clood catfight -o /tmp/battle-001 "Implement binary search"
```

That's it. You're now evaluating models.

---

## The Model Evaluation Toolkit

### Core Commands

| Command | Purpose |
|---------|---------|
| `clood catfight` | Run same prompt on multiple models |
| `clood catfight-live` | Watch responses stream in real-time |
| `clood bench` | Measure raw speed (tokens/sec) |
| `clood watch` | Review catfight results in TUI |
| `clood ask --show-route` | See how clood would classify a query |

### Supporting Commands

| Command | Purpose |
|---------|---------|
| `clood models` | See available models |
| `clood hosts` | Check which machines are online |
| `clood pull` | Download models you need |
| `clood tune` | Get performance recommendations |

---

## Setting Up a Proper Evaluation

### Step 1: Know Your Hardware

```bash
clood system
```

This shows:
- Available VRAM (determines max model size)
- RAM (for CPU inference)
- Recommended models for your hardware

**Rule of thumb:**
- 8GB VRAM → up to 7B models comfortably
- 16GB VRAM → up to 13B models
- 24GB+ VRAM → 32B+ models

### Step 2: Assemble Your Roster

For a good comparison, you want models from different families:

**Recommended Starter Roster:**
```bash
clood pull qwen2.5-coder:3b    # Fast, small, code-focused
clood pull deepseek-coder:6.7b # Proven winner in battles
clood pull mistral:7b          # General purpose
clood pull llama3.1:8b         # Meta's flagship
clood pull codellama:13b       # Larger code model
```

**If you have more VRAM:**
```bash
clood pull deepseek-r1:14b     # Reasoning model
clood pull qwen2.5-coder:32b   # Large code model
```

### Step 3: Verify Everything Works

```bash
clood health
clood bench qwen2.5-coder:3b   # Quick performance check
```

---

## Running a Catfight

### Basic Catfight

```bash
clood catfight "Write a function to check if a number is prime"
```

Uses default models: Persian (deepseek-coder:6.7b), Tabby (mistral:7b), Siamese (qwen2.5-coder:3b).

### Custom Model Selection

```bash
clood catfight --models "llama3.1:8b,codellama:13b,deepseek-r1:14b" "Implement quicksort"
```

### From a File (Better for Complex Prompts)

```bash
cat > /tmp/prompt.txt << 'EOF'
Implement a Go HTTP fetcher with the following requirements:

- Fetcher struct with CacheDir and http.Client
- NewFetcher constructor
- CachePath method using SHA256 of URL
- IsCached method to check if URL is cached
- Fetch method with FetchOption pattern
- Clear and ClearAll for cache management
- Proper error wrapping

Use only standard library. No external dependencies.
EOF

clood catfight -f /tmp/prompt.txt -o /tmp/fetcher-battle
```

### Multi-Host Catfight

If you have multiple machines with Ollama:

```bash
# Compare same model on different hardware
clood catfight --hosts "ubuntu25,mac-mini" --cross-host --models "qwen2.5-coder:7b" "Write fizzbuzz"

# Use all models across all hosts
clood catfight --hosts "ubuntu25,mac-mini" "Implement a LRU cache"
```

### Save Results to GitHub Issue

```bash
clood catfight --issue --labels "catfight,evaluation" "Implement a rate limiter"
```

---

## Analyzing Results

### What to Look For

1. **Correctness**
   - Does the code compile/run?
   - Does it handle edge cases?
   - Are there syntax errors?

2. **Completeness**
   - Did it implement all requirements?
   - Are there missing imports?
   - Are there TODO stubs?

3. **Style**
   - Is it idiomatic for the language?
   - Good error handling?
   - Reasonable variable names?

4. **Hallucinations**
   - Did it invent non-existent APIs?
   - Did it import fake packages?
   - Did it use wrong function signatures?

5. **Speed**
   - Time to completion
   - Tokens generated

### Scoring Template

```markdown
## Battle: [Task Name]

| Cat | Model | Time | Lines | Status |
|-----|-------|------|-------|--------|
| 🥇 | model:size | Xs | N | Clean/Minor issues |
| 🥈 | model:size | Xs | N | Description |
| 🥉 | model:size | Xs | N | Description |
| ⚠️ | model:size | Xs | N | Errors |

### Winner Analysis
What made the winner good?

### Common Failures
What mistakes did multiple models make?

### Synthesis Notes
What would you combine from different outputs?
```

---

## Chimborazo Experiments: Recreating Codebases

The real power of catfights comes from building complete systems, not just functions.

### The Pattern

1. **Choose a target** - Something you understand well
2. **Break it into modules** - Each module becomes a battle
3. **Run battles sequentially** - Build foundation first
4. **Synthesize as you go** - Best parts from each model
5. **Document everything** - Future you will thank you

### Experiment Ideas

#### Tier 1: Simple Tools (1-3 battles)

**Todo List CLI**
```
Battle 1: Task struct and JSON serialization
Battle 2: Add/List/Complete commands
Battle 3: File persistence
```

**URL Shortener**
```
Battle 1: Short code generation
Battle 2: In-memory storage
Battle 3: HTTP handlers
```

**Markdown to HTML**
```
Battle 1: Lexer/tokenizer
Battle 2: Parser
Battle 3: HTML renderer
```

#### Tier 2: Medium Complexity (4-6 battles)

**Static Site Generator**
```
Battle 1: File walker with frontmatter parsing
Battle 2: Markdown processor
Battle 3: Template engine integration
Battle 4: Asset pipeline
Battle 5: Dev server with hot reload
```

**API Client SDK**
```
Battle 1: HTTP client with auth
Battle 2: Request/response serialization
Battle 3: Error types and handling
Battle 4: Rate limiting
Battle 5: Retry logic
```

**Log Analyzer**
```
Battle 1: Log line parser (multiple formats)
Battle 2: Aggregation logic
Battle 3: Query interface
Battle 4: Output formatters
```

#### Tier 3: Complex Systems (7+ battles)

**Git-lite (Version Control)**
```
Battle 1: Object hashing (blob, tree, commit)
Battle 2: Object storage
Battle 3: Index/staging area
Battle 4: Commit creation
Battle 5: Ref management (branches, HEAD)
Battle 6: Diff algorithm
Battle 7: Merge strategy
```

**HTTP Framework**
```
Battle 1: Router and path matching
Battle 2: Middleware chain
Battle 3: Context and request handling
Battle 4: Response writers
Battle 5: Static file serving
Battle 6: Template rendering
Battle 7: Session management
```

---

## Running a Full Experiment

### Example: Todo List CLI

#### Setup

```bash
mkdir -p ~/experiments/todo-cli
cd ~/experiments/todo-cli
```

#### Battle 1: Data Model

```bash
cat > /tmp/battle-1-prompt.txt << 'EOF'
Implement a Todo item and TodoList for a CLI todo app in Go.

Requirements:
- Todo struct with: ID (int), Title (string), Done (bool), CreatedAt (time.Time)
- TodoList struct that holds []Todo
- Methods: Add(title) Todo, Complete(id) error, Delete(id) error, List() []Todo
- Filter methods: ListPending(), ListCompleted()
- JSON serialization tags on Todo

Package name: todo
File: todo.go
EOF

clood catfight -f /tmp/battle-1-prompt.txt -o ~/experiments/todo-cli/battle-1
```

#### Analyze Battle 1

```bash
# Look at outputs
ls ~/experiments/todo-cli/battle-1/

# Check each for compilation
for f in ~/experiments/todo-cli/battle-1/*.txt; do
  echo "=== $f ==="
  # Extract code block and try to compile
done
```

#### Synthesize Battle 1

Take the best parts from each model:
- Persian's clean struct tags
- Siamese's error handling
- Tabby's filter implementation

Save to `internal/todo/todo.go`.

#### Battle 2: CLI Commands

```bash
cat > /tmp/battle-2-prompt.txt << 'EOF'
Implement CLI commands for a todo app using cobra.

The TodoList type is already implemented with these methods:
- Add(title string) Todo
- Complete(id int) error
- Delete(id int) error
- List() []Todo
- ListPending() []Todo
- ListCompleted() []Todo

Implement these commands:
- add <title>: Add a new todo
- list [--all|--pending|--done]: List todos
- done <id>: Mark todo as complete
- delete <id>: Delete a todo

Package name: cmd
Use github.com/spf13/cobra
EOF

clood catfight -f /tmp/battle-2-prompt.txt -o ~/experiments/todo-cli/battle-2
```

#### Continue Pattern...

Each battle builds on the previous, creating a complete application.

---

## Tips for Better Experiments

### Prompt Engineering

**Be Specific:**
```
❌ "Write a config parser"
✅ "Write a YAML config parser in Go that:
    - Uses gopkg.in/yaml.v3
    - Defines Config struct with these fields...
    - Returns wrapped errors with file path
    - Validates required fields"
```

**Provide Context:**
```
❌ "Add caching to the fetcher"
✅ "Add caching to the HTTP fetcher. The Fetcher struct already has:
    - CacheDir string
    - http.Client

    Implement CachePath(url) that returns SHA256-based path.
    Implement IsCached(url) bool.
    Modify Fetch to check cache first."
```

**Specify Constraints:**
```
❌ "Implement a web server"
✅ "Implement a web server using only the standard library.
    No external dependencies. No gorilla/mux, no gin, no chi."
```

### Common Pitfalls

1. **Hallucinated Imports**
   Models often invent packages. Always verify imports exist.

2. **Wrong API Signatures**
   Models guess at function signatures. Check against actual docs.

3. **Circular Dependencies**
   When building multi-file systems, models forget what package they're in.

4. **Missing Error Handling**
   Small models especially skip error handling.

5. **Incomplete Implementations**
   Watch for `// TODO` comments or stub functions.

### The Synthesis Step

After each battle:

1. **Compile each output** - Eliminate non-starters
2. **Run tests if possible** - Which ones actually work?
3. **Compare approaches** - Different models solve problems differently
4. **Cherry-pick the best** - Take error handling from one, structure from another
5. **Add what's missing** - Models rarely get 100%

---

## Benchmarking for Speed

Sometimes you care about raw performance:

```bash
# Quick benchmark
clood bench qwen2.5-coder:3b

# Compare multiple models
for model in qwen2.5-coder:3b deepseek-coder:6.7b llama3.1:8b; do
  echo "=== $model ==="
  clood bench $model
  echo
done

# Custom prompt for more realistic measurement
clood bench llama3.1:8b --prompt "Explain the visitor pattern with a code example"
```

**Metrics:**
- **Tokens/sec**: Raw generation speed
- **Time to first token**: Latency before response starts
- **Total time**: End-to-end duration

---

## Recording Results

### Battle Log Template

```markdown
# Battle Log: [Project Name]

## Roster
| Model | Size | Host | Notes |
|-------|------|------|-------|
| qwen2.5-coder:3b | 3B | localhost | Fast baseline |
| deepseek-coder:6.7b | 6.7B | localhost | Previous winner |
| ... | ... | ... | ... |

## Battle 1: [Module Name]
**Date:** YYYY-MM-DD
**Prompt:** [Link to prompt file]
**Results:** [Link to output directory]

### Scores
| Model | Time | Lines | Compiles | Works | Score |
|-------|------|-------|----------|-------|-------|
| ... | ... | ... | ✅/❌ | ✅/❌ | X/10 |

### Winner: [Model]
Why: [Analysis]

### Synthesis Notes
- Took X from model A
- Took Y from model B
- Added Z manually

## Battle 2: [Module Name]
...
```

### Lessons Learned Document

After each experiment, capture:

1. **Which models performed best for this task type?**
2. **What mistakes were common?**
3. **How could the prompts be improved?**
4. **What manual work was still required?**
5. **Would you use local models for this again?**

---

## Advanced: The Agent Approach

Instead of just generating code, let models execute and iterate:

```bash
clood agent "Create a file called hello.go with a hello world program, then run it" --verbose
```

The agent can:
- Read files for context
- Write files
- Execute shell commands
- Iterate on errors

This is closer to how a human developer works—try, fail, fix.

---

## Experiment Ideas by Language

### Go
- HTTP client with retries
- CLI tool with cobra
- JSON/YAML config loader
- Simple ORM
- Test helpers

### Python
- FastAPI endpoint
- Pydantic models
- pytest fixtures
- Async task queue
- Data pipeline

### JavaScript/TypeScript
- React component
- Express middleware
- Zod schemas
- State management
- API client

### Rust
- CLI argument parser
- Error types
- Serde serialization
- Async runtime usage
- Iterator implementations

---

## The Chimborazo Philosophy

The goal isn't to prove local models are perfect. They're not. The goal is to:

1. **Understand capabilities** - What can they do well?
2. **Find the sweet spot** - Which models for which tasks?
3. **Develop intuition** - When to trust, when to verify
4. **Save cloud tokens** - Use local for iteration, cloud for polish
5. **Build real things** - Experiments should produce usable code

As the Chimborazo Chronicles noted:
> "We built hello world. The summit is still very far away."

The summit may be far, but every battle gets you closer.

---

## Haiku

```
Nine cats circle code,
Each brings a different answer—
Synthesis is art.
```

---

*"The point on Earth farthest from its center—that's where we're climbing."*
