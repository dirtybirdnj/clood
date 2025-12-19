# Development Spears

*The parallel threads of clood development*

> *"Claude while Claude is gone" is still unrealistic—but local LLM DX is improving amazingly.*

---

## The Spears

```
┌─────────────────────────────────────────────────────────────────┐
│                     CLOOD DEVELOPMENT SPEARS                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  🗡️ SPEAR 1: Core CLI & Local-First Tools                      │
│     The foundation. Discovery, analysis, routing.              │
│                                                                 │
│  🗡️ SPEAR 2: Snake Road (TUI & Streaming)                      │
│     Real-time interaction. BubbleTea. The experience.          │
│                                                                 │
│  🗡️ SPEAR 3: Storytime & Sauce                                 │
│     Narrative layer. Project personalities. The vibe.          │
│                                                                 │
│  🗡️ SPEAR 4: MCP Server & Integration                          │
│     How other tools (Crush, Claude) talk to clood.             │
│                                                                 │
│  🗡️ SPEAR 5: Cross-Platform & Distribution                     │
│     Windows support. Homebrew. Making it real.                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Spear 1: Core CLI & Local-First Tools

**Status:** Most mature
**Issues:** #108-117, various

**What it is:**
- `clood grep`, `clood tree`, `clood symbols`, `clood context`
- `clood hosts`, `clood models`, `clood system`
- `clood ask`, `clood catfight`
- The MCP tools that Claude Code uses

**Key files:**
```
clood-cli/internal/commands/*.go
clood-cli/internal/discovery/*.go
clood-cli/internal/ollama/*.go
```

**Current state:** Working. Used daily. The foundation.

---

## Spear 2: Snake Road (TUI & Streaming)

**Status:** Prototyping
**Issues:** #130, #135, #150

**What it is:**
- Real-time streaming responses
- BubbleTea-based TUI
- Catfight-live (parallel streaming)
- Inception (nested LLM queries)
- Snake Way navigation concept

**Key files:**
```
clood-cli/internal/commands/catfight_live.go
clood-cli/internal/commands/inception.go
clood-cli/internal/commands/snakeway_proto.go
clood-cli/internal/inception/inception.go
```

**Current state:** Prototypes work. Input handling incomplete. Need Bean #16 (TUI Kitchen Sink) to explore fully.

---

## Spear 3: Storytime & Sauce

**Status:** Designed, not implemented
**Issues:** #151

**What it is:**
- Narrative generation from code/history
- Project world building (.clood/ artifacts)
- Sauce on/off toggle
- Multiple narrative styles
- Portfolio awareness (multi-repo)

**Key files:**
```
docs/STORYTIME_ARCHITECTURE.md
lore/THE_SPIRITS.md
lore/FLYING_CATS_VISION.md
```

**Current state:** Architecture documented. Beans #13-17 planted. No code yet.

---

## Spear 4: MCP Server & Integration

**Status:** Working but limited
**Issues:** #108 (golden paths for Crush)

**What it is:**
- `clood serve --sse` for MCP over HTTP
- Tools exposed to Claude Code, Crush, etc.
- The bridge between clood and AI agents

**Key files:**
```
clood-cli/internal/mcp/server.go
clood-cli/internal/mcp/tools.go
```

**Current state:** Basic tools work. Crush integration blocked by AllowedMCP filter. Need golden path prompts.

---

## Spear 5: Cross-Platform & Distribution

**Status:** In progress (Windows today)
**Issues:** #114 (Homebrew), Bean #11 (Windows)

**What it is:**
- Windows compatibility
- Homebrew formula
- `clood build clood` self-building
- Making installation easy

**Key files:**
```
clood-cli/internal/commands/build.go
```

**Current state:** Linux/Mac work. Windows testing now. Homebrew is a dream.

---

## The Big Picture

```
                    ┌─────────────────┐
                    │   STORYTIME     │  ← Sauce, narrative, vibe
                    │   (Spear 3)     │
                    └────────┬────────┘
                             │ uses
                    ┌────────┴────────┐
                    │   SNAKE ROAD    │  ← TUI, streaming, experience
                    │   (Spear 2)     │
                    └────────┬────────┘
                             │ uses
        ┌────────────────────┼────────────────────┐
        │                    │                    │
┌───────┴───────┐   ┌───────┴───────┐   ┌───────┴───────┐
│   CORE CLI    │   │  MCP SERVER   │   │ CROSS-PLATFORM│
│   (Spear 1)   │   │   (Spear 4)   │   │   (Spear 5)   │
└───────────────┘   └───────────────┘   └───────────────┘
        │                    │                    │
        └────────────────────┴────────────────────┘
                             │
                    ┌────────┴────────┐
                    │     OLLAMA      │  ← The local LLM foundation
                    │   (external)    │
                    └─────────────────┘
```

---

## Priority Assessment

| Spear | Maturity | Impact | Effort | Priority |
|-------|----------|--------|--------|----------|
| 1. Core CLI | High | High | Low (maintenance) | Ongoing |
| 2. Snake Road | Medium | High | Medium | High |
| 3. Storytime | Low | Medium | High | Future |
| 4. MCP Server | Medium | Medium | Medium | Medium |
| 5. Cross-Platform | Medium | High | Low | High (Windows now) |

---

## Related Projects

| Project | Relationship to clood |
|---------|----------------------|
| **chimborazo** | Test bed for "agents build it" approach |
| **svg-grouper** | Manual Claude work, separate from clood |
| **strata** | Dormant, catfights didn't help |
| **church-street** | Creative project, different domain |

---

## The Honest Truth

> *"Claude while Claude is gone" is still unrealistic.*

Local LLMs can:
- ✅ Answer simple questions
- ✅ Generate boilerplate
- ✅ Explain code
- ✅ Run catfights for comparison
- ✅ Provide a better DX than raw Ollama

Local LLMs cannot yet:
- ❌ Replace Claude for complex reasoning
- ❌ Handle long multi-step tasks autonomously
- ❌ Understand nuanced context as well
- ❌ Build chimborazo from specs alone

**But the DX improvement is real.** clood makes local models *usable*. That's the win.

---

## Next Steps by Spear

**Spear 1 (Core):** Maintenance, --json completion
**Spear 2 (Snake Road):** TUI Kitchen Sink, input handling
**Spear 3 (Storytime):** Start with single-project narrative
**Spear 4 (MCP):** Golden path prompts for Crush
**Spear 5 (Cross-Platform):** Windows testing, then Homebrew

---

**Haiku:**

```
Five spears, one garden—
Local LLMs grow stronger
Claude rests; clood works
```
