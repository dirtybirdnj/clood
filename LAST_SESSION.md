# Last Session - 2025-12-15

## Big Win: clood-cli Go Project Scaffolded!

Designed and scaffolded a Go CLI tool called `clood` - a unified interface for local LLM workflows with tiered inference.

---

## Decision: Go + Charm (not Rust)

**Why Go:**
- Crush and mods are both Go/Charm - same ecosystem
- Faster iteration than Rust
- Bubble Tea + Lipgloss for TUI
- Single binary distribution

---

## Project Created: `clood-cli/`

```
clood-cli/
├── cmd/clood/main.go           # Cobra entry point
├── internal/
│   ├── commands/
│   │   ├── ask.go              # Routes queries to mods (Tier 1) or crush (Tier 2)
│   │   ├── context.go          # Generate LLM-optimized context
│   │   ├── health.go           # Check Ollama, SearXNG, LiteLLM status
│   │   ├── summary.go          # Project type detection, JSON output
│   │   ├── symbols.go          # Extract functions/classes/types
│   │   └── tree.go             # Smart directory tree (respects .gitignore)
│   ├── context/context.go      # Project detection
│   ├── git/git.go              # Git utilities
│   ├── parser/parser.go        # Language-specific parsing
│   ├── router/router.go        # Tier 1/2 query classification
│   └── tui/branding.go         # ASCII ART + COLOR PALETTE (customize here!)
├── pkg/manifest/manifest.go    # Public API
└── go.mod
```

---

## Commands Implemented

| Command | Purpose |
|---------|---------|
| `clood` | Show banner, launch TUI (eventually) |
| `clood ask "query"` | Auto-route to mods (simple) or crush (complex) |
| `clood tree [path]` | Directory tree with gitignore support |
| `clood summary [path]` | JSON project manifest |
| `clood context [path]` | Generate LLM context blob |
| `clood symbols [path]` | Extract functions/types from code |
| `clood health` | Check all services status |

---

## Tiered Inference Router

In `internal/router/router.go`:
- **Tier 1 (mods)**: Simple queries - "what is", "how to", "one-liner"
- **Tier 2 (crush)**: Complex queries - "refactor", "implement", "debug", multi-step

Router uses keyword matching + length heuristics. Can be enhanced with local LLM classification later.

---

## ASCII Art Customization

Edit `internal/tui/branding.go`. Character palette documented inline:

```
BOX DRAWING: ─ │ ┌ ┐ └ ┘ ╭ ╮ ╰ ╯ ═ ║ ╔ ╗ ╚ ╝
BLOCKS: █ ▓ ▒ ░ ▄ ▀ ▌ ▐
WEATHER: ☁ ⛅ ⚡ ✨ 💧
GEOMETRIC: ◆ ◇ ○ ● ▲ ▼ ★ ☆
```

Color palette (lightning/cloud theme):
- `ColorPrimary`: #00D4FF (electric blue)
- `ColorAccent`: #FFD700 (lightning gold)
- `ColorSecondary`: #B8C5D0 (cloud gray)

ASCII generators:
- https://patorjk.com/software/taag/
- https://www.ascii-art-generator.org/

---

## Reviewed: Local LLM Outputs

Found 5 untracked files from local LLM experiments:
- `clood-cli-spec.md` - Basic CLI spec
- `clood-outline-1.md` - 5-step install process
- `clood-outline-2.md` - Rust feature list
- `clood-analysis-1.md` - Confused comparison
- `clood-refine-1.md` - Hallucinated completion

Quality was inconsistent - classic local model behavior. The `improve-clood-ask-gemini.md` spec was much better (likely human or stronger model).

---

## Blockers

**Go not installed** - Cannot build yet.

```bash
# To complete setup:
brew install go
cd ~/Code/clood/clood-cli
go mod tidy
go build -o clood ./cmd/clood
./clood --help
```

---

## Integration Notes

For coordinating with aider agent, share this context:

- Config location: `~/.config/clood/` or `.clood/` in project root
- Context files: `CLAUDE.md`, `AGENTS.md`, `README.md` auto-injected
- Ollama endpoints: localhost:11434/11435, ubuntu25:11434, mac-mini:11434
- LiteLLM proxy: localhost:4000

---

## Files Created This Session

```
clood-cli/go.mod
clood-cli/cmd/clood/main.go
clood-cli/internal/commands/ask.go
clood-cli/internal/commands/context.go
clood-cli/internal/commands/health.go
clood-cli/internal/commands/summary.go
clood-cli/internal/commands/symbols.go
clood-cli/internal/commands/tree.go
clood-cli/internal/context/context.go
clood-cli/internal/git/git.go
clood-cli/internal/parser/parser.go
clood-cli/internal/router/router.go
clood-cli/internal/tui/branding.go
clood-cli/pkg/manifest/manifest.go
```

---

## Next Session TODO

1. **Install Go**: `brew install go`
2. **Build clood**: `go mod tidy && go build -o clood ./cmd/clood`
3. **Test commands**: `./clood tree`, `./clood summary`, `./clood health`
4. **Customize ASCII art**: Edit `internal/tui/branding.go`
5. **Integrate with aider**: Share config patterns
6. **Add TUI mode**: Bubble Tea interactive interface

---

*Go CLI scaffolded. Need Go installed to build. ASCII art ready for customization!*
