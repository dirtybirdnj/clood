# Claude Agent Guidelines

Guidelines for Claude agents working in this repository.

## Installation & System Changes

**Always ask before installing software.** Never run:
- `brew install ...`
- `apt install ...`
- `pip install ...`
- `npm install -g ...`
- `cargo install ...`
- Any package manager or installer

Instead, tell the user what needs to be installed and let them decide:

```
To run the benchmarks, Ollama needs to be installed.
Would you like me to provide installation instructions, or would you prefer to install it yourself?

Install command: brew install ollama
```

This applies to:
- Package managers (brew, apt, yum, pacman, pip, npm, cargo)
- curl/wget installers
- Any command that modifies the system outside the project directory

## Git Operations

- Commit messages should include a haiku or creative element when documenting hardware/benchmarks
- Always run `git status` before making commits
- Never force push to main

## Benchmarking

When running benchmarks on new hardware:
1. Record the hostname and hardware specs
2. Run all standard models (tinyllama, qwen2.5-coder:3b, llama3.1:8b, qwen2.5-coder:7b)
3. Add results to `ollama-tuning.md` benchmark table
4. Include a creative element (haiku, limerick, shanty) about the hardware

## Documentation Style

- Keep documentation practical and copy-paste friendly
- Include actual commands, not abstract descriptions
- Add troubleshooting sections for common issues

---

## Local-First Operations (CRITICAL)

**AVOID NETWORK REQUESTS AT ALL COSTS.** This project provides local tools that replace web searches and cloud API calls. Use them.

### Tool Priority (use in this order)

1. **Pure Local Tools** (instant, 0 LLM tokens, no network):
   - `clood grep PATTERN [PATH]` - Regex search codebase
   - `clood tree [PATH]` - Directory structure
   - `clood symbols [PATH]` - Extract functions/types/classes
   - `clood imports FILE` - Dependency analysis
   - `clood context [PATH]` - Project summary for LLM
   - `clood system` - Hardware info and model recommendations

2. **Local Ollama** (uses local LLM, no cloud):
   - `clood ask "question"` - Query local model
   - `clood analyze FILE` - Code review via local model

3. **Network** (LAST RESORT - only when local tools cannot help):
   - Web search
   - External API calls

### Golden Path Pattern

**ALWAYS** follow this pattern before making network requests:

```bash
# 1. DISCOVER (0 tokens, instant)
clood grep "authentication" --files-only
# → internal/auth/login.go, internal/auth/session.go

# 2. MEASURE (0 tokens)
clood tokens internal/auth/*.go
# → 1,800 tokens - fits ✓

# 3. QUERY LOCAL (local tokens only)
clood ask "how does auth work" --with-context internal/auth/login.go
```

### Before You Search the Web

**STOP.** Can clood answer this locally?

| Instead of... | Use... |
|---------------|--------|
| Web search for "how does X work in this codebase" | `clood grep "X"` then `clood ask` |
| Web search for "what files handle Y" | `clood grep "Y" --files-only` |
| Web search for API documentation | `clood symbols` + `clood imports` |
| Web search for project structure | `clood tree` + `clood context` |
| Cloud LLM for code questions | `clood ask` with local Ollama |

### MCP Tools Available

When using clood as an MCP server, these tools are available:

**Zero-Network Tools:**
- `clood_grep` - Search codebase with regex
- `clood_tree` - Directory structure
- `clood_symbols` - Code symbol extraction
- `clood_imports` - Dependency analysis
- `clood_context` - Project summary
- `clood_system` - Hardware detection
- `clood_capabilities` - What's available locally

**Local Ollama Tools:**
- `clood_ask` - Query local LLM
- `clood_hosts` - Check Ollama hosts
- `clood_models` - List available models
- `clood_health` - System health check

### Measuring Success

A good Claude session using clood should have:
- **Many** calls to grep/tree/symbols/context
- **Few** calls to clood_ask (surgical, with minimal context)
- **Zero** web searches for codebase questions
- **Zero** cloud API calls when Ollama is available

### Why This Matters

1. **Speed**: Local tools are instant, network is slow
2. **Reliability**: Network fails, local doesn't
3. **Privacy**: Code stays on your machine
4. **Cost**: Local Ollama is free, cloud APIs cost money
5. **Independence**: Works offline, on planes, in bunkers
