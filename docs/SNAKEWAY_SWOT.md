# Snake Way SWOT Analysis

> *EVERYBODY BEE SWOT - The Awful Waffle Ska Band performs analysis*

**Date:** December 18, 2025
**Epic:** #135

---

## Catfight Test Results Summary

### Question Generation Patterns

| Model | Host | Questions Generated | Format |
|-------|------|---------------------|--------|
| qwen2.5-coder:3b | localhost | 5 main + 5 sub | Numbered + bold headers + bullet sub-questions |
| qwen2.5-coder:3b | ubuntu25 | 5 | Simple numbered list |
| mistral:7b | ubuntu25 | 5 | Numbered paragraphs with context |

### Question Parsing Test

qwen2.5-coder:3b successfully parsed ALL 10 questions from complex response:
```
[Q1] through [Q10] - correctly extracted including sub-questions
```

**Key Finding:** Coding models can BOTH generate AND parse questions effectively.

---

## Component Breakdown

### Component 1: Question Detection/Parsing

**What it does:** Analyze AI responses and extract individual questions.

#### SWOT

| **Strengths** | **Weaknesses** |
|---------------|----------------|
| Regex patterns work for numbered lists | Multi-format responses vary by model |
| Models can self-parse with prompts | Sub-questions add complexity |
| Fast execution (3-8 seconds) | Contextual sentences may be missed |
| qwen2.5-coder handles both gen/parse | No training data for edge cases |

| **Opportunities** | **Threats** |
|-------------------|-------------|
| Train models to emit `[Q]` markers | Models may change output format |
| Use reasoning model for ambiguous cases | False positives (rhetorical questions) |
| Cache common patterns | Performance hit on long responses |
| Multi-model consensus | Dependency on model availability |

#### Status: **NOT BUILT**

#### Implementation Path:
```go
func parseQuestions(content string) []Question {
    // Tier 1: Regex for numbered patterns
    // Tier 2: Sentence-ending with ?
    // Tier 3: Model-assisted parsing for complex cases
}
```

---

### Component 2: Viewport with Section Navigation

**What it does:** Scrollable TUI with hotkey navigation between sections.

#### SWOT

| **Strengths** | **Weaknesses** |
|---------------|----------------|
| Already built in watch.go | Tied to file watching, not chat |
| bubbletea viewport proven | Section struct needs adaptation |
| Hotkey patterns working | No input zones currently |
| Auto-follow implemented | Fixed section detection |

| **Opportunities** | **Threats** |
|-------------------|-------------|
| Direct port to snakeway.go | Bubbletea API changes |
| Reuse 90% of code | Memory issues with huge scrolls |
| Add dynamic section updates | Terminal compatibility |
| Smooth animation polish | Lipgloss rendering edge cases |

#### Status: **80% BUILT** (in watch.go)

#### What's Missing:
- Adapt Section → Question struct
- Wire to chat instead of file
- Add dynamic content updates

---

### Component 3: Input Zones

**What it does:** Text input fields within the scrollable viewport.

#### SWOT

| **Strengths** | **Weaknesses** |
|---------------|----------------|
| bubbles/textinput available | Not integrated with viewport yet |
| bubbles/textarea for multi-line | Focus management complex |
| Existing styling patterns | Cursor positioning in scroll |
| Clear model for input state | Multiple simultaneous inputs |

| **Opportunities** | **Threats** |
|-------------------|-------------|
| Learn from kicli multi-pane | Input conflicts with navigation |
| Inline editing experience | Scroll position jumps |
| Copy/paste support | Terminal input limitations |
| History/autocomplete | Accessibility concerns |

#### Status: **NOT BUILT**

#### Implementation Challenge:
```go
type snakewayModel struct {
    viewport      viewport.Model
    questions     []Question
    currentQ      int
    inputMode     bool          // Toggle between nav and input
    inputs        []textinput.Model  // One per question
}
```

---

### Component 4: Question State Management

**What it does:** Track state of each question (awaiting, answered, skipped, ignored, avoided).

#### SWOT

| **Strengths** | **Weaknesses** |
|---------------|----------------|
| Simple enum/const pattern | State sync with input zones |
| Clear visual mapping | Undo/redo complexity |
| Fits existing Go patterns | Persistence between sessions |
| Progress indicator simple | Multi-session state recovery |

