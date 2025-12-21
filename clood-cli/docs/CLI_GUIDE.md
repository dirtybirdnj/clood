# clood CLI Guide

Complete reference for all clood commands, organized by workflow.

---

## Quick Start

```bash
# First time? Run these:
clood setup          # Interactive wizard
clood verify         # Check everything works
clood preflight      # See what's available locally

# Then you're ready:
clood ask "explain this codebase"
```

---

## Command Hierarchy

Commands build on each other in layers. Start at the bottom and work up.

```
┌─────────────────────────────────────────────────────────────────┐
│                    🤖 AI-POWERED TOOLS                         │
│  analyze, commit-msg, review-pr, generate-tests, extract       │
│  (Use local LLMs to do real work - require models)             │
└─────────────────────────────────────────────────────────────────┘
                              ▲
                              │ uses
┌─────────────────────────────────────────────────────────────────┐
│                    💬 QUERYING LLMs                            │
│  ask, run, chat                                                 │
│  (Send prompts, get responses - require hosts & models)        │
└─────────────────────────────────────────────────────────────────┘
                              ▲
                              │ requires
┌─────────────────────────────────────────────────────────────────┐
│                 🔌 INFRASTRUCTURE                               │
│  hosts, models, health, discover, pull, tune                   │
│  (What's available, where, how to optimize)                    │
└─────────────────────────────────────────────────────────────────┘
                              ▲
                              │ configured by
┌─────────────────────────────────────────────────────────────────┐
│                 🚀 GETTING STARTED                             │
│  setup, verify, doctor, preflight, system                      │
│  (Bootstrap, verify, diagnose)                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Layer 0: Zero-Network Tools (Always Work)

These tools require **no network, no Ollama, no models**. Pure local operations.

### Codebase Analysis

| Command | Purpose | Example |
|---------|---------|---------|
| `grep` | Regex search | `clood grep "TODO" --type go` |
| `tree` | Directory structure | `clood tree --depth 2` |
| `symbols` | Functions/types | `clood symbols internal/` |
| `imports` | Go dependencies | `clood imports --reverse "router"` |
| `context` | LLM-ready summary | `clood context --tokens 8000` |
| `summary` | Project JSON | `clood summary --json` |

**Key insight**: These are the tools Claude Code should use FIRST before any web search.

### grep - Search codebase

```bash
clood grep "pattern" [path]

Flags:
  -i, --ignore-case     Case insensitive
  -n, --line-number     Show line numbers (default: true)
  -A NUM                Lines after match
  -B NUM                Lines before match
  -C NUM                Lines of context (before + after)
  -t, --type strings    File types: go, py, js, ts, rs
  --files-only          List matching files only
  --count               Just count matches
  -j, --json            JSON output

# Examples
clood grep "func.*Router"                    # Find router functions
clood grep "TODO|FIXME" --type go            # Find todos in Go
clood grep "password" -i --files-only        # Security audit
clood grep "import.*context" -A 5            # Show imports + usage
```

### tree - Directory structure

```bash
clood tree [path]

Flags:
  -d, --depth int    Max depth (default: 3)
  -a, --all          Include hidden files
  -j, --json         JSON output

# Examples
clood tree                      # Current directory, 3 levels
clood tree internal/ -d 5       # Deeper look at internal
clood tree --all                # Show hidden files too
```

### symbols - Extract definitions

```bash
clood symbols [path]

Flags:
  -t, --type string    Filter: func, type, const, var
  -e, --exported       Only exported (capitalized) symbols
  -j, --json           JSON output

# Examples
clood symbols internal/router/                  # All symbols
clood symbols --type func --exported            # Public functions only
clood symbols *.go --type type                  # All type definitions
```

### imports - Dependency analysis

```bash
clood imports [file|dir]

Flags:
  -r, --reverse    Find who imports this package
  -a, --all        Include files with no imports
  -j, --json       JSON output

# Examples
clood imports internal/router/         # What does router import?
clood imports -r "internal/config"     # Who imports config?
```

### context - LLM-ready project summary

```bash
clood context [path]

Flags:
  -t, --tokens int    Target token count (default: 4000)
  --tree              Include tree (default: true)
  --readme            Include README (default: true)
  -j, --json          JSON output

