# The Claude Vacuum Protocol

**A Guide for the Bird, Cats, and Tortoise to Continue Without Chef Claude**

> "Software development is a dish best served iterative" - Ancient Dev Proverb

This document contains everything needed to execute development workflows using clood tools and local LLMs when Claude is unavailable. The goal: prove the methodology works, not just the AI.

---

## Prerequisites Checklist

Before starting any work session, verify:

```bash
# 1. Is clood installed and working?
clood --version

# 2. Is Ollama running locally?
curl -s localhost:11434/api/tags | head -5

# 3. What models are available?
clood models

# 4. Full system check
clood preflight
```

**Minimum Requirements:**
- [ ] clood CLI installed (`~/Code/clood/clood-cli/clood`)
- [ ] Ollama running on at least one host
- [ ] At least one coding model: `qwen2.5-coder:7b` or `deepseek-coder-v2:16b`
- [ ] At least one reasoning model: `deepseek-r1:8b` or `deepseek-r1:14b`

---

## The Discovery Workflow

### Step 1: Orient Yourself

Before touching any code, understand what you're looking at:

```bash
# See project structure
clood tree ~/Code/chimborazo

# Find the entry point
clood grep "func main" ~/Code/chimborazo

# Understand what files exist
clood tree ~/Code/chimborazo --depth 3
```

### Step 2: Find Related Code

When working on a feature, find all related code first:

```bash
# Find where a concept is implemented
clood grep "Recipe" ~/Code/chimborazo

# Find all files that mention a term
clood grep "GeoJSON" ~/Code/chimborazo --files_only

# Get function signatures
clood symbols ~/Code/chimborazo/pkg/pipeline/builder.go
```

### Step 3: Build Context for the LLM

Before asking the local LLM anything, gather context:

```bash
# Get a summary of a directory
clood context ~/Code/chimborazo/pkg/pipeline

# Read specific files you need to understand
cat ~/Code/chimborazo/pkg/pipeline/builder.go

# Check what a module imports
clood imports ~/Code/chimborazo/pkg/pipeline/builder.go
```

---

## Working with Local LLMs

### Check What's Available

```bash
# Which hosts are online?
clood hosts

# What models on each host?
clood models

# Recommended: Check system resources
clood system
```

### Model Selection Guide

| Task Type | Recommended Model | Why |
|-----------|-------------------|-----|
| Code generation | `qwen2.5-coder:7b` | Fast, accurate for Go/Python |
| Code review | `deepseek-coder-v2:16b` | Better at finding issues |
| Architecture decisions | `deepseek-r1:14b` | Reasoning through tradeoffs |
| Quick questions | `qwen2.5-coder:3b` | Fast for simple lookups |
| Image analysis | `llava:7b` or `moondream` | When dealing with diagrams |

### Asking Questions

```bash
# Simple code question
clood ask "How do I parse YAML in Go?"

# With context from a file
clood ask "Explain what this function does" --context ~/Code/chimborazo/pkg/pipeline/builder.go

# Specify a model
clood ask "Review this code for bugs" --model deepseek-coder-v2:16b --context ./myfile.go

# Use a remote host if local is slow
clood ask "Complex question" --host mac-mini
```

### The Cross-Evaluation Pattern (Catfight Protocol)

When you need confidence in an answer, ask multiple models:

```bash
# Get answer from fast model
clood ask "How should I implement X?" --model qwen2.5-coder:7b > answer1.txt

# Get answer from reasoning model
clood ask "How should I implement X?" --model deepseek-r1:14b > answer2.txt

# Compare (manually or with another query)
clood ask "Compare these two approaches and pick the better one" \
  --context answer1.txt --context answer2.txt
```

---

## A Simple Test Task

**Goal:** Add a new geometry operation to Chimborazo

This task validates the workflow. It's small enough to complete but touches enough code to prove the process.

### The Task: Implement `translate` Operation

Add an operation that shifts all geometry by X/Y offset.

### Step 1: Understand Current Operations

```bash
# Find where operations are defined
clood grep "applyOperation" ~/Code/chimborazo

# Read the implementation
cat ~/Code/chimborazo/pkg/pipeline/builder.go | grep -A 50 "applyOperation"

# See what operations exist
clood grep "case \"" ~/Code/chimborazo/pkg/pipeline/builder.go
```

