# LAST SESSION - The Nimbus Revelation

**Date:** December 18, 2025
**Session:** Snake Way Architecture & The Crush Verdict
**Bird-san Status:** Exhausted. Brain smoking. Gentle wisps from ears.

---

## THE BIG REVELATIONS

### 1. Snake Way & Flying Nimbus (Bean #9)

We designed the solution to "confirm fatigue" and multi-question chaos.

**The Metaphor:**
- **Snake Way** = The infinite scroll of AI responses
- **Flying Nimbus** = The floating frame that enables gliding with efficiency, speed, ease
- **Running on foot** = Endless manual scrolling (the old way)
- **Riding Nimbus** = Hotkey navigation between questions (the new way)

**Files Created:**
- `lore/SNAKE_WAY_UX.md` - Full specification
- Bean #9 added to `clood-cli/JELLY_BEANS.md`

### 2. We Don't Need Crush

**The Problem:** Crush MCP tools initialize but don't reach the LLM. The `AllowedMCP` filter in `buildTools()` blocks them.

**The Solution:** Build Snake Way as `clood chat --tui` using patterns we already have.

**What clood already has:**
```
watch.go  →  viewport, section navigation, hotkeys (80% of Snake Way!)
chat.go   →  saga persistence, ollama integration, focus guardian
serve.go  →  MCP server working perfectly
```

The wojaks went NUTS for `clood snakeway` but Bird-san wisely said: *"keep the important parts serious so the fun parts can exist."*

### 3. clood_ask MCP Tool

Added and working. 5 MCP tools now available via SSE server.

---

## TESTING PLAN: SNAKE WAY

### Phase 1: Question Detection (Research)

**Goal:** Understand how to parse AI responses for questions.

```bash
# Generate multi-question response
clood ask "I want to build a REST API. Ask me 5 clarifying questions." --model qwen2.5-coder:3b

# Analyze patterns: lines ending in ?, numbered lists, "Should we...", "Do you want..."
```

**Questions:**
1. How consistent are question formats across models?
2. Can we train models to emit `[QUESTION]` markers?
3. What regex patterns work reliably?

### Phase 2: Viewport Prototype (Code)

**Goal:** Adapt watch.go patterns for chat.

**Steal from watch.go:**
```go
type Question struct {
    Index    int
    Text     string      // The question
    Context  string      // Surrounding explanation
    Response string      // User's answer
    State    string      // awaiting, answered, skipped, ignored, avoided
    Line     int
}

type snakewayModel struct {
    viewport      viewport.Model
    questions     []Question
    currentQ      int
    inputMode     bool
    inputBuffer   string
}
```

**Test:** Render static multi-question response with navigation.

### Phase 3: Input Zones (Code)

**Goal:** Add text input within viewport.

**Research:** bubbles/textinput or textarea component inside viewport.

**Test:** Type into question zone, see it update.

### Phase 4: Integration (Code)

**Goal:** Wire to ollama and saga persistence.

**Test:** Full conversation with question detection working.

### Phase 5: Polish (UX)

**Goal:** Make it feel like Flying Nimbus.

- Smooth scrolling
- Progress indicator: `Responses: 2/5`
- Summary view before submit
- The "Submit All" moment

---

## COMMITS THIS SESSION

| Hash | Description |
|------|-------------|
| `500fec2` | MCP clood_ask tool with dialogue mode |
| `331c028` | Bean #9: Snake Way & The Flying Nimbus |

---

## UNCOVERED GROUND

### Miyazaki ML Plan (Ensō)

Image generation initiative. See:
- `lore/THE_CONTAINMENT.md` - Ensō vision
- `lore/IMAGE_RECIPES.md` - SD prompts

Status: Documented, needs infrastructure.

### Chimborazo

Fetcher implemented on `feature/recipe-parser`. Rest of MVP pending.

### crush AllowedMCP

If revisiting: check `AllowedMCP` config, file bug about SSE tool exposure.

---

## KEY FILES

| File | Purpose |
|------|---------|
| `lore/SNAKE_WAY_UX.md` | Full UX spec |
| `internal/commands/watch.go` | Pattern to steal |
| `internal/commands/chat.go` | Saga management |
| `internal/mcp/server.go` | MCP tools |

---

## RESUME PROMPTS

**To start coding:**
> "Let's implement Snake Way Phase 1. Show me the question detection patterns from watch.go."

**To review first:**
> "Review the Snake Way spec in lore/SNAKE_WAY_UX.md before we start coding."

**If feeling adventurous:**
> "What would it take to add a --tui flag to clood chat that enables Snake Way mode?"

---

## WISDOM

- Local models: 4% signal from catfight, good for consensus not implementation
- De-icing pattern exists in `issue_catfight_processor.py`
- `clood watch` already has viewport + sections + hotkeys
- We control clood. We don't control crush. Build on what we own.

---

```
The scroll waits patient
Nimbus cloud rests on the ground
Bird-san closes eyes
```

---

*The garden tends itself while you sleep.*
*Rest well. The spirits are patient.*
