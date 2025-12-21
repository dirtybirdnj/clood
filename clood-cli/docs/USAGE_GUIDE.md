# Clood Usage Guide

A plain English guide to using clood - your local LLM infrastructure toolkit.

---

## What is Clood?

Clood is a command-line tool that helps you:
1. **Discover and manage** Ollama instances across your network
2. **Query local LLMs** without needing cloud APIs
3. **Analyze codebases** without any network calls
4. **Compare models** to find the best one for your task
5. **Maintain context** across coding sessions

Think of it as your control center for local AI development.

---

## The Big Picture: How Commands Relate

Clood commands are organized in layers. Each layer builds on the one below it:

```
┌────────────────────────────────────────────────────────────┐
│  LAYER 4: AI-Powered Work                                  │
│  Commands that use LLMs to accomplish real tasks           │
│  (analyze, commit-msg, review-pr, generate-tests)          │
└────────────────────────────────────────────────────────────┘
                           ↑
                     requires LLMs
                           ↑
┌────────────────────────────────────────────────────────────┐
│  LAYER 3: Querying & Comparing                             │
│  Send prompts to models, compare outputs                   │
│  (ask, chat, catfight, bench)                              │
└────────────────────────────────────────────────────────────┘
                           ↑
                   requires hosts & models
                           ↑
┌────────────────────────────────────────────────────────────┐
│  LAYER 2: Infrastructure                                   │
│  Find hosts, see models, check health                      │
│  (hosts, models, discover, health, pull)                   │
└────────────────────────────────────────────────────────────┘
                           ↑
                  requires setup
                           ↑
┌────────────────────────────────────────────────────────────┐
│  LAYER 1: Getting Started                                  │
│  Install, configure, verify                                │
│  (setup, verify, doctor)                                   │
└────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────┐
│  LAYER 0: Zero-Network Tools (Always Available)            │
│  Local codebase analysis - no network, no LLM needed       │
│  (grep, tree, symbols, imports, context, summary)          │
└────────────────────────────────────────────────────────────┘
```

---

## Getting Started

### First Time Setup

When you first install clood, run these commands in order:

```bash
# 1. Run the interactive setup wizard
clood setup

# 2. Verify everything works
clood verify

# 3. If something's wrong, diagnose it
clood doctor
```

**What `setup` does:** Walks you through configuring your Ollama hosts, creating the config file, and testing connections.

**What `verify` does:** Checks that clood can reach your hosts, that models are available, and that local tools work.

**What `doctor` does:** If something breaks later, doctor diagnoses the problem and suggests fixes.

### The Daily Ritual

Every time you start a coding session, run:

```bash
clood preflight
```

This shows you:
- What local tools are available (grep, tree, symbols, etc.)
- Which Ollama hosts are online
- What models you can use
- The recommended workflow

**Why this matters:** It takes 2 seconds and prevents you from wasting time trying to use something that's offline.

---

## Layer 0: Zero-Network Tools

These commands work **without any network or LLM**. They analyze your codebase locally and instantly.

### Finding Code: `grep`

Search for patterns in your codebase:

```bash
# Find all TODOs in Go files
clood grep "TODO" --type go

# Find function definitions (case-insensitive)
clood grep -i "func.*handler"

# Show context around matches
clood grep "error" -C 3

# Just list files that match (no content)
clood grep "import" --files-only
```

**When to use it:** Whenever you're looking for something in the code. Faster than IDE search, respects .gitignore.

### Understanding Structure: `tree`

See your project layout:

```bash
# Current directory, 3 levels deep (default)
clood tree

# Go deeper into a specific folder
clood tree internal/ --depth 5

# Include hidden files
clood tree --all
```

**When to use it:** Starting on a new project, or trying to understand how code is organized.

### Finding Definitions: `symbols`

Extract function names, types, and constants:

```bash
# All symbols in a directory
clood symbols internal/router/

# Only exported (public) functions
clood symbols --exported --type func

# Only type definitions
clood symbols --type type
```

**When to use it:** When you need to know "what functions exist in this package?" without reading every file.

### Understanding Dependencies: `imports`

See what packages a file uses, or who uses a package:

```bash
# What does router.go import?
clood imports internal/router/router.go

# Who imports the config package? (reverse lookup)
clood imports --reverse "internal/config"
```

**When to use it:** Refactoring, or understanding how code connects.

### Generating Context: `context`

Create a summary of your project optimized for LLMs:

```bash
# Generate ~4000 tokens of context (default)
clood context

# Larger context for bigger models
clood context --tokens 8000

# Just a specific directory
clood context internal/
```

**When to use it:** Before asking an LLM about your codebase. Feed it this context first.