| **Opportunities** | **Threats** |
|-------------------|-------------|
| Integrate with saga persistence | State corruption |
| Add timestamps to states | UX confusion on state meaning |
| Export decision history | Over-engineering risk |
| Analytics on patterns | |

#### Status: **NOT BUILT** (simple)

#### Implementation:
```go
type QuestionState int
const (
    StateAwaiting QuestionState = iota
    StateAnswered
    StateSkipped
    StateIgnored
    StateAvoided
)
```

---

### Component 5: Batch Submit & Summary

**What it does:** Collect all responses, show summary, submit as single formatted message.

#### SWOT

| **Strengths** | **Weaknesses** |
|---------------|----------------|
| Clear UX goal | Summary formatting complex |
| Reduces confirm fatigue | Large payload handling |
| Single API call | Error recovery on partial submit |
| Saga integration clear | Response ordering |

| **Opportunities** | **Threats** |
|-------------------|-------------|
| Preview before submit | AI confusion on format |
| Edit summary inline | Token limits on large batches |
| Template responses | Lost context between Q&A |
| Save drafts | Rate limiting |

#### Status: **NOT BUILT**

#### Format Design:
```
Here are my responses to your questions:

1. **Main Features:** [user response]
2. **Authentication:** [user response]
3. **Data Model:** SKIPPED - will address later
4. **Scalability:** IGNORED - not relevant for MVP
5. **API Endpoints:** [user response]
```

---

### Component 6: Saga Integration

**What it does:** Wire Snake Way to existing chat persistence.

#### SWOT

| **Strengths** | **Weaknesses** |
|---------------|----------------|
| Saga already built | JSON structure may need extension |
| Message history works | Multi-question messages complex |
| Focus guardian available | State recovery on crash |
| Project context loading | Memory growth |

| **Opportunities** | **Threats** |
|-------------------|-------------|
| Extend Message struct | Breaking saga format |
| Track question/response pairs | Migration complexity |
| Export conversation trees | File corruption |
| Analytics on Q&A patterns | |

#### Status: **90% BUILT** (in chat.go)

#### Extension Needed:
```go
type Message struct {
    Role      string    `json:"role"`
    Content   string    `json:"content"`
    Questions []Question `json:"questions,omitempty"` // NEW
    Timestamp time.Time `json:"timestamp"`
}
```

---

## The Layer Cake Architecture

### Model Tiers for Snake Way

```
┌─────────────────────────────────────────────────────────────┐
│  TIER 3: Reasoning (deepseek-r1:14b)                       │
│  - Complex question understanding                          │
│  - Ambiguous case resolution                               │
│  - Summary generation                                       │
├─────────────────────────────────────────────────────────────┤
│  TIER 2: Coding (qwen2.5-coder:7b/14b)                     │
│  - Question parsing                                        │
│  - Response formatting                                      │
│  - Code-related questions                                   │
├─────────────────────────────────────────────────────────────┤
│  TIER 1: Fast (qwen2.5-coder:3b)                           │
│  - Quick question detection                                │
│  - Pattern matching                                        │
│  - Simple transforms                                        │
└─────────────────────────────────────────────────────────────┘
```

### Sequential Pipeline

```
User sends prompt
       ↓
┌──────────────────┐
│ Tier 1: Generate │  qwen2.5-coder:3b asks clarifying questions
│ questions        │  (fast, 3-8 seconds)
└────────┬─────────┘
         ↓
┌──────────────────┐
│ Tier 1: Parse    │  qwen2.5-coder:3b extracts [Q1] [Q2] etc
│ questions        │  (fast, 3-8 seconds)
└────────┬─────────┘
         ↓
┌──────────────────┐
│ Snake Way UI     │  User navigates, responds, submits
│ (no model)       │
└────────┬─────────┘
         ↓
┌──────────────────┐
│ Tier 2: Process  │  qwen2.5-coder:7b implements based on answers
│ responses        │  (medium, 15-30 seconds)
└────────┬─────────┘
         ↓
┌──────────────────┐
│ Tier 3: Review   │  deepseek-r1:14b validates/reasons (optional)
│ (optional)       │  (slow, 30-60 seconds)
└──────────────────┘
```

### Async Opportunities

