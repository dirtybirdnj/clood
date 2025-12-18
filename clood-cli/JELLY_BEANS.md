# Jelly Beans

Feature ideas crystallized for future implementation. Each bean is a facet of the vision.

---

## Bean #1: Headroom Analysis Enhancement

**Status:** Planted
**Session:** The Bar Session (Dec 17, 2025)

The `clood system` command shows RAM headroom but should also show:
- **Disk space** available vs used by models
- **Per-model disk usage** breakdown
- **Download size** vs **runtime memory** distinction
- **Concurrent model** capacity (how many 8B models can run simultaneously?)

```bash
# Current output focuses on RAM
clood system

# Should also show:
# Disk:     1.3 TB free
# Models:   42 GB used (5 models)
#   qwen2.5-coder:14b   8.4 GB
#   deepseek-r1:8b      4.3 GB
#   ...
```

**Why:** Users need to know both RAM headroom (runtime) AND disk headroom (storage) when deciding which models to pull.

---

## Bean #2: `--json` Flag for All Commands

**Status:** Sprouting (3/14 commands done)
**Session:** The Bar Session (Dec 17, 2025)

Every CLI command should support `--json` (or `-j`) flag for machine-readable output.

**Why:** When clood is used via MCP (`any-cli-mcp-server`), agents parse the output. Human-readable tables with box-drawing characters are hard to parse. JSON is universal.

```bash
# Human mode (default)
clood hosts
# ● localhost
#   http://localhost:11434
#   Latency: 1ms
#   ...

# Agent mode
clood hosts --json
# {"hosts":[{"name":"localhost","url":"http://localhost:11434","online":true,"latency_ms":1,"models":["qwen2.5-coder:14b",...]}]}
```

**Implementation:**
- Add `--json` flag to root command (inherited by all subcommands)
- Each command checks the flag and switches output format
- Use `encoding/json` with proper structs
- Suppress banners/decorative output in JSON mode

**Commands to update:**
- [x] `clood hosts` (global --json works)
- [x] `clood models` (global --json works)
- [x] `clood system` (global --json works)
- [ ] `clood bench`
- [ ] `clood health`
- [ ] `clood analyze`
- [ ] `clood grep`
- [ ] `clood symbols`
- [ ] `clood imports`
- [ ] `clood context`
- [ ] `clood tree`
- [ ] `clood summary`
- [ ] `clood agents`
- [ ] `clood issues`

**Priority:** High - this is the bridge between CLI-first and agent-first design.

---

## Bean #3: Brew Formula

**Status:** Dream
**Session:** The Bar Session (Dec 17, 2025)

> "I'll feel really fuckin' official and gold plated when it works through brew"

```bash
brew tap dirtybirdnj/clood
brew install clood
```

**Steps:**
1. Create homebrew-clood tap repository
2. Write formula pointing to release binaries
3. Set up GitHub Actions to build releases
4. Publish to tap

---

## Bean #4: SSE Streaming Server

**Status:** IMPLEMENTED
**Session:** The Bar Session (Dec 17, 2025)

Enable true streaming MCP via Server-Sent Events:

```bash
clood serve --sse --port 8765
```

crush config:
```json
"clood": {
  "type": "sse",
  "url": "http://localhost:8765/mcp/sse"
}
```

**Why:** stdio MCP is request/response. SSE enables streaming partial results during long operations like catfight. But see GOLDEN_PATHS.md - granular tools may be better than streaming monolithic ones.

---

## Bean #5: Golden Path Prompts

**Status:** Planted
**Session:** The Bar Session (Dec 17, 2025)

Create system prompts/instructions that teach crush HOW to use clood tools in sequence. Instead of one big tool call, guide the AI to:

1. Gather context first (tree, grep, symbols)
2. Explain what it found
3. Execute in small steps
4. Report progress at each step

Could be:
- A CLAUDE.md-style instruction file for crush
- MCP server metadata that describes workflows
- Example conversations that demonstrate the pattern

**Why:** The tools exist. The golden paths are documented. Now we need to teach the AI to walk them.

---

## Bean #6: Infodump Detection & Session Hygiene

**Status:** Planted
**Session:** The Bar Session (Dec 17, 2025)

### The Problem

