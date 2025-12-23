# LAST SESSION - Crush Meets the Conductor

**Date:** December 23, 2025 (late night)
**Session:** Configuring Crush to Talk to Ubuntu25's Conductor
**Status:** Almost ready to test - clood MCP server now has conductor tool

---

## THE GOAL

Configure Crush (Charm's CLI chat interface) on mac-laptop to communicate with ubuntu25's conductor LLM and have it create HTML files through the agentic interface.

---

## WHAT WE DID

### 1. Fixed Crush Configuration

**Problem:** `crush.json` had wrong IP for ubuntu25 (`192.168.4.63` instead of `192.168.4.64`)

**Fixed:** `~/.config/crush/crush.json`
```json
"ubuntu25": {
  "name": "ubuntu25 (RX 590) - Conductor",
  "base_url": "http://192.168.4.64:11434/v1/",
  "type": "openai",
  "api_key": "ollama",
  "supports_tools": true,
  "models": [
    {
      "name": "🎭 Conductor (Tool Use 8B)",
      "id": "llama3-groq-tool-use:8b",
      "context_window": 8192,
      "default_max_tokens": 4096,
      "supports_tools": true
    },
    ...
  ]
}
```

### 2. Added `clood_conductor` MCP Tool

**File:** `clood-cli/internal/mcp/server.go`

Added a new tool that invokes the orchestrator on ubuntu25 via SSH:

```go
func (s *Server) conductorTool() mcp.Tool {
    return mcp.NewTool("clood_conductor",
        mcp.WithDescription(`🎭 Invoke the Conductor agent on ubuntu25 to create files...`),
        mcp.WithString("task", mcp.Required(), mcp.Description("The task for the conductor to perform")),
        mcp.WithString("conductor_model", mcp.Description("Conductor model (default: llama3-groq-tool-use:8b)")),
        mcp.WithNumber("max_iterations", mcp.Description("Max agent iterations (default: 10)")),
    )
}
```

The handler SSHs to ubuntu25 and runs:
```bash
cd /data/repos/workspace && python3 orchestrator.py --conductor MODEL --max-iterations N "TASK"
```

### 3. Built Successfully

```bash
cd ~/Code/clood/clood-cli && go build -o clood ./cmd/clood
```

Build succeeded - clood CLI now has the conductor tool.

---

## ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────────┐
│  MAC LAPTOP                                                      │
│  ┌─────────────┐         ┌──────────────────────────────────┐   │
│  │   Crush     │ ──MCP──▶│  clood MCP Server                │   │
│  │  (UI Chat)  │         │  (localhost:8765)                │   │
│  └─────────────┘         │  - clood_conductor tool          │   │
│                          └───────────────┬──────────────────┘   │
│                                          │ SSH                   │
│  ┌─────────────┐         ┌───────────────▼──────────────────┐   │
│  │  Ollama     │◀────────│  Ubuntu25: Conductor             │   │
│  │  32B models │ delegate│  (llama3-groq-tool-use:8b)       │   │
│  │  (the beef) │─────────▶  - Orchestrates tasks            │   │
│  └─────────────┘         │  - Writes to /workspace          │   │
│                          └──────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## TO TEST (NEXT SESSION)

### Step 1: Start the clood MCP server

```bash
# In one terminal
~/Code/clood/clood-cli/clood mcp
```

This starts the MCP server on localhost:8765.

### Step 2: Start Crush

```bash
# In another terminal
crush
```

Crush should see the clood MCP tools including `clood_conductor`.

### Step 3: Test the conductor

In Crush, try:
```
Create a simple hello world HTML file using the conductor on ubuntu25
```

Crush should use the `clood_conductor` tool which SSHs to ubuntu25 and runs the orchestrator.

### Alternative: Test directly via CLI

```bash
# Test the orchestrator directly
ssh ubuntu25 "cd /data/repos/workspace && python3 orchestrator.py 'Create a hello.html with nice CSS'"

# Check result
ssh ubuntu25 "cat /data/repos/workspace/hello.html"
```

---

## KEY FILES

| File | Purpose |
|------|---------|
| `~/.config/crush/crush.json` | Crush config (ubuntu25 IP fixed) |
| `~/.config/crush/mcp.json` | Crush MCP server config (clood on 8765) |
| `clood-cli/internal/mcp/server.go` | MCP server with conductor tool |
| `scripts/orchestrator.py` | The agentic conductor on ubuntu25 |

---

## VERIFIED CONNECTIVITY

| What | Status |
|------|--------|
| SSH to ubuntu25 | Working |
| Ollama on ubuntu25 (localhost) | Working |
| Ollama on ubuntu25 (network) | Working at 192.168.4.64:11434 |
| llama3-groq-tool-use:8b on ubuntu25 | Available |
| clood CLI build | Successful |

---

## REMAINING TODOS

1. [x] Fix crush.json with correct ubuntu25 IP
2. [x] Add clood_conductor tool to MCP server
3. [x] Build clood CLI
4. [ ] Start clood MCP server and test
5. [ ] Test crush -> conductor -> file creation flow

---

## RESUME PROMPTS

**To test immediately:**
> Start the clood MCP server and use crush to create an HTML file via the conductor

**To verify conductor works:**
> SSH to ubuntu25 and run the orchestrator directly to create a test file

**To see if crush sees the conductor:**
> Start crush and check if clood_conductor tool is available

---

```
Conductor awaits—
MCP bridge is complete,
test at morning light.
```

---

*The plumbing is done. Tomorrow we make music.*