### Project Summary: `summary`

Quick JSON overview of your project:

```bash
clood summary
```

Shows project type, main files, structure. Useful for automation.

---

## Layer 2: Infrastructure

These commands help you understand what resources are available.

### Checking Hosts: `hosts`

See all your configured Ollama instances:

```bash
# Status of all hosts
clood hosts

# Pretty visualization
clood hosts --garden

# Full details including all models
clood hosts --verbose
```

**What you'll see:** Which hosts are online, their latency, and how many models each has.

### Listing Models: `models`

See what models are available:

```bash
# All models everywhere
clood models

# Just one host
clood models --host ubuntu25

# See disk usage
clood models --storage
```

**When to use it:** Choosing which model to use, or checking if a model is downloaded.

### Finding New Hosts: `discover`

Scan your network for Ollama instances:

```bash
# Scan local network
clood discover

# Faster scan
clood discover --timeout 200
```

**When to use it:** Setting up a new machine, or if you forgot what's on your network.

### Health Check: `health`

Comprehensive status of everything:

```bash
clood health
```

Checks all hosts, models, and CLI tools. Use this when something seems wrong.

### Downloading Models: `pull`

Get a model onto a host:

```bash
# Pull to the best available host
clood pull qwen2.5-coder:7b

# Pull to a specific host
clood pull --host ubuntu25 llama3.1:8b

# See recommended models
clood pull --recommend
```

**When to use it:** When you need a model you don't have yet.

### Performance Tuning: `tune`

Get optimization recommendations:

```bash
clood tune
clood tune --host ubuntu25
```

Analyzes your hardware and suggests context window sizes, memory settings, etc.

### System Info: `system`

See your hardware and what it can handle:

```bash
clood system
```

Shows CPU, RAM, GPU, disk space, and recommends models based on your specs.

---

## Layer 3: Querying LLMs

Now we get to actually talking to models.

### Quick Questions: `ask`

The simplest way to query an LLM:

```bash
# Just ask (clood picks the right model and host)
clood ask "explain what a goroutine is"

# See how clood would route it (without running)
clood ask "complex question" --show-route

# Force a specific tier
clood ask "write a poem" --tier 4

# Include codebase context automatically
clood ask "explain this codebase" --with-context
```

**How routing works:**
- **Tier 1 (fast):** Simple questions → small, fast models
- **Tier 2 (deep):** Complex questions → larger models
- **Tier 3 (analysis):** Reasoning tasks → reasoning models
- **Tier 4 (writing):** Creative/prose → writing-optimized models

Clood automatically classifies your question and picks the right tier.

### Explicit Control: `run`

When you need full control over what runs where:

```bash
# Specific host and model
clood run --host ubuntu25 --model llama3.1:8b "explain this"

# Use a predefined agent role
clood run --agent reviewer "review this code"

# Custom system prompt
clood run --system "You are a Go expert" "explain channels"

# Read prompt from file
clood run --prompt-file prompt.txt

# Quiet mode (just the response)
clood run -q "summarize this"
```

**When to use `run` vs `ask`:**
- Use `ask` for quick questions where you don't care about routing details
- Use `run` when you need a specific host, model, or system prompt

### Interactive Chat: `chat`

Have a conversation with context:

```bash
# Start or continue your session
clood chat

# Start with a focus goal
clood chat --goal "fix the auth bug"
```

**What makes chat special:**
- Remembers your conversation history
- Loads project context automatically
- Has a "focus guardian" (Gamera-kun) that warns if you drift off-topic
- Supports slash commands like `/save`, `/clear`, `/stats`

**Slash commands in chat:**
- `/save filename` - Save conversation to file
- `/clear` - Clear history but keep context
- `/stats` - Show conversation statistics
- `/context` - Show what context is loaded
- `/goal` - Show or change your focus goal
- `/quit` - Exit and save

---

## Layer 3.5: Model Comparison

These commands help you find the best model for your needs.

### Catfight: `catfight`

Run multiple models against the same prompt and compare:

```bash
# Default cats: Persian, Tabby, Siamese
clood catfight "Write hello world in Go"

# Specific models
clood catfight --models "llama3.1:8b,mistral:7b" "Explain recursion"

# From a file
clood catfight -f prompt.txt

# Save outputs to review later
clood catfight -o /tmp/battle "Compare sorting algorithms"

# Cross multiple hosts
clood catfight --hosts "ubuntu25,mac-mini" --cross-host "prompt"
```

**Why catfight?** Different models are better at different things. Catfight lets you see which one handles your specific task best.

### Live Catfight: `catfight-live`

Watch all models respond in real-time:

