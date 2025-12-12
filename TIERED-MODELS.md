# Tiered Model Strategy

Use fast models to gather info, slow models to produce quality output.

## Model Tiers

### Tier 1: Fast (GPU, instant feedback)
- `qwen2.5-coder:3b` - drafts, exploration, throwaway code
- `qwen2.5-coder:7b` - solid code generation
- `deepseek-coder:6.7b` - code-focused tasks

**Use for:** Quick iterations, gathering context, first drafts, exploration

### Tier 2: Quality (CPU+GPU, wait for it)
- `qwen2.5-coder:14b` - better reasoning, fewer mistakes
- `deepseek-coder:33b` - best local quality (slow)

**Use for:** Final implementations, complex logic, important code

### Tier 3: Cloud (when local isn't enough)
- Claude via `claude` CLI - complex architecture, planning

---

## Workflow: Fast → Review → Slow

### Phase 1: Exploration (Fast Model)

Use 3B/7B to quickly gather information:

```
# In Crush with qwen2.5-coder:3b

> List all the files in src/ that handle authentication
> Summarize what each one does in 1 sentence

> Show me the function signatures in auth/middleware.py

> What libraries are used for JWT handling?
```

Fast model produces rough notes. Save output to a file:
```bash
# Copy output to a staging file
cat > /tmp/auth-notes.md << 'EOF'
[paste fast model output here]
EOF
```

### Phase 2: Human Review

Read the notes. Decide what to do. Write a clear task:

```markdown
# Task: Add refresh token support

## Context (from fast model exploration):
- Auth is in src/auth/
- JWT handled by python-jose
- Current flow: login() returns access_token only
- No refresh token table exists

## What I want:
- Add refresh_tokens table
- Modify login() to return both tokens
- Add /auth/refresh endpoint
- Access token: 15min, Refresh token: 7 days
```

### Phase 3: Implementation (Slow Model)

Feed the context + task to 14B/33B:

```
# In Crush with qwen2.5-coder:14b

[paste your task document]

Implement this. Show me the code for each file that needs to change.
```

Slow model has full context, produces better code.

---

## Practical Commands

### Quick model switching in Crush

Crush remembers your model choice per session. To switch:
1. Exit Crush (`Ctrl+C` or `/exit`)
2. Restart and select different model

### Running models in parallel

You can have multiple terminals:

```bash
# Terminal 1: Fast exploration
cd ~/Code/myproject
crush  # select 3b

# Terminal 2: Quality implementation
cd ~/Code/myproject
crush  # select 14b
```

Both hit the same Ollama server.

### Saving context between models

```bash
# After fast model exploration, save to file
cat > context.md << 'EOF'
## Codebase Notes

[paste fast model output]

## Task

[write what you want done]
EOF

# Feed to slow model
cat context.md
# Copy and paste into slow model session
```

---

## Example: Adding a Feature

### Step 1: Fast exploration (3b, ~2 min)

```
> What does the /api/users endpoint do? Show me the route handler.

> What validation exists for user creation?

> How are errors handled in this codebase? Show me an example.
```

Save notes.

### Step 2: Write task spec (human, ~5 min)

```markdown
# Add email verification to user registration

## Current state (from exploration):
- POST /api/users creates user immediately
- No email field validation beyond format
- Errors return {"error": "message"}

## Requirements:
- Add email_verified boolean field (default false)
- Generate verification token on registration
- Add GET /api/verify?token=xxx endpoint
- Don't allow login until verified

## Constraints:
- Use existing error handling pattern
- Token expires in 24h
```

### Step 3: Implementation (14b, ~5 min)

```
[paste task spec]

Implement this following the existing patterns in the codebase.
Show each file separately.
```

### Step 4: Review output

- Check for obvious errors
- Run linter/tests
- Iterate if needed

---

## When to Use Each Tier

| Task | Model | Why |
|------|-------|-----|
| "What files handle X?" | 3b | Just grepping/listing |
| "Summarize this function" | 3b/7b | Simple comprehension |
| "Write a simple util function" | 7b | Straightforward code |
| "Implement feature with context" | 14b | Needs reasoning |
| "Complex refactor" | 14b/33b | Needs to hold lots of context |
| "Architecture decisions" | Claude | Beyond local model capability |

---

## Anti-patterns

❌ Using 33b for quick exploration (wastes time)
❌ Using 3b for complex implementation (poor quality)
❌ Skipping human review phase (garbage in, garbage out)
❌ Not providing context to slow model (it can't read your mind)

---

## Model Config for Crush

Make sure all tiers are in your Crush config:

```json
{
  "models": [
    {"name": "⚡ Qwen 3B (fast)", "id": "qwen2.5-coder:3b", ...},
    {"name": "⚡ Qwen 7B (fast)", "id": "qwen2.5-coder:7b", ...},
    {"name": "⚡ DeepSeek 6.7B (fast)", "id": "deepseek-coder:6.7b", ...},
    {"name": "🧠 Qwen 14B (quality)", "id": "qwen2.5-coder:14b", ...},
    {"name": "🧠 DeepSeek 33B (slow)", "id": "deepseek-coder:33b", ...}
  ]
}
```

The emoji prefixes help you quickly identify fast vs slow models.