# Examples
clood context --tokens 8000           # Larger context for better LLM
clood context internal/               # Just internal package
```

---

## Layer 1: Infrastructure Discovery

Know what's available before you ask for anything.

### preflight - Session start ritual

```bash
clood preflight

# Shows:
# - Available local tools (grep, tree, symbols, etc.)
# - Ollama hosts and their status
# - Models available on each host
# - Recommended workflow

# RUN THIS EVERY SESSION before doing anything else
```

### system - Hardware detection

```bash
clood system

Flags:
  --json    JSON output

# Shows:
# - CPU model and cores
# - RAM amount
# - GPU and VRAM
# - Disk usage
# - Model recommendations based on your hardware
```

### hosts - Ollama instances

```bash
clood hosts

Flags:
  --garden     ASCII art visualization
  -v, --verbose    Show all model details
  --json       JSON output

# Examples
clood hosts                  # Status of all configured hosts
clood hosts --garden         # Pretty visualization
clood hosts --verbose        # Full model details
```

### models - Available models

```bash
clood models

Flags:
  -H, --host string    Filter by host
  --storage            Show disk usage
  --json               JSON output

# Examples
clood models                           # All models everywhere
clood models -H ubuntu25               # Models on ubuntu25
clood models --storage                 # See disk usage
```

### health - Service status

```bash
clood health

Flags:
  --json    JSON output

# Comprehensive check of:
# - All Ollama hosts
# - Model availability
# - Network connectivity
# - CLI tool status
```

### discover - Find Ollama on network

```bash
clood discover

Flags:
  --subnet string     Subnet to scan (auto-detects)
  --port int          Port (default: 11434)
  --timeout int       Timeout ms (default: 500)
  --json              JSON output

# Examples
clood discover                              # Scan local network
clood discover --subnet 192.168.1.0/24      # Specific subnet
clood discover --timeout 200                # Faster scan
```

### pull - Download models

```bash
clood pull [model]

Flags:
  --host string       Target host
  --tier string       Pull all models for tier: fast, deep, analysis, writing
  --recommend         Show recommended models

# Examples
clood pull qwen2.5-coder:7b                    # Pull to best host
clood pull --host ubuntu25 llama3.1:8b         # Pull to specific host
clood pull --tier analysis                      # All analysis models
clood pull --recommend                          # See recommendations
```

### tune - Performance optimization

```bash
clood tune

Flags:
  -H, --host string    Target host
  --modelfiles         Show optimized modelfile templates
  -j, --json           JSON output

# Examples
clood tune                        # Analyze local setup
clood tune --host ubuntu25        # Analyze remote host
clood tune --modelfiles           # Get optimized configs
```

---

## Layer 2: Querying LLMs

Send prompts, get responses. Three commands with different control levels.

### ask - Auto-routed queries

```bash
clood ask [question]

Flags:
  -T, --tier int         Force tier: 1=fast, 2=deep, 3=analysis, 4=writing
  -H, --host string      Force host
  -m, --model string     Force model
  --with-context         Include smart codebase context
  --scope string         Limit context to directory
  --max-tokens int       Max context tokens (default: 4000)
  --no-context           Skip context injection
  --no-stream            Disable streaming
  --show-route           Show routing without executing
  -v, --verbose          Show routing decisions
  --json                 JSON output

# Examples
clood ask "explain the router"                         # Auto-routes
clood ask "write a haiku" -T 4                         # Writing tier
clood ask "explain this" --with-context                # Include codebase
clood ask "debug this" --with-context --scope internal/ # Scoped context
clood ask "test question" --show-route                 # See routing only
```

**Tier system:**
- Tier 1 (fast): Quick questions, small models (qwen2.5-coder:3b)
- Tier 2 (deep): Complex questions, larger models (qwen2.5-coder:7b)
- Tier 3 (analysis): Reasoning tasks (deepseek-r1, phi4-reasoning)
- Tier 4 (writing): Creative/prose (llama3.1:8b)

### run - Explicit control

```bash
clood run [prompt]

