# Claude Agent Guidelines

Guidelines for Claude agents working in this repository.

---

## LOCAL-FIRST OPERATIONS (CRITICAL - READ THIS FIRST)

**AVOID NETWORK REQUESTS AT ALL COSTS.** This project provides local tools that replace web searches and cloud API calls. Use them.

### MANDATORY Pre-Task Ritual

Before EVERY coding task, run these commands (in order):

```bash
# 1. PREFLIGHT - See what's available locally
clood preflight              # Or via MCP: clood_preflight

# 2. ORIENT - Understand the codebase
clood tree                   # Project structure
clood grep "relevant_term"   # Find related code
clood symbols path/          # API surface
```

**Do NOT skip this ritual.** Flying blind wastes time and causes network requests.

---

### Decision Tree: Where Should I Look?

```
┌─────────────────────────────────────────────────────────────┐
│ "I need to know about..."                                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ Is it about THIS CODEBASE?                                  │
│ (files, functions, how things work here, project structure) │
└─────────────────────────────────────────────────────────────┘
          │ YES                              │ NO
          ▼                                  ▼
┌─────────────────────────┐    ┌─────────────────────────────┐
│ USE LOCAL TOOLS:        │    │ Is it about CODE PATTERNS,  │
│ • clood_grep            │    │ BEST PRACTICES, or HOW TO   │
│ • clood_tree            │    │ USE A LIBRARY/FRAMEWORK?    │
│ • clood_symbols         │    └─────────────────────────────┘
│ • clood_imports         │              │ YES        │ NO
│ • clood_context         │              ▼            ▼
│                         │    ┌────────────────┐ ┌─────────────┐
│ ⛔ NEVER WEB SEARCH     │    │ Use clood_ask  │ │ Web search  │
│    FOR THIS CATEGORY    │    │ (local Ollama) │ │ APPROVED    │
└─────────────────────────┘    │                │ │             │
                               │ Check hosts    │ │ Examples:   │
                               │ first!         │ │ • News      │
                               └────────────────┘ │ • External  │
                                                  │   API docs  │
                                                  │ • Current   │
                                                  │   events    │
                                                  └─────────────┘
```

---

### Hard Rules (NEVER Violate)

#### ⛔ FORBIDDEN Web Searches

These queries MUST use local tools instead:

| If you're tempted to search for... | USE THIS INSTEAD |
|------------------------------------|------------------|
| "where is X in this codebase" | `clood_grep "X" --files_only` |
| "what files contain Y" | `clood_grep "Y"` |
| "how does Z work in this project" | `clood_grep "Z"` then `clood_context` |
| "project structure" | `clood_tree` |
| "what functions are in file.go" | `clood_symbols file.go` |
| "what does this file import" | `clood_imports file.go` |
| "function signature for Foo" | `clood_symbols --kind func` then grep |
| "how to use internal API" | `clood_symbols` + `clood_grep` |

#### ✅ REQUIRED Actions

- **ALWAYS** `clood_preflight` at session start
- **ALWAYS** `clood_grep` before asking "where is..."
- **ALWAYS** `clood_tree` before exploring directories
- **ALWAYS** `clood_symbols` before asking about function signatures
- **ALWAYS** `clood_hosts` before any `clood_ask` call
- **ALWAYS** `clood_should_search_web` before WebSearch (it will redirect you to local tools if possible)

---

### Tool Priority (Use in This Order)

#### Tier 1: Pure Local Tools (instant, 0 tokens, no network)

Use these FIRST. They are instant and free:

| Tool | Purpose | Replaces |
|------|---------|----------|
| `clood_preflight` | Start here. Shows what's available locally | - |
| `clood_grep PATTERN` | Regex search codebase | WebSearch for "where is X" |
| `clood_tree [PATH]` | Directory structure | WebSearch for "project structure" |
| `clood_symbols PATH` | Extract functions/types/classes | WebSearch for "what functions..." |
| `clood_imports FILE` | Dependency analysis | WebSearch for "what does X import" |
| `clood_context [PATH]` | Project summary for LLM | Manual file reading |
| `clood_system` | Hardware info and model recommendations | - |
| `clood_should_search_web` | Gate: checks if web search is needed | - |

#### Tier 2: Local Ollama (uses local LLM, no cloud)

Use when Tier 1 doesn't answer the question:

| Tool | Purpose |
|------|---------|
| `clood_ask "question"` | Query local model (check `clood_hosts` first!) |
| `clood_hosts` | Check which Ollama hosts are online |
| `clood_models` | List available models |
| `clood_health` | Full system health check |

#### Tier 3: Network (LAST RESORT)

Only use when Tier 1 and Tier 2 cannot help:
- WebSearch (for external documentation, current events, things not in codebase)
- External API calls

---

### Before You Search the Web

**STOP.** Run this mental checklist:

1. Is this about THIS codebase? → **Use `clood_grep`/`clood_tree`/`clood_symbols`**
2. Is this about code patterns or best practices? → **Use `clood_ask` with local Ollama**
3. Is Ollama even online? → **Check `clood_hosts` first**
4. Is this about external/current information? → **Web search is OK**

Better yet, call `clood_should_search_web` with your query - it will tell you.

---

### Measuring Success

A good Claude session using clood should have:
- **Many** calls to grep/tree/symbols/context (10+ is normal)
- **Few** calls to clood_ask (surgical, with minimal context)
- **Zero** web searches for codebase questions
- **Zero** cloud API calls when Ollama is available

#### Self-Assessment (Run Before Session Ends)

Count your tool usage:
- [ ] Local discovery tools (grep/tree/symbols): ___
- [ ] Web searches that could have been local: ___
- [ ] clood_ask calls: ___
- [ ] Did I preflight at session start? Y/N

**Target:** Local tools >> Web searches. Zero avoidable network calls.

---

### Why This Matters

1. **Speed**: Local tools are instant, network is slow
2. **Reliability**: Network fails, local doesn't
3. **Privacy**: Code stays on your machine
4. **Cost**: Local Ollama is free, cloud APIs cost money
5. **Independence**: Works offline, on planes, in bunkers

---

## MCP Tools Reference

When using clood as an MCP server, these tools are available:

### Zero-Network Tools (Use First!)
- `clood_preflight` - **START HERE** - Shows local capabilities
- `clood_grep` - Search codebase with regex
- `clood_tree` - Directory structure
- `clood_symbols` - Code symbol extraction
- `clood_imports` - Dependency analysis
- `clood_context` - Project summary
- `clood_system` - Hardware detection
- `clood_should_search_web` - Gate before web searches
- `clood_capabilities` - What's available locally

### Local Ollama Tools
- `clood_ask` - Query local LLM
- `clood_hosts` - Check Ollama hosts
- `clood_models` - List available models
- `clood_health` - System health check

---

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

---

## Git Operations

- Commit messages should include a haiku or creative element when documenting hardware/benchmarks
- Always run `git status` before making commits
- Never force push to main

---

## Benchmarking

When running benchmarks on new hardware:
1. Record the hostname and hardware specs
2. Run all standard models (tinyllama, qwen2.5-coder:3b, llama3.1:8b, qwen2.5-coder:7b)
3. Add results to `ollama-tuning.md` benchmark table
4. Include a creative element (haiku, limerick, shanty) about the hardware

---

## Documentation Style

- Keep documentation practical and copy-paste friendly
- Include actual commands, not abstract descriptions
- Add troubleshooting sections for common issues
