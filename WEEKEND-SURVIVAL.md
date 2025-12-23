# Weekend Survival Guide

No Claude until Monday? No problem. Everything you need is running.

---

## TL;DR - You're Ready

```
ubuntu25 (192.168.4.63)
├── Ollama ✅ port 11434
├── SearXNG ✅ port 8888
└── llama3-groq-tool-use:8b ✅ (tool calling!)
```

---

## Quick Test (Run This First)

From any machine on your network:

```bash
# 1. Ollama alive?
curl -s http://192.168.4.63:11434/api/tags | jq '.models[].name'

# 2. SearXNG alive?
curl -s "http://192.168.4.63:8888/search?q=hello&format=json" | jq '.results[0].title'

# 3. Generate text?
curl -s http://192.168.4.63:11434/api/generate -d '{
  "model": "llama3-groq-tool-use:8b",
  "prompt": "Hello!",
  "stream": false
}' | jq '.response'
```

If all three work → you're golden.

---

## Workflow 1: clood + MCP (Recommended)

### On ubuntu25

```bash
# Start clood
clood

# Select: llama3-groq-tool-use:8b

# Test filesystem MCP:
> List the files in /home/mgilbert/Code/clood

# Test web search MCP:
> Search the web for "python async tutorial"
```

**If tools work:** You can read files, search the web, and generate code.

**If tools don't work:** The model will try to respond without calling tools. Check:
- Is the MCP server configured in `~/.config/clood/clood.json`?
- Try: `cat ~/.config/clood/clood.json | grep mcp`

### From Mac (Remote to ubuntu25)

Option A: SSH and run clood on ubuntu25
```bash
ssh mgilbert@192.168.4.63
clood
```

Option B: Configure clood on Mac to use ubuntu25's Ollama
```bash
mkdir -p ~/.config/clood
cat > ~/.config/clood/clood.json << 'EOF'
{
  "providers": {
    "ubuntu25": {
      "type": "openai",
      "base_url": "http://192.168.4.63:11434/v1/",
      "api_key": "ollama",
      "models": ["llama3-groq-tool-use:8b", "qwen2.5-coder:7b", "mistral:7b"]
    }
  }
}
EOF
clood
```

---

## Workflow 2: Aider (Git-Integrated Coding)

### Install (one time)
```bash
pip install aider-chat
```

### Use
```bash
cd ~/Code/your-project
aider --model ollama/llama3-groq-tool-use:8b --ollama-api-base http://192.168.4.63:11434
```

**What Aider does:**
- Reads your codebase
- You describe changes in natural language
- It edits files directly
- You approve before git commit

### Example Session
```
$ aider --model ollama/llama3-groq-tool-use:8b
> Add a function to calculate fibonacci numbers to utils.py
> /diff  (see what it wants to change)
> /yes   (approve the change)
> /commit "Add fibonacci function"
```

---

## Workflow 3: Direct API (Simple)

For quick one-off generations without tools:

```bash
# Simple prompt
curl -s http://192.168.4.63:11434/api/generate -d '{
  "model": "qwen2.5-coder:7b",
  "prompt": "Write a Python function to merge two sorted lists",
  "stream": false
}' | jq -r '.response'
```

---

## Model Selection

| Task | Model | Speed |
|------|-------|-------|
| **Tool calling / MCP** | `llama3-groq-tool-use:8b` | ~30 tok/s |
| **Code generation** | `qwen2.5-coder:7b` | ~32 tok/s |
| **Quick questions** | `mistral:7b` | ~32 tok/s |
| **Fast/simple** | `llama3.2:3b` | ~60 tok/s |

---

## Troubleshooting

### "Connection refused" to 192.168.4.63

ubuntu25 might be asleep or Ollama stopped:
```bash
# SSH in and restart
ssh mgilbert@192.168.4.63
sudo systemctl start ollama
sudo systemctl status ollama
```

### Ollama running but slow

CPU governor might be in powersave:
```bash
ssh mgilbert@192.168.4.63
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### SearXNG not responding

```bash
ssh mgilbert@192.168.4.63
cd ~/Code/clood/infrastructure
docker compose up -d searxng
docker compose logs searxng
```

### clood can't find models

Make sure the config points to ubuntu25:
```bash
cat ~/.config/clood/clood.json
# Should have: "base_url": "http://192.168.4.63:11434/v1/"
```

### Model generates garbage / doesn't follow instructions

Try a different model. `llama3-groq-tool-use:8b` is best for tools, but `mistral:7b` or `qwen2.5-coder:7b` might work better for pure code gen.

---

## Network Reference

| Machine | IP | Services |
|---------|-----|----------|
| ubuntu25 | 192.168.4.63 | Ollama :11434, SearXNG :8888 |
| MacBook Air | 192.168.4.47 | Ollama :11434 (local) |
| Mac Mini | 192.168.4.41 | Ollama :11434 (local) |

---

## Emergency: Nothing Works

1. **Check ubuntu25 is on:** Can you ping 192.168.4.63?
2. **Check services:** SSH in, run `sudo systemctl status ollama`
3. **Restart everything:**
   ```bash
   ssh mgilbert@192.168.4.63
   sudo systemctl restart ollama
   cd ~/Code/clood/infrastructure && docker compose restart
   ```

---

## Weekend Mantra

```
I don't need Claude for everything.
Local models can:
- Generate code (qwen2.5-coder)
- Use tools (llama3-groq-tool-use)
- Search the web (via SearXNG)
- Read my files (via MCP)

I approve all changes before commit.
I am in control.
```

---

## Quick Reference Card

```bash
# Test ubuntu25
curl http://192.168.4.63:11434/api/tags

# Start clood (on ubuntu25)
ssh mgilbert@192.168.4.63 -t "clood"

# Aider (from any machine)
aider --model ollama/llama3-groq-tool-use:8b --ollama-api-base http://192.168.4.63:11434

# Generate code directly
curl -s http://192.168.4.63:11434/api/generate -d '{"model":"qwen2.5-coder:7b","prompt":"YOUR PROMPT","stream":false}' | jq -r '.response'
```

Good luck! You've got this.