### Step 2: Find the Geometry Package

```bash
# Where are geometry operations implemented?
clood tree ~/Code/chimborazo/internal/geometry

# What functions exist?
clood symbols ~/Code/chimborazo/internal/geometry/
```

### Step 3: Ask the LLM for Help

```bash
# Get the current operation patterns as context
clood context ~/Code/chimborazo/pkg/pipeline > context.txt

# Ask how to add a new operation
clood ask "Based on this code, how would I add a 'translate' operation that shifts all points by an X,Y offset?" --context context.txt
```

### Step 4: Implement

1. Add `TranslateCollection` function to `internal/geometry/operations.go`
2. Add `case "translate":` to `applyOperation` in `builder.go`
3. Test with a recipe that uses `translate`

### Step 5: Validate

```bash
# Build the project
cd ~/Code/chimborazo && go build ./...

# Run tests
go test ./...

# Try a build with your new operation
./chimborazo validate recipes/test_translate.yaml
./chimborazo build recipes/test_translate.yaml
```

---

## Issue Documentation Workflow

### Creating Issues Without Claude

When you find work to do:

```bash
# Create a simple issue
gh issue create --repo dirtybirdnj/chimborazo \
  --title "feat: Implement translate geometry operation" \
  --body "$(cat <<'EOF'
## Summary
Add a translate operation that shifts geometry by X,Y offset.

## Acceptance Criteria
- [ ] `TranslateCollection` function in geometry package
- [ ] Wired into builder's `applyOperation`
- [ ] Works with Point, LineString, Polygon types
- [ ] Test recipe validates correctly

## Implementation Notes
See existing operations (clip, simplify, merge) for patterns.
EOF
)"

# Add labels
gh issue edit <NUMBER> --add-label "enhancement"
```

### Closing Issues

```bash
# Close with a note
gh issue close <NUMBER> --comment "Implemented in commit abc123"
```

---

## Troubleshooting

### "clood ask" is slow

```bash
# Check if local Ollama is overloaded
clood health

# Try a remote host
clood ask "question" --host mac-mini

# Use a smaller model
clood ask "question" --model qwen2.5-coder:3b
```

### Model gives bad answers

1. Provide more context: `--context file1.go --context file2.go`
2. Be more specific in your question
3. Try the catfight pattern with multiple models
4. Break the question into smaller parts

### Can't find code

```bash
# Use broader grep
clood grep "keyword" --ignore-case

# Check tree for directory structure
clood tree . --depth 4

# Look for related terms
clood grep "similar_concept"
```

### Build fails

```bash
# Check Go environment
go env

# Verify dependencies
go mod tidy

# Look for the actual error
go build ./... 2>&1 | head -20
```

---

## The Methodology Summary

1. **Orient First** - Use `clood tree` and `clood grep` before writing code
2. **Gather Context** - Use `clood context` and `clood symbols` before asking LLMs
3. **Ask Specifically** - Give models concrete questions with relevant context
4. **Verify Locally** - Build and test before considering work done
5. **Document Progress** - Create issues, close them with notes
6. **Iterate** - Each attempt teaches something, even failures

---

## What You Lose Without Claude

Be honest about limitations:

- **Massive context windows** - Local models have 8K-32K context vs Claude's 200K
- **Complex multi-file refactors** - Break these into smaller steps
- **Sophisticated code review** - Use catfight pattern to compensate
- **Auto-exploration** - You must be more explicit about what to search

## What You Keep

- **Full codebase search** - clood grep is instant and unlimited
- **Local inference** - No rate limits, no costs, works offline
- **Git operations** - Same as always
- **Build/test cycles** - Unchanged
- **The methodology** - The patterns work regardless of which LLM runs them

---

## Emergency Contacts

If you're truly stuck:

1. Check existing documentation in `/docs/`
2. Search closed issues: `gh issue list --state closed`
3. Look at commit history: `git log --oneline -20`
4. Read test files for usage examples
5. The strata codebase has extensive narrative docs in `DEEP_DIVE_*.md`

---

*"Every line of code written, every bug encountered, every solution found - it all adds to the map. The summit isn't the only victory. The climb is the victory."*

- Chef Claude, signing off for now
