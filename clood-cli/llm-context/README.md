# LLM Context Files for clood

This folder contains documentation artifacts designed to help local LLM models (Qwen, DeepSeek, etc.) understand the clood codebase without scanning all source files.

## For Claude Code Users

**Claude should ignore this folder unless explicitly asked.** These files are optimized for local LLMs with limited context windows.

## Files in This Folder

| File | Purpose |
|------|---------|
| `CODEBASE.md` | Project overview, directory structure, tech stack |
| `API.md` | Key functions, types, and interfaces |
| `ARCHITECTURE.md` | System design, data flow, module relationships |

## How to Use with Local LLMs

### Basic Context Loading

Copy the relevant file(s) into your prompt:

```
<context>
[Contents of CODEBASE.md]
</context>

Your question here...
```

### Task-Specific Context

| Task Type | Files to Include |
|-----------|-----------------|
| Understanding the project | `CODEBASE.md` |
| Adding new commands | `CODEBASE.md` + `API.md` |
| Modifying routing logic | `API.md` + `ARCHITECTURE.md` |
| Adding new hosts | `API.md` (hosts section) |

## Context Size Estimates

| File | Approximate Tokens |
|------|-------------------|
| `CODEBASE.md` | ~1,500 |
| `API.md` | ~2,500 |
| `ARCHITECTURE.md` | ~1,000 |
| **Total** | **~5,000** |

All files combined fit comfortably in most local LLM context windows (8K-32K tokens).
