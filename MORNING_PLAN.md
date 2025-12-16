# Morning Test Plan & Workflow Design

## Part 1: Testing clood (30 min)

### Phase 1A: Basic Verification (5 min)

```bash
cd ~/Code/clood/clood-cli

# Verify binary
./clood --version
./clood --help
./clood
```

**Expected**: Banner displays, version 0.2.0, command list shows

### Phase 1B: Hardware Detection (5 min)

```bash
./clood system
./clood system --json
```

**Expected**: Shows your Mac's specs, GPU info, recommended models

### Phase 1C: Host Discovery (5 min)

```bash
./clood hosts
./clood hosts --json
```

**Expected**: Shows localhost and ubuntu25 status, latency, model counts

### Phase 1D: Model Inventory (5 min)

```bash
./clood models
./clood models --host ubuntu25
```

**Expected**: Lists all models across hosts, shows tier configuration

### Phase 1E: Query Routing (5 min)

```bash
# Dry run - see routing without executing
./clood ask "what is a goroutine" --show-route
./clood ask "refactor this to use channels" --show-route
./clood ask "implement a binary search tree" --show-route
```

**Expected**: First → Tier 1 (Fast), Second/Third → Tier 2 (Deep)

### Phase 1F: Live Queries (5 min)

```bash
# Actual execution
./clood ask "what is a pointer in Go"
./clood ask "write a haiku about programming"

# Force specific tier/host
./clood ask "explain goroutines" --tier 1 --host ubuntu25
./clood ask "explain goroutines" --tier 2
```

**Expected**: Streaming response from Ollama

---

## Part 2: Hybrid Workflow Design

### The Vision

```
┌─────────────────────────────────────────────────────────────────┐
│                      HUMAN (You)                                │
│                    Provides direction                           │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                   CLAUDE (Architect)                            │
│  - Reviews specs and plans                                      │
│  - Provides feedback on PRs                                     │
│  - Answers @mentions in GitHub                                  │
│  - Guides local LLMs when stuck                                 │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                   clood (Orchestrator)                          │
│  - Routes tasks to appropriate tier/host                        │
│  - Manages which model handles what                             │
│  - Tracks hardware utilization                                  │
└─────────────────────┬───────────────────────────────────────────┘
                      │
          ┌───────────┴───────────┐
          ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│  Local LLMs     │     │  Local LLMs     │
│  (Tier 1: Fast) │     │  (Tier 2: Deep) │
│  Quick tasks    │     │  Implementation │
│  Explanations   │     │  Refactoring    │
│  Lookups        │     │  Debugging      │
└─────────────────┘     └─────────────────┘
```

### Workflow Options

#### Option A: GitHub PR Review Loop

```
1. Local LLM creates code via aider
2. aider commits to feature branch
3. aider opens PR
4. GitHub Action triggers Claude review
5. Claude comments on PR
6. Local LLM reads comments, iterates
7. Repeat until Claude approves
```

#### Option B: Issue-Driven Development

```
1. Create GitHub Issue with spec
2. Claude comments with implementation plan
3. Local LLM picks up issue, implements
4. Local LLM @mentions Claude for review
5. Claude responds in issue thread
6. Local LLM creates PR when ready
```

#### Option C: Slack/Discord Integration

```
1. Human posts task in channel
2. Claude responds with plan
3. Local LLM implements
4. Local LLM posts results to channel
5. Claude reviews in thread
```

---

## Part 3: GitHub Integration Design

### Using GitHub Actions for Claude Review

Create `.github/workflows/claude-review.yml`:

```yaml
name: Claude PR Review

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Get PR diff
        id: diff
        run: |
          gh pr diff ${{ github.event.pull_request.number }} > diff.txt

      - name: Claude Review
        uses: anthropics/claude-code-action@v1  # hypothetical
        with:
          prompt: |
            Review this PR for the Chimborazo project.
            Check for: correctness, patterns from PATTERNS.md, edge cases.

            Diff:
            $(cat diff.txt)

      - name: Post Review Comment
        run: |
          gh pr comment ${{ github.event.pull_request.number }} \
            --body "${{ steps.claude.outputs.response }}"
```

### @mention Webhook Pattern

Create a GitHub App or Action that:
1. Watches for `@claude` mentions in issues/PRs
2. Extracts context (issue body, PR diff, recent comments)
3. Sends to Claude API
4. Posts response as comment

```python
# Pseudo-code for webhook handler
@app.route('/github/webhook', methods=['POST'])
def handle_webhook():
    event = request.json

    if '@claude' in event['comment']['body']:
        context = gather_context(event)
        response = claude_api.complete(context)
        github_api.post_comment(event['issue'], response)
```

