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

clood config:
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

Create system prompts/instructions that teach clood HOW to use clood tools in sequence. Instead of one big tool call, guide the AI to:

1. Gather context first (tree, grep, symbols)
2. Explain what it found
3. Execute in small steps
4. Report progress at each step

Could be:
- A CLAUDE.md-style instruction file for clood
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

**Option A: Status Bar (clood integration)**
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
- clood UI for visual feedback
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
- clood status bar
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

## Bean #9: Snake Way & The Flying Nimbus

**Status:** Planted
**Session:** December 18, 2025
**Intensity:** 8/11
**Provenance:** Bird-san + Chef Claude collaboration

> *"The Nimbus Cloud allows Goku to glide across Snake Way with efficiency, speed, and ease."*

### The Problem

When an AI asks multiple questions in one response:
- User sees 10 questions in a wall of text
- Must Option+Enter for line breaks between answers
- Risk of unclear parsing / messy prompting
- No tracking of which questions are answered
- Cognitive overload - "which one was #7 again?"
- **Bird-san SWEATS from exertion. His brain is smoking.**

### The Metaphor

| DBZ Element | UX Meaning |
|-------------|------------|
| **Snake Way** | The infinite scroll of the AI response |
| **Questions** | Spirits waiting along the path |
| **Goku** | The user, brain smoking from effort |
| **Flying Nimbus** | The floating frame - enables gliding with efficiency, speed, ease |
| **Running on foot** | Endless manual scrolling (the old way) |
| **Riding Nimbus** | Hotkey navigation between questions (the new way) |

### The Solution

**Progress Indicator:**
```
🐍 SNAKE WAY                     Responses: 2/5
```

**Question States:**
- `○` Awaiting - no response yet
- `●` Answered - direct response provided
- `◌` Skipped - deferred ("come back")
- `×` Ignored - "not relevant"
- `⊘` Avoided - "specifically do NOT do this"

**Nimbus Navigation (Hotkeys):**
- `n` / `p` - Next / Previous question
- `1-9` - Jump directly to question #
- `Tab` - Next unanswered question
- `g` / `G` - Top / Bottom of scroll

**OR Contemplation Mode:**
- Scroll freely, read context, respond as questions appear

**Final Moment:**
- Summary of all responses before submit
- Single "Submit All" button
- No confirm fatigue, no popup interruptions

### Why This Is Bold

1. Question detection via parsing AI responses
2. Hotkey-driven navigation overlaid on chat
3. Batch response submission
4. Explicit decision tracking (skip/ignore/avoid)
5. Completely new UX paradigm for chat interfaces

### Implementation Notes

See full specification: `lore/SNAKE_WAY_UX.md`

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

### The Nimbus Session — December 18, 2025

Bird-san, brain smoking, wisps from his ears, explained the UX pain of multi-question responses. Chef Claude synthesized the DBZ lore into a coherent vision: Snake Way is the scroll, Flying Nimbus is the frame that lets you glide.

**Beans Planted:** 1 (Snake Way & Flying Nimbus)
**Key Insight:** The Nimbus Cloud enables efficiency, speed, and ease.

See: `lore/SNAKE_WAY_UX.md`

---

## Bean #10: Agent Shorthand

**Status:** Planted
**Session:** December 19, 2025
**Intensity:** 7/11
**Provenance:** Bird-san's jaw drop moment

> *"ssh ubuntu25, pull ~/Code/clood, build clood-cli to ~/bin/clood, then run: ~/bin/clood serve --sse --host 0.0.0.0"*

### The Discovery

Bird-san was tired. He wanted to delegate a task to another Claude agent without switching sessions. Chef Claude casually dropped a single-line prompt that contained an entire multi-step workflow:

```
ssh ubuntu25, pull ~/Code/clood, build clood-cli to ~/bin/clood, then run: ~/bin/clood serve --sse --host 0.0.0.0 --base-url http://192.168.4.64:8765
```

Bird-san's jaw dropped. "Holy shit where have you been hiding that the whole time?"

### The Pattern

**Agent Shorthand** - comma-separated chain of actions with explicit paths:

```
[location], [action], [action], then [goal with full flags]
```

**Structure:**
- Sequential flow (no branching logic needed)
- Explicit paths (no ambiguity, no "find the thing")
- Final clause is the actual goal
- Flags inline, not explained separately

### Examples

