# LLM Inception: Molecules on Snake Road

## The Atom

We built the **atomic interaction**: one LLM synchronously queries another mid-stream.

```
Main LLM (streaming) → detects <sub-query> → PAUSE → Expert LLM → response → RESUME
```

Files:
- `internal/inception/inception.go` - Core engine
- `internal/commands/inception.go` - TUI command (`clood inception`)
- `internal/mcp/server.go` - MCP tool (`clood_inception`)

## The Molecules

Molecules are compositions of atomic interactions through different pathways.

### Molecule 1: Direct CLI (Working)

```
Human → clood inception → Main LLM → inception engine → Expert LLM → Human
```

**Test:**
```bash
clood inception --model qwen2.5-coder:7b
# Prompt: "Write code to calculate escape velocity. Ask science for the formula."
```

### Molecule 2: MCP via any-cli-mcp-server (Testable)

```
Claude Code → any-cli-mcp-server → clood inception → Expert LLM → Claude Code
        │                                                              │
        └──────────────────────────────────────────────────────────────┘
```

**Setup:** Add to Claude Code's MCP config:
```json
{
  "mcpServers": {
    "clood": {
      "command": "npx",
      "args": ["-y", "any-cli-mcp-server", "clood"]
    }
  }
}
```

**Test:** In Claude Code, call `clood inception ...`

### Molecule 3: Native MCP Server (Working)

```
MCP Client → clood serve --sse → clood_inception tool → Expert LLM → MCP Client
```

**Test:**
```bash
# Terminal 1: Start MCP server
clood serve --sse

# Terminal 2: Call inception via curl
curl -X POST http://localhost:8765/tools/clood_inception \
  -H "Content-Type: application/json" \
  -d '{"expert": "science", "query": "What is orbital velocity?"}'
```

### Molecule 4: clood Integration (Limited)

⚠️ **Known Issue:** clood's `AllowedMCP` filter blocks MCP tools from reaching the LLM.
See commit `d33975d` - "clood tool-calling bug documented"

**Workaround Options:**
1. Use `tool-proxy.py` to inject tools into Ollama requests
2. Use any-cli-mcp-server to wrap clood
3. Build in clood (Snake Road with inception - what we did!)

```
clood → tool-proxy.py → Ollama → (tools injected) → tool call → clood → Expert
```

### Molecule 5: Multi-Expert Chain (Future)

```
Coder LLM ─┬→ Science Expert ─→ response ─┐
           │                              │
           ├→ Math Expert ────→ response ─┼→ Coder continues
           │                              │
           └→ Code Expert ────→ response ─┘
```

This requires depth > 1 and parallel sub-queries. Currently limited to depth=1.

## Testing Matrix

| Molecule | Path | Status | Test Command |
|----------|------|--------|--------------|
| 1 | Direct CLI | ✅ Working | `clood inception` |
| 2 | any-cli-mcp | 🔄 Testable | `npx any-cli-mcp-server clood` |
| 3 | Native MCP | ✅ Working | `clood serve --sse` |
| 4 | clood | ⚠️ Blocked | See workarounds |
| 5 | Multi-Expert | 📋 Future | Jelly Bean #150 |

## The Scrolls Remind the Scrolls

From git history (xbibit postulates):

**Commit c093608** - tool-proxy.py:
> Proxy works - model calls tools
> clood ignores - the loop stays broken
> Forty lines wait

**Commit d33975d** - Bug documented:
> The `AllowedMCP` filter in `buildTools()` blocks them.

**Commit daaae73** - any-cli-mcp-server revelation:
> any-cli-mcp-server sees all, Twenty-two new tools.

## Quick Test Script

```bash
#!/bin/bash
# test_molecules.sh

echo "=== Molecule 1: Direct CLI ==="
echo "Write a function to calculate escape velocity" | timeout 60 clood inception --model qwen2.5-coder:3b

echo ""
echo "=== Molecule 3: MCP Server ==="
# Start server in background
clood serve --sse &
sleep 2

# Test inception tool
curl -s -X POST http://localhost:8765/tools/clood_inception \
  -H "Content-Type: application/json" \
  -d '{"expert": "science", "query": "What is the gravitational constant G?"}'

# Cleanup
pkill -f "clood serve"

echo ""
echo "=== Done ==="
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           MOLECULE PATHWAYS                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Human ───────────────────────────────────────────────────────────┐        │
│    │                                                               │        │
│    ├──→ clood inception (TUI) ──→ inception engine ──→ Expert     │        │
│    │                                      │                        │        │
│    │                               ┌──────┴──────┐                 │        │
│    │                               │  Ollama API │                 │        │
│    │                               └──────┬──────┘                 │        │
│    │                                      │                        │        │
│    ├──→ Claude Code ──→ any-cli-mcp ──→ clood ──→ inception ──→ Expert    │
│    │                                                               │        │
│    ├──→ MCP Client ──→ clood serve (SSE) ──→ clood_inception ──→ Expert   │
│    │                                                               │        │
│    └──→ clood ──→ (blocked by AllowedMCP) ──→ ❌                   │        │
│              │                                                     │        │
│              └──→ tool-proxy.py ──→ Ollama ──→ inception ──→ Expert        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Next Steps

1. **Test Molecule 1** - Run `clood inception` interactively
2. **Test Molecule 3** - Start MCP server, call via curl
3. **Investigate clood** - Can we patch AllowedMCP or use tool-proxy?
4. **Document Results** - Update this file with findings
5. **Bean #150** - Track multi-expert chains as future work
