# clood + crush Integration Design

## Overview

**clood** = Infrastructure layer (hardware, routing, tuning, context gathering)
**crush** = User interface (TUI, conversation, file editing)

```
┌─────────────────────────────────────────────────────────────────┐
│  User                                                           │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│  crush (Charmbracelet)                                          │
│  - TUI interface                                                │
│  - Conversation management                                      │
│  - File editing with diffs                                      │
│  - LSP integration                                              │
│  - MCP server support                                           │
└─────────────────────┬───────────────────────────────────────────┘
                      │
          ┌───────────┴───────────┐
          │                       │
          ▼                       ▼
┌─────────────────┐     ┌─────────────────────────────────────────┐
│  Cloud APIs     │     │  clood (this project)                   │
│  - Anthropic    │     │  - Hardware detection                   │
│  - OpenAI       │     │  - Multi-host Ollama routing            │
│  - Gemini       │     │  - Model recommendations                │
└─────────────────┘     │  - Performance tuning                   │
                        │  - Codebase context gathering           │
                        └─────────────────────┬───────────────────┘
                                              │
                                              ▼
                        ┌─────────────────────────────────────────┐
                        │  Local Ollama instances                 │
                        │  - ubuntu25 (RTX 4070)                  │
                        │  - mac-mini (M2)                        │
                        │  - localhost                            │
                        └─────────────────────────────────────────┘
```

## Integration Options

### Option 1: MCP Server (Recommended)

clood exposes its capabilities as an MCP server that crush can consume.

**clood MCP server provides:**
- `clood_grep`: Search codebase
- `clood_imports`: Analyze dependencies
- `clood_symbols`: List functions/types
- `clood_context`: Generate LLM context
- `clood_analyze`: Code review
- `clood_route`: Get best host/model for a task

**crush config (~/.config/crush/crush.json):**
```json
{
  "mcp": {
    "clood": {
      "transport": "stdio",
      "command": "clood",
      "args": ["mcp-serve"]
    }
  }
}
```

### Option 2: Pre-flight Context Generation

Before starting crush, run clood to generate context:

```bash
# Generate context for crush session
clood context > .crush-context.md

# Or inject into crush's project config
cat > .crush.json << EOF
{
  "system_prompt": "$(clood context --inline)"
}
EOF
```

### Option 3: Ollama Proxy Mode

clood acts as an Ollama proxy with intelligent routing:

```bash
# Start clood as Ollama-compatible proxy
clood serve --port 11435

# Configure crush to use clood proxy
cat > .crush.json << EOF
{
  "providers": {
    "clood": {
      "type": "openai-compat",
      "base_url": "http://localhost:11435/v1",
      "models": ["auto"]
    }
  }
}
EOF
```

## Workflow: Using crush with clood Backend

### Setup (one-time)

```bash
# 1. Install clood
go install github.com/dirtybirdnj/clood/cmd/clood@latest

# 2. Configure Ollama hosts
clood init

# 3. Tune Ollama on your GPU machine
ssh ubuntu25 'export OLLAMA_FLASH_ATTENTION=1 && export OLLAMA_KV_CACHE_TYPE=q8_0'
clood tune --modelfiles | ssh ubuntu25 'cat > ~/.ollama/Modelfile.analysis && ollama create qwen-analysis -f ~/.ollama/Modelfile.analysis'

# 4. Install crush
brew install charmbracelet/tap/crush
```

### Daily Use

```bash
# Option A: Quick context injection
cd my-project
clood context > .crush-context.md
crush

# Option B: Full context with analysis
clood analyze . --focus architecture > .crush-context.md
crush

# Option C: Let crush call clood tools via MCP
crush  # (with MCP config pointing to clood)
```

### In-Session Commands

When using crush with clood MCP:
```
/tool clood_grep "TODO|FIXME"
/tool clood_imports internal/router
/tool clood_analyze internal/commands/ask.go --focus bugs
```

## clood Commands for crush Users

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `clood context` | Generate project overview | Before starting crush session |
| `clood grep PATTERN` | Search codebase | Finding relevant code |
| `clood imports FILE` | Show dependencies | Understanding file relationships |
| `clood symbols PATH` | List functions/types | Exploring unfamiliar code |
| `clood analyze FILE` | Code review | Before modifying complex code |
| `clood tune` | Ollama recommendations | Setting up new machine |

## Why clood + crush > aider

| Feature | aider | crush + clood |
|---------|-------|---------------|
| TUI quality | Basic REPL | Beautiful Bubble Tea TUI |
| Local LLM | Basic Ollama | Multi-host routing + tuning |
| Context gathering | Manual | clood grep/imports/symbols |
| Hardware optimization | None | clood tune + modelfiles |
| MCP support | No | Yes (crush) |
| LSP integration | No | Yes (crush) |
| Cost control | Per-token | Free local + cloud fallback |

## Implementation Plan

### Phase 1: CLI Interop (Current)
- [x] clood grep, imports, symbols, analyze
- [x] clood tune for Ollama optimization
- [ ] clood context --inline for embedding
- [ ] Documentation for manual workflow

### Phase 2: MCP Server
- [ ] `clood mcp-serve` command
- [ ] Implement MCP stdio transport
- [ ] Tools: grep, imports, symbols, analyze, context
- [ ] crush config examples

### Phase 3: Ollama Proxy
- [ ] `clood serve` as OpenAI-compatible proxy
- [ ] Intelligent routing based on query
- [ ] Request logging and metrics
- [ ] Fallback to cloud APIs

## Testing on Another Workstation

To test clood on chimborazo or another machine:

```bash
# On the remote machine
git clone https://github.com/dirtybirdnj/clood.git
cd clood/clood-cli
go build -o clood ./cmd/clood
./clood --help

# Verify Ollama connectivity
./clood hosts

# Test analysis
./clood analyze some-file.go
```