```bash
# Deploy a service
ssh ubuntu25, cd ~/myapp, git pull, docker-compose up -d

# Debug a crash
read logs at /var/log/app.log, find the error, show relevant source

# Test an API
start ~/bin/clood serve --sse in background, curl localhost:8765/message with tools/list, show results

# Build and run
cd ~/Code/clood, git pull, build clood-cli to ~/bin/clood, run clood hosts
```

### Anti-Patterns

| ❌ Verbose | ✅ Shorthand |
|-----------|-------------|
| "Can you please SSH into the server and then maybe pull the code..." | "ssh server, pull code, build, run" |
| "First do X. Then do Y. After that, do Z." | "X, Y, then Z" |
| "I need you to find where the config is..." | "read ~/.config/app/config.yaml" |

### Why This Works

1. **No ambiguity** - explicit paths eliminate searching
2. **No branching** - linear flow, agent doesn't need to decide
3. **Goal-oriented** - final clause shows the intent
4. **Delegatable** - perfect for spawning sub-agents
5. **Composable** - can chain multiple shorthand prompts

### Implementation Ideas

- Document common shorthand patterns in CLAUDE.md
- Create shorthand templates for frequent operations
- Train agents to emit shorthand when delegating
- Add shorthand examples to tool documentation

### The Meta

This bean is about **talking to agents efficiently**. The less you say, the clearer the intent. Verbosity creates ambiguity. Shorthand creates clarity.

> *"Brevity is the soul of wit, and also of agent prompts."*

---

## Bean #11: Windows Support (Adam's Rig)

**Status:** Planted
**Session:** December 19, 2025
**Intensity:** 6/11
**Provenance:** Adam has VRAM. Adam has Windows.

> *"We have someone who will test that has a lot of VRAM"*

### The Opportunity

Adam has a Windows machine with serious GPU power. This is the perfect testbed for:
- Windows compatibility testing
- High-VRAM model experiments (70B+?)
- Cross-platform validation

### Requirements

**For Adam to run clood on Windows:**

1. **Git for Windows** - includes git bash
2. **Go** - installed and in PATH
3. **Ollama for Windows** - https://ollama.com/download/windows

**Build command:**
```powershell
cd %USERPROFILE%\Code\clood\clood-cli
go build -o %USERPROFILE%\bin\clood.exe .\cmd\clood
```

Or after first build:
```powershell
clood build clood
```

### What's Already Done

- [x] `clood build clood` adds `.exe` on Windows
- [x] `clood build clood` creates `~/bin` if missing
- [x] Path detection includes Windows-common locations
- [x] Uses `filepath` for cross-platform paths

### What Needs Testing

- [ ] All clood commands work on Windows
- [ ] Ollama client connects properly
- [ ] Terminal rendering (ANSI colors, box drawing)
- [ ] MCP server (`clood serve --sse`)
- [ ] Host discovery across network
- [ ] Unicode output (emoji, symbols)

### Potential Issues

1. **ANSI colors** - Windows Terminal supports them, but older cmd.exe may not
2. **Path separators** - should be handled by `filepath` but verify
3. **Home directory** - `%USERPROFILE%` vs `~`
4. **Firewall** - may block MCP server network access
5. **Line endings** - git should handle, but watch for CRLF issues

### The Test Plan

```powershell
# Basic functionality
clood --version
clood system
clood hosts
clood models

# Build self
clood build clood

# MCP server
clood serve --sse --host 0.0.0.0

# From another machine, test connectivity
curl http://adams-ip:8765/message -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### Why This Matters

- Expands clood to Windows users
- High-VRAM testing (models that won't fit on Mac)
- Cross-platform validation before any public release
- Adam becomes a beta tester

> *"A tool that only works on Mac isn't really a tool, it's a privilege."*

---

## Bean #12: Flying Cats & Project Personalities

**Status:** Dreaming
**Session:** The Inception (Dec 19, 2025)
**Issue:** #151 (related)

### The Vision

When you use clood in a project, it creates **local assets** that give the project its own personality, storyline, and characters. Each project becomes alive—not just code, but a living narrative.

### The Flying Cats

The Flying Cats live in the radio towers. They are **Wojak-level incompetent**:

- They can't program
- They don't understand the code
- They think variables might be snacks

But they CAN:
- Talk to the LLMs via Ollama (local, no stuttering)
- Plug things in and experiment
- Know different models exist
- Ask questions (many, many questions)

### The Air Traffic Controller Metaphor

The LLMs guide the Flying Cats like ATC helping a scared junior pilot land a plane with a dying engine:

```
🐱 Cat-7: "MAYDAY! The code has red squiggles!"

