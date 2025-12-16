# clood Codebase Overview

## What is clood?

clood is a CLI infrastructure layer for local LLM workflows. It provides:
- Hardware discovery and analysis
- Multi-host Ollama management
- Query routing across Ollama instances
- Model benchmarking

**clood is NOT a coding assistant** - it's the infrastructure that powers them.

## Tech Stack

- **Language**: Go 1.22
- **CLI Framework**: spf13/cobra
- **Styling**: charmbracelet/lipgloss
- **Config**: gopkg.in/yaml.v3

## Directory Structure

```
clood-cli/
├── cmd/clood/
│   └── main.go              # CLI entrypoint, command registration
├── internal/
│   ├── ollama/
│   │   └── client.go        # Ollama HTTP client (generate, stream, list)
│   ├── system/
│   │   └── hardware.go      # Hardware detection (GPU, RAM, disk)
│   ├── hosts/
│   │   └── hosts.go         # Multi-host management
│   ├── config/
│   │   └── config.go        # YAML config loading (~/.config/clood/)
│   ├── router/
│   │   └── router.go        # Query classification + host routing
│   ├── commands/
│   │   ├── system.go        # `clood system` - hardware info
│   │   ├── hosts.go         # `clood hosts` - list/check hosts
│   │   ├── models.go        # `clood models` - list models
│   │   ├── bench.go         # `clood bench` - benchmarking
│   │   ├── ask.go           # `clood ask` - query execution
│   │   └── health.go        # `clood health` - full health check
│   └── tui/
│       └── branding.go      # ASCII art, color palette, styles
├── llm-context/             # This folder (LLM documentation)
└── go.mod
```

## Commands

| Command | Description |
|---------|-------------|
| `clood` | Show banner and quick status |
| `clood system` | Display hardware info and recommendations |
| `clood hosts` | List and check Ollama hosts |
| `clood models` | List models across all hosts |
| `clood bench [model]` | Benchmark a model |
| `clood ask "query"` | Route and execute a query |
| `clood health` | Full health check |
| `clood init` | Create default config |

## Configuration

Location: `~/.config/clood/config.yaml`

```yaml
hosts:
  - name: local-gpu
    url: http://localhost:11434
    priority: 1
  - name: ubuntu25
    url: http://192.168.4.63:11434
    priority: 1

tiers:
  fast:
    model: qwen2.5-coder:3b
  deep:
    model: qwen2.5-coder:7b

routing:
  strategy: fastest
  fallback: true

defaults:
  stream: true
  timeout: 30s
```

## Building

```bash
cd clood-cli
go mod tidy
go build -o clood ./cmd/clood
./clood --help
```

## Key Concepts

### Tiers
- **Tier 1 (Fast)**: Simple queries → smaller model
- **Tier 2 (Deep)**: Complex queries → larger model

### Routing
Queries are classified by keywords and routed to the best available host that has the required model.

### Hosts
Multiple Ollama instances can be configured with priorities. clood checks health and routes to the best available host.