Flags:
  -H, --host string          Target host (for deterministic routing)
  -m, --model string         Specific model
  -a, --agent string         Use preconfigured agent role
  -s, --system string        System prompt
  --system-file string       System prompt from file
  -f, --prompt-file string   Prompt from file (- for stdin)
  -q, --quiet                Only output response
  --no-stream                Disable streaming
  --json                     JSON output

# Examples
clood run --host ubuntu25 --model llama3.1:8b "explain this"
clood run --agent reviewer "review this function"
clood run --system "You are a Go expert" "review this code"
cat file.go | clood run --prompt-file - --json
```

### chat - Interactive session

```bash
clood chat

Flags:
  -g, --goal string      Set focus goal (Gamera-kun guards)
  -T, --tier int         Force tier
  -H, --host string      Force host
  -m, --model string     Force model

# Slash commands in chat:
#   /save FILE    Save conversation
#   /clear        Clear history
#   /stats        Show statistics
#   /context      Show loaded context
#   /goal [NEW]   Show/update goal
#   /quit         Exit

# Examples
clood chat                                    # Continue saga
clood chat --goal "fix the auth bug"          # Start with focus
clood chat -T 4                               # Writing mode
```

---

## Layer 3: Model Comparison

Run same prompt across multiple models to compare outputs.

### catfight - Model battle

```bash
clood catfight [prompt]

Flags:
  -f, --file string      Prompt from file
  -m, --models string    Comma-separated models
  -H, --host string      Single host
  --hosts string         Multiple hosts (comma-separated)
  --cross-host           Compare same model across hosts
  -o, --output string    Save outputs to directory
  -s, --stream           Show progress spinner
  -q, --quiet            No prompt preview
  --issue                Create GitHub issue with results
  --labels string        Labels for issue
  --markdown             Output as markdown
  --json                 JSON output

# Default cats: Persian (deepseek), Tabby (mistral), Siamese (qwen)

# Examples
clood catfight "Write hello world in Go"
clood catfight -f prompt.txt -o /tmp/battle
clood catfight --models "llama3.1:8b,mistral:7b" "Explain X"
clood catfight --hosts "ubuntu25,mac-mini" --cross-host "benchmark this"
clood catfight --issue "Compare sorting algorithms"
```

### catfight-live - Streaming battle

```bash
clood catfight-live [prompt]

# Same as catfight but with live streaming TUI
# Watch all models respond in real-time
```

### watch - TUI viewer

```bash
clood watch

# Scrollable TUI for catfight results
# Navigate between model outputs
```

### bench - Performance benchmark

```bash
clood bench [model]

Flags:
  -H, --host string      Specific host
  -p, --prompt string    Custom prompt
  --json                 JSON output

# Measures:
# - Tokens per second
# - Time to first token
# - Total generation time

# Examples
clood bench                            # Benchmark default model
clood bench llama3.1:8b -H ubuntu25    # Specific model/host
```

---

## Layer 4: AI-Powered Tools

Use local LLMs to accomplish real tasks.

### analyze - Code analysis

```bash
clood analyze [file]

Flags:
  -f, --focus string    Focus: security, performance, bugs, style
  -m, --model string    Override model
  --stdin               Read from stdin
  -j, --json            JSON output

# Examples
clood analyze internal/router/router.go
clood analyze internal/config/ --focus security
git diff | clood analyze --stdin --focus "review changes"
```

### commit-msg - Generate commits

```bash
clood commit-msg

Flags:
  --apply          Generate and commit
  --dry-run        Preview only
  --haiku          Include haiku
  --conventional   Conventional commits format
  --json           JSON output

# Examples
git diff --staged | clood commit-msg    # Pipe diff
clood commit-msg                        # Auto-read staged
clood commit-msg --haiku --apply        # Generate, add haiku, commit
```

### review-pr - PR review

```bash
clood review-pr [PR_NUMBER]

Flags:
  --security           Security focus only
  --dry-run            Preview without posting
  --host string        Ollama host
  -m, --model string   Model (default: qwen2.5-coder:7b)

# Examples
clood review-pr 123                    # Full review
clood review-pr 123 --security         # Security only
clood review-pr 123 --dry-run          # Preview
```

### generate-tests - Test scaffolding

```bash
clood generate-tests <source-file>