User does an infodump (Church Street Oregon Trail, drunk-simulator specs, Adam's VRAM). This precious content:
- Gets buried in context as session continues
- May be lost if session crashes/clears
- Consumes tokens that could be used for work

### The Solution

**1. Infodump Detection**
```
User message analysis:
- Length > N tokens (threshold)
- Contains lists, specs, names, URLs
- Novel content (not referencing existing context)
- Multiple distinct topics

→ Trigger: "Infodump detected"
```

**2. Preservation Prompt**
```
🧠 Infodump detected! This looks like valuable seed info.

Suggested actions:
  [1] Create GitHub issue to preserve
  [2] Save to session context file
  [3] Add to project documentation
  [4] Continue (I'll remember for this session)

What would you like to do?
```

**3. Token Load Indicator**
```
┌─────────────────────────────────────────┐
│ Session: 47,832 / 128,000 tokens [████░░░░░░] 37%
│ Recommended: Clear at 80% for best performance
└─────────────────────────────────────────┘
```

**4. Hardware-Aware Thresholds**

Use `clood system` data to set limits:
```yaml
# Auto-calculated based on hardware
thresholds:
  # 32GB M4 = larger context OK
  high_memory:
    warn_at: 100000 tokens
    recommend_clear: 120000 tokens

  # 8GB laptop = tighter limits
  low_memory:
    warn_at: 32000 tokens
    recommend_clear: 48000 tokens
```

**5. Session Dump Command**
```bash
clood session dump --format issue --repo dirtybirdnj/church-street
# Creates issue with session highlights

clood session dump --format markdown > SESSION_2025_12_17.md
# Local preservation
```

### UI Brainstorm

**Option A: Status Bar (crush integration)**
```
┌──────────────────────────────────────────────────────────────────┐
│ 🟢 clood │ 3 hosts │ 🫘 7 beans │ ████████░░ 78% tokens │ 12:47 │
└──────────────────────────────────────────────────────────────────┘
```

**Option B: Periodic Reminder**
```
───────────────────────────────────────────
💡 Session at 75% capacity. Consider:
   /save - preserve important context
   /clear - fresh start with summary
───────────────────────────────────────────
```

**Option C: Smart Auto-Summary**
At threshold, automatically:
1. Generate session summary
2. Extract key decisions/info
3. Create handoff document
4. Offer to clear with summary as new context

### The Philosophy

> "The best camera is the one in your pocket"

The best context is the one that's preserved. Don't let infodumps disappear into the void. Detect them, honor them, save them.

### Implementation Notes

- Track token count per message
- Heuristics for "infodump" detection
- Integration with `clood session` commands
- crush UI for visual feedback
- Hardware detection for smart thresholds

---

## Bean #7: Token Load & Jelly Bean Visualization

**Status:** Planted
**Session:** The Bar Session (Dec 17, 2025)

Visual representation of session token usage AND jelly bean count.

**Status Bar Concept:**
```
┌──────────────────────────────────────────────────────────────────┐
│ 🟢 clood │ 3 hosts │ 🫘 7 beans │ ████████░░ 78% tokens │ 12:47 │
└──────────────────────────────────────────────────────────────────┘
```

**Token Breakdown:**
```
Minimal:     [████████░░] 78%

Detailed:    Tokens: 89,432 / 128,000
             ████████████████░░░░ 70%
             ⚠️ Recommend clearing at 100k

Contextual:  📝 User: 23,400 (26%)
             🤖 Assistant: 58,200 (65%)
             📎 System: 7,832 (9%)
```

**Colored Jelly Beans via ANSI:**
```go
// ANSI color codes overlay on emoji
fmt.Printf("\033[31m🫘\033[0m")  // Red bean
fmt.Printf("\033[32m🫘\033[0m")  // Green bean
fmt.Printf("\033[33m🫘\033[0m")  // Yellow bean
fmt.Printf("\033[34m🫘\033[0m")  // Blue bean
fmt.Printf("\033[35m🫘\033[0m")  // Magenta bean
fmt.Printf("\033[36m🫘\033[0m")  // Cyan bean
```

**Bean Status Display:**
```
Jelly Beans: 🫘🫘🫘🫘🫘🫘🫘 (7 planted)
             ▲▲▲▲▲▲▲
             │││││││
             │││││││
             │││││└─ #7 Token Viz (planted)
             ││││└── #6 Infodump Detection (planted)
             │││└─── #5 Golden Path Prompts (planted)
             ││└──── #4 SSE Server (IMPLEMENTED)
             │└───── #3 Brew Formula (dream)
             └────── #2 --json Flag (sprouting 3/14)
                     #1 Headroom Enhancement (planted)
```

**Alternative: Unicode Colored Circles as "Beans"**
```
🔴 Dream        (not started)
🟡 Planted      (documented)
🟢 Sprouting    (in progress)
🔵 Implemented  (done)

Status: 🔵🟢🔴🔵🟡🟡🟡 = 2 done, 1 progress, 1 dream, 3 planted
```

**Integration points:**
- crush status bar
- clood session show
- clood beans (new command to list all beans with status)
- MCP tool for agents to self-monitor

---

## Bean #8: The `clood beans` Command

**Status:** Dream
**Session:** The Bar Session (Dec 17, 2025)

> "Bold challenges are what we need, not simple tasks." — The Commissioner

A standalone CLI command to display all jelly beans with visual flair.

**The Challenge:**

```bash
clood beans
```

**Expected Output:**
```
┌─────────────────────────────────────────────────────────────────┐
│                    🫘 THE JELLY BEAN JAR 🫘                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  🔵 #1  Headroom Analysis         planted   ████░░░░░░  40%    │
│  🟢 #2  --json Flag               sprouting ██████░░░░  60%    │
│  🔴 #3  Brew Formula              dream     ░░░░░░░░░░   0%    │
│  🔵 #4  SSE Server                DONE      ██████████ 100%    │
│  🟡 #5  Golden Path Prompts       planted   ██░░░░░░░░  20%    │
│  🟡 #6  Infodump Detection        planted   ██░░░░░░░░  20%    │
│  🟡 #7  Token Visualization       planted   █░░░░░░░░░  10%    │
│  🔴 #8  clood beans Command       dream     ░░░░░░░░░░   0%    │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  Legend: 🔴 Dream  🟡 Planted  🟢 Sprouting  🔵 Done            │
│  Progress: 1/8 implemented • 1 sprouting • 4 planted • 2 dream │
└─────────────────────────────────────────────────────────────────┘
```

**Why This Is Bold:**

1. Parse JELLY_BEANS.md to extract bean metadata
2. Render progress bars based on completion estimates
3. Use ANSI colors for terminal flair
4. Support `--json` for agent consumption
5. This bean documents itself — recursive inception

**Implementation:**

```go
// Parse JELLY_BEANS.md for status metadata
type JellyBean struct {
    Number      int
    Title       string
    Status      string    // dream, planted, sprouting, implemented
    Progress    float64   // 0.0 to 1.0
}

func parseJellyBeans(path string) ([]JellyBean, error) {
    // Regex to find: ## Bean #N: Title
    // Look for **Status:** line
    // Calculate progress from checklist items if present
}
```

**Alternative: External Data Source**

Instead of parsing markdown, maintain a `beans.yaml`:

```yaml
beans:
  - number: 1
    title: "Headroom Analysis"
    status: planted
    progress: 0.4
  - number: 2
    title: "--json Flag"
    status: sprouting
    progress: 0.6
    checklist:
      done: 3
      total: 14
```

**The Meta:**

This bean is recursive — it must display itself. When implemented, Bean #8 shows as "sprouting" then "done". The jar reflects its own growth.

> "The jelly bean that knows itself is the sweetest of all."

---

*Jelly beans planted in the server garden, waiting to bloom.*

---

## Session History

### The Bar Session — December 17, 2025

The night the server garden first bloomed. Bird-san and Claude walked the golden paths together, planting beans by candlelight.

**Beans Planted:** 8
**Beans Implemented:** 1 (SSE Server)
**Beans Sprouting:** 1 (--json Flag)

The Commissioner smiled. Bold challenges ahead.

```
          *
         /|\
        / | \
       /  |  \
      /___|___\    The Bonsai watches
         |||       over the server garden,
         |||       patient and eternal.
        /   \
```