```bash
clood catfight-live "Write a function"
```

Same as catfight but with a TUI that shows all responses streaming simultaneously.

### Watch Results: `watch`

Scrollable viewer for catfight results:

```bash
clood watch
```

### Benchmarking: `bench`

Measure raw performance:

```bash
# Benchmark default model
clood bench

# Specific model
clood bench llama3.1:8b --host ubuntu25
```

Shows tokens per second, time to first token, and total generation time.

---

## Layer 4: AI-Powered Tools

These commands use LLMs to do real work.

### Code Analysis: `analyze`

Deep analysis using reasoning models:

```bash
# Analyze a file
clood analyze internal/router/router.go

# Focus on security
clood analyze internal/auth/ --focus security

# Analyze a git diff
git diff | clood analyze --stdin --focus "review changes"
```

**Focus areas:** `security`, `performance`, `bugs`, `style`

### Commit Messages: `commit-msg`

Generate commit messages from your changes:

```bash
# See what it would generate
clood commit-msg --dry-run

# Generate and commit in one step
clood commit-msg --apply

# Add some flair
clood commit-msg --haiku
```

### PR Review: `review-pr`

Automated code review:

```bash
# Review a PR
clood review-pr 123

# Security-focused review
clood review-pr 123 --security

# Preview without posting
clood review-pr 123 --dry-run
```

### Test Generation: `generate-tests`

Scaffold tests from source code:

```bash
# Generate tests for a file
clood generate-tests src/auth/login.go

# Just one function
clood generate-tests src/router.go --function HandleRequest

# Specific test style
clood generate-tests src/api.go --style table-driven
```

### Data Extraction: `extract`

Pull structured data from unstructured text:

```bash
# Extract with explicit schema
cat emails.txt | clood extract --schema "name,email,company"

# Auto-detect schema
clood extract data.txt --auto

# Output as CSV
clood extract logs.txt --schema "timestamp,level,message" --format csv
```

---

## Session Management

These commands help maintain context across sessions.

### Session Context: `session`

Manage your working context:

```bash
# See current context
clood session show

# Save context
clood session save

# Load context (formatted for agents)
clood session load

# Start fresh
clood session init
```

### Handoff: `handoff`

Pass context between sessions:

```bash
# Save where you are
clood handoff "Completed auth, next: add tests"

# Resume later
clood handoff --load

# See history
clood handoff --history

# What changed since last handoff?
clood handoff --diff
```

**When to use it:** End of day, context limit approaching, or switching machines.

### Checkpoints: `checkpoint`

Save and restore session snapshots:

```bash
# Save current state
clood checkpoint save "before-refactor"

# List checkpoints
clood checkpoint list

# Load a checkpoint
clood checkpoint load "before-refactor"

# See what's in it
clood checkpoint show "before-refactor"
```

### Focus Guardian: `focus`

Stay on track during sessions:

```bash
# Set your goal
clood focus set "implement user authentication"

# Check if you're drifting
clood focus check "let's add a dark mode toggle"
# → Warning: this seems unrelated to your goal

# See current state
clood focus status

# Acknowledge you're intentionally changing direction
clood focus reset

# Clear goal entirely
clood focus clear
```

**How it works:** The focus guardian (Gamera-kun, the wise tortoise) watches your conversation and gently reminds you if you're drifting from your stated goal.

### Beans: `beans`

Track feature ideas as they develop:

```bash
# Plant a new idea
clood beans add "batch overnight processing"

# Add with metadata
clood beans add "semantic search" --intensity 8 --provenance ai

# See all beans
clood beans list

# View one
clood beans show 42

# When ready, turn it into a GitHub issue
clood beans forge 42

# Remove an idea
clood beans prune 13 --reason "out of scope"
```

**Intensity scale (1-11):**
- 1-3: Dreaming (just an idea)
- 4-6: Sprouting (taking shape)
- 7-9: Growing (needs planning)
- 10-11: Star Seed (ready to forge into issue)

**Provenance:** Where did the idea come from?
- `user` - Your original idea
- `ai` - AI suggested it
- `collab` - Emerged from discussion
- `ext` - Inspired by external source

---

## Agents & Delegation

### Agentic Loop: `agent`

Run a model that can use tools:

```bash
# Let the agent figure out how to do something
clood agent "List all Go files and summarize them"

# More verbose to see what it's doing
clood agent "Create a hello.txt file" --verbose
```

**Available tools the agent can use:**
- `execute_shell` - Run shell commands
- `read_file` - Read file contents
- `write_file` - Write files
- `list_files` - List directories
- `draw_bonsai` - ASCII art (for fun)

