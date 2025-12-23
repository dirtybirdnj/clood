# Task: Create `clood-ask` Wrapper Script

## Context
I am working with a local LLM infrastructure called `clood` (repo: dirtybirdnj/clood). It uses a "Tiered Intelligence" architecture:
1.  **Tier 1 (Router):** Fast models (TinyLlama/Llama3-8B) via `mods`.
2.  **Tier 2/3 (Agent):** Complex reasoning models (Llama3-Groq-Tool-Use) via `clood` (CLI tool by CharmBracelet) which supports MCP (Filesystem, Search).

## Goal
I need a bash script called `clood-ask` that mimics the experience of `claude-cli`. It should act as a smart router that transparently switches between a fast pipe-based tool (`mods`) and a heavy agentic tool (`clood`) depending on the complexity of the user's request.

## Requirements

### 1. The Router (Tier 1)
- When I run `clood-ask "my query"`, the script should first pipe the query to `mods` (using a fast model like `tinyllama` or `llama3:8b`) with a system prompt to classify the intent.
- **Classification Categories:**
    - `SIMPLE`: General questions, shell one-liners, git commit messages.
    - `COMPLEX`: Multi-step reasoning, coding tasks requiring file access, or web search.

### 2. Execution Logic
- **If SIMPLE:** Pipe the original query to `mods` immediately for a fast response.
- **If COMPLEX:** Launch `clood` in interactive mode.

### 3. Context Injection (Project Awareness)
- The script must check the current directory for context files (specifically `projects_manifest.json`, `README.md`, or `package.json`).
- If found, read the content of these files and inject them as a "System Prompt" or initial context.
- **For `mods`:** Pass it via stdin or arguments.
- **For `clood`:** Pass it as the initial prompt context so the agent knows "I am working in the `auth-service` repository" immediately.

### 4. Interface Unification
- Support piping: `cat error.log | clood-ask "fix this"` (Should default to `mods` unless specified otherwise).
- Support arguments: `clood-ask "Refactor main.py"` (Should trigger `clood`).
- Support REPL: `clood-ask` (No args) should launch `clood` directly.

## Output
Please write the complete `clood-ask` bash script. Ensure it handles:
- Dependency checks (is `mods` and `clood` installed?).
- Safe string handling for the prompts.
- Colored output (optional but nice) to indicate which "Brain" is being used (e.g., "⚡ Tier 1: Speed Mode" vs "🧠 Tier 3: Agent Mode").