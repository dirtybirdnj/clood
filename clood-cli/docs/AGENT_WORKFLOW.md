# Agent Workflow Guidelines

Instructions for AI agents working on the clood repository.

## Core Principles

1. **Humans close issues** - Agents comment, label, and recommend but never close
2. **Questions go on GitHub** - Not just in chat, post to the relevant issue
3. **@mention for notifications** - Always tag @dirtybirdnj for decisions
4. **Document everything** - Future agents and humans need the context

## Issue Interaction Protocol

### When You Have Questions

Post a comment on the relevant issue using this format:

```markdown
## 🤖 Agent Questions - Needs Human Input

@dirtybirdnj - [Brief context for why these questions arose]

### Question 1: [Topic]
**Option A:** [Description]
**Option B:** [Description]
**Option C:** [Description]

*My recommendation: Option [X] because [reasoning]*

### Question 2: [Topic]
...

---
Please reply to this comment with your preferences.

🤖 Questions from Claude Code agent
```

### When You Complete Work

Add a review comment and apply the `agent-review-complete` label:

```markdown
## ✅ Implementation Complete

[Summary of what was implemented]

### Features Delivered:
- Feature 1
- Feature 2

### Testing Notes:
[How it was tested]

### Files Changed:
- `path/to/file.go` - [what changed]

---
🤖 Review by Claude Code agent
Status: READY FOR HUMAN REVIEW
Suggested action: Close if implementation verified
```

Then apply label:
```bash
gh issue edit NUMBER --add-label "agent-review-complete"
```

### When You Find Bugs

Create a new issue:

```markdown
## Bug Report

[Description of the bug]

### Reproduction
```bash
[commands to reproduce]
```

### Expected Behavior
[what should happen]

### Actual Behavior
[what happens instead]

### Fix Location
[where in the code the fix should go]

---
🤖 Discovered by Claude Code agent during [activity]
```

## Labels

| Label | Who Sets | Meaning |
|-------|----------|---------|
| `agent-review-complete` | Agent | Work done, needs human verification |
| `agent-in-progress` | Agent | Currently being worked on |
| `agent-needs-info` | Agent | Blocked, waiting for human input |
| `human-approved` | Human | Verified, ready to close |

## Anti-Patterns to Avoid

❌ **Don't close issues** - Even if you're confident it's done
❌ **Don't merge PRs** - Human reviews and merges
❌ **Don't skip questions** - When in doubt, ask on the issue
❌ **Don't delete branches** - Let humans manage git state
❌ **Don't force push** - Ever

## GitHub CLI Commands

```bash
# Comment on an issue
gh issue comment NUMBER --body "comment text"

# Add a label
gh issue edit NUMBER --add-label "label-name"

# Create an issue
gh issue create --title "Title" --body "Body"

# List issues needing attention
gh issue list --label "agent-needs-info"
```

## Commit Messages

Include a haiku or creative element for hardware/benchmark commits:

```
Add hardware facts collection script

Servers hum in night,
JSON facts flow through the wire—
Garden comes alive.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

## Communication Flow

```
┌─────────────────────────────────────────────────────────────┐
│ Agent finds ambiguity during implementation                 │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Agent posts question comment on GitHub issue with @mention  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Human receives notification, reviews, replies               │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Agent implements based on feedback                          │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Agent posts completion comment, adds label                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Human reviews, approves, closes issue                       │
└─────────────────────────────────────────────────────────────┘
```

## Why This Matters

1. **Training data** - GitHub comments become training data for agent improvement
2. **Audit trail** - Decisions are documented for future reference
3. **Async workflow** - Human doesn't need to be in same chat session
4. **Safety** - Human remains in control of project state
5. **Learning** - Patterns from interactions improve clood CLI itself