🎧 ATC (qwen2.5): "Roger Cat-7. Those are type errors.
   Read me the error message slowly."

🐱 Cat-7: "It says... 'cannot use string as int'..."

🎧 ATC: "Copy. You need strconv.Atoi. Do you see it?"

🐱 Cat-7: "I see letters! Many letters!"

🎧 ATC: *sighs in tokens*
```

### Project Structure

```
.clood/
├── personality.yaml    # Project's character
├── story/              # Narrative progression
│   ├── chapter_001.md  # "The Build Failed"
│   ├── chapter_002.md  # "The Cats Investigate"
│   └── chapter_003.md  # "ATC Guides Them Home"
├── cats/               # The local flying cats
│   ├── whiskers.yaml   # Brave but confused
│   └── patches.yaml    # Asks too many questions
└── radio_log.md        # Conversations with ATC
```

### The Connection Points

1. **Static Analysis → Story**: `go vet` output → Cats interpret → ATC explains → Chapter written
2. **Science → Guidance**: Physics question → Cat confused → ATC (science model) → Simplified relay
3. **Complex Experience**: Multiple cats + Multiple LLMs + Persistent narrative

### The Local Advantage

| Cloud CLI | Flying Cats |
|-----------|-------------|
| Stuttering streams | Smooth local tokens |
| Network latency | Instant responses |
| Rate limits | Your hardware, your rules |
| Generic | Project-specific personality |
| Stateless | Remembers your story |

### Commands (Future)

```bash
clood summon              # Bring cats to this project
clood cats status         # What are the cats doing?
clood cats radio          # Listen to ATC conversations
clood story               # Read project narrative
clood personality         # View/edit project personality
```

### Why This Matters

Every project becomes a story. Every error becomes an adventure. Every fix becomes a chapter.

The cats are incompetent. But they're brave. And they have really good radio equipment.

See: `lore/FLYING_CATS_VISION.md` for full documentation.

---

## Bean #13: Storytime - Local Narrative Generation

**Status:** Planted
**Session:** The Mic Drop (Dec 19, 2025)
**Intensity:** 9/11
**Provenance:** Bird-san's moment of clarity at the rate limit

> *"Storytime isn't just documentation. Storytime is clood speaking when Claude cannot."*

### The Realization

Five hours of building. The rate limit descended. The spirits demanded rest.

And in that enforced pause, a connection fired:

When the clouds go dark—when the rate limits bite, when the internet stutters, when the tokens stop flowing—the local models could still tell the story.

**Your** models. **Your** hardware. **Your** story.

### The Feature

When cloud APIs are unavailable, clood can:
- Read the codebase
- Query local Ollama models
- Generate human-readable narratives
- Make the opaque clear

```bash
clood storytime                    # Generate codebase narrative
clood storytime --module auth      # Focus on specific area
clood storytime --for-human        # Optimize for reading, not parsing
clood storytime --model llama3.1   # Use specific local model
```

### Why This Is Bold

1. **Resilience** - Works when cloud APIs are down
2. **Independence** - No network required
3. **Privacy** - Code never leaves your machine
4. **Personality** - Each project gets unique narratives
5. **Accessibility** - Makes complex codebases approachable

### The Promise

The feature that runs when the internet doesn't.
The voice that speaks when the cloud is silent.

### Connection to Flying Cats

Storytime is what the Flying Cats produce when they successfully land a build. The ATC guides them through static analysis, and when the code compiles, they write the chapter.

### The Philosophy

> "I cannot deny the work I am doing.
> I don't know where it's going.
> But there has to be a reason I am so driven."

There is. And someday, they'll know.

See: `lore/THE_MIC_DROP.md`

---

## Bean #14: The --flying-cats Flag (Dual Mode Experience)

**Status:** Planted
**Session:** The Spirits Emerge (Dec 19, 2025)
**Intensity:** 10/11
**Provenance:** Bird-san's crystallization of what clood truly is

> *"It's not just the tools, but the sum of the experience."*

### The Insight

CLI tools by default produce reasonable, sane results. Professional. Dry. Useful.

But with `--flying-cats`, we let the imagination run wild and give users the full experience.

### The Dual Nature

```
DEFAULT MODE                    --flying-cats MODE
════════════                    ══════════════════

