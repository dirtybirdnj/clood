# Golden Paths

The art of AI-assisted development isn't one command that does everything.
It's the AI orchestrating many small, focused tools in sequence.

**Unix philosophy:** Do one thing well, compose into pipelines.
**clood philosophy:** Granular tools, AI as conductor.

---

## Why Granular > Monolithic

When crush/AI calls `clood catfight` and waits 30 seconds for results:
- User sees loading spinner
- No feedback during execution
- Feels like a black box

When crush/AI breaks it into steps:
- "Checking available models..." → `clood models`
- "Analyzing your hardware..." → `clood system`
- "Running qwen2.5-coder:14b..." → `clood run --model qwen2.5-coder:14b "prompt"`
- "Running deepseek-r1:8b..." → `clood run --model deepseek-r1:8b "prompt"`
- "Comparing results..."

Each step gives feedback. The user sees progress. The experience breathes.

---

## Core Workflows

### 1. Context Gathering (Before Any Task)

```
clood tree           → See the shape of the codebase
clood grep "pattern" → Find relevant code
clood symbols path/  → Understand structure
clood context        → Generate LLM-ready summary
```

**The AI reads the room before speaking.**

Use when: Starting a new session, switching contexts, before major changes.

---

### 2. Code Understanding

```
clood imports file.go  → What does this file depend on?
clood symbols file.go  → What functions/types does it define?
clood analyze file.go  → Deep analysis with reasoning model
```

**Layers: dependencies → structure → meaning**

Use when: Before modifying unfamiliar code, debugging, code review.

---

### 3. Model Selection (The Catfight Flow)

```
clood models         → What's available?
clood system         → What can I run?
clood hosts          → Where can I run it?
[AI decides which models to battle based on capacity]
clood catfight --models "a,b" "prompt"
```

**Know your arsenal, know your limits, then battle.**

Use when: Choosing the right model for a task, comparing approaches.

---

### 4. The Handoff (Session Continuity)

```
clood session show   → Current session state
clood context        → Generate transferable context
clood session save   → Persist for later
clood handoff        → Package for another agent/session
```

**Preserve state across sessions, agents, and time.**

Use when: Ending a session, passing work to another agent, resuming later.

---

### 5. Health → Route → Execute

```
clood hosts          → Which servers are online?
clood health         → Full system check
[AI picks best host based on latency/capacity]
clood run --host X --model Y "prompt"
```

**Check the garden, find the ripest fruit, harvest.**

Use when: Before any LLM call, ensuring optimal routing.

---

### 6. The Development Loop

```
[User describes task]
clood tree + grep + symbols  → Gather context
clood analyze               → Understand current state
[AI proposes changes]
[User approves]
[AI makes changes via crush file tools]
clood analyze               → Verify changes
[Repeat]
```

**Context → Understand → Propose → Execute → Verify**

---

## Designing for Streaming Feel

Even with stdio MCP (no true streaming), we simulate the feel:

1. **Frequent small updates** - Each tool call is a beat
2. **Progressive disclosure** - Start broad, narrow down
3. **Narrate the journey** - AI explains what it's doing and why
4. **Quick wins first** - Fast tools before slow ones

The user should never wonder "what's happening?"

---

## Anti-Patterns

❌ **One giant command** - `clood do-everything --magic`
❌ **Silent waiting** - 30 seconds of spinner
❌ **Skipping context** - Diving into code blind
❌ **Wrong tool for job** - Using 14B model for "hello world"

---

## The Vision

crush + clood should feel like pair programming with a skilled colleague who:
- Looks before leaping
- Explains their thinking
- Uses the right tool for each step
- Keeps you informed
- Learns your codebase

The granular CLI tools are the vocabulary.
The golden paths are the grammar.
The AI is the storyteller.

---

*"Just because you could doesn't mean you should."*
*The most exquisite vision is the one that breathes.*
