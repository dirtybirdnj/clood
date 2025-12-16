# Clood Smooth Git: Session Continuity

Patterns for seamless context handoff between LLM sessions.

---

## The Problem

LLM sessions are stateless. When you:
- Hit context limits
- Need a fresh session
- Switch machines
- Come back the next day

...you lose all the context of what you were working on.

**Current workaround:** Manually update LAST_SESSION.md, commit, push, pull in new session.

**Goal:** Make this invisible infrastructure.

---

## Proposed: `clood handoff` Command

### End of Session

```bash
clood handoff --save "Canaries work done, next: implement pre-flight checks"
```

This:
1. Updates LAST_SESSION.md with structured context
2. Captures git diff summary
3. Lists files changed
4. Records next steps
5. Commits with standard message
6. Pushes to remote

### Start of Session

```bash
clood handoff --load
```

This:
1. Runs `git pull`
2. Reads LAST_SESSION.md
3. Outputs context summary for LLM consumption
4. Suggests: "Continue from: {next steps}"

### View History

```bash
clood handoff --history
```

Shows recent handoffs with timestamps and summaries.

---

## Implementation

### Handoff Data Structure

```go
type Handoff struct {
    Timestamp    time.Time  `json:"timestamp"`
    Summary      string     `json:"summary"`
    FilesChanged []string   `json:"files_changed"`
    NextSteps    []string   `json:"next_steps"`
    GitRef       string     `json:"git_ref"`
    Branch       string     `json:"branch"`
    Context      string     `json:"context"`  // Full blob for LLM
}
```

### Save Operation

```go
func (h *Handoff) Save() error {
    // 1. Capture current state
    h.Timestamp = time.Now()
    h.GitRef = getCurrentCommit()
    h.Branch = getCurrentBranch()
    h.FilesChanged = getRecentlyChangedFiles()

    // 2. Write to handoff storage
    handoffPath := filepath.Join(os.Getenv("HOME"), ".clood", "handoffs")
    os.MkdirAll(handoffPath, 0755)

    filename := fmt.Sprintf("%s.json", h.Timestamp.Format("2006-01-02T15-04-05"))
    data, _ := json.MarshalIndent(h, "", "  ")
    os.WriteFile(filepath.Join(handoffPath, filename), data, 0644)

    // 3. Update LAST_SESSION.md
    updateLastSession(h)

    // 4. Commit and push
    exec.Command("git", "add", "LAST_SESSION.md").Run()
    exec.Command("git", "commit", "-m",
        fmt.Sprintf("Session handoff: %s", h.Summary)).Run()
    exec.Command("git", "push").Run()

    return nil
}
```

### Load Operation

```go
func LoadLatestHandoff() (*Handoff, error) {
    // 1. Pull latest
    exec.Command("git", "pull").Run()

    // 2. Find most recent handoff
    handoffPath := filepath.Join(os.Getenv("HOME"), ".clood", "handoffs")
    files, _ := os.ReadDir(handoffPath)

    // Sort by name (timestamp) descending
    sort.Slice(files, func(i, j int) bool {
        return files[i].Name() > files[j].Name()
    })

    if len(files) == 0 {
        return nil, fmt.Errorf("no handoffs found")
    }

    // 3. Load and return
    data, _ := os.ReadFile(filepath.Join(handoffPath, files[0].Name()))
    var h Handoff
    json.Unmarshal(data, &h)

    return &h, nil
}
```

### LLM-Consumable Output

```go
func (h *Handoff) ToPrompt() string {
    return fmt.Sprintf(`## Session Context

**Last session:** %s
**Branch:** %s
**Commit:** %s

**What was done:**
%s

**Files changed:**
%s

**Next steps:**
%s

---
Ready to continue.
`,
        h.Timestamp.Format("2006-01-02 15:04"),
        h.Branch,
        h.GitRef[:8],
        h.Summary,
        formatFileList(h.FilesChanged),
        formatNextSteps(h.NextSteps),
    )
}
```

---

## Alternative: CLAUDE.md Instructions

If we don't want a CLI command, add to CLAUDE.md:

```markdown
## Session Handoff Protocol

### When user says "handoff", "context dump", or "wrap up":

1. Summarize what was accomplished this session
2. List files created/modified with brief descriptions
3. State clear, actionable next steps
4. Update LAST_SESSION.md with this information
5. Commit with message: "Session handoff: {brief summary}"
6. Push to remote

### When starting a new session:

1. Run `git fetch` to check for remote changes
2. If behind, run `git pull`
3. Read LAST_SESSION.md
4. Orient to previous context before taking action
5. Acknowledge: "Continuing from: {last session summary}"
```

---

## Alternative: Slash Command

Create `.claude/commands/handoff.md`:

```markdown
---
description: Save session context and push for continuity
---

Perform a session handoff:

1. Summarize what we accomplished in this session
2. List all files we created or modified
3. Identify clear next steps
4. Update LAST_SESSION.md with:
   - Session date/time
   - Summary of work done
   - Files changed
   - Next steps checklist
5. Commit with message: "Session handoff: $ARGUMENTS"
6. Push to remote

If no summary provided in arguments, ask what to include.
```

Usage:
```bash
/handoff "Completed canaries design, next: implement pre-flight checks"
```

---

## Alternative: Git Hook

`.git/hooks/pre-push`:

```bash
#!/bin/bash
# Auto-generate session summary before push

# Check if LAST_SESSION.md was modified
if git diff --cached --name-only | grep -q "LAST_SESSION.md"; then
    echo "LAST_SESSION.md already updated, skipping auto-summary"
    exit 0
fi

# Use local LLM to summarize recent commits
SUMMARY=$(git log --oneline -5 | clood ask "Summarize these commits in one sentence")

# Append to LAST_SESSION.md
echo -e "\n---\n## Auto-summary $(date)\n$SUMMARY" >> LAST_SESSION.md
git add LAST_SESSION.md
git commit --amend --no-edit
```

---

## Storage Locations

```
~/.clood/
├── config.yaml          # Main config
├── handoffs/            # Session handoff history
│   ├── 2025-12-16T14-30-00.json
│   ├── 2025-12-16T09-15-00.json
│   └── ...
└── cache/               # Other cached data
```

---

## Integration Points

### With Claude Code

Claude Code could:
1. Auto-detect when context is getting large
2. Suggest: "Context is 80% full. Run `/handoff` to save and continue fresh?"
3. On session start, auto-load latest handoff

### With clood CLI

```bash
# Full workflow
clood handoff --save "summary"    # Save + commit + push
clood handoff --load              # Pull + display context
clood handoff --history           # List recent handoffs
clood handoff --diff              # Show what changed since last handoff
```

### With Other Tools

```bash
# Pipe handoff context to other LLM tools
clood handoff --load --format=json | aider --context -
clood handoff --load --format=markdown | pbcopy
```

---

## Next Steps

1. [ ] Implement basic `clood handoff` command
2. [ ] Create slash command version for Claude Code
3. [ ] Add auto-detection of context limits
4. [ ] Build handoff history viewer
5. [ ] Test cross-machine handoffs (mbp → ubuntu25 → mac-mini)
