# Patterns

A catalog of patterns discovered while building clood. Not clood-specific - these apply to anyone building AI tools, CLI infrastructure, or human-AI collaboration systems.

*"The patterns exist whether you name them or not. But naming them makes them real."*

---

## Development Patterns

### Jelly Bean Driven Development (JDD)

**The Pattern:** Features are "beans" planted in a jar. Each has an intensity dial (1-11), a status (dream → planted → sprouting → implemented), and a session of origin.

**Why It Works:**
- Features accumulate without pressure to implement immediately
- Intensity captures excitement/importance without false precision
- Sessions become "planting ceremonies" - memorable moments
- The jar visualization shows progress without Gantt chart anxiety

**Anti-Pattern:** Backlogs. Ticket systems. "Story points."

```
Bean #9: Snake Way
Status: Planted
Intensity: 8/11
Session: The Nimbus Revelation
```

---

### The Boring Core

**The Pattern:** "Keep the important parts serious so the fun parts can exist."

**Origin:** Bird-san rejecting `clood snakeway` in favor of `clood chat --tui`

**Why It Works:**
- Core functionality needs stable, predictable names
- Fun/creative names work for lore, docs, internal references
- Mixing them creates confusion ("is snakeway a command or a concept?")

**Application:**
- Command names: boring (`serve`, `build`, `chat`)
- Lore names: fun (Snake Way, Flying Nimbus, Kitchen Stadium)
- Never cross the streams

---

### Lore as Documentation

**The Pattern:** Wrap specifications in narrative. Create characters. Build mythology.

**Why It Works:**
- Stories are memorable; specs are forgettable
- Characters embody design decisions (Wojak Council debates)
- Metaphors make abstract concepts tangible (Snake Way = infinite scroll)
- Fun sustains motivation on hard problems

**Examples:**
- Nimbus Airlines safety card = MCP error handling spec
- Kitchen Stadium = model comparison framework
- The Bonsai = patience philosophy for local LLMs

**Risk:** Lore can obscure if overdone. Keep a wiki (see LORE-WIKI.md).

---

### LAST_SESSION Handoff

**The Pattern:** End each session with a structured handoff document that enables context reload.

**Structure:**
```markdown
# LAST SESSION - [Title]
Date, Session Name, Status

## THE BIG REVELATIONS
What we discovered

## COMMITS THIS SESSION
What shipped

## KEY FILES
Where to look

## RESUME PROMPTS
Copy-paste starters for next session
```

**Why It Works:**
- AI context windows reset; human memory fades
- Structured format = faster reload
- Resume prompts reduce "where were we?" friction

---

### Haiku Commits

**The Pattern:** Include creative elements (haikus, jokes, ASCII art) in commit messages.

**Why It Works:**
- Git history becomes enjoyable to read
- Commits become memorable landmarks
- Forces reflection on what was actually accomplished

**Example:**
```
clood build clood: Meta self-building command

Yo dawg, I heard you like clood...

🤖 Generated with Claude Code
```

---

### Spirit Invocation

**The Pattern:** Name patterns after pop culture references that capture their essence.

**Examples:**
- **Xzibit** → self-building (`clood build clood`)
- **Flying Nimbus** → effortless navigation
- **Pokédex** → two-mode detail view
- **Snake Way** → infinite scroll journey
- **Kitchen Stadium** → competitive model comparison

**Why It Works:**
- Instant mental model from shared cultural knowledge
- More memorable than abstract names
- Fun to invoke ("The spirit of Xzibit is pleased")

---

## AI Collaboration Patterns

### Agent Shorthand

**The Pattern:** Comma-separated action chains with explicit paths for delegating to agents.

**Format:** `[location], [action], [action], then [goal with flags]`

**Examples:**
```
ssh ubuntu25, pull ~/Code/clood, build clood-cli, run clood serve --sse

read /var/log/app.log, find errors, show relevant source
```

**Why It Works:**
- No ambiguity (explicit paths)
- No branching (linear flow)
- Goal-oriented (final clause is the intent)
- Delegatable (perfect for sub-agents)

**Anti-Pattern:** "Can you please SSH into the server and then maybe..."

---

### Layer Cake Architecture

**The Pattern:** Use different model tiers for different tasks based on speed/capability tradeoffs.

