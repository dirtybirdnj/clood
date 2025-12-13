# Next Session

## Completed This Session
- [x] BIOS tuned (4.8GHz confirmed)
- [x] MacBook Air Ollama exposed (0.0.0.0:11434)
- [x] llama3-groq-tool-use:8b pulled on MacBook Air
- [x] LiteLLM config updated with all 3 backends
- [x] MCP deps verified on ubuntu25 (node, gh, searxng)

## Remaining Tasks

### 1. Mac Mini Ollama Setup
Run this on the Mac Mini (192.168.4.41) to expose Ollama for the LiteLLM hub.

## Quick Setup

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

# 5. Check models
ollama list
```

## Models to Pull (if not present)

```bash
ollama pull qwen2.5-coder:7b    # 4.7GB - medium coding
ollama pull qwen2.5-coder:14b   # 9GB - large coding (if RAM permits)
```

## Test from ubuntu25

After setup, test from ubuntu25:
```bash
curl http://192.168.4.41:11434/api/tags
```

## Context

This is part of the multi-machine LiteLLM hub setup:
- **ubuntu25** runs LiteLLM proxy on port 4000
- **MacBook Air** (192.168.4.47) - Ollama exposed (DONE)
- **Mac Mini** (192.168.4.41) - Ollama needs exposure (THIS TASK)

LiteLLM config updated at: `/infrastructure/litellm-config.yaml`

## After This

Once Mac Mini is exposed, return to ubuntu25 and:
1. Start LiteLLM: `~/Code/clood/scripts/start-litellm.sh`
2. Test: `curl http://localhost:4000/models`
3. Configure Crush to point to LiteLLM hub

### 2. Start LiteLLM on ubuntu25
```bash
ssh ubuntu25
~/Code/clood/scripts/start-litellm.sh
curl http://localhost:4000/models
```

### 3. Test MCP Tools in Crush
On ubuntu25, open Crush and test:
- "List files in /home/mgilbert/Code/clood"
- "Search for ollama vulkan performance"
- "List my github repos"

### 4. End-to-End PR Test
- Open clood repo in Crush
- "Read README and suggest improvement"
- "Create branch, make change, open PR"

---
*Generated 2025-12-12*
