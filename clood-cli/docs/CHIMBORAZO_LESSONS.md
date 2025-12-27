# Chimborazo Development: Lessons Learned

*From the ATC experiment sessions on 2025-12-26*

---

## What Worked

### 1. Structured Context Documents

The `llm-context/` folder in strata is effective:
- ARCHITECTURE.md with ASCII diagrams - models understand the flow
- OPERATIONS.md with Go implementation hints - reduces hallucination
- TYPES.md with exact struct definitions - copy-paste ready

**Takeaway:** Front-load context. Local models do better with complete information upfront.

### 2. Step-by-Step Validation

Breaking work into phases with validation gates:
- Recipe parser → `go test ./internal/config`
- Fetcher → File appears in cache
- SVG → Opens in browser

**Takeaway:** Don't batch. Validate each component before moving on.

### 3. Model Tiering

Different models for different tasks:
- Quick analysis: qwen2.5-coder:3b (fast, low context)
- Implementation: qwen2.5-coder:7b (good balance)
- Complex geometry: deepseek-r1:8b (reasoning traces)
- Design/review: Opus 4.5 (when local fails)

**Takeaway:** Match model to task complexity. Escalate on failure.

---

## What Didn't Work

### 1. ATC Dashboard Visibility

The ATC experiment mode showed activity but not enough detail:
- "Running step 1" isn't useful without seeing the actual prompt/output
- Event feeds need larger content (120 chars, not 50)
- Session timers and gap detection help but aren't core

**Status:** Improvements made but concept needs more thought.

### 2. Auto-Expanding Output Panels

Tried to show model output auto-expanded, but:
- CSS specificity issues or browser cache caused it to stay collapsed
- User had to click to expand

**Takeaway:** Test UI changes in fresh browser, check CSS cascade.

### 3. Endpoint Confusion

Initially POSTed to `/event` instead of `/experiment`:
- Different endpoints have different message formats
- Wrapper messages (`{type: "experiment", data: {...}}`) need unwrapping

**Takeaway:** Document API contracts clearly. Test endpoints before building UI.

---

## Agent Behavior Patterns

### What Models Do Well

1. **Code generation from specs** - Given types + interface + examples, they produce working code
2. **Test writing** - Given implementation, they write reasonable tests
3. **Explaining existing code** - Good at summarizing what code does

### What Models Struggle With

1. **Architectural decisions** - Need human/Claude guidance
2. **Cross-file refactoring** - Lose context, introduce inconsistencies
3. **Debugging complex issues** - Often circle back to same wrong answer
4. **Knowing when to stop** - Will keep iterating without progress

### Mitigations

| Problem | Solution |
|---------|----------|
| Invents dependencies | List exact imports in spec |
| Over-engineers | Add "DO NOT add" constraints |
| Wrong file paths | Specify exact paths |
| Ignores patterns | Always include PATTERNS.md in context |
| Infinite iteration | Max 3 iterations, then escalate |

---

## Effective Prompt Patterns

### For Implementation

```markdown
# Task: [SHORT_DESCRIPTION]

## Context
You are implementing [component] for Chimborazo.

### Types (copy these exactly)
```go
type Feature struct { ... }
```

### Interface to Implement
```go
type Clipper interface { ... }
```

## Requirements
1. Create file at `internal/geometry/clip.go`
2. Implement [specific function]
3. Handle these errors: [list]

## Constraints
- DO NOT add dependencies
- DO NOT modify other files
- Use orb.Polygon, not custom types

## Output
Only the Go code. No explanations.
```

### For Debugging

```markdown
# Bug: [DESCRIPTION]

## Current Behavior
[What happens]

## Expected Behavior
[What should happen]

## Relevant Code
```go
// paste the broken function
```

## Constraints
- Only fix this function
- Do not refactor
- Add a test case for this bug
```

---

## Iteration Limits

From the experiment:

| Condition | Action |
|-----------|--------|
| First attempt | Try with current model |
| Second attempt | Provide more context |
| Third attempt | Escalate to larger model |
| After 3 failures at Opus | Human intervention |

**Never let agents iterate indefinitely.** They get stuck in loops.

---

## Workflow: Claude + Local Agents

### Phase 1: Design (Claude)
- Analyze requirements
- Write specs with exact interfaces
- Define test cases
- Create prompt for local agent

### Phase 2: Implement (Local Agent)
- Aider with qwen2.5-coder:7b
- Context from llm-context/ folder
- One file at a time
- Validate before commit

### Phase 3: Review (Claude)
- `git show HEAD` - review the diff
- Check for bugs, deviations from spec
- Approve or request fixes

### Phase 4: Integrate (Human + Claude)
- Wire components together
- Run end-to-end tests
- Compare to reference outputs

---

## Technical Notes

### Go Geometry with orb

```go
import (
    "github.com/paulmach/orb"
    "github.com/paulmach/orb/clip"
    "github.com/paulmach/orb/simplify"
)

// Clip polygon to bounds
bound := orb.Bound{Min: orb.Point{-73, 43}, Max: orb.Point{-71, 45}}
clipped := clip.Geometry(bound, polygon)

// Simplify with tolerance
simplified := simplify.DouglasPeucker(0.001).Simplify(polygon)
```

### SVG Rendering

Key insight from strata: latitude correction for geographic accuracy.
- Longitude degrees are narrower at higher latitudes
- Scale factor: `cos(latitude * pi / 180)`

### Caching

Use URI as cache key:
```
~/.cache/chimborazo/{scheme}/{path}
```

---

## What the ATC Experiment Revealed

The Air Traffic Control concept for monitoring agent work has potential but needs:

1. **More visibility** - Show actual prompts/outputs, not just status
2. **Timing metrics** - Session duration, step times, latency gaps
3. **Post-analysis** - Summary of what was accomplished
4. **Iteration tracking** - Which attempts succeeded/failed

The core idea (monitor distributed agent work from a dashboard) is sound. The implementation needs iteration.

---

## Recommended Next Session

1. **Focus on chimborazo core** - Recipe parser, fetcher, cache
2. **Use strata llm-context/** - Already optimized for this
3. **Small commits** - One component at a time
4. **Validate early** - Tests before integration
5. **Compare outputs** - Vermont map should match

Don't get distracted by tooling (ATC, etc.) - the goal is working maps.