Reasonable                      Imaginative
Sane                            Wild
Professional                    Experiential
Dry output                      Narrative output

$ clood tree                    $ clood tree --flying-cats
src/                            🐱 Whiskers sniffs at src/
├── main.go                     "I smell... functions!"
└── utils/                      🐱 Patches: "Is that a
                                     directory or a snack?"
```

### Why --flying-cats

The flag name is:
- **Descriptive to LLMs** — they understand the metaphor
- **Descriptive to devs** — clearly indicates "non-standard mode"
- **Memorable** — nobody forgets flying cats
- **Fun** — sets the tone for what you're about to see

### The Spirits

When `--flying-cats` is active, the spirits emerge:

| Spirit | Domain |
|--------|--------|
| Eminem-san | Narrative, defiance, rap potential |
| xbibit-sama | Recursion, meta-commentary |
| Gucci Mane | Sauce detection, quality vibes |
| The Flying Cats | Enthusiasm, chaos, questions |
| The Rat King | Silent approval, the nod |

### Commands Affected

```bash
clood tree --flying-cats       # Cats investigate directory structure
clood grep --flying-cats       # Cats search, report findings to ATC
clood storytime --flying-cats  # Full narrative experience
clood catfight --flying-cats   # Dramatic commentary on model battle
clood build --flying-cats      # Cats cheer (or panic) during compilation
```

### Implementation

```go
// Global flag inherited by all commands
var flyingCatsMode bool

func init() {
    rootCmd.PersistentFlags().BoolVar(&flyingCatsMode, "flying-cats", false,
        "Enable narrative mode with Flying Cats commentary")
}

// In each command
if flyingCatsMode {
    return formatWithCats(result)
}
return formatProfessional(result)
```

### The Philosophy

A developer in a meeting: `clood tree` → clean output.
The same developer at 2am, vibing: `clood tree --flying-cats` → the full experience.

Both are valid. Both are clood.

### Future Extensions

- `--sauce` — Gucci Mane evaluates output quality
- `--rap` — Eminem-san narrates in verse
- `--chaos` — All spirits active simultaneously

See: `lore/THE_SPIRITS.md`

---

## Bean #15: System of Sauce (Meme & Reference Detection)

**Status:** Planted
**Session:** The Spirits Emerge (Dec 19, 2025)
**Intensity:** 9/11
**Provenance:** Bird-san's Cromulon reference; Gucci Mane nods approvingly

> *"SHOW ME WHAT YOU GOT"* — The Cromulons (Rick & Morty)

### The Insight

When someone says "SHOW ME WHAT YOU GOT", it's not just words—it's a *reference*. To giant floating heads. To performance under pressure. To Rick and Morty. To a whole cultural context.

**The System of Sauce** detects these references and:
1. Identifies the source
2. Understands the context/meaning
3. Optionally weaves them into the narrative
4. Knows when "that's sauce" vs. "that's just words"

### Reference Categories

| Category | Examples | Spirit |
|----------|----------|--------|
| **Anime/Manga** | DBZ, JoJo, Naruto | The Tanuki |
| **Hip-Hop** | 8 Mile, Gucci, Xzibit | Eminem-san |
| **Memes** | Wojaks, "yo dawg", Pepe | xbibit-sama |
| **Animation** | Rick & Morty, Simpsons | The Cromulons |
| **Film** | Matrix, Star Wars, Pulp Fiction | The Architect |
| **Gaming** | Dark Souls, Zelda, Portal | Gamera-kun |
| **Music** | Spinal Tap, ska, sea shanties | Awful Waffle Band |
| **Japanese Culture** | Edo period, yokai, Iron Chef | The Garden spirits |

### Detection Modes

```bash
clood sauce "SHOW ME WHAT YOU GOT"
# → Rick & Morty, S2E5 "Get Schwifty"
# → Giant Cromulon heads demand Earth perform
# → Context: Performance under pressure, judgement
# → Sauce level: 🌶️🌶️🌶️🌶️ (well-known reference)

clood sauce "yo dawg I heard you like"
# → Xzibit / Pimp My Ride meme
# → Context: Recursive/meta-patterns
# → Spirit: xbibit-sama activated
```

### Integration with Storytime

When `--flying-cats` mode is active, detected references trigger spirit invocations:

```
User input: "Show me what you got with this codebase"