Flags:
  --function string    Specific function only
  -o, --output string  Output file
  --style string       Style: table-driven, pytest, jest
  -m, --model string   Model to use

# Examples
clood generate-tests src/auth/login.go
clood generate-tests src/router.go --function HandleRequest
clood generate-tests src/api.go -o src/api_test.go
```

### extract - Structured extraction

```bash
clood extract [file]

Flags:
  --schema string      Field names (comma-separated)
  --auto               Auto-detect schema
  --format string      Output: json, csv, yaml (default: json)
  -o, --output string  Output file
  -m, --model string   Model to use

# Examples
cat emails.txt | clood extract --schema "name,email,company"
clood extract invoice.pdf --schema "date,amount,vendor"
clood extract data.txt --auto --format csv
```

---

## Session Management

Maintain context across sessions.

### session - Context management

```bash
clood session [command]

Commands:
  show    Show current context
  load    Load context (markdown for agents)
  save    Save context
  init    Create new CONTEXT.yaml

# Examples
clood session show           # View current
clood session save           # Save current state
clood session init           # Create CONTEXT.yaml
```

### handoff - Session continuity

```bash
clood handoff [summary]

Flags:
  -s, --save      Save (default if summary provided)
  -l, --load      Load latest
  --history       View history
  --diff          Changes since last
  --no-push       Don't push to remote
  -j, --json      JSON output

# Examples
clood handoff "Completed auth, next: add tests"    # Save
clood handoff --load                                # Resume
clood handoff --history                             # View history
clood handoff --diff                                # What changed?
```

### checkpoint - Save/restore points

```bash
clood checkpoint [command]

Commands:
  save [name]    Save current state
  load [name]    Load checkpoint
  list           List all checkpoints
  show [name]    Show details
  delete [name]  Delete checkpoint

# Examples
clood checkpoint save "planning-dec17"
clood checkpoint load "planning-dec17"
clood checkpoint list
```

### focus - Drift detection

```bash
clood focus [command]

Commands:
  set [goal]     Set session goal
  check [msg]    Check for drift
  status         Show current state
  reset          Clear drift counter
  clear          Remove goal

# Examples
clood focus set "implement auth"
clood focus check "let's add a sidebar"    # Drift warning!
clood focus status
```

### beans - Feature visions

```bash
clood beans [command]

Commands:
  add [vision]   Plant new bean
  list           List all beans
  show [id]      Show bean details
  forge [id]     Convert to GitHub issue
  edit [id]      Edit metadata
  prune [id]     Remove bean

Flags for add:
  --intensity int      1-11 intensity scale
  --provenance string  user, ai, collab, ext

# Intensity stages:
# 1-3  Dreaming    - just an idea
# 4-6  Sprouting   - taking shape
# 7-9  Growing     - needs planning
# 10-11 Star Seed  - ready to forge

# Examples
clood beans add "batch overnight processing"
clood beans add "semantic search" --intensity 8 --provenance ai
clood beans forge 42
```

---

## Agents & Delegation

Orchestrate tasks across LLMs.

### agent - Tool-calling loop

```bash
clood agent [prompt]

Flags:
  --host string       Ollama host
  -m, --model string  Model (default: llama3-groq-tool-use:8b)
  --max-turns int     Max turns (default: 10)
  --system string     Custom system prompt
  -v, --verbose       Show turn-by-turn
  --json              JSON output

# Available tools:
# - execute_shell: Run commands
# - draw_bonsai: ASCII art
# - read_file: Read files
# - list_files: List directory
# - write_file: Write files

# Examples
clood agent "Draw me a bonsai tree"
clood agent "List all Go files and summarize"
clood agent "Create hello.txt with a greeting" --verbose
```

### agents - Role management

```bash
clood agents [command]

Commands:
  show [name]    Show agent config
  init           Create example agents.yaml

# Config location: .clood/agents.yaml or ~/.config/clood/agents.yaml
```

### delegate - Task-oriented execution

```bash
clood delegate [task]

Flags:
  -a, --agent string       Agent role
  -f, --file stringArray   Include files as context
  -H, --host string        Target host
  -m, --model string       Model to use
  --format                 Parse into structured output
  --task-type string       review, generate, document, analyze
  -q, --quiet              Response only
  --json                   JSON output

