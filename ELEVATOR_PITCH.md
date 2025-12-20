# clood - Elevator Pitch

## The One-Liner

**clood** is a local-first LLM orchestration layer that routes queries across multiple Ollama instances, enabling fast/cheap/private AI workflows without cloud dependencies.

---

## The Problem

You have Ollama running on your laptop. Maybe a Mac Mini. Maybe a Linux box with a GPU. Each has different models. You want to:

- Ask quick questions → fast small model
- Get code review → capable coding model
- Run complex reasoning → large reasoning model
- Compare outputs → multiple models in parallel

Currently, you SSH between machines, remember which models are where, and manually route everything. It's tedious.

---

## The Solution

```bash
# One command, automatic routing
clood ask "quick question"           # → routes to fastest available model
clood ask "review this code" -f x.go # → routes to coding model
clood catfight "compare approaches"  # → runs on multiple models, shows winner
```

clood handles:
- **Multi-host routing** - finds models across your network
- **Tiered inference** - fast/deep/reasoning model selection
- **Parallel execution** - catfight runs same prompt on multiple models
- **Zero cloud** - everything stays local, no API costs, works offline

---

## The Technical Stack

- **Go CLI** - single binary, no dependencies
- **Ollama backend** - any model Ollama supports
- **MCP integration** - works as Claude Code tool provider
- **Multi-platform** - macOS, Linux, Windows

---

## Who It's For

- **Developers** with multiple machines running Ollama
- **Privacy-conscious** teams who can't send code to cloud APIs
- **Cost-conscious** users tired of API bills
- **Hobbyists** with home labs wanting to maximize their hardware

---

## What Makes It Different

1. **Multi-host by default** - not an afterthought, the core design
2. **Catfight pattern** - built-in model comparison for confidence
3. **Claude Code native** - MCP server integration for seamless use
4. **Actually documented** - quickstart to running in 5 minutes

---

## The Demo

```bash
# Install
brew install dirtybirdnj/tap/clood

# See what's available
clood health

# Ask across your network
clood ask "explain dependency injection"

# Compare models
clood catfight "write fizzbuzz in Go"
```

---

## The Numbers

- **215+ commits** in active development
- **3 host routing** tested (laptop, mac-mini, linux GPU box)
- **Sub-second** model discovery across network
- **191 tok/s** achieved on consumer hardware

---

## The Vision

Local LLMs are getting good. Hardware is getting cheap. The missing piece is *infrastructure* - the glue that makes multiple models on multiple machines feel like one coherent system.

clood is that glue.

---

## Links

- **GitHub:** github.com/dirtybirdnj/clood
- **Install:** `brew install dirtybirdnj/tap/clood`
- **Quickstart:** See QUICKSTART.md (coming soon)

---

*clood = claude + collude. Local-first AI infrastructure for people who ship.*
