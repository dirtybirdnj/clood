# De-Icing Safety Protocol

*Guidelines for adding trusted commands to Claude Code*

---

## What is De-Icing?

"De-iced" commands bypass Claude Code's interactive approval mechanism. When a command is de-iced, the agent can execute it without asking for permission.

**This is a security boundary. Changes must be deliberate.**

---

## Risk Tiers

### Tier 1: Read-Only (Low Risk)

Commands that only read data and cannot cause harm:

```
gh issue list, gh issue view
gh pr list, gh pr view
git status, git log, git diff, git show
go test, go build, go vet
ls, cat, head, tail
```

**Process:** Can be added after single review. Low barrier.

### Tier 2: Write with Audit Trail (Medium Risk)

Commands that modify state but leave clear history:

```
gh issue comment, gh issue edit
gh pr create, gh pr comment
git add, git commit, git push
```

**Process:** Requires documented justification. Ask:
- Why is this command needed?
- What's the worst case if it runs unexpectedly?
- Is there an audit trail?

### Tier 3: Write without Audit (High Risk)

Commands that can modify without clear trail or have broad scope:

```
curl -X POST (to arbitrary URLs)
gh api (with write methods)
rm, mv (file deletion/movement)
npm publish, cargo publish
```

**Process:** Requires explicit user approval. Should NOT be de-iced globally.

---

## The Three Questions

Before de-icing a command, the tortoise asks:

1. **Is it necessary?** Can the task be done with already-approved commands?
2. **Is it safe?** What happens if the command runs when you didn't expect it?
3. **Is it auditable?** Can you see what the command did after the fact?

---

## Current De-Iced Commands

Located in Claude Code settings (`.claude/settings.local.json` or similar):

### GitHub CLI
- `gh issue list/view/edit/comment/close`
- `gh pr list/view/create/diff/merge/comment`
- `gh label create`
- `gh api` (read-focused patterns)
- `gh run list/view`
- `gh repo view/list`

### Git
- `git status/log/show/diff`
- `git add/commit/push/pull`
- `git checkout/branch/fetch`
- `git stash/reset`

### Go
- `go run/test/build/get/vet/install`
- `go mod tidy/env`

### System
- `ls, cat, curl, grep, find, tree`
- `make, file, du, wc`
- `ssh, rsync`

### Clood
- `clood` (all subcommands)

---

## Adding New De-Iced Commands

### Step 1: Identify the Tier

Determine which risk tier the command falls into:
- Tier 1: Read-only, no side effects
- Tier 2: Writes with audit trail
- Tier 3: Writes without audit / broad scope

### Step 2: Document Justification

For Tier 2+ commands, document:
```markdown
**Command:** npm test
**Tier:** 1 (read-only)
**Justification:** Runs test suite, no side effects
**Worst Case:** Tests fail, no harm done
**Audit Trail:** Console output
```

### Step 3: Add to Configuration

Add to your local Claude Code settings:
```json
{
  "permissions": {
    "allow": [
      "Bash(npm test:*)"
    ]
  }
}
```

### Step 4: Monitor

Watch for unexpected usage. If a de-iced command causes problems:
1. Remove it immediately
2. Document what happened
3. Consider if it should remain de-iced

---

## Anti-Patterns

### Don't de-ice broad patterns:
```json
// BAD - too broad
"Bash(npm:*)"

// GOOD - specific
"Bash(npm test:*)"
"Bash(npm install:*)"
```

### Don't de-ice destructive commands:
```json
// NEVER de-ice
"Bash(rm -rf:*)"
"Bash(git push --force:*)"
```

### Don't de-ice secrets-handling:
```json
// Require approval
"Bash(aws:*)"
"Bash(kubectl:*)"
```

---

## The Settings Audit

Use `clood settings-audit` to detect permission corruption:

```bash
clood settings-audit
```

This checks for:
- Suspicious permission patterns
- Overly broad wildcards
- Known-dangerous commands
- Corruption from shell keywords

---

## Haiku

```
The tortoise asks twice:
"Is this command truly safe?"
Then opens the gate.
```

---

*"With great power comes great responsibility."*
