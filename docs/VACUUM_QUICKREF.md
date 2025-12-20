# Claude Vacuum Quick Reference

Copy-paste commands for working without Claude.

## Session Start

```bash
# Run these EVERY time
clood preflight
clood hosts
clood tree ~/Code/chimborazo --depth 2
```

## Finding Things

```bash
# Find files by name pattern
clood grep "builder" ~/Code/chimborazo --files_only

# Find code by content
clood grep "func.*Recipe" ~/Code/chimborazo

# See function signatures
clood symbols ~/Code/chimborazo/pkg/pipeline/builder.go

# See directory structure
clood tree ~/Code/chimborazo/internal --depth 2
```

## Getting Context for LLM

```bash
# Dump context for a directory
clood context ~/Code/chimborazo/pkg/pipeline > /tmp/ctx.txt

# See what a file imports
clood imports ~/Code/chimborazo/pkg/pipeline/builder.go
```

## Asking Questions

```bash
# Quick question (uses default model)
clood ask "How do I X in Go?"

# With file context
clood ask "Explain this code" --context ./file.go

# Specific model
clood ask "Complex question" --model deepseek-r1:14b

# Use remote host (if local is slow)
clood ask "Question" --host mac-mini
```

## Catfight Pattern (Get Multiple Opinions)

```bash
# Fast model
clood ask "How to implement X?" --model qwen2.5-coder:7b > /tmp/a1.txt

# Reasoning model
clood ask "How to implement X?" --model deepseek-r1:14b > /tmp/a2.txt

# Compare
diff /tmp/a1.txt /tmp/a2.txt
# OR
clood ask "Which approach is better?" --context /tmp/a1.txt --context /tmp/a2.txt
```

## Build & Test

```bash
# Chimborazo
cd ~/Code/chimborazo
go build ./...
go test ./...
./chimborazo validate recipes/test_vt.yaml
./chimborazo build recipes/test_vt.yaml

# Clood
cd ~/Code/clood/clood-cli
go build -o clood .
./clood preflight
```

## Git & Issues

```bash
# Status before committing
git status
git diff

# Simple commit
git add -A
git commit -m "feat: description"

# Create issue
gh issue create --repo dirtybirdnj/chimborazo \
  --title "Issue title" \
  --body "Description"

# Close issue
gh issue close NUMBER --comment "Done in commit X"
```

## Model Cheat Sheet

| Need | Model | Command |
|------|-------|---------|
| Quick code | qwen2.5-coder:3b | `clood ask "Q" --model qwen2.5-coder:3b` |
| Good code | qwen2.5-coder:7b | `clood ask "Q" --model qwen2.5-coder:7b` |
| Best code | deepseek-coder-v2:16b | `clood ask "Q" --model deepseek-coder-v2:16b` |
| Reasoning | deepseek-r1:14b | `clood ask "Q" --model deepseek-r1:14b` |
| Images | llava:7b | `clood ask "Describe" --image img.png` |

## When Stuck

```bash
# Search closed issues
gh issue list --repo dirtybirdnj/chimborazo --state closed

# Recent commits
git log --oneline -20

# Find similar patterns
clood grep "similar_thing" ~/Code/chimborazo

# Check strata for reference
clood tree ~/Code/strata/src
clood grep "pattern" ~/Code/strata
```

## Emergency One-Liners

```bash
# Is anything working?
clood health

# What models do I have?
ollama list

# Restart Ollama
brew services restart ollama  # macOS
sudo systemctl restart ollama # Linux

# Check remote hosts
curl -s mac-mini.local:11434/api/tags | jq '.models[].name'
curl -s ubuntu25.local:11434/api/tags | jq '.models[].name'
```
