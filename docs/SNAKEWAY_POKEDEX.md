# Snake Way: The Pokédex Pattern

> *"It's like a Pokédex, but for questions."*

---

## The Insight

Instead of complex inline inputs, we create **two visually identical modes**:

1. **SCROLL MODE** - Browse questions, read context (read-only)
2. **ENTRY MODE** - Focus on ONE question with full context (input active)

**The user doesn't notice mode switching** because the UI looks the same. They think they're just "scrolling to a question and typing."

---

## The Two Modes

### Scroll Mode (The Journey)

```
┌─────────────────────────────────────────────────────────────────┐
│  🐍 SNAKE WAY                              Responses: 2/5       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  I'll help you build a REST API. First, let me understand      │
│  your requirements better...                                    │
│                                                                 │
│  ┌─ Q1 ─────────────────────────────────────────────────────┐  │
│  │ AUTHENTICATION                                      [●]  │  │
│  │ What authentication method should we use?                │  │
│  │ Your answer: JWT with refresh tokens                     │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Q2 ─────────────────────────────────────────────────────┐  │
│  │ DATA MODEL                                          [○]  │  │
│  │ Can you describe your data model?                        │  │
│  │ Press ENTER to respond...                                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ► Q3 ──────────────────────────────────────────────────────   │
│  │ SCALABILITY                                         [○]  │  │
│  │ Are there expected growth patterns?                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  [j/k] Scroll  [n/p] Next/Prev Q  [ENTER] Respond  [S] Submit  │
└─────────────────────────────────────────────────────────────────┘
```

**User presses ENTER on Q3...**

### Entry Mode (The Pokédex)

```
┌─────────────────────────────────────────────────────────────────┐
│  🐍 SNAKE WAY                              Question 3 of 5      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Q3: SCALABILITY ────────────────────────────────────────┐  │
│  │                                                          │  │
│  │  QUESTION                                                │  │
│  │  ─────────                                               │  │
│  │  Are there expected growth patterns for your             │  │
│  │  application that you need to account for in             │  │
│  │  your API design?                                        │  │
│  │                                                          │  │
│  │  CONTEXT                                                 │  │
│  │  ───────                                                 │  │
│  │  This affects database indexing, caching strategy,       │  │
│  │  and whether you need horizontal scaling. Consider:      │  │
│  │  • Expected concurrent users                             │  │
│  │  • Data volume growth                                    │  │
│  │  • Read vs write ratio                                   │  │
│  │                                                          │  │
│  │  RELATED                                                 │  │
│  │  ───────                                                 │  │
│  │  ← Q2: Data Model (affects schema design)                │  │
│  │  → Q4: Caching (depends on this answer)                  │  │
│  │                                                          │  │
│  ├──────────────────────────────────────────────────────────┤  │
│  │  YOUR RESPONSE                                           │  │
│  │  ─────────────                                           │  │
│  │  > Expecting 10k users initially, growing to 100k_       │  │
│  │                                                          │  │
│  │                                                          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  [ESC] Back to scroll  [TAB] Next Q  [ENTER] Save & Next       │
└─────────────────────────────────────────────────────────────────┘
```

---

## Why This Works

### Visual Continuity

| Element | Scroll Mode | Entry Mode |
|---------|-------------|------------|
| Header | Same | Same |
| Question box | Collapsed card | Expanded card |
| Border style | Same | Same |
| Colors | Same | Same |
| Status indicator | Same position | Same position |

**The user perceives:** "I zoomed into a question" not "I changed modes"

### The Pokédex Mental Model

Just like a Pokédex entry:

```
┌─────────────────────────────────────────┐
│  #025 PIKACHU                           │
│  ═══════════════════════════════════════│
│  TYPE: Electric                         │
│  HEIGHT: 0.4m                           │
│  WEIGHT: 6.0kg                          │
│                                         │
│  DESCRIPTION                            │
│  When several of these Pokémon gather,  │
│  their electricity could build and      │
│  cause lightning storms.                │
│                                         │
│  EVOLUTION                              │
│  ← Pichu  → Raichu                      │
└─────────────────────────────────────────┘
```

Becomes:

```
┌─────────────────────────────────────────┐
│  Q3: SCALABILITY                        │
│  ═══════════════════════════════════════│
│  STATUS: Awaiting                       │
│  PRIORITY: High                         │
│  DEPENDS ON: Q2 (Data Model)            │
│                                         │
│  QUESTION                               │
│  Are there expected growth patterns...  │
│                                         │
│  CONTEXT                                │
│  This affects database indexing...      │
│                                         │
│  RELATED                                │
│  ← Q2  → Q4                             │
│                                         │
│  YOUR RESPONSE                          │
│  > ____________________________         │
└─────────────────────────────────────────┘
```

---

## The Chat Sequence Reframe

### Traditional Chat (What We're Escaping)

```
AI: Here are 5 questions:
    1. Auth?
    2. Data?
    3. Scale?
    4. Cache?
    5. API?

User: 1. JWT
      2. PostgreSQL
      3. 10k users
      4. Redis
      5. REST

AI: [tries to parse this mess]
```

### Snake Way Chat (The New Pattern)

**Turn 1: AI generates questions**
```
AI Response → Parsed into 5 Question entries
User enters Entry Mode for each
```

**Turn 2: AI acknowledges (optional)**
```
AI: "Got it. JWT auth, PostgreSQL, expecting 10k users.
     Let me clarify Q4 about caching..."
```

**Turn 3: Follow-up questions**
```
New questions parsed, added to existing entries
User can revisit Q1-Q5 AND see new Q6-Q8
```

