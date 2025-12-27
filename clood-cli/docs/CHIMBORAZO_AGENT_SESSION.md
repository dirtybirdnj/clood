# Chimborazo Agent Session Guide

*For focused agent work on the strata→chimborazo port*

---

## The Mission

Rename and complete the **strata** Python project as **chimborazo** in Go, bringing it to feature parity. This is about making plotter-ready SVG maps from geospatial data.

### What Chimborazo Does

```
Recipe (YAML) → Fetch GIS Data → Process Geometry → Render SVG → Plotter
```

The Python version (strata) works. The Go port (chimborazo) needs to match its output.

---

## Key Resources

### Strata Codebase (the reference implementation)

```
~/Code/strata/
├── llm-context/           # LLM-optimized documentation (START HERE)
│   ├── ARCHITECTURE.md    # System overview, data flow diagrams
│   ├── OPERATIONS.md      # Geometry operations with Go hints
│   ├── TYPES.md           # Data structures
│   ├── PATTERNS.md        # Code patterns
│   └── AGENT_GUIDE.md     # Workflow for agent development
├── src/strata/            # Python source
│   ├── thoreau/           # Data fetching (→ chimborazo/internal/sources)
│   ├── humboldt/          # Geometry ops (→ chimborazo/internal/geometry)
│   ├── kelley/            # SVG output (→ chimborazo/internal/output)
│   └── maury/             # Pipeline (→ chimborazo/pkg/pipeline)
├── examples/              # Recipe files (.strata.yaml)
└── output/                # Reference SVG outputs to match
```

### Chimborazo Rebuild Workspace

```
~/Code/chimborazo-rebuild/   # Clean rebuild workspace
```

### GitHub Issue

```bash
gh issue view 191 --repo dirtybirdnj/clood
```

Contains full session structure, success criteria, and validation gates.

---

## Module Name Mapping

| Python (strata) | Go (chimborazo) | Named After |
|-----------------|-----------------|-------------|
| `thoreau/` | `internal/sources/` | Henry David Thoreau, surveyor |
| `humboldt/` | `internal/geometry/` | Alexander von Humboldt, explorer |
| `kelley/` | `internal/output/` | Florence Kelley, social mapper |
| `maury/` | `pkg/pipeline/` | Matthew Fontaine Maury, oceanographer |

---

## Implementation Order

Based on the session structure from issue #191:

### Phase 1: Analysis (Read-only)
1. Understand strata architecture via `llm-context/`
2. Analyze recipe schema (`examples/*.strata.yaml`)

### Phase 2: Core Implementation
1. **Recipe parser** - YAML → Go structs
2. **HTTP fetcher + cache** - Download Census/CanVec data
3. **GeoJSON parser** - Parse downloaded shapefiles
4. **Geometry operations** - clip, subtract, merge, simplify
5. **SVG renderer** - Geometry → SVG paths

### Phase 3: Integration
1. **Pipeline builder** - Wire modules together
2. **CLI** - `chimborazo build recipe.yaml`

### Phase 4: Validation
1. Build Vermont map
2. Compare to `~/Code/strata/output/vermont_12x18/svg/`

---

## Validation Gates

| Component | Test | Pass Criteria |
|-----------|------|---------------|
| Recipe Parser | `go test ./internal/config` | Parses example recipes |
| HTTP Fetcher | Fetch census URL | File cached locally |
| GeoJSON | Parse testdata | No errors |
| Geometry Ops | Unit tests | All pass |
| SVG Output | Open in browser | Valid, renders |
| Pipeline | `chimborazo build` | SVG generated |
| **Final** | Compare outputs | Visual match to strata |

---

## Historical Narrative

The project name "Chimborazo" references the Ecuadorian volcano that Alexander von Humboldt famously attempted to summit. The rename from "strata" serves multiple purposes:

1. **Avoid name collision** with another GIS tool
2. **Honor the expedition metaphor** - climbing toward a summit of capabilities
3. **Document the AI development journey** - the expedition includes the failures and learnings from trying to get local LLMs to complete the work

The narrative should be updated to reflect:
- The summit attempt (feature parity goal)
- The expedition team (Claude + local models)
- The base camps (working milestones)
- The setbacks (agent failures, iteration limits)

---

## Commands Reference

```bash
# Build chimborazo
cd ~/Code/chimborazo-rebuild
go build -o chimborazo ./cmd/chimborazo

# Run tests
go test ./...

# Build a map
./chimborazo build examples/vermont_12x18.strata.yaml

# Compare outputs
open ~/Code/strata/output/vermont_12x18/svg/combined.svg
open ~/Code/chimborazo-rebuild/output/vermont_12x18/combined.svg
```

---

## Agent Workflow

### Using clood for Context

```bash
# Get project health
clood preflight

# Explore strata codebase
clood tree ~/Code/strata/src/strata
clood grep "def subtract" ~/Code/strata
clood symbols ~/Code/strata/src/strata/humboldt/geometry.py

# Generate context for focused work
clood context ~/Code/strata/llm-context

# Ask local model about patterns
clood ask "How does strata handle polygon clipping?"
```

### Using Aider for Implementation

```bash
cd ~/Code/chimborazo-rebuild

# Start with context
aider --model ollama/qwen2.5-coder:7b \
      --read ~/Code/strata/llm-context/TYPES.md \
      --read ~/Code/strata/llm-context/OPERATIONS.md

# In aider:
> Implement the clip operation in internal/geometry/clip.go
```

---

## Success Criteria (from issue #191)

### Primary: Visual Match

Given `vermont_12x18.strata.yaml`, chimborazo must produce SVG that visually matches strata's output.

### Secondary: Recipe Compatibility

Same YAML format, same layer operations, same output structure.

---

## Related Issues

- #191 - Chimborazo Rebuild via Agent Swarm
- #195 - ATC collapsible output
- #196 - ATC post-experiment summary

---

## Next Steps for Agent

1. Read `~/Code/strata/llm-context/ARCHITECTURE.md`
2. Read `~/Code/strata/llm-context/OPERATIONS.md`
3. Check existing code in `~/Code/chimborazo-rebuild/`
4. Pick the next unimplemented component
5. Write tests first, then implementation
6. Validate against strata output
