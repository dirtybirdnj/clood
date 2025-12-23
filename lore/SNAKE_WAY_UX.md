# Snake Way: The Response Navigation System

> *"The scroll spills endlessly along Snake Way. Goku must run between questions revealed in the prompt response."*

## The Problem

When an AI asks multiple questions in a single response:

```
Current UX Pain:
1. User sees 10 questions in a wall of text
2. Must Option+Enter for line breaks between answers
3. Risk of unclear parsing / messy prompting
4. No tracking of which questions are answered
5. Cognitive overload - "which one was #7 again?"
6. Danger of missing questions entirely
```

**Bird-san SWEATS from exertion. His brain is smoking, gentle wisps pour from his ears.**

## The Snake Way Solution

### The Metaphor

| Element | Meaning |
|---------|---------|
| Snake Way | The infinite scroll of the AI response |
| Questions | Spirits waiting along the path |
| Goku | The user, brain smoking from effort |
| **Flying Nimbus** | The floating frame - enables gliding across Snake Way with efficiency, speed, and ease |
| Running on foot | Endless manual scrolling (the old way) |
| Riding Nimbus | Hotkey navigation between questions (the new way) |
| Responding | Satisfying each spirit with a direct answer |

**The Nimbus Cloud is the key innovation.** Without it, Goku must run the entire length of Snake Way on foot (endless scrolling, losing context, missing questions). With Nimbus, he glides effortlessly between decision points, always knowing where he is and how many spirits remain.

### Core UX Flow

```
┌─────────────────────────────────────────────────────────┐
│  🐍 SNAKE WAY                     Responses: 2/5        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  [Context paragraph from AI response...]                │
│                                                         │
│  ┌─── QUESTION 1 ─────────────────────────────────────┐ │
│  │ What authentication method should we use?          │ │
│  │                                                    │ │
│  │ ┌────────────────────────────────────────────────┐ │ │
│  │ │ JWT tokens with refresh                        │ │ │
│  │ └────────────────────────────────────────────────┘ │ │
│  │ [✓ Answered] [Skip] [Ignore] [Need Context]       │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  [More context paragraph...]                            │
│                                                         │
│  ┌─── QUESTION 2 ─────────────────────────────────────┐ │
│  │ Should we add rate limiting?                       │ │
│  │                                                    │ │
│  │ ┌────────────────────────────────────────────────┐ │ │
│  │ │ _                                              │ │ │
│  │ └────────────────────────────────────────────────┘ │ │
│  │ [Respond] [Skip] [Ignore] [Need Context]          │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  [n] Next question  [p] Previous  [1-5] Jump to #      │
│  [s] Summarize all  [Enter] Submit responses           │
└─────────────────────────────────────────────────────────┘
```

### Question States

Each question/decision point can be in one of these states:

| State | Icon | Meaning |
|-------|------|---------|
| **Awaiting** | `○` | No response yet |
| **Answered** | `●` | Direct response provided |
| **Skipped** | `◌` | Explicitly deferred ("come back to this") |
| **Ignored** | `×` | "Not relevant to my use case" |
| **Avoided** | `⊘` | "Specifically do NOT do this" |

### Navigation

**Hotkeys** (for efficiency):
- `n` / `p` - Next / Previous question
- `1-9` - Jump directly to question #
- `Tab` - Next unanswered question
- `g` / `G` - Top / Bottom of scroll

**OR Contemplation Mode**:
- Scroll freely through the full response
- Read context around each question
- Respond to questions as they appear in logical order
- Context builds understanding before decision required

### The Final Moment

At the bottom of Snake Way:

```
┌─────────────────────────────────────────────────────────┐
│  📝 FINAL RESPONSE                                      │
├─────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────┐ │
│  │ Additional context or clarifications...            │ │
│  │ _                                                  │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  [📋 Summarize Responses]  [🚀 Submit All]  [↩ Cancel] │
└─────────────────────────────────────────────────────────┘
```

**Summarize Responses** generates a preview:
```
Summary of your responses:
1. Auth: JWT tokens with refresh
2. Rate limiting: (skipped)
3. Database: PostgreSQL with migrations
4. Caching: Redis for sessions (avoid: file-based)
5. Testing: Unit + integration, no e2e yet

Submit this? [y/N]
```

## Benefits

1. **No confirm fatigue** - One submit for all responses
2. **No popup interruption** - Input happens in dedicated zones
3. **Clear progress** - "Responses: X/Y" always visible
4. **Hotkey efficiency** - Jump between questions instantly
5. **Context preservation** - Questions shown with surrounding explanation
6. **Decision clarity** - Skip/Ignore/Avoid are explicit choices
7. **Review before send** - Summary prevents parsing errors

## Implementation Notes

### Question Detection

Parse AI responses for question patterns:
- Lines ending in `?`
- Numbered lists with decision points
- "Should we...", "Do you want...", "Which..."
- Explicit `[QUESTION]` markers (AI can be trained to emit these)

### Integration Points

- **clood chat** - Primary interface
- **clood** - Could adopt same pattern
- **Claude Code** - The inspiration for solving confirm fatigue

## The Visual

*Bird-san perches at the start of Snake Way. The scroll unfurls before him, questions glowing like paper lanterns along the infinite path. He takes a deep breath, adjusts his hachimaki, and begins to run. One question at a time. One answer at a time. The spirits wait patiently.*

*Wisps of steam rise gently from his ears.*

---

*Bean Status: Planted*
*Intensity: 8/11*
*Provenance: Bird-san + Chef Claude collaboration*