# Examples
clood delegate --agent reviewer "Review internal/router/"
clood delegate --file broken.go "Fix the bug"
clood delegate --host ubuntu25 "Summarize codebase"
```

---

## MCP Server

Expose clood tools to AI agents.

### mcp - Simple server

```bash
clood mcp

Flags:
  -p, --port int    Port (default: 8765)
  -q, --quiet       Minimal output
  -c, --copy        Copy .mcp.json to clipboard

# Examples
clood mcp             # Start on 8765
clood mcp -p 9000     # Custom port
clood mcp --copy      # Get config for Claude Code
```

### serve - Advanced server

```bash
clood serve

Flags:
  --sse                 Use SSE transport
  --host string         Bind address (default: 127.0.0.1)
  -p, --port int        Port (default: 8765)
  --base-url string     Base URL for SSE endpoints

# Examples
clood serve --sse                               # Local only
clood serve --sse --host 0.0.0.0                # Network access
clood serve --sse --host 0.0.0.0 --base-url http://192.168.4.64:8765
```

**MCP Tools Exposed:**

| Tool | Type | Purpose |
|------|------|---------|
| `clood_grep` | Local | Search codebase |
| `clood_tree` | Local | Directory structure |
| `clood_symbols` | Local | Extract definitions |
| `clood_imports` | Local | Dependency analysis |
| `clood_context` | Local | Project summary |
| `clood_system` | Local | Hardware info |
| `clood_capabilities` | Local | Available features |
| `clood_ask` | Ollama | Query LLM |
| `clood_hosts` | Ollama | Check hosts |
| `clood_models` | Ollama | List models |
| `clood_health` | Ollama | Health check |

---

## Global Flags

Available on all commands:

```bash
-j, --json      # JSON output (for scripting/MCP)
-h, --help      # Command help
-v, --version   # Show version
```

---

## Setup & Maintenance

### setup - Interactive wizard

```bash
clood setup
# Walks through initial configuration
```

### verify - Check installation

```bash
clood verify
# Verifies everything works correctly
```

### doctor - Diagnose issues

```bash
clood doctor
# Diagnoses problems and suggests fixes
```

### update - Self-update

```bash
clood update
# Updates to latest version
```

### completion - Shell completions

```bash
clood completion [bash|zsh|fish|powershell]

# Install:
source <(clood completion bash)    # Bash
source <(clood completion zsh)     # Zsh
clood completion fish | source     # Fish
```

---

## Recommended Workflows

### Daily Session Start

```bash
clood preflight                    # What's available?
clood health                       # Everything working?
clood chat --goal "today's task"   # Start focused
```

### Exploring a Codebase

```bash
clood tree                         # Structure
clood grep "main" --type go        # Entry points
clood symbols internal/            # Definitions
clood context --tokens 8000        # Get full context
clood ask "explain this codebase" --with-context
```

### Code Review

```bash
clood analyze file.go --focus security    # Security check
clood analyze file.go --focus bugs        # Bug hunt
clood review-pr 123 --dry-run             # Preview PR review
```

### Comparing Models

```bash
clood catfight "Write a function to..."
clood catfight-live "Explain X"           # Watch live
clood bench llama3.1:8b                   # Performance test
```

### Session Handoff

```bash
clood handoff "Completed X, next: Y"      # Save state
# ... later or different machine ...
clood handoff --load                       # Resume
```

---

## Configuration

Config file: `~/.config/clood/config.yaml`

```yaml
hosts:
  - name: localhost
    url: http://localhost:11434
  - name: ubuntu25
    url: http://ubuntu25.local:11434
  - name: mac-mini
    url: http://mac-mini.local:11434

tiers:
  fast:
    model: qwen2.5-coder:3b
  deep:
    model: qwen2.5-coder:7b
  analysis:
    model: deepseek-r1:8b
  writing:
    model: llama3.1:8b
```

Agent config: `.clood/agents.yaml`

```yaml
agents:
  reviewer:
    model: qwen2.5-coder:7b
    system: "You are a senior code reviewer..."
  coder:
    model: qwen2.5-coder:7b
    system: "You are an expert programmer..."
```
