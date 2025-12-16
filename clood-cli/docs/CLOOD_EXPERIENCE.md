# The Clood Experience

For humans. The vision, the flow, the feel.

> **See also:** [CLOOD_TOOLS.md](./CLOOD_TOOLS.md) for command reference and agent usage.

---

## The Saga

Every project has one saga - an ongoing conversation about its development.

### Beginning a New Saga

```
$ cd ~/Code/my-project
$ clood chat

No saga found. Beginning discovery...

📁 Project Analysis:
   Type: Go module (go.mod found)
   Size: 47 files, 12 packages
   Docs: README.md ✓, CLAUDE.md ✗

🔍 Generating context artifacts...
   → llm-context/CODEBASE.md (created)
   → llm-context/API.md (created)
   → llm-context/ARCHITECTURE.md (created)

The Saga of my-project begins.
Context loaded: 3,200 tokens

You:
```

### Continuing a Saga

```
$ clood chat

Continuing The Saga of my-project...
Last session: 2 days ago (47 exchanges)

You:
```

### The Chat Loop

```
You: help me implement user authentication

[response streams...]

You: that's good, but add session management too

[remembers previous context, continues...]

You: /save auth-design.md
Saved conversation to auth-design.md

You: /quit
Saga saved. See you next time.
```

---

## Context Health

The saga tracks context usage with a visual health meter.

### Healthy (Green)
```
╭─ The Saga of my-project ─────────────────────────────────────────╮
│ Context: ████░░░░░░░░░░░░ 25% (4,000 / 16,000 tok)              │
╰──────────────────────────────────────────────────────────────────╯
```

### Warning (Yellow)
```
╭─ The Saga of my-project ─────────────────────────────────────────╮
│ Context: ██████████░░░░░░ 65% ⚠️                                 │
╰──────────────────────────────────────────────────────────────────╯
```

### Critical (Red)
```
╭─ The Saga of my-project ─────────────────────────────────────────╮
│ Context: █████████████░░░ 85% 🔴 COMPRESS SOON                   │
╰──────────────────────────────────────────────────────────────────╯
```

### Compression Flow

When context reaches 80%, clood prompts for human-guided compression:

```
Context at 80%. Time to compress.

I'll ask you 3 questions to capture key decisions:

1. What's the main goal we're working toward?
   > implementing user auth with OAuth2

2. What key decisions have we made?
   > using JWT tokens, storing refresh tokens in Redis

3. What's blocked or still unclear?
   > need to decide on session timeout policy

Compressing... New context: 2,400 tokens (15%)
Archived 45 messages to .clood/saga-archive/

Ready to continue.
```

---

## The Server Garden

Your local LLM infrastructure - machines working together.

### Viewing the Garden

```
$ clood garden

╭─ The Server Garden ──────────────────────────────────────────────╮
│                                                                  │
│  ┌─────────────────┐                                            │
│  │  MacBook Air    │  DRIVER                                    │
│  │  clood chat     │  Orchestration                             │
│  └────────┬────────┘                                            │
│           │                                                      │
│     ┌─────┴─────┐                                               │
│     ▼           ▼                                               │
│  ┌──────────┐  ┌──────────┐                                     │
│  │ ubuntu25 │  │ mac-mini │                                     │
│  │ ● ONLINE │  │ ● ONLINE │                                     │
│  │ 8 models │  │ 2 models │                                     │
│  │ 35 tok/s │  │ ?? tok/s │                                     │
│  └──────────┘  └──────────┘                                     │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯
```

### Tier Routing

```
$ clood tiers

╭─ Tier Routing ───────────────────────────────────────────────────╮
│                                                                  │
│  ⚡ Fast      qwen2.5-coder:3b    → mac-mini (fastest)          │
│  🧠 Deep      qwen2.5-coder:7b    → ubuntu25                     │
│  🔬 Analysis  llama3.1:8b         → ubuntu25                     │
│  ✍️  Writing   llama3.1:8b         → ubuntu25                     │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯
```

### Architecture

```
DRIVER (your laptop)
    │
    ├── Runs: clood chat (saga management)
    ├── Shows: Context health, routing decisions
    └── Orchestrates: Worker queries
          │
          ▼
WORKERS (ubuntu25, mac-mini)
    │
    ├── Run: Ollama with various models
    ├── Execute: LLM queries routed by tier
    └── Use: Same clood CLI tools for surgical ops
```

---

## Chat Commands

While in `clood chat`, these slash commands are available:

| Command | Action |
|---------|--------|
| `/save FILE` | Save conversation to file |
| `/clear` | Clear history (keep context) |
| `/context` | Show loaded context |
| `/context add FILE` | Add file to context |
| `/stats` | Show saga statistics |
| `/compress` | Trigger compression flow |
| `/tier N` | Switch tier for next message |
| `/quit` | Exit and save saga |

---

## Workflow Patterns

### Starting a New Feature

```
$ clood chat

You: I want to add user authentication

[clood suggests approach, you refine]

You: /save docs/auth-design.md
You: let's start with the login endpoint
```

### Code Review

```
$ clood chat --tier 3

You: review internal/auth/login.go for security issues

[detailed analysis]

You: fix the SQL injection on line 45
```

### Creative Writing (The Saga of Clood)

```
$ cd ~/Code/clood
$ clood chat --tier 4

You: continue the legend of Lord Clood and the copper mines

[narrative streams...]

You: /save japanese-history.md
```

---

## Opting Out

Create `.cloodignore` to disable saga in a directory:

```bash
echo "# No saga here" > .cloodignore
```

```
$ clood chat
Saga disabled for this directory (.cloodignore found)
Use --force to override
```

---

## The Philosophy

```
Discovery before query.     (grep, symbols, imports)
Measure before send.        (tokens --check)
Human guides compression.   (not auto-summarize)
One saga per project.       (not per-feature)
CLI tools for agents.       (same toolkit everywhere)
```
