# Multi-Machine Setup

Run models across your server garden from any workstation.

---

## The Vision

```
┌─────────────────────────────────────────────────────────────┐
│                    YOUR WORKSTATION                         │
│  (ubuntu25 / MacBook Air / Mac Mini - wherever you are)     │
│                                                             │
│  ┌─────────────┐                                            │
│  │   crush     │──┐                                         │
│  │   aider     │  │                                         │
│  │   chain.sh  │  │                                         │
│  └─────────────┘  │                                         │
│                   ▼                                         │
│         ┌─────────────────┐                                 │
│         │  LiteLLM Proxy  │  ← Single endpoint              │
│         │  localhost:4000 │    for ALL models               │
│         └────────┬────────┘                                 │
└──────────────────┼──────────────────────────────────────────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │ubuntu25 │ │MacBook  │ │Mac Mini │
   │ Ollama  │ │ Ollama  │ │ Ollama  │
   │ :11434  │ │ :11434  │ │ :11434  │
   │         │ │         │ │         │
   │ RX 590  │ │ M4 16GB │ │ M4 24GB │
   │ 8GB     │ │ unified │ │ unified │
   └─────────┘ └─────────┘ └─────────┘
```

---

## Prerequisites

### Each Machine Needs:
1. Ollama installed
2. Ollama exposed to network
3. Models pulled appropriate for that machine's capability

### Network:
- All machines on same network (or VPN)
- Ports open: 11434 (Ollama), 4000 (LiteLLM)

---

## Step 1: Expose Ollama on Each Machine

### Ubuntu/Linux
```bash
# Create override file
sudo mkdir -p /etc/systemd/system/ollama.service.d/
sudo tee /etc/systemd/system/ollama.service.d/override.conf << 'EOF'
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
EOF

sudo systemctl daemon-reload
sudo systemctl restart ollama
```

### macOS
```bash
# Set environment for launchd
launchctl setenv OLLAMA_HOST "0.0.0.0:11434"

# Restart Ollama (quit and reopen, or)
pkill ollama
ollama serve &
```

### Test Remote Access
From another machine:
```bash
curl http://192.168.4.63:11434/api/tags  # ubuntu25
curl http://192.168.x.x:11434/api/tags   # Mac
```

---

## Step 2: Set Up LiteLLM Proxy

### Install
```bash
pip install 'litellm[proxy]'
```

### Configure
Edit `infrastructure/litellm-config.yaml` with your machine IPs:
```yaml
model_list:
  - model_name: "fast/tinyllama"
    litellm_params:
      model: "ollama/tinyllama"
      api_base: "http://192.168.4.63:11434"  # ubuntu25

  - model_name: "code/deepseek"
    litellm_params:
      model: "ollama/deepseek-coder:6.7b"
      api_base: "http://192.168.4.63:11434"  # ubuntu25

  - model_name: "heavy/qwen-14b"
    litellm_params:
      model: "ollama/qwen2.5-coder:14b"
      api_base: "http://192.168.x.x:11434"   # Mac Mini
```

### Run
```bash
litellm --config infrastructure/litellm-config.yaml --port 4000
```

### Test
```bash
curl http://localhost:4000/v1/models
```

---

## Step 3: Model Distribution Strategy

| Machine | VRAM | Models | Role |
|---------|------|--------|------|
| **ubuntu25** | 8GB (RX 590) | tinyllama, llama3.2:3b, deepseek-coder:6.7b | Fast queries, medium code |
| **MacBook Air** | 16GB unified | mistral:7b, qwen2.5-coder:7b | Reasoning, code review |
| **Mac Mini** | 24GB unified | qwen2.5-coder:14b, deepseek-coder:33b | Heavy lifting |

---

## Step 4: Use the Chain Script

### Direct Ollama (single machine)
```bash
./scripts/chain.sh "analyze this codebase for bottlenecks"
```

### With LiteLLM (multi-machine)
```bash
./scripts/chain.sh --litellm http://localhost:4000 \
  --fast "fast/tinyllama" \
  --deep "heavy/qwen-14b" \
  "find security vulnerabilities"
```

### Pipe code into it
```bash
cat src/main.py | ./scripts/chain.sh "review this code"
```

---

## Step 5: Point Aider at LiteLLM

```bash
# Use LiteLLM as OpenAI-compatible API
export OPENAI_API_BASE=http://localhost:4000
export OPENAI_API_KEY=dummy  # LiteLLM doesn't need a real key

# Run Aider with any model from your pool
aider --model "code/deepseek"
```

---

## Workflow Examples

### Analyze a New Codebase
```bash
cd ~/Code/some-project

# Fast triage with TinyLlama, deep dive with DeepSeek
./scripts/chain.sh "what does this codebase do and what are the main components"
```

### Review a PR
```bash
git diff main..feature-branch | ./scripts/chain.sh "review these changes for bugs and style issues"
```

### Research + Apply
```bash
# TinyLlama searches, DeepSeek applies
./scripts/chain.sh --fast tinyllama --deep deepseek-coder:6.7b \
  "find best practices for async Python and show how to apply to this code"
```

---

## Troubleshooting

### Can't connect to remote Ollama
```bash
# Check Ollama is listening on 0.0.0.0
sudo ss -tlnp | grep 11434

# Check firewall
sudo ufw allow 11434
```

### LiteLLM model not found
```bash
# List available models through LiteLLM
curl http://localhost:4000/v1/models | jq '.data[].id'

# Make sure model is actually pulled on target machine
ssh ubuntu25 "ollama list"
```

### Slow cross-machine responses
- Check network speed between machines
- Consider running LiteLLM on the machine with most models
- Use wired ethernet instead of WiFi

---

## Security Notes

- This setup is for **local network only**
- Ollama has no authentication - don't expose to internet
- For remote access, use SSH tunneling or VPN

```bash
# SSH tunnel example
ssh -L 11434:localhost:11434 ubuntu25
# Then use localhost:11434 from your laptop
```