### The Restatement Pattern

When entering Entry Mode, the UI **restates** everything relevant:

```
┌─────────────────────────────────────────────────────────────────┐
│  Q3: SCALABILITY                                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  THE QUESTION                                                   │
│  "Are there expected growth patterns for your application       │
│   that you need to account for in your API design?"             │
│                                                                 │
│  WHY THIS MATTERS                                               │
│  Your answer affects:                                           │
│  • Database indexing strategy                                   │
│  • Caching architecture                                         │
│  • Horizontal vs vertical scaling decisions                     │
│  • API rate limiting configuration                              │
│                                                                 │
│  WHAT YOU'VE ALREADY SAID                                       │
│  • Q1: Using JWT authentication                                 │
│  • Q2: PostgreSQL with users, tasks, projects tables            │
│                                                                 │
│  WHAT COMES NEXT                                                │
│  • Q4: Caching strategy (depends on your scale answer)          │
│  • Q5: API endpoints (informed by all previous answers)         │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  YOUR RESPONSE                                                  │
│  > _                                                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Implementation: Two Components, Same Style

### Component 1: ScrollView

```go
type ScrollView struct {
    viewport    viewport.Model
    questions   []QuestionCard  // Collapsed cards
    currentIdx  int
    styles      SharedStyles    // Same styles as EntryView
}

func (s ScrollView) View() string {
    // Render collapsed question cards in scrollable viewport
    // Highlight current question
    // Show answered/pending status
}
```

### Component 2: EntryView

```go
type EntryView struct {
    question    Question
    context     QuestionContext  // AI-generated context
    related     []QuestionRef    // Links to related questions
    input       textinput.Model
    styles      SharedStyles     // Same styles as ScrollView
}

func (e EntryView) View() string {
    // Render expanded Pokédex-style entry
    // Full question text
    // Context section
    // Related questions
    // Input field at bottom
}
```

### The Transition

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.mode == ModeScroll && msg.String() == "enter" {
            // Seamless transition to Entry Mode
            m.mode = ModeEntry
            m.entryView = NewEntryView(m.questions[m.currentIdx])
            return m, m.entryView.Focus()
        }
        if m.mode == ModeEntry && msg.String() == "esc" {
            // Seamless transition back to Scroll Mode
            m.mode = ModeScroll
            return m, nil
        }
    }
}
```

---

## The Context Engine

### Where Context Comes From

```
┌─────────────────────────────────────────────────────────────────┐
│                        CONTEXT ENGINE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  INPUT                           OUTPUT                          │
│  ─────                           ──────                          │
│  Original AI response    →       Question entries                │
│  Question relationships  →       "Related" links                 │
│  Previous answers        →       "What you've said" summary      │
│  Question dependencies   →       "Why this matters" section      │
│                                                                  │
│  OPTIONAL: Tier 2 model call for rich context generation        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Context Generation (Async)

```go
// When entering Entry Mode, optionally enrich context
func enrichQuestionContext(q Question, prevAnswers []Answer) QuestionContext {
    // Option 1: Static analysis (fast)
    ctx := analyzeQuestionDependencies(q, prevAnswers)

    // Option 2: Model-assisted (richer, async)
    go func() {
        richCtx := callModel("qwen:3b", fmt.Sprintf(
            "Given this question: %s\n"+
            "And these previous answers: %v\n"+
            "Explain why this question matters and what it affects.",
            q.Text, prevAnswers,
        ))
        contextChan <- richCtx
    }()

    return ctx
}
```

---

## Risk Mitigation Summary

| Original Risk | Mitigation |
|---------------|------------|
| Complex inline inputs | Two separate views, same style |
| Focus management | Only one input ever active |
| Cursor positioning | Full-screen input area |
| Multiple simultaneous inputs | One at a time, modal style |
| User confusion | Visual continuity masks mode switch |
| Lost context | Pokédex restates everything |

---

## ASCII Art: The Experience

```
USER JOURNEY
════════════════════════════════════════════════════════════════

  ┌──────────────────┐
  │   AI RESPONSE    │
  │   5 Questions    │
  └────────┬─────────┘
           │
           ▼
  ┌──────────────────┐     User scrolls
  │   SCROLL MODE    │◄────through questions
  │   (Read-only)    │     j/k/n/p
  └────────┬─────────┘
           │ ENTER
           ▼
  ┌──────────────────┐     User types response
  │   ENTRY MODE     │◄────full context visible
  │   (Pokédex)      │     single input field
  └────────┬─────────┘
           │ ENTER (save) or ESC (cancel)
           ▼
  ┌──────────────────┐     Back to browsing
  │   SCROLL MODE    │     answer saved
  │   Q marked [●]   │
  └────────┬─────────┘
           │ ... repeat for each question ...
           ▼
  ┌──────────────────┐
  │  ALL ANSWERED    │
  │  Press S to      │
  │  Submit All      │
  └────────┬─────────┘
           │
           ▼
  ┌──────────────────┐
  │   BATCH SUBMIT   │     All responses sent
  │   to AI          │     as formatted message
  └──────────────────┘
```

---

## The Seamless Illusion

The magic is that from the user's perspective:

1. They see a scrollable list of questions
2. They move to a question and press ENTER
3. The question "expands" with more context
4. They type their response
5. They press ENTER and it "collapses" back
6. They continue scrolling

**They never think "I'm in a different mode."**

They think: *"I'm just scrolling through questions and answering them. This is how chat should always work."*

---

```
Two modes, one face
The Pokédex knows your path
Context guides the way
```
