# clood

![We need clood](soyjacks-pointing-clood-efw.png)

We want Claude! Too bad, we're out of tokens. We have clood instead.

Local LLM toolkit for a Claude-like experience on your own hardware.

## What is this?

clood is a CLI and MCP server for running local LLMs with:

1. **Multi-Host Routing** - Distribute queries across your hardware fleet
2. **Tiered Model Inference** - Fast models for context, powerful models for reasoning
3. **MCP Integration** - Works as an MCP server for Claude Code
4. **Local-First Tools** - grep, tree, symbols, imports - no network needed

## Quick Start

```bash
# Install clood
cd clood-cli
go build -o clood ./cmd/clood
./clood setup

# Check your hosts
./clood preflight
./clood hosts

# Ask a question
./clood ask "What is the best way to structure a Go CLI?"

# Use as MCP server with Claude Code
./clood mcp
```

See [clood-cli/docs/USAGE_GUIDE.md](clood-cli/docs/USAGE_GUIDE.md) for complete documentation.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  Claude Code (or any MCP client)                                    │
└───────────────────────────┬─────────────────────────────────────────┘
                            │ MCP
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│  clood MCP server                                                   │
│  • clood_grep, clood_tree, clood_symbols (local, instant)           │
│  • clood_ask (routes to best available Ollama host)                 │
│  • clood_context, clood_imports (project awareness)                 │
└───────────────────────────┬─────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
     ┌───────────┐   ┌───────────┐   ┌───────────┐
     │ mac-mini  │   │ ubuntu25  │   │mac-laptop │
     │ localhost │   │ .64       │   │ .47       │
     │ M2        │   │ RX 590    │   │ M-series  │
     └───────────┘   └───────────┘   └───────────┘
```

## The Server Garden

| Host | IP | GPU | Models | Role |
|------|-----|-----|--------|------|
| mac-mini | localhost | M2 | 16 models | Always-on, fast routing |
| ubuntu25 | 192.168.4.64 | RX 590 8GB | 17 models | GPU muscle, reasoning |
| mac-laptop | 192.168.4.47 | M-series | 12 models | Heavy lifting (32B models) |

## Key Commands

| Command | Purpose |
|---------|---------|
| `clood preflight` | Start-of-session check - what's available |
| `clood hosts` | Check Ollama host status and latency |
| `clood ask "question"` | Query local LLM (auto-routes) |
| `clood grep PATTERN` | Search codebase (instant, local) |
| `clood tree` | Project structure |
| `clood symbols PATH` | Extract functions/types |
| `clood mcp` | Start MCP server for Claude Code |
| `clood analyze FILE` | Code review with reasoning model |

## MCP Tools (for Claude Code)

When running as an MCP server, clood provides:

| Tool | Purpose | Network |
|------|---------|---------|
| `clood_preflight` | Session startup check | None |
| `clood_grep` | Regex search codebase | None |
| `clood_tree` | Directory structure | None |
| `clood_symbols` | Extract functions/types | None |
| `clood_imports` | Dependency analysis | None |
| `clood_context` | Project summary | None |
| `clood_ask` | Query local Ollama | LAN only |
| `clood_hosts` | Check host status | LAN only |

## Directory Structure

```
clood/
├── clood-cli/              # The CLI tool
│   ├── cmd/clood/          # Main entry point
│   ├── internal/           # Implementation
│   ├── docs/               # CLI documentation
│   │   ├── USAGE_GUIDE.md  # How to use clood
│   │   └── CLI_GUIDE.md    # Command reference
│   └── scripts/            # Build and utility scripts
│
├── docs/                   # Project documentation
├── hardware/               # Machine-specific tuning
├── infrastructure/         # Docker, SSH setup
├── lore/                   # Project history and narrative
└── scripts/                # Utility scripts
```

## Documentation

| Doc | Purpose |
|-----|---------|
| [clood-cli/docs/USAGE_GUIDE.md](clood-cli/docs/USAGE_GUIDE.md) | How to use clood |
| [clood-cli/docs/CLI_GUIDE.md](clood-cli/docs/CLI_GUIDE.md) | Command reference |
| [CLAUDE.md](CLAUDE.md) | Instructions for Claude agents |
| [ollama-tuning.md](ollama-tuning.md) | Performance optimization |
| [hardware/](hardware/) | Machine-specific tuning |

## Requirements

- Go 1.21+ (to build clood)
- Ollama running on at least one host
- Optional: Multiple machines for distributed inference

## TODO

- [ ] llama.cpp integration on ubuntu25
- [ ] Speculative decoding experiments
- [ ] Model evaluation framework (Chimborazo)

---

*Lightning in a Bottle*
