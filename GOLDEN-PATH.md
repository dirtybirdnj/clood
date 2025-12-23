# Golden Path - Local LLM Self-Sufficiency

**Goal:** Get clood to feature parity for Claude-free local work

---

## Phase 1: Fix Infrastructure (You Run)

```bash
# 1. Fix Ollama models path
sudo systemctl stop ollama
sudo mv /home/ollama-models/models/* /home/ollama-models/
sudo rmdir /home/ollama-models/models
sudo rm -f /home/ollama-models/id_ed25519*
sudo chown -R ollama:ollama /home/ollama-models
sudo systemctl start ollama
ollama list

# 2. Pull tool-capable model
ollama pull llama3-groq-tool-use:8b

# 3. Fix CPU governor
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

---

## Phase 2: Test MCP Tools

In clood, select `llama3-groq-tool-use:8b`, then test:

```
List files in /home/mgilbert/Code/clood
```

If it works → MCP filesystem is functional.

```
Search the web for "ollama vulkan performance"
```

If it works → SearXNG MCP is functional.

---

## Phase 3: Multi-Machine Setup

### ubuntu25 (already done)
- Ollama exposed: `OLLAMA_HOST=0.0.0.0:11434`
- SearXNG: `http://192.168.4.62:8888`

### MacBook Air / Mac Mini
1. Install Ollama + clood
2. Copy clood.json, change `base_url` to ubuntu25:
```json
"base_url": "http://192.168.4.62:11434/v1/"
```

---

## Phase 4: Local Workflow (No More Claude)

### For Coding Tasks
1. Open clood with `llama3-groq-tool-use:8b`
2. Use MCP to read files, search web
3. Generate code, copy to editor
4. Run tests locally: `pytest` / `cargo test` / `npm test`

### For PRs
```bash
git checkout -b feature/my-change
# make changes
git add . && git commit -m "description"
git push -u origin feature/my-change
gh pr create --title "..." --body "..."
# HUMAN REVIEWS AND MERGES
```

---

## What's Working Now

- [x] Ollama + GPU (Vulkan on ubuntu25)
- [x] clood configured with MCP servers
- [x] SearXNG running (192.168.4.63:8888)
- [x] Documentation consolidated
- [x] BIOS tuning guide created
- [x] Multi-machine IPs documented

## What Needs Testing/Verification

- [ ] MCP filesystem in clood
- [ ] MCP searxng in clood
- [ ] MCP github in clood
- [ ] llama3-groq-tool-use:8b tool calling
- [ ] Post-BIOS benchmark (after applying settings)

## What Can Wait

- [ ] Multi-machine coordination
- [ ] Tiered routing prompts
- [ ] Embeddings/RAG
- [ ] Auto-manifest generation

---

## Recommended Tools (Research Summary)

| Tool | Purpose | Install |
|------|---------|---------|
| **Aider** | Terminal coding agent + Git | `pip install aider-chat` |
| **LiteLLM** | Multi-model routing | `pip install litellm` |
| **Continue.dev** | VSCode extension | VS Code marketplace |

### Aider (Primary Recommendation)
```bash
pip install aider-chat
cd ~/Code/project
aider --model ollama/llama3-groq-tool-use:8b
```
- 84.9% benchmark accuracy
- Native Git integration
- Works offline with Ollama
- Human approves before commit

### LiteLLM (For Multi-Machine)
```bash
pip install litellm
litellm --model ollama/llama3-groq-tool-use:8b --port 4000
```
- Route queries to different machines/models
- Self-hosted proxy
- Implements your tiered strategy

### Multi-Machine Distribution
- **OLOL** (`pip install olol`) - Ollama load balancer
- **Hive** - Distributed inference framework
- Both are early-stage but functional

---

## Commands Cheat Sheet

```bash
# Check Ollama
ollama list
curl http://localhost:11434/api/tags

# Check SearXNG
curl "http://localhost:8888/search?q=test&format=json" | head

# Check clood config
cat ~/.config/clood/clood.json | head -20

# Test model
ollama run llama3-groq-tool-use:8b "Hello"
```
