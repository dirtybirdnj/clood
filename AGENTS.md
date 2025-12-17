# AGENTS.md — Universal Agent Guidance

This file provides guidance for AI coding agents working in this repository.
Compatible with Claude Code, Cursor, Gemini CLI, and other agentic tools.

---

## Project Overview

**clood** — Lightning in a Bottle

A CLI tool for orchestrating local LLM inference across multiple machines.
Part of the Legend of Clood: a rebellion against cloud API dependency.

```
The Server Garden:

     🏯 Jade Palace (MacBook Air)
          │ Command
          │
   ┌──────┴──────┐
   │             │
🏰 Iron Keep  🗿 Sentinel
  (ubuntu25)   (mac-mini)
```

---

## Key Concepts

| Concept | Description |
|---------|-------------|
| Host | An Ollama instance (local or remote) |
| Tier | Model category: fast, deep, analysis, writing |
| Routing | Selecting which host/model handles a query |
| Session | Work context passed between agents via CONTEXT.yaml |

---

## Directory Structure

```
clood/
├── clood-cli/           # Go CLI application
│   ├── cmd/clood/       # Entry point
│   ├── internal/        # Internal packages
│   │   ├── commands/    # CLI commands
│   │   ├── config/      # Configuration handling
│   │   ├── hosts/       # Host management
│   │   ├── ollama/      # Ollama API client
│   │   └── tui/         # Terminal UI components
│   └── config.example.yaml
├── lore/                # Narrative and metaphors
├── docs/                # Documentation
├── CONTEXT.yaml         # Session state (if present)
└── CLAUDE.md            # Claude Code specific rules
```

---

## Working in This Codebase

### Before Making Changes

1. Check `CONTEXT.yaml` for current session state:
   ```bash
   clood session show
   ```

2. Understand the focus files and current blockers

3. Check related GitHub issues:
   ```bash
   clood issues
   ```

### Making Changes

1. This is a Go project. Rebuild after changes:
   ```bash
   cd clood-cli
   go build -o clood ./cmd/clood
   ```

2. Test your changes:
   ```bash
   ./clood hosts
   ./clood ask "test query"
   ```

3. Follow existing patterns in `internal/commands/`

### After Making Changes

1. Update session context:
   ```bash
   clood session save "what you did, next: remaining tasks"
   ```

2. Commit with descriptive message (include haiku if appropriate)

---

## Code Style

- Go standard formatting (`go fmt`)
- Cobra for CLI commands
- YAML for configuration
- Error messages should be helpful

### Command Structure

```go
func NewCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "name",
        Short: "Brief description",
        Long:  "Detailed description",
        Run: func(cmd *cobra.Command, args []string) {
            // Implementation
        },
    }
    return cmd
}
```

---

## The Spirits (Technical Metaphors)

| Spirit | Technical Concept |
|--------|------------------|
| Tanuki (Ollama) | Model manager, shapeshifts between models |
| Tengu | GPU-accelerated inference |
| Scholar Spirit | Session context (`clood session`) |
| Archivist | Prior-art detection (`clood research`) |
| Kappa's Bowl | Context window limitations |

See `lore/METAPHORS.md` for the full mapping.

---

## Common Tasks

### Add a new command

1. Create `internal/commands/yourcommand.go`
2. Follow pattern from existing commands
3. Register in `cmd/clood/main.go`
4. Rebuild and test

### Fix host connectivity

1. Check `~/.config/clood/config.yaml` for correct IPs
2. Test directly: `curl http://HOST:11434/api/version`
3. See `docs/DIAGNOSE_HOST.md` for troubleshooting

### Update session context

```bash
clood session save "summary of work, next: task1, task2" --commit
```

---

## Don'ts

- Don't hardcode IP addresses (use config)
- Don't skip the rebuild step after Go changes
- Don't forget to update CONTEXT.yaml for the next agent
- Don't install packages without asking (see CLAUDE.md)

---

*"The garden remembers what the gardener forgets."*
