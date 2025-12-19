# Clood CLI Tools Reference

For agents and power users. No vision, just facts.

> **See also:** [CLOOD_EXPERIENCE.md](./CLOOD_EXPERIENCE.md) for UX patterns and the Saga flow.

---

## MCP Tools (for AI Agents)

When clood runs as an MCP server (`clood serve --sse`), these tools are exposed to AI agents like Claude Code:

### Zero-Network Discovery Tools (USE FIRST!)

| Tool | Description | Network | LLM Tokens |
|------|-------------|---------|------------|
| `clood_grep` | Search codebase with regex | 0 | 0 |
| `clood_tree` | Directory structure | 0 | 0 |
| `clood_symbols` | Extract functions/types/classes | 0 | 0 |
| `clood_imports` | Dependency analysis | 0 | 0 |
| `clood_context` | Project summary | 0 | 0 |
| `clood_capabilities` | List available tools | 0 | 0 |
| `clood_system` | Hardware detection | 0 | 0 |

### Local Ollama Tools (Requires Local LLM)

| Tool | Description | Network | LLM Tokens |
|------|-------------|---------|------------|
| `clood_ask` | Query local LLM | Local only | Yes |
| `clood_hosts` | Check Ollama hosts | Local only | 0 |
| `clood_models` | List available models | Local only | 0 |
| `clood_health` | System health check | Local only | 0 |

### MCP Tool Usage Examples

```json
// Search for authentication code
{"tool": "clood_grep", "args": {"pattern": "auth", "files_only": true}}

// Get directory structure
{"tool": "clood_tree", "args": {"path": "internal/", "depth": 2}}

// Extract all functions
{"tool": "clood_symbols", "args": {"path": ".", "kind": "func"}}

// Check what's available locally
{"tool": "clood_capabilities", "args": {}}
```

---

## Discovery Tools (0 LLM tokens)

These tools scan the codebase without calling any LLM. Use them for surgical context discovery.

### clood grep PATTERN [PATH]
Search codebase with regex.
```bash
clood grep "func.*Router" internal/
clood grep "TODO|FIXME" --files-only
clood grep "error" -C 3              # 3 lines context
clood grep "Auth" --type go          # Go files only
```

### clood symbols PATH
Extract functions, types, variables from code.
```bash
clood symbols internal/router/router.go
# → L139 [func] ClassifyQuery
# → L286 [type] RouteResult
# → L301 [func] NewRouter
```

### clood imports FILE
Analyze Go imports and dependencies.
```bash
clood imports internal/commands/ask.go
# → internal: config, router, tui
# → external: github.com/spf13/cobra
# → stdlib: fmt, os, strings

clood imports internal/router/ --reverse  # Who imports this?
```

### clood tree [PATH]
Directory tree respecting .gitignore.
```bash
clood tree internal/
clood tree --depth 2
```

### clood context
Generate project structure overview.
```bash
clood context
# → Project type, file counts, directory structure
```

### clood summary [PATH]
Project structure detection as JSON.
```bash
clood summary --json
```

---

## Measurement Tools (0 LLM tokens)

Measure context size before making LLM calls.

### clood tokens FILE
Count estimated tokens in files.
```bash
clood tokens internal/router/router.go
# → 1,247 tokens (estimated)

clood tokens internal/commands/*.go
# → 8,400 tokens total (14 files)

clood tokens --check internal/router/
# → 2,400 tokens - fits in 16K context ✓
```

### clood saga --stats
Show current saga context usage.
```bash
clood saga --stats
# → History: 6,200 tokens (47 messages)
# → Project: 3,200 tokens
# → Available: 6,600 tokens (41%)
```

---

## Query Tools (uses LLM)

These make LLM API calls. Use discovery tools first to minimize tokens.

### clood ask "question"
One-shot query with auto-routing.
```bash
clood ask "what does this function do"
clood ask "explain routing" --with-context internal/router/router.go
clood ask "review this" --tier 3        # Force analysis tier
clood ask "write docs" --host ubuntu25  # Force specific host
```

**Flags:**
- `--with-context FILE` - Load specific file(s) as context
- `--tier N` - Force tier (1=fast, 2=deep, 3=analysis, 4=writing)
- `--host NAME` - Force specific host
- `--model NAME` - Force specific model
- `--no-stream` - Wait for complete response

### clood analyze FILE
Code review using analysis tier.
```bash
clood analyze internal/router/router.go
clood analyze internal/commands/ --focus security
cat file.go | clood analyze --stdin
git diff | clood analyze --stdin --focus "review changes"
```

**Focus areas:** security, performance, bugs, style

---

## Infrastructure Tools

### clood hosts
List configured hosts and status.
```bash
clood hosts
# → ● ubuntu25  ONLINE  73ms  8 models
# → ● mac-mini  ONLINE  12ms  2 models
# → ○ localhost OFFLINE
```

### clood models
List available models across all hosts.
```bash
clood models
clood models --host ubuntu25
```

### clood bench MODEL
Benchmark model performance.
```bash
clood bench llama3.1:8b --host ubuntu25
# → gen: 34 tok/s, prompt: 97 tok/s
```

### clood garden
Display Server Garden architecture diagram.
```bash
clood garden
# → ASCII diagram of hosts, models, connections
```

### clood tiers
Show tier routing configuration.
```bash
clood tiers
# → Which model/host handles each tier
```

### clood health
Full system health check.
```bash
clood health
# → Hosts, models, connectivity, versions
```

### clood tune
Ollama performance recommendations.
```bash
clood tune
# → Environment variables, modelfile suggestions
```

---

## Surgical Context Pattern

The recommended pattern for agents:

```bash
# 1. Discover (0 tokens)
clood grep "authentication" --files-only
# → internal/auth/login.go, internal/auth/session.go

# 2. Measure (0 tokens)
clood tokens internal/auth/*.go --check
# → 1,800 tokens - fits ✓

# 3. Query (minimal tokens)
clood ask "how does auth work" --with-context internal/auth/login.go,internal/auth/session.go
```

**NOT this:**
```bash
# BAD: Loading everything
clood ask "how does auth work" --with-context .
# → 50,000 tokens, slow, expensive
```

---

## Model Tiers

| Tier | Name | Default Model | Use Case |
|------|------|---------------|----------|
| 1 | Fast | qwen2.5-coder:3b | Quick questions, simple tasks |
| 2 | Deep | qwen2.5-coder:7b | Complex reasoning, code generation |
| 3 | Analysis | llama3.1:8b | Code review, bug finding |
| 4 | Writing | llama3.1:8b | Documentation, prose, narratives |

---

## Server Garden (Current)

| Host | Hardware | Models | Speed | Best For |
|------|----------|--------|-------|----------|
| ubuntu25 | GPU | 8 | 35 tok/s | Deep, Analysis, Writing |
| mac-mini | M4 16GB | 2 | TBD | Fast tier |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | No hosts available |
| 3 | Model not found |
| 4 | Context overflow |