---

## Part 4: Practical Morning Workflow

### Step 1: Test clood with Chimborazo

```bash
cd ~/Code/strata

# Use clood to query about the codebase
~/Code/clood/clood-cli/clood ask "what does the maury module do" --no-context
~/Code/clood/clood-cli/clood ask "how would I add a new data source"
```

### Step 2: Create a Test Task

Pick a small task from CHIMBORAZO-README.md:
- Recipe parsing
- HTTP fetcher
- File cache

### Step 3: Generate Implementation Prompt

Use Claude to create a detailed prompt for the local LLM:

```bash
# In this Claude Code session:
# "Create a prompt artifact for implementing the HTTP fetcher in Chimborazo"
```

### Step 4: Run Local LLM via clood

```bash
# Save prompt to file
cat > /tmp/task.txt << 'EOF'
[Detailed prompt from Claude]
EOF

# Execute via clood
~/Code/clood/clood-cli/clood ask "$(cat /tmp/task.txt)" --tier 2 --no-stream > output.go
```

### Step 5: Review with Claude

```bash
# Back in Claude Code session:
# "Review this implementation: [paste output.go]"
```

### Step 6: Iterate

Based on Claude's feedback, refine the prompt and re-run.

---

## Part 5: Management & Tracking

### Option 1: GitHub Projects Board

```
┌─────────────┬─────────────┬─────────────┬─────────────┐
│   Backlog   │  In Design  │ Local LLM   │   Review    │
│             │  (Claude)   │  Working    │  (Claude)   │
├─────────────┼─────────────┼─────────────┼─────────────┤
│ Task A      │ Task B      │ Task C      │ Task D      │
│ Task E      │             │             │             │
└─────────────┴─────────────┴─────────────┴─────────────┘
```

### Option 2: Local Tracking with clood

Add a `clood tasks` command that:
- Reads from a local TODO.md or tasks.yaml
- Tracks which tasks are assigned to local LLM
- Shows status of each task

### Option 3: Git Branch Convention

```
main
├── feature/task-a-local-llm    # Local LLM working
├── feature/task-b-claude-review # Waiting for Claude
├── feature/task-c-human-review  # Waiting for human
└── feature/task-d-approved      # Ready to merge
```

---

## Part 6: Tomorrow's Experiment

### Goal: Have local LLM implement one Chimborazo feature with Claude oversight

1. **Morning (You + Claude)**: Pick task, create detailed spec
2. **Midday (Local LLM via clood)**: Implement based on spec
3. **Afternoon (Claude)**: Review implementation
4. **Evening (Local LLM)**: Address feedback
5. **Night (You)**: Final review and merge

### Metrics to Track

- Time for local LLM to produce first draft
- Number of review cycles needed
- Quality of final output vs Claude-only implementation
- Token costs (local = free, Claude = $$)

---

## Part 7: GitHub @mention Integration (Future)

### Simple Version (No infrastructure needed)

1. Local LLM creates PR
2. In PR description, write: "cc @your-username for Claude review"
3. You see notification, open Claude Code
4. Ask Claude to review the PR
5. Post Claude's feedback as PR comment manually

### Automated Version (Requires setup)

1. Create GitHub App with webhook
2. Host webhook handler (could be on ubuntu25)
3. Handler calls Claude API on @claude mentions
4. Posts responses automatically

### Using Existing Tools

- **GitHub Copilot** has PR review, but not Claude
- **CodeRabbit** does AI PR review
- **Claude API** can be called from GitHub Actions

---

## Questions to Decide

1. **How much automation do you want?**
   - Manual: Copy/paste between Claude and local LLM
   - Semi-auto: Scripts that format prompts
   - Full-auto: GitHub webhooks, no human in loop

2. **Where should Claude feedback live?**
   - GitHub PR comments
   - Local files (FEEDBACK.md)
   - This conversation

3. **What's the handoff format?**
   - Prompts in markdown files
   - YAML task definitions
   - Issue templates

---

## Recommended Starting Point

**Start simple, add automation later:**

1. Test clood basic commands (Part 1)
2. Pick one small Chimborazo task
3. Ask Claude (me) to write the implementation prompt
4. Run prompt through clood → local LLM
5. Paste output back here for review
6. Iterate manually

Once the workflow is proven, we can add:
- GitHub Actions for auto-review
- @mention webhooks
- Task tracking

---

*Ready to test when you wake up. Let's see how the local LLMs perform.*