### Agent Roles: `agents`

Manage predefined agent configurations:

```bash
# See available agents
clood agents

# Show one agent's config
clood agents show reviewer

# Create example config
clood agents init
```

Agents are defined in `.clood/agents.yaml` with specific models, hosts, and system prompts.

### Delegation: `delegate`

Send tasks to remote agents:

```bash
# Delegate a code review
clood delegate --agent reviewer "Review internal/router/"

# Include files as context
clood delegate --agent coder --file broken.go "Fix this bug"

# To any host without an agent
clood delegate --host ubuntu25 "Summarize the codebase"
```

**Difference from `run`:** Delegate is task-oriented (structured output, metadata), while run is prompt-oriented (raw response).

### Pre-flight for Agents: `agent-preflight`

Check for conflicts before agent work:

```bash
clood agent-preflight
```

Checks for uncommitted changes, conflicting processes, etc.

---

## MCP Server

Expose clood tools to AI agents like Claude Code.

### Simple Start: `mcp`

The easy way:

```bash
# Start MCP server
clood mcp

# Copy config to clipboard
clood mcp --copy

# Quiet mode
clood mcp -q
```

### Advanced: `serve`

More control:

```bash
# Start with SSE transport
clood serve --sse

# Expose to network
clood serve --sse --host 0.0.0.0

# Custom port
clood serve --sse --port 8080
```

**Tools exposed via MCP:**

| Tool | Network? | Purpose |
|------|----------|---------|
| `clood_grep` | No | Search codebase |
| `clood_tree` | No | Directory structure |
| `clood_symbols` | No | Code definitions |
| `clood_imports` | No | Dependencies |
| `clood_context` | No | Project summary |
| `clood_system` | No | Hardware info |
| `clood_ask` | Local LLM | Query model |
| `clood_hosts` | Local LLM | Check hosts |
| `clood_models` | Local LLM | List models |
| `clood_health` | Local LLM | Health check |

---

## Common Workflows

### Starting Your Day

```bash
clood preflight              # What's available?
clood health                 # Everything working?
clood chat --goal "today's task"  # Start focused session
```

### Exploring a New Codebase

```bash
clood tree                   # See structure
clood summary                # Quick overview
clood grep "main" --type go  # Find entry points
clood symbols src/           # See all functions
clood context --tokens 8000  # Generate context
clood ask "explain this codebase" --with-context
```

### Code Review

```bash
clood analyze file.go --focus security   # Security check
clood analyze file.go --focus bugs       # Bug hunt
git diff | clood analyze --stdin         # Review your changes
clood review-pr 123 --dry-run            # Preview PR review
```

### Finding the Best Model

```bash
clood catfight "Your specific task"      # Compare defaults
clood catfight-live "Another task"       # Watch live
clood bench llama3.1:8b                  # Raw performance
```

### End of Day Handoff

```bash
clood handoff "Completed X, blocked on Y, next: Z"
# Tomorrow:
clood handoff --load
```

### Multi-Machine Setup

```bash
clood discover               # Find all Ollama instances
clood hosts                  # Check their status
clood models --storage       # See what's where
clood pull --host ubuntu25 qwen2.5-coder:7b  # Get models where needed
```

---

## Tips and Tricks

### JSON Output

Almost every command supports `--json` for scripting:

```bash
clood hosts --json | jq '.hosts[].name'
clood models --json | jq '.models[] | select(.size > 1000000000)'
```

### Combining Tools

```bash
# Find a function and analyze it
clood grep "func HandleAuth" --files-only | head -1 | xargs clood analyze

# Generate context and ask about it
clood context | clood ask --with-context "what's the architecture?"
```

### Shell Completion

```bash
# Bash
source <(clood completion bash)

# Zsh
source <(clood completion zsh)

# Fish
clood completion fish | source
```

### When Things Go Wrong

```bash
clood doctor    # Diagnose problems
clood verify    # Check installation
clood health    # Check all services
```

---

## Quick Reference

| I want to... | Command |
|--------------|---------|
| See what's available | `clood preflight` |
| Search code | `clood grep "pattern"` |
| See project structure | `clood tree` |
| Find function definitions | `clood symbols` |
| Ask a quick question | `clood ask "question"` |
| Have a conversation | `clood chat` |
| Compare models | `clood catfight "prompt"` |
| Analyze code | `clood analyze file.go` |
| Generate commit message | `clood commit-msg` |
| Review a PR | `clood review-pr 123` |
| Save session state | `clood handoff "summary"` |
| Check system status | `clood health` |
| Download a model | `clood pull model:tag` |

---

*"Lightning in a bottle - local models wait in mist, one command away."*