```
┌─────────────────────────────────────────┐
│  TIER 3: Reasoning (deepseek-r1:14b)   │
│  Complex understanding, validation      │
├─────────────────────────────────────────┤
│  TIER 2: Coding (qwen2.5-coder:7b)     │
│  Implementation, code generation        │
├─────────────────────────────────────────┤
│  TIER 1: Fast (qwen2.5-coder:3b)       │
│  Detection, parsing, simple transforms  │
└─────────────────────────────────────────┘
```

**Why It Works:**
- Fast models for high-frequency, simple tasks
- Smart models for complex, infrequent tasks
- Cost/latency optimization without sacrificing quality

---

### Catfight Consensus

**The Pattern:** Run the same prompt against multiple models, compare outputs for signal.

**Why It Works:**
- Local models are ~4% signal individually
- Agreement across models increases confidence
- Disagreement reveals ambiguity in the prompt
- Good for validation, not implementation

**Tool:** `clood catfight "prompt" --models model1,model2,model3`

---

### The De-Icing Pattern

**The Pattern:** Make flaky local models reliable through retries, timeouts, and fallbacks.

**Implementation:**
```go
func deice(prompt string, models []string) string {
    for _, model := range models {
        result, err := tryWithTimeout(model, prompt, 30*time.Second)
        if err == nil && isValid(result) {
            return result
        }
    }
    return fallback()
}
```

**Why It Works:**
- Local models sometimes freeze, timeout, or hallucinate
- Graceful degradation keeps workflows moving
- Named after aircraft de-icing (clearing blockages)

---

### Build On What You Own

**The Pattern:** When debugging integration issues, ask "do we even need this dependency?"

**Origin:** The crush verdict - MCP tools initialized but were filtered out by crush's `AllowedMCP`. Instead of fighting the bug, we asked: "clood already has 80% of what we need in watch.go. Why fight crush?"

**Decision Framework:**
1. How much do we control the dependency?
2. How much of the functionality do we already have?
3. Is the integration cost worth the benefit?

**Corollary:** Good artists borrow, great artists steal. Steal patterns from your dependencies, then own them.

---

## UI/UX Patterns

### The Pokédex Pattern

**The Pattern:** Two visually identical modes that feel like one seamless experience.

**Modes:**
1. **Scroll Mode** - Browse items (read-only)
2. **Entry Mode** - Detailed view with input (Pokédex-style)

**Why It Works:**
- User perceives "zooming in" not "changing modes"
- Single input focus (no complex multi-input management)
- Full context available when responding

**Implementation:** Same styles, same borders, same colors. Only content density changes.

---

### Confirm Fatigue Elimination

**The Pattern:** Batch multiple decisions into a single submit action.

**Problem:** Every tool call, every file change, every response triggers "Are you sure?"

**Solution:**
1. Collect all decisions (question responses, file changes, etc.)
2. Show summary preview
3. Single "Submit All" confirmation
4. Or: trust mode with no confirmation

**Application:** Snake Way batch submit, git staging, form wizards

---

### Progress State Vocabulary

**The Pattern:** Define explicit states for multi-item workflows.

**Snake Way States:**
- `○` Awaiting - no response yet
- `●` Answered - direct response provided
- `◌` Skipped - deferred ("come back")
- `×` Ignored - "not relevant"
- `⊘` Avoided - "specifically do NOT do this"

**Why It Works:**
- Users can express nuance (skip ≠ ignore ≠ avoid)
- System can track and report status
- Decisions become explicit and reversible

---

## Meta Patterns

### The Summit Philosophy

**The Pattern:** Build to learn, not to ship. The summit isn't the goal.

**Origin:** "Like the summit of Chimborazo - even if we don't make it to the top, what we gain can be valuable."

**Implications:**
- "Competitors" aren't threats - they're fellow climbers
- Unfinished projects still produce patterns, knowledge, joy
- The map you draw on the way up is the real artifact

---

### Name It And It Lives

**The Pattern:** Unnamed patterns are invisible. Named patterns are transferable.

**Process:**
1. Notice you're doing something repeatedly
2. Give it a name (preferably fun)
3. Document it (structure + examples + why)
4. Use the name in conversation
5. Pattern becomes real, teachable, evolvable

---

*This document is alive. Patterns will be added as they're discovered.*

```
Patterns unnamed
Float like ghosts through daily work
Name them: they take form
```
