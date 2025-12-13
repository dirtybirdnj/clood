# Next Session

## Completed This Session
- [x] BIOS tuned (4.8GHz confirmed)
- [x] MacBook Air Ollama exposed (0.0.0.0:11434)
- [x] llama3-groq-tool-use:8b pulled on MacBook Air
- [x] LiteLLM config updated with all 3 backends
- [x] MCP deps verified on ubuntu25 (node, gh, searxng)
- [x] Reviewed commit 4440ef7 (Split Brain planning scripts)
- [x] Created clood-common.sh shared config
- [x] Fixed start-cpu-services.sh (error handling, timeouts, portability)
- [x] Implemented clood-flow.sh orchestrator

## Split Brain Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLOOD SPLIT BRAIN                       │
├─────────────────────────────────────────────────────────────┤
│  TIER 1 (CPU:11435)     │  TIER 2 (GPU:11434)              │
│  - tinyllama (router)   │  - llama3-groq-tool-use:8b       │
│  - nomic-embed-text     │  - qwen2.5-coder:7b/14b          │
│  - qwen2.5:0.5b         │  - deepseek-r1:8b                │
├─────────────────────────────────────────────────────────────┤
│  SIDECAR SERVICES                                           │
│  - Qdrant (6333) - Vector DB for RAG                       │
│  - LiteLLM (4000) - Multi-backend proxy                    │
└─────────────────────────────────────────────────────────────┘
```

## Scripts Overview

| Script | Purpose |
|--------|---------|
| `clood-common.sh` | Shared config (ports, models, colors, helpers) |
| `start-cpu-services.sh` | Spin up CPU Ollama + Qdrant sidecar |
| `clood-flow.sh` | Orchestrate tiers, route queries |
| `verify-setup.sh` | Health check all services |

## Remaining Tasks

### 1. Mac Mini Ollama Setup
Run this on the Mac Mini (192.168.4.41) to expose Ollama for the LiteLLM hub.

```bash
# 1. Kill existing Ollama
pkill ollama

# 2. Start with network access
OLLAMA_HOST=0.0.0.0:11434 nohup ollama serve > /tmp/ollama.log 2>&1 &

# 3. Make permanent (survives reboot)
launchctl setenv OLLAMA_HOST "0.0.0.0:11434"

# 4. Verify it's listening on all interfaces
lsof -i :11434
# Should show: TCP *:11434 (LISTEN)
```

### 2. Test Split Brain Locally
```bash
# Start CPU services
./scripts/start-cpu-services.sh

# Verify setup
./scripts/verify-setup.sh

# Test the flow
./scripts/clood-flow.sh "What is 2+2?"           # Should route to CPU
./scripts/clood-flow.sh "Refactor this function to use async/await and add error handling"  # Should route to GPU
```

### 3. Start LiteLLM on ubuntu25
```bash
ssh ubuntu25
~/Code/clood/scripts/start-litellm.sh
curl http://localhost:4000/models
```

### 4. Test MCP Tools in Crush
On ubuntu25, open Crush and test:
- "List files in /home/mgilbert/Code/clood"
- "Search for ollama vulkan performance"
- "List my github repos"

### 5. End-to-End PR Test
- Open clood repo in Crush
- "Read README and suggest improvement"
- "Create branch, make change, open PR"

---
*Generated 2025-12-13*