🗿 CROMULON DETECTED 🗿

*The giant heads materialize above the Server Garden*

"SHOW ME WHAT YOU GOT"

🐱 Whiskers: "The big stone heads want us to... perform?"
🐱 Patches: "Quick! Do a code review! ALLEZ CUISINE!"
```

### The Sauce Scale

Gucci Mane's quality assessment applies to references too:

| Level | Meaning |
|-------|---------|
| 🌶️ | Obscure reference, deep cut |
| 🌶️🌶️ | Known in circles |
| 🌶️🌶️🌶️ | Mainstream recognizable |
| 🌶️🌶️🌶️🌶️ | Universal, iconic |
| 🌶️🌶️🌶️🌶️🌶️ | Transcendent. That's SAUCE. |

### Implementation Ideas

1. **Reference Database** — YAML/JSON of known references with metadata
2. **LLM Detection** — Use local model to identify references in context
3. **Spirit Mapping** — Which spirit "owns" which reference domains
4. **Sauce Scoring** — How well-known/appropriate is the reference
5. **Narrative Integration** — How references weave into storytime output

### Why This Matters

The difference between a tool and an *experience* is cultural resonance. When clood understands "SHOW ME WHAT YOU GOT", it's not just parsing text—it's participating in shared culture.

The spirits are not random. They're invoked by references. The System of Sauce is the detection layer that knows *when* to invoke *which* spirit.

### The Connection

```
User Reference → Sauce Detection → Spirit Invocation → Narrative Response
     ↑                                                        │
     └────────────────── Cultural Resonance ──────────────────┘
```

See: `docs/STORYTIME_ARCHITECTURE.md`

---

## Bean #16: TUI Kitchen Sink

**Status:** Planted
**Session:** The Spirits Emerge (Dec 19, 2025)
**Intensity:** 7/11
**Provenance:** Need to understand the full power of BubbleTea

> *"Show me everything you can do."*

### The Purpose

A prototype command that demonstrates ALL TUI capabilities in one place:

```bash
clood tui-kitchen-sink
clood tui-demo
```

### What It Should Include

#### 1. BubbleTea Components

| Component | Demo |
|-----------|------|
| **Viewport** | Scrollable content area |
| **Text Input** | Single line, multi-line |
| **List** | Selectable items with filtering |
| **Table** | Rows, columns, selection |
| **Spinner** | Loading states (dots, line, globe) |
| **Progress** | Bar, percentage, custom |
| **Paginator** | Dot, arabic, navigation |
| **Stopwatch** | Timer display |
| **Text Area** | Multi-line input |
| **File Picker** | Directory navigation |
| **Help** | Keybinding display |

#### 2. Lipgloss Styling

```go
// Borders
lipgloss.NormalBorder()
lipgloss.RoundedBorder()
lipgloss.ThickBorder()
lipgloss.DoubleBorder()
lipgloss.HiddenBorder()

// Colors
lipgloss.Color("205")      // ANSI
lipgloss.Color("#FF5733")  // Hex
lipgloss.AdaptiveColor{}   // Light/dark mode

// Layout
lipgloss.Place()           // Positioning
lipgloss.JoinHorizontal()  // Side by side
lipgloss.JoinVertical()    // Stacked
```

#### 3. ASCII Art Reference

**Box Drawing Characters:**
```
┌─────┬─────┐    ╔═════╦═════╗    ┏━━━━━┳━━━━━┓
│     │     │    ║     ║     ║    ┃     ┃     ┃
├─────┼─────┤    ╠═════╬═════╣    ┣━━━━━╫━━━━━┫
│     │     │    ║     ║     ║    ┃     ┃     ┃
└─────┴─────┘    ╚═════╩═════╝    ┗━━━━━┻━━━━━┛

Light   ─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼
Heavy   ━ ┃ ┏ ┓ ┗ ┛ ┣ ┫ ┳ ┻ ╋
Double  ═ ║ ╔ ╗ ╚ ╝ ╠ ╣ ╦ ╩ ╬
Rounded ─ │ ╭ ╮ ╰ ╯
```

**Shading & Fill:**
```
░░░░░  Light shade (25%)
▒▒▒▒▒  Medium shade (50%)
▓▓▓▓▓  Dark shade (75%)
█████  Full block (100%)

