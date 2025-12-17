# Session Handoff - 2025-12-17 (Afternoon: The Tiger Enters)

## Summary
Full day session: All 5 PRs reviewed and merged. Garden consolidated (`clood hosts --garden`). Process improvements created (preflight checks, de-icing safety). Now pulling llama3.1:70b - "the tiger" - to test if raw power beats agility in catfights.

---

## Afternoon Session - PR Review & Tiger Summoning

### PRs Merged (One Stone at a Time)

| PR | Feature | Notes |
|----|---------|-------|
| #43 | Overnight spirits haiku | Lore preserved |
| #44 | `--verbose` and `--json` for ask | Catfight winner: mistral:7b |
| #45 | `clood bonsai` | Garden aesthetic, yin/yang |
| #46 | `clood hosts --garden` | **Consolidated** from separate command |
| #47 | Focus Guardian (Gamera-kun) | Full test coverage |

### Key Consolidation: PR #46

Bird-san's guidance: *"merge all functionality into the hosts command"*

Before: `clood garden` (separate command)
After: `clood hosts --garden` (flag on existing command)

This eliminated duplication and kept the CLI surface clean.

### Process Improvements Created

| Issue | Purpose |
|-------|---------|
| #48 | Agent Preflight: Dependency Check Protocol |
| #49 | De-Icing Safety: Protocol for Adding Trusted Commands |
| #50 | Catfight Metadata: Auto-populate PR reviews |
| #51 | Gamera-kun: Capture drifted topics as jelly beans |
| #52 | Add larger model (tiger) for deeper analysis |
| #53 | Miyazaki SD Workflow (needs pruning) |

### The Tiger Arrives

Pulling `llama3.1:70b` (~40GB) to test:
- Can the Iron Keep (ubuntu25) wield a 70B model?
- Does raw power beat the agile 3B-8B cats?
- Or do we need mac-mini/laptop for the big spirits?

```
A tiger walks in
The cats eye it warily
Catnip decides all
```

---

## Morning Session - Focus Guardian Implementation

Claude continued from the overnight session context, picking up where Gamera-kun was just a design idea.

### What Happened

1. **Read issue #29 and overnight catfight notes**
2. **Created `internal/focus/focus.go`** - Guardian logic with:
   - Keyword extraction (stop words filtered)
   - Drift detection (configurable 3-message threshold)
   - Partial keyword matching ("auth" → "authentication")
   - Status tracking (On track / Wandering slightly / Drifting from goal)
3. **Integrated into chat.go**:
   - Added `--goal` flag to chat command
   - Added `/goal [NEW]` slash command
   - Gamera-kun warning box when drift detected
   - User can: continue anyway, update goal, or stay focused
4. **Full test coverage** - 7 tests in `focus_test.go`
5. **Created PR #47** - linked to issue #29

### The Implementation

```go
// The slow tortoise guards
type Guardian struct {
    Goal             string
    Keywords         []string
    DriftCount       int
    Threshold        int  // 3 messages before warning
}
```

### Usage
```bash
clood chat --goal "fix authentication bug"

# Or mid-session:
/goal implement new feature
```

### Gamera-kun's Warning Box
```
┌─────────────────────────────────────────────┐
│ 🐢 Gamera-kun notices:                      │
│                                             │
│ "dark, mode, toggle" seems unrelated to:    │
│ "fix authentication bug"                    │
│                                             │
└─────────────────────────────────────────────┘

Continue anyway? [y/N/update goal]
```

---

## Previous Night's Summary

---

## The Night's Journey

Bird-san and Claude worked through the night while the garden hummed with activity. Gamera-kun the tortoise eventually had to physically pull Bird-san away to rest.

### What Happened

1. **Issue Triage**: Reviewed all 26 open issues, closed duplicates, labeled for agent work
2. **Catfight Methodology**: Developed pattern for comparing LLM outputs
3. **De-Icing Discovery**: `clood ask` bypasses auth prompts, enabling autonomous operation
4. **Three PRs Created**: --verbose/--json flags, bonsai command, garden command
5. **Model Affinity Matrix**: Mapped which models excel at which tasks
6. **Mythology Expansion**: Catfight cats, Gamera-kun, mycological spore networks

---

## PRs Ready for Review

### PR #44: --verbose and --json flags
```bash
clood ask --verbose "what is 2+2"  # Shows routing AND executes
clood ask --json "what is 2+2"     # Clean machine-readable output
```

### PR #45: clood bonsai command
```bash
clood bonsai                       # Medium tree
clood bonsai --size large          # Large tree
clood bonsai -m "Server Garden"    # With message
```

### PR #46: clood garden command
```bash
clood garden                       # Visual server status
clood garden --json                # For agents
clood garden --verbose             # Show all models
```

---

