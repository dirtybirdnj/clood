# clood Architecture

## Design Philosophy

clood is an **infrastructure layer**, not a coding assistant. It answers:
- "What can my hardware run?"
- "Which Ollama host should handle this query?"
- "What models should I have installed?"

Applications like aider, crush, or custom tools sit on top of clood.

## System Diagram

```
┌─────────────────────────────────────────────────────────┐
│  Applications (aider, crush, mods, custom)              │
├─────────────────────────────────────────────────────────┤
│  clood CLI                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │  Router  │ │  Config  │ │  Hosts   │ │  System  │   │
│  │          │ │          │ │ Manager  │ │ Hardware │   │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────────┘   │
│       │            │            │                       │
│       └────────────┴────────────┘                       │
│                    │                                    │
│            ┌───────┴───────┐                            │
│            │ Ollama Client │                            │
│            └───────┬───────┘                            │
├────────────────────┼────────────────────────────────────┤
│                    │                                    │
│  ┌─────────────────┼─────────────────┐                  │
│  │                 │                 │                  │
│  ▼                 ▼                 ▼                  │
│ Ollama          Ollama           Ollama                 │
│ localhost       ubuntu25         mac-mini               │
│ :11434          :11434           :11434                 │
└─────────────────────────────────────────────────────────┘
```

## Data Flow

### Query Execution (`clood ask`)

```
1. User: clood ask "what is a goroutine"
              │
              ▼
2. Router: ClassifyQuery() → TierFast (keyword match)
              │
              ▼
3. Config: GetTierModel(1) → "qwen2.5-coder:3b"
              │
              ▼
4. Hosts: GetHostWithModel("qwen2.5-coder:3b")
          → Check all hosts concurrently
          → Filter by model availability
          → Sort by priority, then latency
          → Return best host
              │
              ▼
5. Client: GenerateStream() → Ollama API
              │
              ▼
6. Output: Stream chunks to stdout
```

### Host Discovery (`clood hosts`)

```
1. Config: Load hosts from YAML
              │
              ▼
2. Manager: For each host, concurrently:
            - Ping for latency
            - Get version
            - List models
              │
              ▼
3. Output: Status table with online/offline, latency, model count
```

## Module Responsibilities

| Module | Responsibility |
|--------|----------------|
| `ollama/client` | HTTP communication with Ollama API |
| `hosts/hosts` | Multi-host management, health checks |
| `config/config` | YAML config loading, defaults |
| `router/router` | Query classification, host selection |
| `system/hardware` | Hardware detection (GPU, RAM) |
| `commands/*` | CLI command implementations |
| `tui/branding` | Styling, colors, ASCII art |

## Routing Algorithm

### Query Classification

```go
complexIndicators := []string{
    "refactor", "implement", "debug", "fix",
    "modify", "change", "analyze", "compare",
}

simpleIndicators := []string{
    "what is", "how do i", "why does",
    "convert", "syntax for", "example of",
}

// Score each category, pick higher score
// Length > 200 chars → +1 complex
// Multiple sentences → +1 complex
```

### Host Selection

```go
1. Filter: hosts with required model
2. Filter: online hosts only
3. Sort: by priority (lower first)
4. Sort: by latency (lower first)
5. Return: first match
6. Fallback: if enabled, try any online host
```

## Configuration Hierarchy

```
Built-in defaults
        │
        ▼
~/.config/clood/config.yaml (global)
        │
        ▼
Command-line flags (--tier, --model, --host)
```

## Error Handling

- **No hosts online**: Show error, suggest `clood hosts`
- **Model not found**: Warn, fallback to any host (if enabled)
- **Ollama error**: Show HTTP error, continue to next host
- **Config parse error**: Fall back to defaults

## Extension Points

### Adding a New Command

1. Create `internal/commands/mycommand.go`
2. Implement `func MyCmd() *cobra.Command`
3. Register in `cmd/clood/main.go`

### Adding a New Host Type

1. Implement health check in `hosts/hosts.go`
2. Add to default config in `config/config.go`

### Modifying Routing

1. Edit `complexIndicators` / `simpleIndicators` in `router/router.go`
2. Or override with `--tier` flag