```go
// Parallel parsing while user reads
go func() {
    tier1Response := callModel("qwen:3b", generatePrompt)
    questions := parseQuestions(tier1Response)
    questionsChan <- questions
}()

// User starts reading immediately
// Questions populate as parsed
// No blocking on full parse
```

---

## Priority Matrix

| Component | Effort | Value | Priority |
|-----------|--------|-------|----------|
| Question Detection | Medium | High | **1** |
| Viewport (port watch.go) | Low | High | **2** |
| Question State | Low | Medium | **3** |
| Input Zones | High | High | **4** |
| Saga Integration | Low | Medium | **5** |
| Batch Submit | Medium | High | **6** |

---

## Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Model format inconsistency | High | Medium | Multi-pattern parser, model-assisted fallback |
| Input zone complexity | Medium | High | Start with single active input, iterate |
| Terminal compatibility | Low | High | Test on iTerm2, Terminal.app, tmux |
| Performance on large scrolls | Medium | Medium | Virtualize viewport, lazy render |
| User confusion on states | Low | Medium | Clear visual indicators, help text |

---

## Plan B: MVP Without Input Zones

**The Problem:** Input Zones are HIGH effort and the riskiest component.

**The Insight:** We can get 80% of the value with 20% of the effort by NOT doing inline input zones initially.

### MVP Snake Way (No Input Zones)

```
┌─────────────────────────────────────────────────────────────┐
│  🐍 SNAKE WAY                          Questions: 5         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [Context paragraph from AI...]                             │
│                                                             │
│  ► Q1: What authentication method?                    [○]   │
│    Q2: Do you need rate limiting?                     [○]   │
│    Q3: What's your data model?                        [○]   │
│    Q4: Scalability requirements?                      [○]   │
│    Q5: API endpoint structure?                        [○]   │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  Press ENTER on highlighted question to respond             │
│  [n/p] Navigate  [s] Skip  [i] Ignore  [a] Avoid           │
└─────────────────────────────────────────────────────────────┘
```

**When user presses ENTER:**
```
┌─────────────────────────────────────────────────────────────┐
│  Responding to Q1: What authentication method?              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  > JWT with refresh tokens_                                 │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  [Enter] Save  [Esc] Cancel                                │
└─────────────────────────────────────────────────────────────┘
```

### Why This Works

| Full Vision | MVP | Value Retained |
|-------------|-----|----------------|
| Inline input zones | Modal input popup | 90% |
| See all responses while typing | See one at a time | 70% |
| Complex focus management | Simple modal toggle | 100% simpler |
| bubbletea nested inputs | Single textinput | Much easier |

### What MVP Still Delivers

✅ Question detection and parsing
✅ Hotkey navigation (n/p/1-9)
✅ Progress tracking (Responses: X/5)
✅ Question states (answered/skipped/ignored/avoided)
✅ Batch submit at the end
✅ No confirm fatigue
✅ Nimbus navigation experience

### What MVP Defers

❌ Inline editing (respond in place)
❌ See all responses simultaneously
❌ Multi-input focus management

### The Upgrade Path

```
MVP (Modal)                    →    Full (Inline)
─────────────────────────────────────────────────
Press Enter → Modal popup      →    Tab into inline input
One input at a time            →    All inputs visible
Simple focus                   →    Complex focus management
Ship in days                   →    Ship in weeks
```

**Build MVP first. Validate the UX. THEN tackle inline inputs if needed.**

---

## Next Steps (When Bird-san Returns)

1. **Phase 1 Start:** Copy watch.go → snakeway.go, adapt Section → Question
2. **Regex Library:** Build question detection patterns from catfight data
3. **Simple UI:** Static render of parsed questions with navigation
4. **Input POC:** Single textinput in viewport as proof of concept
5. **Iterate:** Add states, submit, saga integration

---

## Files Created This Session

| File | Purpose |
|------|---------|
| `/tmp/snakeway_test_prompt.txt` | Question generation test prompt |
| `/tmp/snakeway_catfight/` | Catfight outputs for analysis |
| `/tmp/parse_questions_prompt.txt` | Question parsing test prompt |
| `/tmp/parse_catfight/` | Parsing test outputs |

---

*The Awful Waffle Ska Band bows. EVERYBODY BEE SWOT.*