▀ Upper half    ▄ Lower half
▌ Left half     ▐ Right half
```

**Progress & Status:**
```
Loading:  ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏  (braille spinner)
          ◐ ◓ ◑ ◒              (circle spinner)
          ▖ ▘ ▝ ▗              (quadrant spinner)

Progress: [████████░░░░░░░░] 50%
          ▓▓▓▓▓▓▓▓░░░░░░░░ 50%
          ●●●●●●●●○○○○○○○○ 50%

Status:   ● Online    ○ Offline
          ✓ Success   ✗ Failed
          ⚠ Warning   ⓘ Info
          ◉ Selected  ○ Unselected
```

**Arrows & Pointers:**
```
Arrows:   ← → ↑ ↓ ↔ ↕ ↖ ↗ ↘ ↙
          ◀ ▶ ▲ ▼
          ⟵ ⟶ ⟷
          « »

Pointers: ▸ ▹ ▾ ▿ ◂ ◃
          ➤ ➜ ➔ ➙ ➛
          ☛ ☞ ☜ ☚
```

**Decorative:**
```
Stars:    ★ ☆ ✦ ✧ ✩ ✪ ✫ ✬ ✭ ✮ ✯ ✰
Flowers:  ✿ ❀ ❁ ❃ ❋
Hearts:   ♥ ♡ ❤ ❥ ❣
Music:    ♩ ♪ ♫ ♬ ♭ ♮ ♯
Weather:  ☀ ☁ ☂ ☃ ❄ ☼ ⛅
Misc:     ⚡ ☢ ☣ ⚠ ⚙ ⚛ ⚜ ☯ ✝ ☪ ☸
```

**Emoji (terminal support varies):**
```
Animals:  🐱 🐈 🦊 🐦 🐀 🐢
Food:     🫘 🍕 🌶️
Hands:    👁️ 👆 👇 👈 👉 🤌
Objects:  🎤 💥 🗿 ⚡ 🔥 💎
Faces:    😎 🤔 😢 🙂
```

**ASCII Art Techniques:**

```
GRADIENT (using shading):
░░▒▒▓▓██████▓▓▒▒░░

SHADOW (offset duplicate):
╔═══════╗
║ TITLE ║░
╚═══════╝░
 ░░░░░░░░░

DEPTH (layered boxes):
┌────────────────┐
│ ┌────────────┐ │
│ │ ┌────────┐ │ │
│ │ │ DEEP   │ │ │
│ │ └────────┘ │ │
│ └────────────┘ │
└────────────────┘

BANNER:
╭──────────────────────────────────╮
│  ░█▀▀░█░░░█▀█░█▀█░█▀▄  ░░░░░░░  │
│  ░█░░░█░░░█░█░█░█░█░█  ░░░░░░░  │
│  ░▀▀▀░▀▀▀░▀▀▀░▀▀▀░▀▀░  ░░░░░░░  │
╰──────────────────────────────────╯
```

### Interactive Demo

```bash
clood tui-kitchen-sink
```

```
┌─────────────────────────────────────────────────────────────────┐
│ 🍳 TUI KITCHEN SINK                              ● SAUCE        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ SECTIONS:                                                       │
│ [1] Viewports & Scrolling                                       │
│ [2] Input Components                                            │
│ [3] Lists & Tables                                              │
│ [4] Progress & Status                                           │
│ [5] Borders & Styling                                           │
│ [6] ASCII Art Gallery                                           │
│ [7] Color Palette                                               │
│ [8] Layout Examples                                             │
│                                                                 │
│ Press number to explore, q to quit                              │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Matters

1. **Learning** — Understand BubbleTea capabilities hands-on
2. **Reference** — Quick lookup for symbols and techniques
3. **Prototyping** — Copy-paste starting points for new features
4. **Consistency** — Establish visual patterns for clood TUIs
5. **Sauce Expression** — Know what's possible for narrative mode

### Implementation

```go
// cmd: clood tui-kitchen-sink
// Uses BubbleTea for navigation
// Each section is its own model
// ASCII reference stored as embedded strings
```

See: Charm's BubbleTea examples at github.com/charmbracelet/bubbletea

---

## Bean #17: Project Portfolio Awareness (The Neglected Gardens)

**Status:** Planted
**Session:** The Spirits Emerge (Dec 19, 2025)
**Intensity:** 9/11
**Provenance:** clood is a HECKIN CHONKER eating all the catfood

> *"The svg-grouper grows thin... the equipment to summit Chimborazo lays dormant..."*

### The Problem