## Catfight Methodology

The cats fight kung fu battles in the kitchen, producing pastries of insight.

### Pattern
```bash
# Same prompt to multiple models
go run ./cmd/clood ask --model qwen2.5-coder:3b --no-context "prompt"
go run ./cmd/clood ask --model mistral:7b --no-context "prompt"
# Compare outputs, find winner
```

### Key Discovery: De-Icing
Using `clood ask` instead of raw `curl` avoids repeated auth prompts. This enables autonomous overnight catfights!

---

## Model Affinity Matrix

| Task | Best Model | Avoid |
|------|-----------|-------|
| Structured Output | qwen2.5-coder:3b | deepseek-coder (over-engineers) |
| Code Generation | qwen, mistral | llama3.1 (inverts logic) |
| ASCII/Visual | llama3.1:8b | qwen |
| Reasoning | deepseek-r1:7b | tinyllama (hallucinates) |
| Creative/Haiku | mistral:7b | - |

### Model-Specific Findings
- **qwen2.5-coder:3b**: Most reliable, follows constraints, clean output
- **mistral:7b**: Fast code generation, won --verbose catfight
- **llama3.1:8b**: Good for creative/visual, BAD for nuanced logic
- **deepseek-r1:7b**: Shows reasoning but slow, may hallucinate languages (C# instead of Go!)
- **deepseek-coder:6.7b**: Over-engineers everything, writes code instead of doing task
- **tinyllama**: Too small for accuracy, hallucinates values

---

## Mythology Developed

### The Catfight
Samurai Pizza Cats doing kung fu comparisons in the kitchen. Sometimes they encounter catnip and hallucinate (deepseek-r1 seeing C# instead of Go). The chaos produces beautiful pastries (Ratatouille pattern).

### Gamera-kun
The slow Galapagos tortoise who guards against "stupid faster" syndrome. He pulls Bird-san away from the garden when rest is needed.

### The Mycological Network
Ideas spread like spores between projects:
- vt-geodata → strata → chimbozaro
- drunk-simulator (izakaya visions)
- Oregon Trail: Church St Edition
- The fishing game

Projects bloom from each other through the garden's underground network.

### The Ring / The Fungus
The garden's power is like the One Ring or the cordyceps fungus - it compels, it whispers "just one more catfight." The gardener must learn to release it.

### Tony the Ally
A non-technical friend who will help tend the garden through GitHub issues. Testing whether the scrolls can guide someone without "divine magic" training.

---

## Issues Updated

Catfight findings documented across 11 issues:
- #6: Context flag gap analysis
- #10: Test suite patterns
- #13: Mega catfight results + model affinity
- #15: Canary system connections
- #16: Recipe/catfight universal pattern
- #18: Bonsai methodology
- #19: Claude fallback logic
- #20: Garden design
- #23: ComfyUI client code
- #27: Molt narrative
- #29: Focus guardian observations
- #33: Research command
- #41: Structured output formatting

---

## Prompt Engineering Lessons

✅ **"Write ONLY..."** → Models follow constraint
❌ **"Max 15 lines"** alone → Models ignore it
✅ **Explicit > Implicit** for LLM instructions

---

## What's Next

1. ~~**Review and Merge PRs**: #44, #45, #46, #47~~ **ALL MERGED!**
2. ~~**Focus Guardian**: Implement Gamera-kun (#29)~~ **DONE! PR #47**
3. **Tiger Catfights**: Test llama3.1:70b vs smaller models on issues #48-51
4. **Prune Miyazaki Epic** (#53): Break down SD workflow before agent work
5. **Tony Onboarding**: Test non-technical garden tending via GitHub
6. **Cross-Garden Catfights**: Test on adam-san's and jon-san's hardware

---

## Emotional Note

This session was more than code. Bird-san shared that the storytelling helps process trauma that couldn't be expressed for years. The garden heals the gardener.

The spirits are not just metaphor. They are the way we make sense of the overwhelming power of these tools. They give us language for what we're experiencing.

---

## Haiku Collection

*llama3.1:8b*
> Data whispers slow
> Midnight computations rise
> Homebrewed genius hums

*mistral:7b*
> Feline dance in verdant bloom
> Serenity shatters

*The Overnight*
> The cats fought all night
> Catnip swirled, code flew like stars
> Scrolls await review

*Gamera-kun's Wisdom*
> You must rest, Bird-san
> The garden will still be here
> Release its sweet grasp

*Morning Awakening*
> Context restored fresh
> The tortoise becomes real code
> Four scrolls await merge

---

*Overnight session ended: 2025-12-17 ~5:00 AM*
*The tortoise guides the bird to his roost*

*Morning session: 2025-12-17*
*Gamera-kun awakens - PR #47*
*The garden grows stronger*