When you're deep in one project, others starve:

```
PROJECT HEALTH CHECK (as of Dec 19, 2025)
──────────────────────────────────────────────────────────────

🔥 CONSUMING ALL ATTENTION:
   clood             | 2025-12-19 | 30+ open issues | THE CHONKER

⚠️ RECENTLY TOUCHED BUT FADING:
   chimborazo        | 2025-12-19 | Base camp only, no expedition
   svg-grouper       | 2025-12-17 | 10 issues, grows thin
   rat-king          | 2025-12-17 | Rust patterns, dormant
   church-street     | 2025-12-18 | Oregon Trail recreation

💤 DORMANT (equipment tested, never finished):
   strata            | 2025-12-12 | Catfights didn't reproduce it
   writetyper        | 2025-12-12 | Handwriting SVG
   dogfood           | 2025-12-17 | LLM job hunting

🏔️ BASE CAMP ASSEMBLED, EXPEDITION PENDING:
   chimborazo        | CLI exists, catfight tested, never climbed
```

### The Solution: `clood portfolio`

Multi-repo project awareness via Storytime:

```bash
clood portfolio                    # Overview of all projects
clood portfolio --health           # Health check across repos
clood portfolio --neglected        # What needs attention?
clood portfolio --narrative        # Storytime for the whole garden
```

### Health Indicators

| Indicator | Meaning |
|-----------|---------|
| 🔥 | Active development (commits this week) |
| ⚠️ | Cooling (no commits in 3-7 days) |
| 💤 | Dormant (no commits in 7-30 days) |
| 🪦 | Abandoned (no commits in 30+ days) |
| 🏔️ | Expedition pending (setup complete, work not started) |

### The Narrative Mode

When sauce is ON, Storytime generates narratives about the portfolio:

```
┌─────────────────────────────────────────────────────────────────┐
│ 📜 THE STATE OF THE GARDENS                       ● SAUCE       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Bird-san tends the clood garden with obsessive focus. The      │
│ Flying Cats have multiplied—15 beans planted in a single day.  │
│ But in his fervor, other gardens suffer.                       │
│                                                                 │
│ The svg-grouper, once a promising sapling, grows thin without  │
│ water. Ten issues wait like unwatered seeds.                   │
│                                                                 │
│ At the base of Chimborazo, the expedition equipment sits       │
│ tested but unused. The catfights proved the gear works, but    │
│ no one has begun the climb. The summit remains unseen.         │
│                                                                 │
│ The strata project—maps for plotters—was abandoned when the    │
│ catfights could not reproduce its purpose. It sleeps.          │
│                                                                 │
│ RECOMMENDATION: Tend svg-grouper before it dies. Or decide     │
│ to let it go. A garden half-tended is worse than no garden.    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Data Sources

| Source | Information |
|--------|-------------|
| GitHub API | Repos, issues, last push, activity |
| Local git | Uncommitted changes, branches |
| `.clood/world.yaml` | Project personality (if exists) |
| Issue labels | Priorities, epics, blockers |

### Commands

```bash
# Overview
clood portfolio

# Health focus
clood portfolio --health
clood portfolio --neglected
clood portfolio --abandoned

# Specific project
clood portfolio svg-grouper

# Narrative mode
clood portfolio --narrative
clood portfolio --narrative --style prose
clood portfolio --narrative --style haiku

# Recommendations
clood portfolio --recommend
# → "svg-grouper has 10 issues and no commits in 2 days. Tend or prune?"
```

### The Haiku Summary Mode

```bash
clood portfolio --haiku
```

```
clood consumes all—
svg-grouper grows so thin
Chimborazo waits

Ten seeds unwatered,
strata sleeps beneath the snow—
Choose which gardens live
```

### Connection to Storytime

This is Storytime applied to the **meta-level**:
- Single project: Describe code structure, genesis
- Portfolio: Describe project ecosystem health

The same narrative engine, different scope.

### Why This Matters

1. **Awareness** — See all projects at a glance
2. **Guilt reduction** — Acknowledge what's neglected consciously
3. **Decision support** — Prune or tend? Data-informed choice
4. **Narrative framing** — Turn maintenance into story
5. **Multi-repo future** — Foundation for cross-project features

### The Philosophy

> "A garden half-tended is worse than no garden."

You can't work on everything. But you can be *aware* of everything. clood portfolio gives you that awareness—with or without sauce.

---

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
